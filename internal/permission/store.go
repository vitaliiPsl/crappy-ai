package permission

import (
	"context"
	"fmt"

	"github.com/vitaliiPsl/crappy-ai/internal/config"
	"github.com/vitaliiPsl/crappy-ai/internal/permission/model"
)

type Store struct {
	config *config.Store
}

func NewStore(configStore *config.Store) *Store {
	return &Store{config: configStore}
}

func (s *Store) Load(_ context.Context) (model.Permissions, error) {
	return s.config.Get().Permissions, nil
}

func (s *Store) Add(_ context.Context, decision model.Decision, rule model.Rule) error {
	cfg := s.config.Get()
	cfg.Permissions.Add(decision, rule)

	if err := s.config.Save(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	return nil
}
