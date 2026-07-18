package settings

import (
	"strconv"

	"github.com/vitaliiPsl/crappy-adk/kit"

	coresettings "github.com/vitaliiPsl/crappy-ai/internal/settings"
	"github.com/vitaliiPsl/crappy-ai/internal/utils"
)

const (
	conversationSection = "Conversation"
	modelSection        = "Model"
	providerSection     = "Provider Credentials"

	promptLabel          = "Prompt"
	providerLabel        = "Provider"
	modelLabel           = "Model"
	thinkingLabel        = "Thinking"
	temperatureLabel     = "Temperature"
	maxOutputTokensLabel = "Max Output Tokens"
	apiKeyLabel          = "API Key"
	apiKeyEnvLabel       = "API Key Env"
	baseURLLabel         = "Base URL"
	authTypeLabel        = "Authentication"
	oauthDriverLabel     = "OAuth Driver"
	oauthLabel           = "Subscription"
)

type fieldKind int

const (
	fieldText fieldKind = iota
	fieldTextarea
	fieldOption
	fieldModel
	fieldOAuth
)

type fieldDef struct {
	section string
	label   string
	kind    fieldKind
	masked  bool
	visible func(Model) bool
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
			section: modelSection,
			label:   temperatureLabel,
			kind:    fieldText,
			get:     func(m Model) string { return formatFloat32Ptr(m.cfg.Temperature) },
			set:     func(m *Model, value string) { m.cfg.Temperature = parseFloat32Ptr(value) },
		},
		{
			section: modelSection,
			label:   maxOutputTokensLabel,
			kind:    fieldText,
			get:     func(m Model) string { return formatInt32Ptr(m.cfg.MaxOutputTokens) },
			set:     func(m *Model, value string) { m.cfg.MaxOutputTokens = parseInt32Ptr(value) },
		},
		{
			section: providerSection,
			label:   authTypeLabel,
			kind:    fieldOption,
			options: authTypeOptions,
			get:     func(m Model) string { return string(m.provider().Auth.Type) },
			set: func(m *Model, value string) {
				p := m.provider()

				p.Auth = coresettings.ProviderAuthSettings{Type: coresettings.ProviderAuthType(value)}
				if p.Auth.Type == coresettings.ProviderAuthOAuth {
					drivers := m.server.ProviderOAuthDrivers(m.cfg.Provider)
					if len(drivers) > 0 {
						p.Auth.Driver = drivers[0]
					}
				}

				m.setProvider(p)
				m.oauthStatus = "disconnected"
				m.oauthErr = nil
			},
		},
		{
			section: providerSection,
			label:   oauthDriverLabel,
			kind:    fieldOption,
			visible: func(m Model) bool { return m.provider().Auth.Type == coresettings.ProviderAuthOAuth },
			options: oauthDriverOptions,
			get:     func(m Model) string { return m.provider().Auth.Driver },
			set: func(m *Model, value string) {
				p := m.provider()
				p.Auth.Driver = value
				m.setProvider(p)
				m.oauthStatus = "disconnected"
				m.oauthErr = nil
			},
		},
		{
			section: providerSection,
			label:   oauthLabel,
			kind:    fieldOAuth,
			visible: func(m Model) bool { return m.provider().Auth.Type == coresettings.ProviderAuthOAuth },
			get:     func(m Model) string { return m.oauthStatus },
		},
		{
			section: providerSection,
			label:   apiKeyLabel,
			kind:    fieldText,
			masked:  true,
			visible: func(m Model) bool { return m.provider().Auth.Type == coresettings.ProviderAuthAPIKey },
			get:     func(m Model) string { return m.provider().Auth.APIKey },
			set: func(m *Model, value string) {
				p := m.provider()
				p.Auth.APIKey = value
				m.setProvider(p)
			},
		},
		{
			section: providerSection,
			label:   apiKeyEnvLabel,
			kind:    fieldText,
			visible: func(m Model) bool { return m.provider().Auth.Type == coresettings.ProviderAuthAPIKey },
			get:     func(m Model) string { return m.provider().Auth.APIKeyEnv },
			set: func(m *Model, value string) {
				p := m.provider()
				p.Auth.APIKeyEnv = value
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

func authTypeOptions(m Model) []string {
	options := []string{string(coresettings.ProviderAuthAPIKey)}
	if m.server != nil && m.server.ProviderSupportsOAuth(m.cfg.Provider) {
		options = append(options, string(coresettings.ProviderAuthOAuth))
	}

	return options
}

func oauthDriverOptions(m Model) []string {
	if m.server == nil {
		return nil
	}

	return m.server.ProviderOAuthDrivers(m.cfg.Provider)
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

	return coresettings.ProviderSettings{
		ID:   m.cfg.Provider,
		API:  m.cfg.Provider,
		Auth: coresettings.ProviderAuthSettings{Type: coresettings.ProviderAuthAPIKey},
	}
}

func formatFloat32Ptr(v *float32) string {
	if v == nil {
		return ""
	}

	return strconv.FormatFloat(float64(*v), 'f', -1, 32)
}

func parseFloat32Ptr(raw string) *float32 {
	v, err := utils.ParseFloat32Ptr(raw)
	if err != nil {
		return nil
	}

	return v
}

func formatInt32Ptr(v *int32) string {
	if v == nil {
		return ""
	}

	return strconv.FormatInt(int64(*v), 10)
}

func parseInt32Ptr(raw string) *int32 {
	v, err := utils.ParseNonnegativeInt32Ptr(raw)
	if err != nil {
		return nil
	}

	return v
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
	m.refreshOAuthStatus()

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
