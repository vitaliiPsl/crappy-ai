package settings

import (
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
	fields    fieldsModel

	state       state
	returnState state
	saveErr     error

	input            component.Input
	modelSuggestions modelSuggestions

	width  int
	height int
}

func New(srv *server.Server) Model {
	m := Model{
		server:    srv,
		cfg:       srv.GetConfig(),
		settings:  srv.GetSettings(),
		providers: srv.GetProviders(),
		input:     component.NewInput(),
	}
	m.fields = newFieldsModel(buildFields())
	m.fields.SetRows(m.fieldRows())
	m.modelSuggestions = newModelSuggestions(m.modelOptions())

	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if action, ok := msg.(fieldAction); ok {
		return m.handleFieldAction(action)
	}

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

	if m.state == statePickingModel {
		return m.updatePickingModel(msg)
	}

	return m.updateBrowsing(msg)
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.input.SetWidth(width)
	m.resizeViewport()
	m.refreshContent()
}

func (m Model) updateBrowsing(msg tea.Msg) (Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "s":
			m.state = stateSaving
			m.saveErr = nil

			return m, m.save()

		case "esc":
			return m, func() tea.Msg { return ClosedMsg{} }
		}
	}

	var (
		cmd tea.Cmd
	)

	m.fields, cmd = m.fields.Update(msg)

	return m, cmd
}

func (m Model) handleFieldAction(action fieldAction) (Model, tea.Cmd) {
	switch action.kind {
	case fieldActionEdit:
		return m.startEditing(action.field)
	case fieldActionPickModel:
		return m.startModelPicking()
	case fieldActionCycle:
		m.cycleOption(action.field, action.delta)
		m.refreshContent()
	}

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
