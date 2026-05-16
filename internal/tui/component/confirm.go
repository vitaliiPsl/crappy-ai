package component

import (
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vitaliiPsl/crappy-ai/internal/tui/theme"
)

const (
	defaultConfirmLabel = "Confirm"
	defaultCancelLabel  = "Cancel"
	confirmKeySep       = " "
	confirmActionSep    = " • "
)

var (
	defaultConfirmKeys = []string{"y", "enter"}
	defaultCancelKeys  = []string{"n", "esc"}
)

type ConfirmOption func(*confirmOptions)

type confirmOptions struct {
	prompt       string
	confirmLabel string
	cancelLabel  string
	confirmKeys  []string
	cancelKeys   []string
}

type Confirm struct {
	prompt       string
	confirmLabel string
	cancelLabel  string
	confirmKeys  []string
	cancelKeys   []string
}

func NewConfirm(opts ...ConfirmOption) Confirm {
	options := confirmOptions{
		confirmLabel: defaultConfirmLabel,
		cancelLabel:  defaultCancelLabel,
		confirmKeys:  defaultConfirmKeys,
		cancelKeys:   defaultCancelKeys,
	}
	for _, opt := range opts {
		opt(&options)
	}

	return Confirm(options)
}

func WithConfirmPrompt(prompt string) ConfirmOption {
	return func(o *confirmOptions) { o.prompt = prompt }
}

func WithConfirmLabel(label string) ConfirmOption {
	return func(o *confirmOptions) { o.confirmLabel = label }
}

func WithCancelLabel(label string) ConfirmOption {
	return func(o *confirmOptions) { o.cancelLabel = label }
}

func WithConfirmKeys(keys ...string) ConfirmOption {
	return func(o *confirmOptions) { o.confirmKeys = keys }
}

func WithCancelKeys(keys ...string) ConfirmOption {
	return func(o *confirmOptions) { o.cancelKeys = keys }
}

func (c Confirm) Update(msg tea.Msg) (Confirm, tea.Cmd, tea.Msg) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return c, nil, nil
	}

	switch keyStr := key.String(); {
	case slices.Contains(c.confirmKeys, keyStr):
		return c, nil, ConfirmMsg{}
	case slices.Contains(c.cancelKeys, keyStr):
		return c, nil, CancelMsg{}
	}

	return c, nil, nil
}

func (c Confirm) View() string {
	style := lipgloss.NewStyle().Foreground(theme.Default.Warning).Bold(true)

	parts := []string{
		c.prompt,
		c.confirmKeys[0] + confirmKeySep + c.confirmLabel,
		c.cancelKeys[0] + confirmKeySep + c.cancelLabel,
	}

	return style.Render(strings.Join(parts, confirmActionSep))
}
