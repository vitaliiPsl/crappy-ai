package component

import (
	"strings"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vitaliiPsl/crappy-ai/internal/tui/theme"
)

const (
	inputPlaceholder = "Type a message..."
	inputPrompt      = "> "
	inputMinHeight   = 1
	inputMaxHeight   = 8
	inputPaddingX    = 1
)

type SubmitMsg struct {
	Text string
}

type Input struct {
	textarea textarea.Model
	width    int
	initCmd  tea.Cmd
}

func NewInput() Input {
	thm := theme.Default

	input := textarea.New()
	input.Placeholder = inputPlaceholder
	input.Prompt = inputPrompt
	input.CharLimit = 0
	input.ShowLineNumbers = false
	input.DynamicHeight = true
	input.MinHeight = inputMinHeight
	input.MaxHeight = inputMaxHeight
	input.SetHeight(inputMinHeight)
	input.SetPromptFunc(lipgloss.Width(inputPrompt), func(info textarea.PromptInfo) string {
		if info.LineNumber == 0 {
			return inputPrompt
		}

		return ""
	})

	styles := input.Styles()
	styles.Focused.Text = lipgloss.NewStyle().Foreground(thm.Text)
	styles.Focused.Placeholder = lipgloss.NewStyle().Foreground(thm.SubtleText)
	styles.Focused.Prompt = lipgloss.NewStyle().Foreground(thm.Primary)
	styles.Focused.Base = lipgloss.NewStyle().Background(thm.SurfaceAlt)
	styles.Focused.CursorLine = lipgloss.NewStyle()
	styles.Focused.EndOfBuffer = lipgloss.NewStyle().Foreground(thm.SurfaceAlt).Background(thm.SurfaceAlt)
	styles.Blurred.Text = lipgloss.NewStyle().Foreground(thm.SubtleText)
	styles.Blurred.Placeholder = lipgloss.NewStyle().Foreground(thm.Muted)
	styles.Blurred.Prompt = lipgloss.NewStyle().Foreground(thm.Muted)
	styles.Blurred.Base = lipgloss.NewStyle().Background(thm.SurfaceAlt)
	styles.Blurred.CursorLine = lipgloss.NewStyle()
	styles.Blurred.EndOfBuffer = lipgloss.NewStyle().Foreground(thm.SurfaceAlt).Background(thm.SurfaceAlt)
	input.SetStyles(styles)

	return Input{textarea: input, initCmd: input.Focus()}
}

func (i Input) Init() tea.Cmd {
	return i.initCmd
}

func (i Input) Update(msg tea.Msg) (Input, tea.Cmd, tea.Msg) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "enter":
			return i.submit()
		case "shift+enter":
			i.textarea.InsertRune('\n')

			return i, nil, nil
		case "pgup", "pgdown", "up", "down":
			return i, nil, msg
		}
	}

	if _, ok := msg.(tea.MouseWheelMsg); ok {
		return i, nil, msg
	}

	var cmd tea.Cmd

	i.textarea, cmd = i.textarea.Update(msg)

	return i, cmd, nil
}

func (i Input) View() string {
	box := lipgloss.NewStyle().
		Width(i.width).
		Background(theme.Default.SurfaceAlt).
		Padding(0, inputPaddingX)

	return strings.TrimRight(box.Render("\n"+i.textarea.View()+"\n"), "\n")
}

func (i Input) Height() int {
	return lipgloss.Height(i.View())
}

func (i *Input) SetWidth(width int) {
	i.width = width
	i.textarea.SetWidth(max(width-inputPaddingX*2, 1))
}

func (i Input) submit() (Input, tea.Cmd, tea.Msg) {
	text := i.textarea.Value()
	if strings.TrimSpace(text) == "" {
		return i, nil, nil
	}

	i.textarea.Reset()

	return i, func() tea.Msg { return SubmitMsg{Text: text} }, nil
}
