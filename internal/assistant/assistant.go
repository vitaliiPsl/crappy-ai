package assistant

import (
	"context"
	"errors"
	"fmt"

	"github.com/vitaliiPsl/crappy-adk/agent"
	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/config"
	"github.com/vitaliiPsl/crappy-ai/internal/models"
	"github.com/vitaliiPsl/crappy-ai/internal/session"
	"github.com/vitaliiPsl/crappy-ai/internal/tools"

	"github.com/vitaliiPsl/crappy-ai/internal/assistant/memory"
)

type Assistant struct {
	configStore   *config.Store
	sessionStore  session.Store
	modelRegistry *models.Registry
	toolRegistry  *tools.Registry
}

func New(
	configStore *config.Store,
	sessionStore session.Store,
	modelRegistry *models.Registry,
	toolRegistry *tools.Registry,
) *Assistant {
	return &Assistant{
		configStore:   configStore,
		sessionStore:  sessionStore,
		modelRegistry: modelRegistry,
		toolRegistry:  toolRegistry,
	}
}

func (a *Assistant) Run(ctx context.Context, sessionID, text string) (*kit.Stream[session.Event, struct{}], error) {
	cfg := a.configStore.Get()

	model, err := a.modelRegistry.Build(cfg)
	if err != nil {
		return nil, fmt.Errorf("build model: %w", err)
	}

	mem := memory.New(a.sessionStore, sessionID)

	ag, err := agent.New(model, mem, a.buildAgentOpts(cfg, model)...)
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

		var ev session.Event
		var err error
		if runErr != nil {
			ev, err = a.handleRunError(ctx, sessionID, runErr)
		} else {
			ev, err = a.handleRunResult(ctx, sessionID, model.Config(), resp.Usage)
		}
		if err != nil {
			return struct{}{}, err
		}

		if err := emit.Emit(ev); err != nil {
			return struct{}{}, err
		}

		return struct{}{}, nil
	}), nil
}

func (a *Assistant) buildAgentOpts(cfg config.Config, model kit.Model) []agent.Option {
	sources := []string{cfg.SystemPrompt}

	opts := []agent.Option{
		agent.WithInstructions(sources...),
		agent.WithTools(a.toolRegistry.GetTools()...),
		newSessionSummarization(model),
	}

	if cfg.Thinking != "" {
		opts = append(opts, agent.WithThinking(kit.ThinkingLevel(cfg.Thinking)))
	}

	return opts
}

func (a *Assistant) handleRunResult(
	ctx context.Context,
	sessionID string,
	modelConfig kit.ModelConfig,
	usage kit.Usage,
) (session.Event, error) {
	sess, err := a.sessionStore.Get(ctx, sessionID)
	if err != nil {
		return session.Event{}, fmt.Errorf("get session error: %w", err)
	}

	sess.Usage.Add(usage)

	if err := a.sessionStore.Save(ctx, sess); err != nil {
		return session.Event{}, fmt.Errorf("save session error: %w", err)
	}

	return session.NewTurnCompleteEvent(sess.ID, session.TurnStats{
		Usage:         sess.Usage,
		ContextUsed:   usage.InputTokens,
		ContextWindow: int64(modelConfig.InputLimit),
	}), nil
}

func (a *Assistant) handleRunError(
	ctx context.Context,
	sessionID string,
	runErr error,
) (session.Event, error) {
	if errors.Is(runErr, context.Canceled) || errors.Is(runErr, context.DeadlineExceeded) {
		return session.NewTurnCancelledEvent(sessionID), nil
	}

	ev := session.NewErrorEvent(sessionID, runErr)
	if appendErr := a.sessionStore.AppendEvents(ctx, sessionID, ev); appendErr != nil {
		return session.Event{}, fmt.Errorf("append error event: %v", appendErr)
	}

	return ev, nil
}
