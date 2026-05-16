package server

import (
	"github.com/vitaliiPsl/crappy-ai/internal/config"
	"github.com/vitaliiPsl/crappy-ai/internal/settings"
	settingsmodels "github.com/vitaliiPsl/crappy-ai/internal/settings/models"
)

func (s *Server) GetConfig() config.Config {
	return s.configStore.Get()
}

func (s *Server) UpdateConfig(cfg config.Config) error {
	return s.configStore.Save(cfg)
}

func (s *Server) GetSettings() settings.Settings {
	return s.settingsStore.Get()
}

func (s *Server) UpdateSettings(st settings.Settings) error {
	return s.settingsStore.Save(st)
}

func (s *Server) GetProviders() []settingsmodels.ProviderSettings {
	return s.registry.GetProviders()
}
