package runtime

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/vitaliiPsl/crappy-adk/kit"

	appagent "github.com/vitaliiPsl/crappy-ai/internal/agent"
	"github.com/vitaliiPsl/crappy-ai/internal/ask"
	"github.com/vitaliiPsl/crappy-ai/internal/background"
	"github.com/vitaliiPsl/crappy-ai/internal/config"
	"github.com/vitaliiPsl/crappy-ai/internal/eventbus"
	"github.com/vitaliiPsl/crappy-ai/internal/mcp"
	"github.com/vitaliiPsl/crappy-ai/internal/models"
	"github.com/vitaliiPsl/crappy-ai/internal/permission"
	"github.com/vitaliiPsl/crappy-ai/internal/session"
	"github.com/vitaliiPsl/crappy-ai/internal/skills"

	bgext "github.com/vitaliiPsl/crappy-ai/internal/extensions/background"
	mcpext "github.com/vitaliiPsl/crappy-ai/internal/extensions/mcp"
	planningext "github.com/vitaliiPsl/crappy-ai/internal/extensions/planning"
	skillsext "github.com/vitaliiPsl/crappy-ai/internal/extensions/skills"
)

type Session struct {
	id string

	configStore  *config.Store
	sessionStore session.Store

	permissions *permission.Service

	modelRegistry *models.Registry
	skillRegistry *skills.Registry

	mcpManager        *mcp.Manager
	backgroundManager *background.Manager

	events  *eventbus.Bus[session.Event]
	prompts *ask.Broker

	mu     sync.Mutex
	cancel context.CancelFunc
}

func newSession(
	id string,
	configStore *config.Store,
	sessionStore session.Store,
	permissions *permission.Service,
	modelRegistry *models.Registry,
	skillRegistry *skills.Registry,
	mcpManager *mcp.Manager,
	backgroundManager *background.Manager,
) *Session {
	return &Session{
		id:                id,
		configStore:       configStore,
		sessionStore:      sessionStore,
		permissions:       permissions,
		modelRegistry:     modelRegistry,
		skillRegistry:     skillRegistry,
		mcpManager:        mcpManager,
		backgroundManager: backgroundManager,
		events:            eventbus.New[session.Event](),
		prompts:           ask.NewBroker(),
	}
}

func (s *Session) ID() string {
	return s.id
}

func (s *Session) Subscribe() *eventbus.Subscription[session.Event] {
	return s.events.Subscribe()
}

func (s *Session) Send(ctx context.Context, req Request) error {
	return s.start(ctx, func(turnCtx context.Context) error {
		return s.run(turnCtx, req)
	})
}

func (s *Session) Compact(ctx context.Context) error {
	return s.start(ctx, func(turnCtx context.Context) error {
		return s.compact(turnCtx)
	})
}

func (s *Session) Cancel() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cancel != nil {
		s.cancel()
	}
}

func (s *Session) start(ctx context.Context, fn func(context.Context) error) error {
	s.mu.Lock()
	if s.cancel != nil {
		s.mu.Unlock()

		return fmt.Errorf("session %q already has an active turn", s.id)
	}

	turnCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.mu.Unlock()

	go func() {
		defer func() {
			cancel()
			s.clearCancel()
		}()

		_ = fn(turnCtx)
	}()

	return nil
}

func (s *Session) run(ctx context.Context, req Request) error {
	cfg := s.configStore.Get()
	mem := newMemory(s.sessionStore, s.id)

	model, err := s.modelRegistry.Build(cfg.Provider, cfg.Model)
	if err != nil {
		return s.fail(fmt.Errorf("build model: %w", err))
	}

	ag, err := appagent.Build(ctx, appagent.Request{
		SessionID:  s.id,
		Config:     cfg,
		Model:      model,
		Memory:     mem,
		Asker:      s,
		Background: s.backgroundManager,
	},
		coreContributor{},
		permissionsContributor{s.permissions},
		compactionContributor{},
		bgext.New(s.backgroundManager),
		skillsext.New(s.skillRegistry),
		planningext.New(s.sessionStore),
		mcpext.New(s.mcpManager),
	)
	if err != nil {
		return s.fail(err)
	}

	input := kit.NewUserMessage(kit.NewTextContent(req.Text))
	s.events.Publish(session.NewMessageEvent(s.id, input))

	stream := ag.Stream(ctx, input)
	for ev := range stream.Iter() {
		if sev, ok := session.FromKitEvent(s.id, ev); ok {
			s.events.Publish(sev)
		}
	}

	resp, runErr := stream.Result()
	if runErr != nil {
		return s.fail(runErr)
	}

	s.finish(ctx, model.Config(), resp.Usage, resp.LastUsage)

	return nil
}

func (s *Session) compact(ctx context.Context) error {
	cfg := s.configStore.Get()

	model, err := s.modelRegistry.Build(cfg.Provider, cfg.Model)
	if err != nil {
		return s.fail(fmt.Errorf("build model: %w", err))
	}

	usage, err := summarize(ctx, model, newMemory(s.sessionStore, s.id))
	if err != nil {
		return s.fail(err)
	}

	s.finish(ctx, model.Config(), usage, usage)

	return nil
}

func (s *Session) fail(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		s.events.Publish(session.NewTurnCancelledEvent(s.id))

		return err
	}

	s.events.Publish(session.NewErrorEvent(s.id, err))

	return err
}

func (s *Session) finish(ctx context.Context, modelConfig kit.ModelConfig, usage, lastUsage kit.Usage) {
	total := usage
	if sess, err := s.sessionStore.Get(ctx, s.id); err == nil {
		sess.Usage.Add(usage)
		_ = s.sessionStore.Save(ctx, sess)
		total = sess.Usage
	}

	s.events.Publish(session.NewTurnCompleteEvent(s.id, session.TurnStats{
		Usage:         total,
		ContextUsed:   lastUsage.InputTokens,
		ContextWindow: int64(modelConfig.InputLimit),
	}))
}

func (s *Session) clearCancel() {
	s.mu.Lock()
	s.cancel = nil
	s.mu.Unlock()
}
