package session

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

const (
	emptyLogoText = "" +
		"  ██████╗██████╗  █████╗ ██████╗ ██████╗ ██╗   ██╗\n" +
		" ██╔════╝██╔══██╗██╔══██╗██╔══██╗██╔══██╗╚██╗ ██╔╝\n" +
		" ██║     ██████╔╝███████║██████╔╝██████╔╝ ╚████╔╝ \n" +
		" ██║     ██╔══██╗██╔══██║██╔═══╝ ██╔═══╝   ╚██╔╝  \n" +
		" ╚██████╗██║  ██║██║  ██║██║     ██║        ██║   \n" +
		"  ╚═════╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝     ╚═╝        ╚═╝   "
	emptyCompactLogoText = "CRAPPY"

	systemLabel   = "System"
	thinkingLabel = "Thinking"

	emptyHeadline = "What do you want to understand today?"
	emptySubtitle = "Notice patterns, untangle thoughts, or decide what matters next."

	errorPrefix    = "Error: "
	modelPrefix    = "Model: "
	providerPrefix = "Provider: "

	toolPendingIcon = "⏺"
	toolDoneIcon    = "✓"
	toolErrorIcon   = "!"
	toolIndent      = "  "
	toolCommandMark = "$ "
	truncatedMark   = "..."
	compactingText  = "Compacting context..."

	systemPad     = " "
	systemDivider = "-"

	messagePadding = 2
	emptyPadFactor = 4
	maxResultLen   = 120
	toolLines      = 5
)

func (conv *conversation) refreshContent() {
	if len(conv.messages) == 0 && !conv.runActive && !conv.hasDraft() && !conv.compacting {
		conv.viewport.SetContent(conv.renderEmpty())

		return
	}

	var b strings.Builder

	for i, msg := range conv.messages {
		if i > 0 && conv.messages[i-1].role != msg.role {
			b.WriteByte('\n')
		}

		b.WriteString(conv.renderMessage(msg))
		b.WriteByte('\n')
	}

	if conv.compacting {
		if len(conv.messages) > 0 {
			b.WriteByte('\n')
		}

		b.WriteString(conv.renderSummaryProgress())
		b.WriteByte('\n')
	}

	if conv.runActive || conv.hasDraft() {
		b.WriteString(conv.renderAssistantMessage(conv.draft))
	}

	conv.viewport.SetContent(b.String())
}

func (conv *conversation) renderEmpty() string {
	var b strings.Builder

	b.WriteString(textStyle.Render(conv.emptyLogo()) + "\n\n")
	b.WriteString(textStyle.Render(emptyHeadline) + "\n")
	b.WriteString(subtleTextStyle.Render(emptySubtitle) + "\n\n")

	if conv.model != "" {
		b.WriteString(subtleTextStyle.Render(modelPrefix) + textStyle.Render(conv.model) + "\n")
	} else if conv.provider != "" {
		b.WriteString(subtleTextStyle.Render(providerPrefix) + textStyle.Render(conv.provider) + "\n")
	}

	content := b.String()
	lines := strings.Count(content, "\n")
	pad := max((conv.height-lines-emptyPadFactor)/2, 0)

	return strings.Repeat("\n", pad) + emptyCenterStyle.Width(conv.width).Render(content)
}

func (conv conversation) emptyLogo() string {
	if conv.width > 0 && lipgloss.Width(emptyLogoText) > conv.width {
		return emptyCompactLogoText
	}

	return emptyLogoText
}

func (conv *conversation) renderMessage(msg chatMessage) string {
	if msg.error != "" {
		return errorStyle.Render(errorPrefix+msg.error) + "\n"
	}

	switch msg.role {
	case messageRoleUser:
		return conv.renderUserMessage(msg)
	case messageRoleModel:
		return conv.renderAssistantMessage(msg)
	case messageRoleSystem:
		return conv.renderSystemMessage(msg)
	case messageRoleTool:
		return subtleTextStyle.Render(msg.text) + "\n"
	default:
		return ""
	}
}

func (conv *conversation) renderUserMessage(msg chatMessage) string {
	content := "\n" + textStyle.Render(msg.text) + "\n"

	return userMessageStyle.Width(conv.width).Render(content)
}

func (conv *conversation) renderAssistantMessage(msg chatMessage) string {
	var b strings.Builder

	contentWidth := max(0, conv.width-messagePadding)

	if msg.thinking != "" && conv.showThinking {
		b.WriteString(renderThinking(msg.thinking, contentWidth))
		b.WriteByte('\n')
	}

	if msg.text != "" {
		b.WriteString(textStyle.Width(contentWidth).Render(msg.text))
		b.WriteByte('\n')
	}

	for _, tool := range msg.tools {
		b.WriteString(conv.renderTool(tool))
	}

	return assistantMessageStyle.Render(strings.TrimRight(b.String(), "\n"))
}

func (conv *conversation) renderSystemMessage(msg chatMessage) string {
	return conv.renderSystemBlock(systemLabel, msg.text)
}

func (conv *conversation) renderSummaryProgress() string {
	return conv.renderSystemBlock(systemLabel, compactingText)
}

func (conv *conversation) renderSystemBlock(name string, text string) string {
	label := systemPad + name + systemPad
	lineLen := max((conv.width-len(label))/2, 1)
	line := strings.Repeat(systemDivider, lineLen)
	header := systemStyle.Render(line + label + line)

	if text == "" {
		return header + "\n"
	}

	return header + "\n" + thinkingStyle.Render(text) + "\n"
}

func (conv *conversation) renderTool(tool toolUse) string {
	icon := warningStyle.Render(toolPendingIcon)
	if tool.Done {
		icon = successStyle.Render(toolDoneIcon)
	}

	if tool.Error != "" {
		icon = errorStyle.Render(toolErrorIcon)
	}

	var b strings.Builder
	b.WriteString(icon + systemPad + subtleTextStyle.Render(toolSummary(tool)) + "\n")

	if tool.Error != "" {
		b.WriteString(toolIndent + errorStyle.Render(truncate(tool.Error, maxResultLen)) + "\n")
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
		return fmt.Sprintf("%s: %s\n%s%s%s", tool.Name, desc, toolIndent, toolCommandMark, truncate(cmd, maxResultLen))
	case desc != "":
		return fmt.Sprintf("%s: %s", tool.Name, desc)
	case rawURL != "":
		return fmt.Sprintf("%s: %s", tool.Name, truncate(rawURL, maxResultLen))
	case path != "":
		return fmt.Sprintf("%s: %s", tool.Name, path)
	case cmd != "":
		return fmt.Sprintf("%s: %s%s", tool.Name, toolCommandMark, truncate(cmd, maxResultLen))
	default:
		return tool.Name
	}
}

func renderToolResult(result string) string {
	lines := strings.SplitN(result, "\n", toolLines+1)
	if len(lines) > toolLines {
		lines = lines[:toolLines]
		lines = append(lines, truncatedMark)
	}

	var b strings.Builder
	for _, line := range lines {
		b.WriteString(toolIndent + subtleTextStyle.Render(line) + "\n")
	}

	return strings.TrimRight(b.String(), "\n")
}

func renderThinking(thinking string, width int) string {
	return thinkingLabelStyle.Render(thinkingLabel) + "\n" +
		thinkingStyle.Width(width).Render(thinking)
}
