package assistant

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/vitaliiPsl/crappy-adk/agent"
	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/config"
	"github.com/vitaliiPsl/crappy-ai/internal/models"
	"github.com/vitaliiPsl/crappy-ai/internal/session"
)

type Assistant struct {
	configStore  *config.Store
	sessionStore session.Store
	registry     *models.Registry
}

func New(configStore *config.Store, sessionStore session.Store, registry *models.Registry) *Assistant {
	return &Assistant{
		configStore:  configStore,
		sessionStore: sessionStore,
		registry:     registry,
	}
}

func (a *Assistant) Run(ctx context.Context, sessionID, text string) (*kit.Stream[session.Event, struct{}], error) {
	sess, err := a.sessionStore.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	history, err := a.loadHistory(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("load history: %w", err)
	}

	userMsg := kit.NewUserMessage([]kit.Content{kit.NewTextContent(text)})
	history = append(history, userMsg)

	cfg := a.configStore.Get()

	model, err := a.registry.Build(cfg)
	if err != nil {
		return nil, fmt.Errorf("build model: %w", err)
	}

	ag, err := agent.New(model, buildAgentOpts(cfg, sess)...)
	if err != nil {
		return nil, fmt.Errorf("build agent: %w", err)
	}

	return kit.NewStream(func(emit kit.Emitter[session.Event]) (struct{}, error) {
		userEvent := session.NewMessageEvent(sessionID, userMsg)
		a.persistEvents(ctx, sessionID, userEvent)

		if err := emit.Emit(userEvent); err != nil {
			return struct{}{}, err
		}

		stream := ag.Stream(ctx, history)

		for kitEvent := range stream.Iter() {
			ev, ok := session.FromKitEvent(sessionID, kitEvent)
			if !ok {
				continue
			}

			if ev.Persistent() {
				a.persistEvents(ctx, sessionID, ev)
			}

			if err := emit.Emit(ev); err != nil {
				return struct{}{}, err
			}
		}

		resp, err := stream.Result()
		if err != nil {
			ev := cancelledOrError(sessionID, err)
			if ev.Persistent() {
				a.persistEvents(ctx, sessionID, ev)
			}

			if emitErr := emit.Emit(ev); emitErr != nil {
				return struct{}{}, emitErr
			}

			return struct{}{}, nil
		}

		stats := a.handleResult(ctx, sess, resp)

		if err := emit.Emit(session.NewTurnCompleteEvent(sessionID, stats)); err != nil {
			return struct{}{}, err
		}

		return struct{}{}, nil
	}), nil
}

func (a *Assistant) loadHistory(ctx context.Context, sessionID string) ([]kit.Message, error) {
	events, err := a.sessionStore.LoadEvents(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	start := 0
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].Type != session.EventMessage || events[i].Message == nil {
			continue
		}

		if isSummary(*events[i].Message) {
			start = i

			break
		}
	}

	var msgs []kit.Message
	for _, e := range events[start:] {
		if e.Type == session.EventMessage && e.Message != nil {
			msgs = append(msgs, *e.Message)
		}
	}

	return msgs, nil
}

func (a *Assistant) persistEvents(ctx context.Context, sessionID string, events ...session.Event) {
	if err := a.sessionStore.AppendEvents(ctx, sessionID, events...); err != nil {
		fmt.Fprintf(os.Stderr, "append events: %v\n", err)
	}
}

func (a *Assistant) handleResult(ctx context.Context, sess *session.Session, resp kit.AgentResponse) session.TurnStats {
	sess.Usage.Add(resp.Usage)

	if err := a.sessionStore.Save(ctx, sess); err != nil {
		fmt.Fprintf(os.Stderr, "warning: save session usage: %v\n", err)
	}

	return session.TurnStats{
		Usage:       sess.Usage,
		ContextUsed: resp.Usage.InputTokens,
	}
}

func buildAgentOpts(cfg config.Config, _ *session.Session) []agent.Option {
	sources := []string{cfg.SystemPrompt}

	opts := []agent.Option{
		agent.WithInstructions(sources...),
	}

	if cfg.Thinking != "" {
		opts = append(opts, agent.WithThinking(kit.ThinkingLevel(cfg.Thinking)))
	}

	return opts
}

func isSummary(msg kit.Message) bool {
	for _, c := range msg.Content {
		if c.Type == kit.ContentTypeSummary {
			return true
		}
	}

	return false
}

func cancelledOrError(sessionID string, err error) session.Event {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return session.NewTurnCancelledEvent(sessionID)
	}

	return session.NewErrorEvent(sessionID, err)
}
