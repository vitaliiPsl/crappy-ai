package component

import (
	"strings"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vitaliiPsl/crappy-ai/internal/tui/theme"
)

const (
	inputMinHeight = 1
	inputMaxHeight = 6
	inputPaddingX  = 1
)

type ConfirmMsg struct {
	Value string
}

type CancelMsg struct{}

type InputOption func(*inputOptions)

type inputOptions struct {
	multiline   bool
	masked      bool
	placeholder string
	prompt      string
	maxHeight   int
}

type Input struct {
	textinput textinput.Model
	textarea  textarea.Model

	multiline bool
	width     int
}

func NewInput(opts ...InputOption) Input {
	options := inputOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	if options.maxHeight == 0 {
		options.maxHeight = inputMaxHeight
	}

	thm := theme.Default

	ti := textinput.New()
	ti.Prompt = options.prompt
	ti.Placeholder = options.placeholder

	ti.EchoMode = textinput.EchoNormal
	if options.masked {
		ti.EchoMode = textinput.EchoPassword
	}

	inputStyles := ti.Styles()
	inputStyles.Focused.Text = lipgloss.NewStyle().Foreground(thm.Text).Background(thm.SurfaceAlt)
	inputStyles.Focused.Placeholder = lipgloss.NewStyle().Foreground(thm.SubtleText).Background(thm.SurfaceAlt)
	inputStyles.Focused.Prompt = lipgloss.NewStyle().Foreground(thm.Primary).Background(thm.SurfaceAlt)
	inputStyles.Focused.Suggestion = lipgloss.NewStyle().Foreground(thm.SubtleText).Background(thm.SurfaceAlt)
	inputStyles.Blurred.Text = lipgloss.NewStyle().Foreground(thm.SubtleText).Background(thm.SurfaceAlt)
	inputStyles.Blurred.Placeholder = lipgloss.NewStyle().Foreground(thm.Muted).Background(thm.SurfaceAlt)
	inputStyles.Blurred.Prompt = lipgloss.NewStyle().Foreground(thm.Muted).Background(thm.SurfaceAlt)
	inputStyles.Blurred.Suggestion = lipgloss.NewStyle().Foreground(thm.Muted).Background(thm.SurfaceAlt)
	ti.SetStyles(inputStyles)

	ta := textarea.New()
	ta.Placeholder = options.placeholder
	ta.Prompt = options.prompt
	ta.ShowLineNumbers = false
	ta.DynamicHeight = true
	ta.MinHeight = inputMinHeight

	ta.MaxHeight = options.maxHeight
	if options.prompt != "" {
		ta.SetPromptFunc(lipgloss.Width(options.prompt), func(info textarea.PromptInfo) string {
			if info.LineNumber == 0 {
				return options.prompt
			}

			return ""
		})
	}

	textareaStyles := ta.Styles()
	textareaStyles.Focused.Text = lipgloss.NewStyle().Foreground(thm.Text).Background(thm.SurfaceAlt)
	textareaStyles.Focused.Placeholder = lipgloss.NewStyle().Foreground(thm.SubtleText).Background(thm.SurfaceAlt)
	textareaStyles.Focused.Prompt = lipgloss.NewStyle().Foreground(thm.Primary).Background(thm.SurfaceAlt)
	textareaStyles.Focused.Base = lipgloss.NewStyle().Background(thm.SurfaceAlt)
	textareaStyles.Focused.CursorLine = lipgloss.NewStyle()
	textareaStyles.Focused.EndOfBuffer = lipgloss.NewStyle().Foreground(thm.SurfaceAlt).Background(thm.SurfaceAlt)
	textareaStyles.Blurred.Text = lipgloss.NewStyle().Foreground(thm.SubtleText).Background(thm.SurfaceAlt)
	textareaStyles.Blurred.Placeholder = lipgloss.NewStyle().Foreground(thm.Muted).Background(thm.SurfaceAlt)
	textareaStyles.Blurred.Prompt = lipgloss.NewStyle().Foreground(thm.Muted).Background(thm.SurfaceAlt)
	textareaStyles.Blurred.Base = lipgloss.NewStyle().Background(thm.SurfaceAlt)
	textareaStyles.Blurred.CursorLine = lipgloss.NewStyle()
	textareaStyles.Blurred.EndOfBuffer = lipgloss.NewStyle().Foreground(thm.SurfaceAlt).Background(thm.SurfaceAlt)
	ta.SetStyles(textareaStyles)

	if options.multiline {
		_ = ta.Focus()
	} else {
		_ = ti.Focus()
	}

	return Input{
		textinput: ti,
		textarea:  ta,
		multiline: options.multiline,
	}
}

func WithMultiline(multiline bool) InputOption {
	return func(o *inputOptions) {
		o.multiline = multiline
	}
}

func WithMasked(masked bool) InputOption {
	return func(o *inputOptions) {
		o.masked = masked
	}
}

func WithPlaceholder(placeholder string) InputOption {
	return func(o *inputOptions) {
		o.placeholder = placeholder
	}
}

func WithPrompt(prompt string) InputOption {
	return func(o *inputOptions) {
		o.prompt = prompt
	}
}

func WithMaxHeight(maxHeight int) InputOption {
	return func(o *inputOptions) {
		o.maxHeight = maxHeight
	}
}

func (i Input) Update(msg tea.Msg) (Input, tea.Cmd, tea.Msg) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc":
			return i, nil, CancelMsg{}
		case "enter":
			return i, nil, ConfirmMsg{Value: i.Value()}
		case "shift+enter":
			if i.multiline {
				i.textarea.InsertRune('\n')

				return i, nil, nil
			}
		}
	}

	var cmd tea.Cmd
	if i.multiline {
		i.textarea, cmd = i.textarea.Update(msg)
	} else {
		i.textinput, cmd = i.textinput.Update(msg)
	}

	return i, cmd, nil
}

func (i Input) View() string {
	box := lipgloss.NewStyle().
		Width(i.width).
		Background(theme.Default.SurfaceAlt).
		Padding(0, inputPaddingX)

	var body string
	if i.multiline {
		body = i.textarea.View()
	} else {
		body = i.textinput.View()
	}

	return strings.TrimRight(box.Render("\n"+body+"\n"), "\n")
}

func (i Input) Value() string {
	if i.multiline {
		return i.textarea.Value()
	}

	return i.textinput.Value()
}

func (i *Input) SetValue(value string) {
	if i.multiline {
		i.textarea.SetValue(value)

		return
	}

	i.textinput.SetValue(value)
}

func (i *Input) Reset() {
	i.textinput.Reset()
	i.textarea.Reset()
}

func (i *Input) SetWidth(width int) {
	i.width = width
	innerWidth := max(width-inputPaddingX*2, 1)
	i.textinput.SetWidth(innerWidth)
	i.textarea.SetWidth(innerWidth)
}

func (i *Input) Focus() tea.Cmd {
	if i.multiline {
		return i.textarea.Focus()
	}

	return i.textinput.Focus()
}

func (i Input) Height() int {
	return lipgloss.Height(i.View())
}
