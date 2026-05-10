package settings

import (
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/vitaliiPsl/crappy-ai/internal/config"
	"github.com/vitaliiPsl/crappy-ai/internal/server"
	appsettings "github.com/vitaliiPsl/crappy-ai/internal/settings"
	"github.com/vitaliiPsl/crappy-ai/internal/tui/component"
)

type Model struct {
	server *server.Server

	cfg       config.Config
	settings  appsettings.Settings
	providers []appsettings.ProviderSettings
	fields    []fieldDef

	cursor  int
	state   state
	saveErr error

	editor   component.Editor
	viewport viewport.Model

	width  int
	height int
}

func New(srv *server.Server) Model {
	vp := viewport.New()
	vp.SoftWrap = true

	m := Model{
		server:    srv,
		cfg:       srv.GetConfig(),
		settings:  srv.GetSettings(),
		providers: srv.GetProviders(),
		editor:    component.NewEditor(),
		viewport:  vp,
	}
	m.fields = buildFields()

	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if saved, ok := msg.(savedMsg); ok {
		if saved.err == nil {
			m.state = stateSaved
			m.saveErr = nil
		} else {
			m.state = stateFailed
			m.saveErr = saved.err
		}

		m.refreshContent()

		return m, nil
	}

	if m.state == stateEditing {
		return m.updateEditing(msg)
	}

	return m.updateBrowsing(msg)
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.viewport.SetWidth(width)
	m.editor.SetWidth(width)
	m.resizeViewport()
	m.refreshContent()
}

func (m Model) updateBrowsing(msg tea.Msg) (Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		var cmd tea.Cmd

		m.viewport, cmd = m.viewport.Update(msg)

		return m, cmd
	}

	field := m.currentField()

	switch key.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < len(m.fields)-1 {
			m.cursor++
		}

	case "left":
		if field.kind == fieldOption {
			m.cycleOption(-1)
		}

	case "right":
		if field.kind == fieldOption {
			m.cycleOption(1)
		}

	case "enter":
		if field.kind == fieldOption {
			m.cycleOption(1)
		} else {
			return m.startEditing()
		}

	case "s":
		m.state = stateSaving
		m.saveErr = nil

		return m, m.save()

	case "esc":
		return m, func() tea.Msg { return ClosedMsg{} }
	}

	m.refreshContent()
	m.scrollToCursor()

	return m, nil
}

func (m Model) save() tea.Cmd {
	cfg := m.cfg
	settings := m.settings

	return func() tea.Msg {
		if err := m.server.UpdateConfig(cfg); err != nil {
			return savedMsg{err: err}
		}

		current := m.server.GetSettings()

		current.Providers = settings.Providers
		if err := m.server.UpdateSettings(current); err != nil {
			return savedMsg{err: err}
		}

		return savedMsg{}
	}
}
