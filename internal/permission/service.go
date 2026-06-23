package permission

import (
	"context"
	"errors"
	"fmt"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/config"
	"github.com/vitaliiPsl/crappy-ai/internal/permission/model"
	"github.com/vitaliiPsl/crappy-ai/internal/permission/strategy"
)

var ErrDenied = errors.New("permission denied")

type Handler interface {
	Ask(ctx context.Context, sessionID string, request model.AskRequest) (model.AskResponse, error)
}

type Context struct {
	SessionID   string
	Mode        config.Mode
	Permissions model.Permissions
	Handler     Handler
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
		if result.AskRequest == nil {
			return fmt.Errorf("permission ask decision missing ask request")
		}

		return s.ask(ctx, auth, *result.AskRequest)
	default:
		return fmt.Errorf("invalid permission decision %q", result.Decision)
	}
}

func (s *Service) ask(ctx context.Context, auth Context, request model.AskRequest) error {
	if auth.Handler == nil {
		return fmt.Errorf("tool %q requires permission but the run cannot prompt: %w", request.Call.Name, ErrDenied)
	}

	resp, err := auth.Handler.Ask(ctx, auth.SessionID, request)
	if err != nil {
		return err
	}

	option, ok := request.Option(resp.OptionID)
	if !ok {
		return fmt.Errorf("invalid permission response option %q", resp.OptionID)
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

func (s *Service) save(decision model.Decision, rule model.Rule) error {
	cfg := s.config.Get()
	cfg.Permissions.Add(decision, rule)

	if err := s.config.Save(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	return nil
}
