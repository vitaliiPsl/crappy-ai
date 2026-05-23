package session

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/vitaliiPsl/crappy-ai/internal/tui/command"
	"github.com/vitaliiPsl/crappy-ai/internal/tui/component"
)

const (
	commandPickerWindow = 4
	commandDescSep      = "  "
)

type commandPicker struct {
	selector component.Selector[command.Definition]
	active   bool
}

func newCommandPicker(registry *command.Registry) commandPicker {
	selected := lipgloss.NewStyle().Foreground(sessionTheme.Primary).Bold(true)
	normal := lipgloss.NewStyle().Foreground(sessionTheme.SubtleText)
	desc := lipgloss.NewStyle().Foreground(sessionTheme.Muted)

	var items []command.Definition
	if registry != nil {
		items = registry.Definitions()
	}

	return commandPicker{
		selector: component.NewSelector(component.SelectorConfig[command.Definition]{
			Items: items,
			Match: func(def command.Definition, query string) bool {
				return strings.HasPrefix(def.Name, query)
			},
			Render: func(def command.Definition, isSelected bool) string {
				style := normal
				if isSelected {
					style = selected
				}

				line := style.Render("/" + def.Name)
				if def.Description != "" {
					line += commandDescSep + desc.Render(def.Description)
				}

				return line
			},
			Window: commandPickerWindow,
		}),
	}
}

func (c *commandPicker) Sync(value string) {
	if !strings.HasPrefix(value, "/") || strings.ContainsAny(value, " \t\r\n") {
		c.active = false

		return
	}

	c.active = true
	c.selector.Filter(strings.ToLower(value[1:]))
}

func (c commandPicker) View() string {
	if !c.Active() {
		return ""
	}

	return c.selector.View()
}

func (c commandPicker) Active() bool {
	return c.active && !c.selector.Empty()
}

func (c *commandPicker) Clear() {
	c.active = false
}

func (c *commandPicker) Previous() {
	if c.Active() {
		c.selector.Previous()
	}
}

func (c *commandPicker) Next() {
	if c.Active() {
		c.selector.Next()
	}
}

func (c commandPicker) Completion(value string) (string, bool) {
	if !c.Active() {
		return "", false
	}

	sel, ok := c.selector.Selected()
	if !ok {
		return "", false
	}

	completion := "/" + sel.Name
	if value == completion {
		return "", false
	}

	return completion, true
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
