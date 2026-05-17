package permission

import (
	"context"
	"errors"
	"fmt"

	"github.com/vitaliiPsl/crappy-adk/kit"
)

var ErrDenied = errors.New("permission denied")

type Store interface {
	Load(ctx context.Context) (Permissions, error)
	Add(ctx context.Context, decision Decision, rule Rule) error
}

type Handler interface {
	Ask(ctx context.Context, sessionID string, call kit.ToolCall) (Response, error)
}

type Service struct {
	store   Store
	handler Handler
}

func NewService(store Store, handler Handler) *Service {
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

	permissions := Merge(Permissions{Default: Ask}, global)

	switch Resolve(permissions, call) {
	case Allow:
		return nil
	case Deny:
		return fmt.Errorf("denied by permission rules: %w", ErrDenied)
	default:
		return s.ask(ctx, sessionID, call)
	}
}

func (s *Service) ask(ctx context.Context, sessionID string, call kit.ToolCall) error {
	if s.handler == nil {
		return fmt.Errorf("tool %q requires permission but no handler is configured", call.Name)
	}

	resp, err := s.handler.Ask(ctx, sessionID, call)
	if err != nil {
		return err
	}

	if err := validateResponse(resp); err != nil {
		return err
	}

	if resp.Scope == ScopeGlobal {
		rule := Rule{Tool: call.Name, Pattern: resp.Pattern}
		if err := s.store.Add(ctx, resp.Decision, rule); err != nil {
			return fmt.Errorf("save permission: %w", err)
		}
	}

	if resp.Decision == Deny {
		return fmt.Errorf("denied by user: %w", ErrDenied)
	}

	return nil
}

func validateResponse(resp Response) error {
	switch resp.Decision {
	case Allow, Deny:
	default:
		return fmt.Errorf("invalid permission response decision %q", resp.Decision)
	}

	switch resp.Scope {
	case ScopeOnce, ScopeGlobal:
	default:
		return fmt.Errorf("invalid permission response scope %q", resp.Scope)
	}

	return nil
}
