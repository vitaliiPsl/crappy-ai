package server

import (
	"github.com/vitaliiPsl/crappy-ai/internal/config"
	"github.com/vitaliiPsl/crappy-ai/internal/settings"
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

func (s *Server) GetProviders() []settings.ProviderSettings {
	return s.registry.GetProviders()
}
