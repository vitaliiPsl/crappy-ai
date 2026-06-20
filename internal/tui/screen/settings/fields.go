package settings

import (
	"github.com/vitaliiPsl/crappy-adk/kit"

	coresettings "github.com/vitaliiPsl/crappy-ai/internal/settings"
)

const (
	conversationSection = "Conversation"
	modelSection        = "Model"
	providerSection     = "Provider Credentials"

	promptLabel    = "Prompt"
	providerLabel  = "Provider"
	modelLabel     = "Model"
	thinkingLabel  = "Thinking"
	apiKeyLabel    = "API Key"
	apiKeyEnvLabel = "API Key Env"
	baseURLLabel   = "Base URL"
)

type fieldKind int

const (
	fieldText fieldKind = iota
	fieldTextarea
	fieldOption
	fieldModel
)

type fieldDef struct {
	section string
	label   string
	kind    fieldKind
	masked  bool
	options func(Model) []string
	get     func(Model) string
	set     func(*Model, string)
}

func buildFields() []fieldDef {
	return []fieldDef{
		{
			section: conversationSection,
			label:   promptLabel,
			kind:    fieldTextarea,
			get:     func(m Model) string { return m.cfg.Prompt },
			set:     func(m *Model, value string) { m.cfg.Prompt = value },
		},
		{
			section: modelSection,
			label:   providerLabel,
			kind:    fieldOption,
			options: providerOptions,
			get:     func(m Model) string { return m.cfg.Provider },
			set:     func(m *Model, value string) { m.setActiveProvider(value) },
		},
		{
			section: modelSection,
			label:   modelLabel,
			kind:    fieldModel,
			get:     func(m Model) string { return m.cfg.Model },
			set:     func(m *Model, value string) { m.cfg.Model = value },
		},
		{
			section: modelSection,
			label:   thinkingLabel,
			kind:    fieldOption,
			options: func(Model) []string { return []string{"disabled", "low", "medium", "high"} },
			get:     func(m Model) string { return m.cfg.Thinking },
			set:     func(m *Model, value string) { m.cfg.Thinking = value },
		},
		{
			section: providerSection,
			label:   apiKeyLabel,
			kind:    fieldText,
			masked:  true,
			get:     func(m Model) string { return m.provider().APIKey },
			set: func(m *Model, value string) {
				p := m.provider()
				p.APIKey = value
				m.setProvider(p)
			},
		},
		{
			section: providerSection,
			label:   apiKeyEnvLabel,
			kind:    fieldText,
			get:     func(m Model) string { return m.provider().APIKeyEnv },
			set: func(m *Model, value string) {
				p := m.provider()
				p.APIKeyEnv = value
				m.setProvider(p)
			},
		},
		{
			section: providerSection,
			label:   baseURLLabel,
			kind:    fieldText,
			get:     func(m Model) string { return m.provider().BaseURL },
			set: func(m *Model, value string) {
				p := m.provider()
				p.BaseURL = value
				m.setProvider(p)
			},
		},
	}
}

func providerOptions(m Model) []string {
	out := make([]string, 0, len(m.providers))
	for _, p := range m.providers {
		out = append(out, p.ID)
	}

	if len(out) == 0 && m.cfg.Provider != "" {
		out = append(out, m.cfg.Provider)
	}

	return out
}

func (m Model) provider() coresettings.ProviderSettings {
	for _, p := range m.settings.Providers {
		if p.ID == m.cfg.Provider {
			return p
		}
	}

	for _, p := range m.providers {
		if p.ID == m.cfg.Provider {
			return p
		}
	}

	return coresettings.ProviderSettings{ID: m.cfg.Provider, API: m.cfg.Provider}
}

func (m *Model) setProvider(provider coresettings.ProviderSettings) {
	for i, p := range m.settings.Providers {
		if p.ID == provider.ID {
			m.settings.Providers[i] = provider
			m.providers = m.settings.Providers

			return
		}
	}

	m.settings.Providers = append(m.settings.Providers, provider)
	m.providers = m.settings.Providers
}

func (m *Model) setActiveProvider(provider string) {
	m.cfg.Provider = provider

	if m.hasModel(m.cfg.Model) {
		return
	}

	models := m.modelOptions()
	if len(models) > 0 {
		m.cfg.Model = models[0].ID
	}
}

func (m Model) hasModel(modelID string) bool {
	if modelID == "" {
		return false
	}

	for _, model := range m.modelOptions() {
		if model.ID == modelID {
			return true
		}
	}

	return false
}

func (m Model) modelOptions() []kit.ModelConfig {
	if len(m.settings.Models) == 0 {
		return nil
	}

	models := m.settings.Models[m.cfg.Provider]
	if len(models) > 0 {
		return models
	}

	return nil
}

func (m *Model) cycleOption(field fieldDef, delta int) {
	options := field.options(*m)
	if len(options) == 0 {
		return
	}

	current := field.get(*m)

	idx := 0
	for i, option := range options {
		if option == current {
			idx = i

			break
		}
	}

	idx = (idx + delta + len(options)) % len(options)
	field.set(m, options[idx])
	m.state = stateDirty
	m.saveErr = nil
}
