package runtime

import (
	"context"
	"fmt"

	adk "github.com/vitaliiPsl/crappy-adk/agent"
	"github.com/vitaliiPsl/crappy-adk/kit"

	appagent "github.com/vitaliiPsl/crappy-ai/internal/agent"
	"github.com/vitaliiPsl/crappy-ai/internal/ask"
	"github.com/vitaliiPsl/crappy-ai/internal/permission"
	"github.com/vitaliiPsl/crappy-ai/internal/session"
)

type permissionsContributor struct {
	service *permission.Service
}

func (c permissionsContributor) Contribute(_ context.Context, req appagent.Request) (appagent.Contribution, error) {
	if c.service == nil {
		return appagent.Contribution{}, nil
	}

	auth := permission.Context{
		SessionID:   req.SessionID,
		Mode:        req.Config.Mode,
		Permissions: req.Config.Permissions,
		Asker:       req.Asker,
	}

	option := adk.WithOnToolCall(func(rc *kit.RunContext, call kit.ToolCall) (kit.ToolCall, error) {
		if err := c.service.Authorize(rc.Context, auth, call); err != nil {
			return call, err
		}

		return call, nil
	})

	return appagent.Contribution{Options: []adk.Option{option}}, nil
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
