package jobs

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vitaliiPsl/crappy-ai/internal/background"
	"github.com/vitaliiPsl/crappy-ai/internal/server"
)

const (
	headerText    = "Background Jobs"
	hintsText     = "j/Down Move • r Refresh • c Cancel • Esc Back"
	emptyTitle    = "No background jobs"
	emptySubtitle = "Start a long-running bash command with background execution."

	cursorPrefix   = "> "
	noCursorPrefix = "  "
	titleSep       = "  "
	metaPad        = "  "
	metaSep        = " · "
	headerLines    = 2
	bottomHeight   = 1
	jobPreviewLen  = 160
)

type Model struct {
	server    *server.Server
	sessionID string

	jobs   []background.Job
	cursor int

	viewport viewport.Model
	width    int
	height   int
}

func New(srv *server.Server, sessionID string) Model {
	vp := viewport.New()
	vp.SoftWrap = false

	return Model{
		server:    srv,
		sessionID: sessionID,
		viewport:  vp,
	}
}

func (m Model) Init() tea.Cmd {
	return m.loadJobs()
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case jobsLoadedMsg:
		m.jobs = msg.jobs
		m.cursor = clampCursor(m.cursor, len(m.jobs))
		m.refreshContent()
		m.scrollToCursor()

		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	var cmd tea.Cmd

	m.viewport, cmd = m.viewport.Update(msg)

	return m, cmd
}

func (m Model) View() string {
	header := lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(headerStyle.Render(headerText))
	bottom := lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(hintsStyle.Render(hintsText))

	return header + "\n\n" + m.viewport.View() + "\n" + bottom
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.viewport.SetWidth(width)
	m.resizeViewport()
	m.refreshContent()
}

func (m Model) handleKey(key tea.KeyMsg) (Model, tea.Cmd) {
	switch key.String() {
	case "up", "k":
		return m.moveCursor(-1)
	case "down", "j":
		return m.moveCursor(1)
	case "r":
		return m, m.loadJobs()
	case "c":
		return m.cancelSelected()
	case "esc":
		return m, func() tea.Msg { return ClosedMsg{} }
	}

	var cmd tea.Cmd

	m.viewport, cmd = m.viewport.Update(key)

	return m, cmd
}

func (m Model) moveCursor(delta int) (Model, tea.Cmd) {
	next := m.cursor + delta
	if next < 0 || next >= len(m.jobs) {
		return m, nil
	}

	m.cursor = next
	m.refreshContent()
	m.scrollToCursor()

	return m, nil
}

func (m Model) cancelSelected() (Model, tea.Cmd) {
	if m.cursor < 0 || m.cursor >= len(m.jobs) {
		return m, nil
	}

	job := m.jobs[m.cursor]
	if job.Status != background.StatusRunning {
		return m, nil
	}

	job.Status = background.StatusCanceled
	m.jobs[m.cursor] = job
	m.refreshContent()

	return m, func() tea.Msg {
		m.server.CancelBackgroundJob(m.sessionID, job.ID)

		return jobsLoadedMsg{jobs: m.server.BackgroundJobs(m.sessionID)}
	}
}

func (m *Model) resizeViewport() {
	m.viewport.SetHeight(max(m.height-headerLines-bottomHeight, 1))
}

func (m *Model) refreshContent() {
	if len(m.jobs) == 0 {
		m.viewport.SetContent(renderEmpty(m.width, m.viewport.Height()))

		return
	}

	m.viewport.SetContent(renderList(m.jobs, m.cursor))
}

func (m Model) loadJobs() tea.Cmd {
	return func() tea.Msg {
		return jobsLoadedMsg{jobs: m.server.BackgroundJobs(m.sessionID)}
	}
}

func (m Model) scrollToCursor() {
	if m.cursor < 0 {
		return
	}

	top := m.cursor * 4
	if top < m.viewport.YOffset() {
		m.viewport.SetYOffset(top)
	}

	bottom := top + 3
	if bottom >= m.viewport.YOffset()+m.viewport.Height() {
		m.viewport.SetYOffset(max(bottom-m.viewport.Height()+1, 0))
	}
}

func renderEmpty(width, height int) string {
	content := emptyStyle.Render(emptyTitle) + "\n" + emptyStyle.Render(emptySubtitle)

	return lipgloss.NewStyle().
		Width(width).
		Height(max(height, 1)).
		Align(lipgloss.Center).
		AlignVertical(lipgloss.Center).
		Render(content)
}

func renderList(jobs []background.Job, cursor int) string {
	var b strings.Builder
	for i, job := range jobs {
		b.WriteString(renderJob(job, i == cursor))
		b.WriteString("\n\n")
	}

	return strings.TrimRight(b.String(), "\n")
}

func renderJob(job background.Job, selected bool) string {
	cursor := noCursorPrefix

	title := itemStyle.Render(jobTitle(job))
	if selected {
		cursor = selectedStyle.Render(cursorPrefix)
		title = selectedStyle.Render(jobTitle(job))
	}

	lines := []string{
		cursor + title + titleSep + statusBadge(job.Status),
		metaStyle.Render(metaPad + strings.Join(jobMeta(job), metaSep)),
	}

	if preview := jobPreview(job); preview != "" {
		lines = append(lines, metaStyle.Render(metaPad+preview))
	}

	return strings.Join(lines, "\n")
}

func jobTitle(job background.Job) string {
	tool := job.Tool
	if tool == "" {
		tool = "job"
	}

	return job.ID + " " + tool
}

func jobMeta(job background.Job) []string {
	parts := []string{fmt.Sprintf("started %s", relativeTime(job.StartedAt))}
	if job.CompletedAt != nil {
		parts = append(parts, fmt.Sprintf("completed %s", relativeTime(*job.CompletedAt)))
	}

	return parts
}

func jobPreview(job background.Job) string {
	text := job.Error
	if text == "" {
		text = job.Output
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}

	text = strings.ReplaceAll(text, "\n", " ")
	if len(text) > jobPreviewLen {
		return text[:jobPreviewLen] + "..."
	}

	return text
}

func statusBadge(status background.Status) string {
	switch status {
	case background.StatusRunning:
		return warningStyle.Render("running")
	case background.StatusSucceeded:
		return successStyle.Render("succeeded")
	case background.StatusFailed:
		return errorStyle.Render("failed")
	case background.StatusCanceled:
		return mutedStyle.Render("canceled")
	default:
		return mutedStyle.Render(string(status))
	}
}

func relativeTime(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}

	d := time.Since(t).Round(time.Second)
	if d < time.Second {
		return "now"
	}

	return d.String() + " ago"
}

func clampCursor(cursor, count int) int {
	if count <= 0 {
		return 0
	}

	if cursor < 0 {
		return 0
	}

	if cursor >= count {
		return count - 1
	}

	return cursor
}
