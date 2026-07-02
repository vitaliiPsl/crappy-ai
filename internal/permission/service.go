package permission

import (
	"context"
	"errors"
	"fmt"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/ask"
	"github.com/vitaliiPsl/crappy-ai/internal/config"
	"github.com/vitaliiPsl/crappy-ai/internal/permission/model"
	"github.com/vitaliiPsl/crappy-ai/internal/permission/strategy"
)

var ErrDenied = errors.New("permission denied")

type Context struct {
	Mode        config.Mode
	Permissions model.Permissions

	Asker ask.Asker
}

type Service struct {
	config *config.Store
}

func NewService(configStore *config.Store) *Service {
	return &Service{config: configStore}
}

func (s *Service) Authorize(ctx context.Context, auth Context, call kit.ToolCall) error {
	switch auth.Mode {
	case config.ModeDefault:
		return s.authorizeDefault(ctx, auth, call)
	case config.ModeYolo:
		return nil
	default:
		return fmt.Errorf("invalid mode %q", auth.Mode)
	}
}

func (s *Service) authorizeDefault(ctx context.Context, auth Context, call kit.ToolCall) error {
	result := strategy.Resolve(auth.Permissions, call)
	switch result.Decision {
	case model.Allow:
		return nil
	case model.Deny:
		return fmt.Errorf("denied by permission rules: %w", ErrDenied)
	case model.Ask:
		if result.Prompt == nil {
			return fmt.Errorf("permission ask decision missing prompt")
		}

		return s.ask(ctx, auth, *result.Prompt)
	default:
		return fmt.Errorf("invalid permission decision %q", result.Decision)
	}
}

func (s *Service) ask(ctx context.Context, auth Context, request model.Prompt) error {
	optionID, err := s.prompt(ctx, auth, request)
	if err != nil {
		return err
	}

	option, ok := request.Option(optionID)
	if !ok {
		return fmt.Errorf("invalid permission response option %q", optionID)
	}

	if option.Scope == model.ScopeGlobal && option.Rule != nil {
		if err := s.save(option.Decision, *option.Rule); err != nil {
			return fmt.Errorf("save permission: %w", err)
		}
	}

	if option.Decision == model.Deny {
		return fmt.Errorf("denied by user: %w", ErrDenied)
	}

	return nil
}

func (s *Service) prompt(ctx context.Context, auth Context, request model.Prompt) (string, error) {
	if auth.Asker == nil {
		return "", fmt.Errorf("tool %q requires permission but the run cannot prompt: %w", request.Call.Name, ErrDenied)
	}

	resp, err := auth.Asker.Ask(ctx, request.Request)
	if err != nil {
		return "", err
	}

	return resp.OptionID, nil
}

func (s *Service) save(decision model.Decision, rule model.Rule) error {
	cfg := s.config.Get()
	cfg.Permissions.Add(decision, rule)

	if err := s.config.Save(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	return nil
}
