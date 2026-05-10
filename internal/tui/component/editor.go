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
	editorMinHeight = 1
	editorMaxHeight = 6
	editorPaddingX  = 1
)

type EditorConfirmMsg struct {
	Value string
}

type EditorCancelMsg struct{}

type EditorOption func(*editorOptions)

type editorOptions struct {
	multiline bool
	masked    bool
}

type Editor struct {
	input    textinput.Model
	textarea textarea.Model

	multiline bool
	width     int
}

func NewEditor(opts ...EditorOption) Editor {
	options := editorOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	thm := theme.Default

	input := textinput.New()
	input.Prompt = ""

	input.EchoMode = textinput.EchoNormal
	if options.masked {
		input.EchoMode = textinput.EchoPassword
	}

	inputStyles := input.Styles()
	inputStyles.Focused.Text = lipgloss.NewStyle().Foreground(thm.Text).Background(thm.SurfaceAlt)
	inputStyles.Focused.Placeholder = lipgloss.NewStyle().Foreground(thm.SubtleText).Background(thm.SurfaceAlt)
	inputStyles.Focused.Prompt = lipgloss.NewStyle().Foreground(thm.Primary).Background(thm.SurfaceAlt)
	inputStyles.Focused.Suggestion = lipgloss.NewStyle().Foreground(thm.SubtleText).Background(thm.SurfaceAlt)
	inputStyles.Blurred.Text = lipgloss.NewStyle().Foreground(thm.SubtleText).Background(thm.SurfaceAlt)
	inputStyles.Blurred.Placeholder = lipgloss.NewStyle().Foreground(thm.Muted).Background(thm.SurfaceAlt)
	inputStyles.Blurred.Prompt = lipgloss.NewStyle().Foreground(thm.Muted).Background(thm.SurfaceAlt)
	inputStyles.Blurred.Suggestion = lipgloss.NewStyle().Foreground(thm.Muted).Background(thm.SurfaceAlt)
	input.SetStyles(inputStyles)

	ta := textarea.New()
	ta.Prompt = ""
	ta.ShowLineNumbers = false
	ta.DynamicHeight = true
	ta.MinHeight = editorMinHeight
	ta.MaxHeight = editorMaxHeight

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

	return Editor{
		input:     input,
		textarea:  ta,
		multiline: options.multiline,
	}
}

func WithEditorMultiline(multiline bool) EditorOption {
	return func(o *editorOptions) {
		o.multiline = multiline
	}
}

func WithEditorMasked(masked bool) EditorOption {
	return func(o *editorOptions) {
		o.masked = masked
	}
}

func (e Editor) Update(msg tea.Msg) (Editor, tea.Cmd, tea.Msg) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc":
			return e, nil, EditorCancelMsg{}
		case "enter":
			return e, nil, EditorConfirmMsg{Value: e.Value()}
		case "shift+enter":
			if e.multiline {
				e.textarea.InsertRune('\n')

				return e, nil, nil
			}
		}
	}

	var cmd tea.Cmd
	if e.multiline {
		e.textarea, cmd = e.textarea.Update(msg)
	} else {
		e.input, cmd = e.input.Update(msg)
	}

	return e, cmd, nil
}

func (e Editor) View() string {
	box := lipgloss.NewStyle().
		Width(e.width).
		Background(theme.Default.SurfaceAlt).
		Padding(0, editorPaddingX)

	var body string
	if e.multiline {
		body = e.textarea.View()
	} else {
		body = e.input.View()
	}

	return strings.TrimRight(box.Render("\n"+body+"\n"), "\n")
}

func (e Editor) Value() string {
	if e.multiline {
		return e.textarea.Value()
	}

	return e.input.Value()
}

func (e *Editor) SetValue(value string) {
	if e.multiline {
		e.textarea.SetValue(value)

		return
	}

	e.input.SetValue(value)
}

func (e *Editor) SetWidth(width int) {
	e.width = width
	innerWidth := max(width-editorPaddingX*2, 1)
	e.input.SetWidth(innerWidth)
	e.textarea.SetWidth(innerWidth)
}

func (e *Editor) Focus() tea.Cmd {
	if e.multiline {
		return e.textarea.Focus()
	}

	return e.input.Focus()
}

func (e Editor) Height() int {
	return lipgloss.Height(e.View())
}
