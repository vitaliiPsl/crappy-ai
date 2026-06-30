package runtime

import (
	"context"
	"fmt"

	"github.com/vitaliiPsl/crappy-adk/agent"
	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/ask"
	"github.com/vitaliiPsl/crappy-ai/internal/config"
	"github.com/vitaliiPsl/crappy-ai/internal/permission"
	"github.com/vitaliiPsl/crappy-ai/internal/session"
)

func (s *Session) permissionOption(cfg config.Config) agent.Option {
	auth := permission.Context{
		SessionID:   s.id,
		Mode:        cfg.Mode,
		Permissions: cfg.Permissions,
		Asker:       s,
	}

	return agent.WithOnToolCall(func(rc *kit.RunContext, call kit.ToolCall) (kit.ToolCall, error) {
		if err := s.permissions.Authorize(rc.Context, auth, call); err != nil {
			return call, err
		}

		return call, nil
	})
}

func (s *Session) Ask(ctx context.Context, req ask.Request) (ask.Response, error) {
	return s.prompts.Await(ctx, req, func(r ask.Request) {
		s.events.Publish(session.NewAskEvent(s.id, r))
	})
}

func (s *Session) Respond(resp ask.Response) error {
	if !s.prompts.Resolve(resp) {
		return fmt.Errorf("no pending request %q", resp.RequestID)
	}

	return nil
}
