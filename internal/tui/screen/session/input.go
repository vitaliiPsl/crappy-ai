package session

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/tui/command"
	"github.com/vitaliiPsl/crappy-ai/internal/tui/component"
)

const (
	inputPlaceholder = "Type a message or /command..."
	inputPrompt      = "> "
	inputMaxHeight   = 8
)

type Focus int

const (
	FocusInput Focus = iota
	FocusPrompt
)

type inputBar struct {
	input       component.Input
	picker      commandPicker
	attachments []attachment
}

func newInputBar(registry *command.Registry) inputBar {
	return inputBar{
		input: component.NewInput(
			component.WithMultiline(true),
			component.WithPlaceholder(inputPlaceholder),
			component.WithPrompt(inputPrompt),
			component.WithMaxHeight(inputMaxHeight),
		),
		picker: newCommandPicker(registry),
	}
}

func (b inputBar) Init() tea.Cmd {
	return b.input.Focus()
}

func (b inputBar) Update(msg tea.Msg) (inputBar, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		if next, consumed := b.handlePickerKey(key); consumed {
			return next, nil
		}
	}

	var (
		cmd tea.Cmd
		out tea.Msg
	)

	b.input, cmd, out = b.input.Update(msg)
	b.picker.Sync(b.input.Value())

	confirm, ok := out.(component.ConfirmMsg)
	if !ok {
		return b, cmd
	}

	text := strings.TrimSpace(confirm.Value)

	b.input.Reset()
	b.picker.Clear()

	if text == "" && len(b.attachments) == 0 {
		return b, cmd
	}

	if cmdMsg, isCmd := parseCommand(text); isCmd {
		return b, tea.Batch(cmd, emitCommand(cmdMsg))
	}

	content := make([]kit.Content, 0, len(b.attachments)+1)
	if text != "" {
		content = append(content, kit.NewTextContent(text))
	}

	for _, item := range b.attachments {
		content = append(content, item.Content)
	}

	b.attachments = nil

	return b, tea.Batch(cmd, emitSubmit(content))
}

func (b inputBar) View() string {
	input := strings.TrimRight(b.input.View(), "\n")
	if len(b.attachments) > 0 {
		labels := make([]string, 0, len(b.attachments))
		for _, item := range b.attachments {
			labels = append(labels, item.label())
		}

		input = strings.Join(labels, " ") + "\n" + input
	}

	suggestions := b.picker.View()
	if suggestions == "" {
		return input
	}

	return suggestions + "\n" + input
}

func (b *inputBar) SetWidth(width int) {
	b.input.SetWidth(width)
}

func (b *inputBar) Reset() {
	b.input.Reset()
	b.picker.Clear()
	b.attachments = nil
}

func (b *inputBar) attach(item attachment) {
	b.attachments = append(b.attachments, item)
}

func (b inputBar) PickerActive() bool {
	return b.picker.Active()
}

func (b *inputBar) ClearPicker() {
	b.picker.Clear()
}

func (b inputBar) handlePickerKey(key tea.KeyMsg) (inputBar, bool) {
	if !b.picker.Active() {
		return b, false
	}

	switch key.String() {
	case "up":
		b.picker.Previous()

		return b, true

	case "down":
		b.picker.Next()

		return b, true

	case "enter":
		value, completed := b.picker.Completion(b.input.Value())
		if !completed {
			return b, false
		}

		b.input.SetValue(value)
		b.picker.Clear()

		return b, true
	}

	return b, false
}

func focusForState(s State) Focus {
	if s.Prompt != nil {
		return FocusPrompt
	}

	return FocusInput
}

func emitSubmit(content []kit.Content) tea.Cmd {
	return func() tea.Msg { return submitMsg{Content: content} }
}

func emitCommand(msg commandMsg) tea.Cmd {
	return func() tea.Msg { return msg }
}
