package permission

import (
	"context"
	"errors"
	"fmt"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/permission/model"
	"github.com/vitaliiPsl/crappy-ai/internal/permission/strategy"
)

var ErrDenied = errors.New("permission denied")

type Handler interface {
	Ask(ctx context.Context, sessionID string, request model.AskRequest) (model.AskResponse, error)
}

type Service struct {
	store   *Store
	handler Handler
}

func NewService(store *Store, handler Handler) *Service {
	return &Service{
		store:   store,
		handler: handler,
	}
}

func (s *Service) SetHandler(h Handler) {
	s.handler = h
}

func (s *Service) Authorize(ctx context.Context, sessionID string, call kit.ToolCall) error {
	global, err := s.store.Load(ctx)
	if err != nil {
		return fmt.Errorf("load permissions: %w", err)
	}

	permissions := model.Merge(model.Permissions{Default: model.Ask}, global)

	result := strategy.Resolve(permissions, call)
	switch result.Decision {
	case model.Allow:
		return nil
	case model.Deny:
		return fmt.Errorf("denied by permission rules: %w", ErrDenied)
	case model.Ask:
		if result.AskRequest == nil {
			return fmt.Errorf("permission ask decision missing ask request")
		}

		return s.ask(ctx, sessionID, *result.AskRequest)
	default:
		return fmt.Errorf("invalid permission decision %q", result.Decision)
	}
}

func (s *Service) ask(ctx context.Context, sessionID string, request model.AskRequest) error {
	if s.handler == nil {
		return fmt.Errorf("tool %q requires permission but no handler is configured", request.Call.Name)
	}

	resp, err := s.handler.Ask(ctx, sessionID, request)
	if err != nil {
		return err
	}

	option, ok := request.Option(resp.OptionID)
	if !ok {
		return fmt.Errorf("invalid permission response option %q", resp.OptionID)
	}

	if option.Scope == model.ScopeGlobal && option.Rule != nil {
		if err := s.store.Add(ctx, option.Decision, *option.Rule); err != nil {
			return fmt.Errorf("save permission: %w", err)
		}
	}

	if option.Decision == model.Deny {
		return fmt.Errorf("denied by user: %w", ErrDenied)
	}

	return nil
}
