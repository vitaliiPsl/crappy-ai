package session

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/vitaliiPsl/crappy-ai/internal/tui/command"
	"github.com/vitaliiPsl/crappy-ai/internal/tui/component"
)

const (
	inputPlaceholder = "Type a message or /command..."
	inputPrompt      = "> "
	inputMaxHeight   = 8
)

type inputBar struct {
	input    component.Input
	commands commandPicker
}

func newInputBar(registry *command.Registry) inputBar {
	return inputBar{
		input: component.NewInput(
			component.WithMultiline(true),
			component.WithPlaceholder(inputPlaceholder),
			component.WithPrompt(inputPrompt),
			component.WithMaxHeight(inputMaxHeight),
		),
		commands: newCommandPicker(registry),
	}
}

func (b inputBar) Init() tea.Cmd {
	return b.input.Focus()
}

func (b inputBar) Update(msg tea.Msg) (inputBar, tea.Cmd, bool) {
	if b.shouldPassThrough(msg) {
		return b, nil, false
	}

	if key, ok := msg.(tea.KeyMsg); ok {
		if next, consumed := b.handleSuggestionKey(key); consumed {
			return next, nil, true
		}
	}

	if !isInputMsg(msg) {
		return b, nil, false
	}

	return b.updateInput(msg)
}

func (b inputBar) View() string {
	input := strings.TrimRight(b.input.View(), "\n")

	suggestions := b.commands.View()
	if suggestions == "" {
		return input
	}

	return suggestions + "\n" + input
}

func (b *inputBar) setSize(width int) {
	b.input.SetWidth(width)
}

func (b inputBar) shouldPassThrough(msg tea.Msg) bool {
	switch msg := msg.(type) {
	case tea.MouseWheelMsg:
		return true
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+o", "ctrl+t", "pgup", "pgdown":
			return true
		case "up", "down":
			return !b.commands.Active()
		}
	}

	return false
}

func (b inputBar) handleSuggestionKey(key tea.KeyMsg) (inputBar, bool) {
	switch key.String() {
	case "up":
		consumed := b.commands.Previous()

		return b, consumed
	case "down":
		consumed := b.commands.Next()

		return b, consumed
	case "esc":
		if !b.commands.Active() {
			return b, false
		}

		b.commands.Clear()

		return b, true
	case "enter":
		value, ok := b.commands.Completion(b.input.Value())
		if !ok {
			return b, false
		}

		b.input.SetValue(value)
		b.commands.Clear()

		return b, true
	default:
		return b, false
	}
}

func isInputMsg(msg tea.Msg) bool {
	switch msg.(type) {
	case tea.KeyMsg, tea.PasteMsg, tea.PasteStartMsg, tea.PasteEndMsg:
		return true
	default:
		return false
	}
}

func (b inputBar) updateInput(msg tea.Msg) (inputBar, tea.Cmd, bool) {
	var (
		cmd tea.Cmd
		out tea.Msg
	)

	b.input, cmd, out = b.input.Update(msg)
	b.commands.Update(b.input.Value())

	if submit, ok := out.(component.ConfirmMsg); ok {
		return b.handleSubmit(submit.Value)
	}

	return b, cmd, true
}

func (b inputBar) handleSubmit(value string) (inputBar, tea.Cmd, bool) {
	if strings.TrimSpace(value) == "" {
		return b, nil, true
	}

	b.input.Reset()
	b.commands.Clear()

	if cmdMsg, ok := parseCommand(value); ok {
		return b, func() tea.Msg { return cmdMsg }, true
	}

	return b, func() tea.Msg { return submitMsg{Text: value} }, true
}
