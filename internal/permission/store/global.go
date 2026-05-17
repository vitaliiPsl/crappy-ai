package store

import (
	"context"
	"fmt"

	"github.com/vitaliiPsl/crappy-ai/internal/config"
	"github.com/vitaliiPsl/crappy-ai/internal/permission"
)

type Global struct {
	config *config.Store
}

func NewGlobal(configStore *config.Store) *Global {
	return &Global{config: configStore}
}

func (s *Global) Load(_ context.Context) (permission.Permissions, error) {
	return s.config.Get().Permissions, nil
}

func (s *Global) Add(_ context.Context, decision permission.Decision, rule permission.Rule) error {
	cfg := s.config.Get()
	cfg.Permissions.Add(decision, rule)

	if err := s.config.Save(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	return nil
}
