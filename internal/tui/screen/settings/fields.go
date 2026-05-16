package settings

import settings "github.com/vitaliiPsl/crappy-ai/internal/settings/models"

const (
	conversationSection = "Conversation"
	modelSection        = "Model"
	providerSection     = "Provider Credentials"

	systemPromptLabel = "System Prompt"
	providerLabel     = "Provider"
	modelLabel        = "Model"
	thinkingLabel     = "Thinking"
	apiKeyLabel       = "API Key"
	apiKeyEnvLabel    = "API Key Env"
	baseURLLabel      = "Base URL"
)

type fieldKind int

const (
	fieldText fieldKind = iota
	fieldTextarea
	fieldOption
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
			label:   systemPromptLabel,
			kind:    fieldTextarea,
			get:     func(m Model) string { return m.cfg.SystemPrompt },
			set:     func(m *Model, value string) { m.cfg.SystemPrompt = value },
		},
		{
			section: modelSection,
			label:   providerLabel,
			kind:    fieldOption,
			options: providerOptions,
			get:     func(m Model) string { return m.cfg.Provider },
			set:     func(m *Model, value string) { m.cfg.Provider = value },
		},
		{
			section: modelSection,
			label:   modelLabel,
			kind:    fieldText,
			get:     func(m Model) string { return m.cfg.Model },
			set:     func(m *Model, value string) { m.cfg.Model = value },
		},
		{
			section: modelSection,
			label:   thinkingLabel,
			kind:    fieldOption,
			options: func(Model) []string { return []string{"", "low", "medium", "high"} },
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
		out = append(out, p.Name)
	}

	if len(out) == 0 && m.cfg.Provider != "" {
		out = append(out, m.cfg.Provider)
	}

	return out
}

func (m Model) provider() settings.ProviderSettings {
	for _, p := range m.settings.Providers {
		if p.Name == m.cfg.Provider {
			return p
		}
	}

	for _, p := range m.providers {
		if p.Name == m.cfg.Provider {
			return p
		}
	}

	return settings.ProviderSettings{Name: m.cfg.Provider, API: m.cfg.Provider}
}

func (m *Model) setProvider(provider settings.ProviderSettings) {
	for i, p := range m.settings.Providers {
		if p.Name == provider.Name {
			m.settings.Providers[i] = provider
			m.providers = m.settings.Providers

			return
		}
	}

	m.settings.Providers = append(m.settings.Providers, provider)
	m.providers = m.settings.Providers
}

func (m Model) currentField() fieldDef {
	if len(m.fields) == 0 {
		return fieldDef{}
	}

	return m.fields[clamp(m.cursor, 0, len(m.fields)-1)]
}

func (m *Model) cycleOption(delta int) {
	field := m.currentField()

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
