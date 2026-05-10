package session

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/vitaliiPsl/crappy-ai/internal/tui/command"
	"github.com/vitaliiPsl/crappy-ai/internal/tui/theme"
)

const maxCommandSuggestions = 4

type commandSuggestions struct {
	commands []command.Definition
	matches  []command.Definition
	selected int
}

func newCommandSuggestions(registry *command.Registry) commandSuggestions {
	if registry == nil {
		return commandSuggestions{}
	}

	return commandSuggestions{commands: registry.Definitions()}
}

func (c *commandSuggestions) Update(value string) {
	if !strings.HasPrefix(value, "/") || strings.ContainsAny(value, " \t\r\n") {
		c.Clear()

		return
	}

	prefix := strings.ToLower(value[1:])
	c.matches = c.matches[:0]
	for _, def := range c.commands {
		if strings.HasPrefix(def.Name, prefix) {
			c.matches = append(c.matches, def)
		}
	}

	if c.selected >= len(c.matches) {
		c.selected = 0
	}
}

func (c *commandSuggestions) Clear() {
	c.matches = nil
	c.selected = 0
}

func (c commandSuggestions) Active() bool {
	return len(c.matches) > 0
}

func (c *commandSuggestions) Previous() bool {
	if !c.Active() {
		return false
	}

	c.selected = (c.selected - 1 + len(c.matches)) % len(c.matches)

	return true
}

func (c *commandSuggestions) Next() bool {
	if !c.Active() {
		return false
	}

	c.selected = (c.selected + 1) % len(c.matches)

	return true
}

func (c commandSuggestions) Completion(value string) (string, bool) {
	if !c.Active() || c.hasExactMatch(value) {
		return "", false
	}

	return "/" + c.matches[c.selected].Name, true
}

func (c commandSuggestions) View() string {
	if !c.Active() {
		return ""
	}

	thm := theme.Default
	selectedStyle := lipgloss.NewStyle().Foreground(thm.Primary).Bold(true)
	normalStyle := lipgloss.NewStyle().Foreground(thm.SubtleText)
	descStyle := lipgloss.NewStyle().Foreground(thm.Muted)

	lines := make([]string, 0, min(len(c.matches), maxCommandSuggestions))
	start, end := c.visibleRange()
	for idx := start; idx < end; idx++ {
		def := c.matches[idx]
		nameStyle := normalStyle
		if idx == c.selected {
			nameStyle = selectedStyle
		}

		line := nameStyle.Render("/" + def.Name)
		if def.Description != "" {
			line += "  " + descStyle.Render(def.Description)
		}

		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

func (c commandSuggestions) hasExactMatch(value string) bool {
	return c.Active() && value == "/"+c.matches[c.selected].Name
}

func (c commandSuggestions) visibleRange() (int, int) {
	if len(c.matches) <= maxCommandSuggestions {
		return 0, len(c.matches)
	}

	start := c.selected - maxCommandSuggestions/2
	start = max(start, 0)

	end := start + maxCommandSuggestions
	if end > len(c.matches) {
		end = len(c.matches)
		start = end - maxCommandSuggestions
	}

	return start, end
}

func parseCommand(value string) (commandMsg, bool) {
	if !strings.HasPrefix(value, "/") || strings.ContainsRune(value, '\n') {
		return commandMsg{}, false
	}

	parts := strings.Fields(value[1:])
	if len(parts) == 0 {
		return commandMsg{}, false
	}

	return commandMsg{Name: parts[0], Args: parts[1:]}, true
}
