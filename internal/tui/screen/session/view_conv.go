package session

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	sessiondata "github.com/vitaliiPsl/crappy-ai/internal/session"
)

const (
	convMessagePadding = 2
	convMaxResultLen   = 120
	convToolLines      = 5
	convToolIndent     = "  "

	convSystemPad     = " "
	convSystemDivider = "-"

	convCompactingText = "Compacting context..."
	convSystemLabel    = "System"
)

type ConvOpts struct {
	Width          int
	ShowThinking   bool
	ShowToolResult bool
}

func renderConversation(s *State, opts ConvOpts) string {
	if isConversationEmpty(s) {
		return ""
	}

	var b strings.Builder

	for i, msg := range s.Messages {
		if i > 0 && s.Messages[i-1].Role != msg.Role {
			b.WriteByte('\n')
		}

		b.WriteString(renderMessage(&msg, opts))
		b.WriteByte('\n')
	}

	if s.Phase == PhaseCompacting {
		if len(s.Messages) > 0 {
			b.WriteByte('\n')
		}

		b.WriteString(renderSystemBlock(convSystemLabel, convCompactingText, opts.Width))
		b.WriteByte('\n')
	}

	if s.Streaming != nil && (s.Phase == PhaseRunning || s.HasDraft()) {
		b.WriteByte('\n')
		b.WriteString(renderAssistantMessage(s.Streaming, opts))
		b.WriteByte('\n')
	}

	for _, queued := range s.Pending {
		b.WriteByte('\n')
		b.WriteString(renderQueuedRequest(queued.Request, opts.Width))
		b.WriteByte('\n')
	}

	return b.String()
}

func renderQueuedRequest(req sessiondata.Request, width int) string {
	text := req.Text
	if req.Skill != nil {
		text = skillInvocationText(*req.Skill)
	}

	if req.MCPPrompt != nil {
		text = mcpPromptInvocationText(*req.MCPPrompt)
	}

	return queuedMessageStyle.Width(width).Render("\n" + subtleTextStyle.Render(text) + "\n")
}

func isConversationEmpty(s *State) bool {
	return len(s.Messages) == 0 && s.Streaming == nil && s.Phase == PhaseIdle
}

func renderMessage(msg *Message, opts ConvOpts) string {
	if msg.Error != "" {
		return errorStyle.Render("Error: " + msg.Error)
	}

	switch msg.Role {
	case RoleUser:
		return renderUserMessage(msg, opts.Width)
	case RoleModel:
		return renderAssistantMessage(msg, opts)
	case RoleSystem:
		return renderSystemBlock(convSystemLabel, msg.Text, opts.Width)
	case RoleTool:
		return subtleTextStyle.Render(msg.Text)
	}

	return ""
}

func renderUserMessage(msg *Message, width int) string {
	return userMessageStyle.Width(width).Render("\n" + textStyle.Render(msg.Text) + "\n")
}

func renderAssistantMessage(msg *Message, opts ConvOpts) string {
	var b strings.Builder

	contentWidth := max(0, opts.Width-convMessagePadding)

	if msg.Thinking != "" {
		b.WriteString(renderThinking(strings.TrimSpace(msg.Thinking), opts.ShowThinking, contentWidth))
		b.WriteString("\n\n")
	}

	if msg.Text != "" {
		b.WriteString(textStyle.Width(contentWidth).Render(msg.Text))
		b.WriteByte('\n')
	}

	for _, tool := range msg.Tools {
		b.WriteString(renderTool(&tool, opts.ShowToolResult))
	}

	return assistantMessageStyle.Render(b.String())
}

func renderSystemBlock(name, text string, width int) string {
	label := convSystemPad + name + convSystemPad
	lineLen := max((width-len(label))/2, 1)
	line := strings.Repeat(convSystemDivider, lineLen)
	header := systemStyle.Render(line + label + line)

	if text == "" {
		return header
	}

	return header + "\n" + thinkingStyle.Render(text)
}

func renderTool(tool *ToolUse, showResult bool) string {
	if tool.Name == planToolName {
		if rendered := renderPlanTool(tool); rendered != "" {
			return rendered
		}
	}

	var b strings.Builder

	head := toolNameStyle.Render(tool.Name)
	if arg := toolInlineArg(tool); arg != "" {
		head += "  " + subtleTextStyle.Render(arg)
	}

	b.WriteString(head)

	if desc := toolSecondaryDescription(tool); desc != "" {
		b.WriteString("\n" + subtleTextStyle.Render(desc))
	}

	switch {
	case tool.Error != "":
		b.WriteString("\n" + errorStyle.Render(truncateInline(tool.Error, convMaxResultLen)))
	case showResult && tool.Done && tool.Result != "":
		b.WriteString("\n" + renderToolResult(tool.Result))
	}

	return toolBlockStyle(tool).Render(b.String()) + "\n"
}

func toolBlockStyle(tool *ToolUse) lipgloss.Style {
	switch {
	case tool.Error != "":
		return toolBlockError
	case tool.Done:
		return toolBlockDone
	default:
		return toolBlockPending
	}
}

func toolInlineArg(tool *ToolUse) string {
	if cmd, _ := tool.Arguments["command"].(string); cmd != "" {
		return truncateInline(cmd, convMaxResultLen)
	}

	if path, _ := tool.Arguments["path"].(string); path != "" {
		return path
	}

	if url, _ := tool.Arguments["url"].(string); url != "" {
		return truncateInline(url, convMaxResultLen)
	}

	if desc, _ := tool.Arguments["description"].(string); desc != "" {
		return desc
	}

	if skill, _ := tool.Arguments["skill"].(string); skill != "" {
		return skill
	}

	return ""
}

func toolSecondaryDescription(tool *ToolUse) string {
	desc, _ := tool.Arguments["description"].(string)
	if desc == "" {
		return ""
	}

	cmd, _ := tool.Arguments["command"].(string)
	path, _ := tool.Arguments["path"].(string)
	url, _ := tool.Arguments["url"].(string)

	if cmd == "" && path == "" && url == "" {
		return ""
	}

	return desc
}

func renderToolResult(result string) string {
	lines := strings.SplitN(result, "\n", convToolLines+1)
	if len(lines) > convToolLines {
		lines = append(lines[:convToolLines], ellipsis)
	}

	var b strings.Builder
	for _, line := range lines {
		b.WriteString(convToolIndent + subtleTextStyle.Render(line) + "\n")
	}

	return strings.TrimRight(b.String(), "\n")
}

func renderThinking(thinking string, expanded bool, width int) string {
	header := thinkingHeaderStyle.Render("Thinking · " + thinkingSize(thinking))
	if !expanded {
		return header
	}

	return header + "\n\n" + thinkingStyle.Width(width).Render(thinking)
}

func thinkingSize(s string) string {
	n := len([]rune(s))
	if n >= 1000 {
		return fmt.Sprintf("%.1fk chars", float64(n)/1000)
	}

	return fmt.Sprintf("%d chars", n)
}
