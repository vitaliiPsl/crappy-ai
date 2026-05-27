package assistant

import (
	"context"
	"errors"
	"fmt"

	"github.com/vitaliiPsl/crappy-adk/agent"
	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/assistant/memory"
	"github.com/vitaliiPsl/crappy-ai/internal/config"
	"github.com/vitaliiPsl/crappy-ai/internal/models"
	"github.com/vitaliiPsl/crappy-ai/internal/permission"
	"github.com/vitaliiPsl/crappy-ai/internal/session"
	coreskills "github.com/vitaliiPsl/crappy-ai/internal/skills"
	"github.com/vitaliiPsl/crappy-ai/internal/tools"
)

type Assistant struct {
	configStore   *config.Store
	sessionStore  session.Store
	artifactStore session.ArtifactStore
	modelRegistry *models.Registry
	skillRegistry *coreskills.Registry
	toolRegistry  *tools.Registry
	permissions   *permission.Service
}

func New(
	configStore *config.Store,
	sessionStore session.Store,
	artifactStore session.ArtifactStore,
	modelRegistry *models.Registry,
	skillRegistry *coreskills.Registry,
	toolRegistry *tools.Registry,
	permissions *permission.Service,
) *Assistant {
	return &Assistant{
		configStore:   configStore,
		sessionStore:  sessionStore,
		artifactStore: artifactStore,
		modelRegistry: modelRegistry,
		skillRegistry: skillRegistry,
		toolRegistry:  toolRegistry,
		permissions:   permissions,
	}
}

func (a *Assistant) Run(ctx context.Context, sessionID, text string) (*kit.Stream[session.Event, struct{}], error) {
	cfg := a.configStore.Get()

	model, err := a.modelRegistry.Build(cfg)
	if err != nil {
		return nil, fmt.Errorf("build model: %w", err)
	}

	mem := memory.New(a.sessionStore, sessionID)

	ag, err := agent.New(model, mem, a.buildAgentOpts(sessionID, cfg, model)...)
	if err != nil {
		return nil, fmt.Errorf("build agent: %w", err)
	}

	userMsg := kit.NewUserMessage(kit.NewTextContent(text))

	return kit.NewStream(func(emit kit.Emitter[session.Event]) (struct{}, error) {
		userEvent := session.NewMessageEvent(sessionID, userMsg)

		if err := emit.Emit(userEvent); err != nil {
			return struct{}{}, err
		}

		stream := ag.Stream(ctx, userMsg)
		for kitEvent := range stream.Iter() {
			ev, ok := session.FromKitEvent(sessionID, kitEvent)
			if !ok {
				continue
			}

			if err := emit.Emit(ev); err != nil {
				return struct{}{}, err
			}
		}

		resp, runErr := stream.Result()

		return struct{}{}, a.handleResult(ctx, sessionID, model.Config(), resp.Usage, resp.LastUsage, runErr, emit)
	}), nil
}

func (a *Assistant) handleResult(
	ctx context.Context,
	sessionID string,
	modelConfig kit.ModelConfig,
	usage, lastUsage kit.Usage,
	runErr error,
	emit kit.Emitter[session.Event],
) error {
	if runErr != nil {
		if errors.Is(runErr, context.Canceled) || errors.Is(runErr, context.DeadlineExceeded) {
			return emit.Emit(session.NewTurnCancelledEvent(sessionID))
		}

		ev := session.NewErrorEvent(sessionID, runErr)
		if err := a.sessionStore.AppendEvents(ctx, sessionID, ev); err != nil {
			return fmt.Errorf("append error event: %w", err)
		}

		return emit.Emit(ev)
	}

	sess, err := a.sessionStore.Get(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}

	sess.Usage.Add(usage)

	if err := a.sessionStore.Save(ctx, sess); err != nil {
		return fmt.Errorf("save session: %w", err)
	}

	return emit.Emit(session.NewTurnCompleteEvent(sess.ID, session.TurnStats{
		Usage:         sess.Usage,
		ContextUsed:   lastUsage.InputTokens,
		ContextWindow: int64(modelConfig.InputLimit),
	}))
}
