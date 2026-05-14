package session

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/vitaliiPsl/crappy-ai/internal/tui/theme"
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

	assistantLabel = "Crappy"
	userLabel      = "You"
	systemLabel    = "System"
	thinkingLabel  = "Thinking"

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
	if len(conv.messages) == 0 && !conv.turnActive && !conv.hasDraft() && !conv.compacting {
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

	if conv.compacting {
		if len(conv.messages) > 0 {
			b.WriteByte('\n')
		}

		b.WriteString(conv.renderSummaryProgress())
		b.WriteByte('\n')
	}

	if conv.turnActive || conv.hasDraft() {
		if (len(conv.messages) > 0 && conv.messages[len(conv.messages)-1].role != messageRoleModel) || conv.compacting {
			b.WriteByte('\n')
		}

		lastIsAssistant := len(conv.messages) > 0 && conv.messages[len(conv.messages)-1].role == messageRoleModel
		b.WriteString(conv.renderDraft(!lastIsAssistant))
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

	return strings.Repeat("\n", pad) +
		lipgloss.NewStyle().Width(conv.width).Align(lipgloss.Center).Render(content)
}

func (conv conversation) emptyLogo() string {
	if conv.width > 0 && lipgloss.Width(emptyLogoText) > conv.width {
		return emptyCompactLogoText
	}

	return emptyLogoText
}

func (conv *conversation) renderMessage(msg chatMessage, showLabel bool) string {
	if msg.error != "" {
		return errorStyle.Render(errorPrefix+msg.error) + "\n"
	}

	switch msg.role {
	case messageRoleUser:
		return conv.renderUserMessage(msg)
	case messageRoleModel:
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
		b.WriteString(textStyle.Width(max(0, conv.width-messagePadding)).Render(msg.text))
		b.WriteByte('\n')
	}

	for _, tool := range msg.tools {
		b.WriteString(conv.renderTool(tool))
	}

	return renderAssistantBlock(strings.TrimRight(b.String(), "\n"))
}

func (conv *conversation) renderDraft(showLabel bool) string {
	var b strings.Builder

	thinkingText := collapseBlankLines(conv.draft.thinking)
	showThinking := thinkingText != "" && conv.showThinking

	if showLabel && (showThinking || conv.draft.text != "" || len(conv.draft.tools) > 0) {
		b.WriteString(assistantLabelStyle.Render(assistantLabel) + "\n")
	}

	if showThinking {
		b.WriteString(renderThinking(thinkingText))

		if conv.draft.text != "" || len(conv.draft.tools) > 0 {
			b.WriteByte('\n')
		}
	}

	if conv.draft.text != "" {
		b.WriteString(textStyle.Width(max(0, conv.width-messagePadding)).Render(conv.draft.text))
		b.WriteByte('\n')
	}

	for _, tool := range conv.draft.tools {
		b.WriteString(conv.renderTool(tool))
	}

	return renderAssistantBlock(strings.TrimRight(b.String(), "\n"))
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
