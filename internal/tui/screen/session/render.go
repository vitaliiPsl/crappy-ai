package session

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/vitaliiPsl/crappy-ai/internal/tui/theme"
)

const (
	assistantLabel = "Assistant"
	userLabel      = "You"
	systemLabel    = "System"
	thinkingLabel  = "Thinking"
	emptyPadFactor = 4
	maxResultLen   = 120
	toolLines      = 5
)

var (
	thm = theme.Default

	userLabelStyle      = lipgloss.NewStyle().Foreground(thm.Primary).Bold(true)
	assistantLabelStyle = lipgloss.NewStyle().Foreground(thm.Secondary).Bold(true)
	thinkingLabelStyle  = lipgloss.NewStyle().Foreground(thm.Muted).Italic(true)

	textStyle       = lipgloss.NewStyle().Foreground(thm.Text)
	subtleTextStyle = lipgloss.NewStyle().Foreground(thm.SubtleText)
	thinkingStyle   = lipgloss.NewStyle().Foreground(thm.Muted)
	errorStyle      = lipgloss.NewStyle().Foreground(thm.Error)
	hintsStyle      = lipgloss.NewStyle().Foreground(thm.SubtleText)
	systemStyle     = lipgloss.NewStyle().Foreground(thm.Muted)
	successStyle    = lipgloss.NewStyle().Foreground(thm.Success)
	warningStyle    = lipgloss.NewStyle().Foreground(thm.Warning)

	userMessageStyle = lipgloss.NewStyle().
				Background(thm.SurfaceAlt).
				Padding(0, 1)

	assistantMessageStyle = lipgloss.NewStyle().
				Padding(0, 1)
)

func (conv *conversation) refreshContent() {
	if len(conv.messages) == 0 && !conv.streaming {
		conv.viewport.SetContent(conv.renderEmpty())

		return
	}

	var b strings.Builder

	for i, msg := range conv.messages {
		if i > 0 && conv.messages[i-1].role != msg.role {
			b.WriteByte('\n')
		}

		showLabel := i == 0 || conv.messages[i-1].role != msg.role
		b.WriteString(conv.renderMessage(msg, showLabel))
		b.WriteByte('\n')
	}

	if conv.streaming {
		if len(conv.messages) > 0 && conv.messages[len(conv.messages)-1].role != messageRoleAssistant {
			b.WriteByte('\n')
		}

		lastIsAssistant := len(conv.messages) > 0 && conv.messages[len(conv.messages)-1].role == messageRoleAssistant
		b.WriteString(conv.renderStreaming(!lastIsAssistant))
	}

	conv.viewport.SetContent(b.String())
}

func (conv *conversation) renderEmpty() string {
	var b strings.Builder

	b.WriteString(textStyle.Render("What do you want to understand today?") + "\n\n")

	if conv.model != "" {
		b.WriteString(subtleTextStyle.Render("Model: ") + textStyle.Render(conv.model) + "\n")
	} else if conv.provider != "" {
		b.WriteString(subtleTextStyle.Render("Provider: ") + textStyle.Render(conv.provider) + "\n")
	}

	content := b.String()
	lines := strings.Count(content, "\n")
	pad := max((conv.height-lines-emptyPadFactor)/2, 0)

	return strings.Repeat("\n", pad) +
		lipgloss.NewStyle().Width(conv.width).Align(lipgloss.Center).Render(content)
}

func (conv *conversation) renderMessage(msg chatMessage, showLabel bool) string {
	if msg.error != "" {
		return errorStyle.Render("Error: "+msg.error) + "\n"
	}

	switch msg.role {
	case messageRoleUser:
		return conv.renderUserMessage(msg)
	case messageRoleAssistant:
		return conv.renderAssistantMessage(msg, showLabel)
	case messageRoleSystem:
		return conv.renderSystemMessage(msg)
	case messageRoleTool:
		return subtleTextStyle.Render(msg.text) + "\n"
	default:
		return ""
	}
}

func (conv *conversation) renderUserMessage(msg chatMessage) string {
	content := userLabelStyle.Render(userLabel) + "\n" + textStyle.Render(msg.text)

	return userMessageStyle.Width(conv.width).Render(content)
}

func (conv *conversation) renderAssistantMessage(msg chatMessage, showLabel bool) string {
	var b strings.Builder

	if showLabel {
		b.WriteString(assistantLabelStyle.Render(assistantLabel) + "\n")
	}

	if thinking := collapseBlankLines(msg.thinking); thinking != "" && conv.showThinking {
		b.WriteString(renderThinking(thinking))

		if msg.text != "" || len(msg.tools) > 0 {
			b.WriteByte('\n')
		}
	}

	if msg.text != "" {
		b.WriteString(textStyle.Width(max(0, conv.width-2)).Render(msg.text))
		b.WriteByte('\n')
	}

	for _, tool := range msg.tools {
		b.WriteString(conv.renderTool(tool))
	}

	return renderAssistantBlock(strings.TrimRight(b.String(), "\n"))
}

func (conv *conversation) renderStreaming(showLabel bool) string {
	var b strings.Builder

	thinkingText := collapseBlankLines(conv.streamingThinking)
	showThinking := thinkingText != "" && conv.showThinking

	if showLabel && (showThinking || conv.streamingText != "" || len(conv.streamingTools) > 0) {
		b.WriteString(assistantLabelStyle.Render(assistantLabel) + "\n")
	}

	if showThinking {
		b.WriteString(renderThinking(thinkingText))

		if conv.streamingText != "" || len(conv.streamingTools) > 0 {
			b.WriteByte('\n')
		}
	}

	if conv.streamingText != "" {
		b.WriteString(textStyle.Width(max(0, conv.width-2)).Render(conv.streamingText))
		b.WriteByte('\n')
	}

	for _, tool := range conv.streamingTools {
		b.WriteString(conv.renderTool(tool))
	}

	return renderAssistantBlock(strings.TrimRight(b.String(), "\n"))
}

func (conv *conversation) renderSystemMessage(msg chatMessage) string {
	label := " " + systemLabel + " "
	lineLen := max((conv.width-len(label))/2, 1)
	line := strings.Repeat("-", lineLen)
	header := systemStyle.Render(line + label + line)

	if msg.text == "" {
		return header + "\n"
	}

	return header + "\n" + thinkingStyle.Render(msg.text) + "\n"
}

func (conv *conversation) renderTool(tool toolUse) string {
	icon := warningStyle.Render("o")
	if tool.Done {
		icon = successStyle.Render("x")
	}

	if tool.Error != "" {
		icon = errorStyle.Render("!")
	}

	var b strings.Builder
	b.WriteString(icon + " " + subtleTextStyle.Render(toolSummary(tool)) + "\n")

	if tool.Error != "" {
		b.WriteString("  " + errorStyle.Render(truncate(tool.Error, maxResultLen)) + "\n")
	}

	if conv.showToolResult && tool.Done && tool.Result != "" {
		b.WriteString(renderToolResult(tool.Result) + "\n")
	}

	return b.String()
}

func toolSummary(tool toolUse) string {
	desc, _ := tool.Arguments["description"].(string)
	cmd, _ := tool.Arguments["command"].(string)
	path, _ := tool.Arguments["path"].(string)
	rawURL, _ := tool.Arguments["url"].(string)

	switch {
	case desc != "" && cmd != "":
		return fmt.Sprintf("%s: %s\n  $ %s", tool.Name, desc, truncate(cmd, maxResultLen))
	case desc != "":
		return fmt.Sprintf("%s: %s", tool.Name, desc)
	case rawURL != "":
		return fmt.Sprintf("%s: %s", tool.Name, truncate(rawURL, maxResultLen))
	case path != "":
		return fmt.Sprintf("%s: %s", tool.Name, path)
	case cmd != "":
		return fmt.Sprintf("%s: $ %s", tool.Name, truncate(cmd, maxResultLen))
	default:
		return tool.Name
	}
}

func renderToolResult(result string) string {
	lines := strings.SplitN(result, "\n", toolLines+1)
	if len(lines) > toolLines {
		lines = lines[:toolLines]
		lines = append(lines, "...")
	}

	var b strings.Builder
	for _, line := range lines {
		b.WriteString("  " + subtleTextStyle.Render(line) + "\n")
	}

	return strings.TrimRight(b.String(), "\n")
}

func renderThinking(thinking string) string {
	return thinkingLabelStyle.Render(thinkingLabel) + "\n" +
		thinkingStyle.Render(thinking) + "\n"
}

func renderAssistantBlock(content string) string {
	if content == "" {
		return ""
	}

	return assistantMessageStyle.Render(content)
}
