package runtime

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	adk "github.com/vitaliiPsl/crappy-adk/agent"
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

func (s *Session) Run(ctx context.Context, req Request) error {
	return s.start(ctx, func(turnCtx context.Context) error {
		return s.run(turnCtx, req)
	})
}

func (s *Session) RunSubagent(ctx context.Context, req SubagentRequest) (SubagentResult, error) {
	return s.runSubagent(ctx, req)
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

	agent, err := s.buildAgent(ctx, s.id, cfg, model, mem)
	if err != nil {
		return s.fail(err)
	}

	input, event, err := s.buildAgentInput(ctx, req)
	if err != nil {
		return s.fail(err)
	}

	s.events.Publish(event)

	stream := agent.Stream(ctx, input)
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
	if updated, ok := s.recordUsage(ctx, s.id, usage); ok {
		total = updated
	}

	s.events.Publish(session.NewTurnCompleteEvent(s.id, session.TurnStats{
		Usage:         total,
		ContextUsed:   lastUsage.InputTokens,
		ContextWindow: int64(modelConfig.InputLimit),
	}))
}

func (s *Session) recordUsage(ctx context.Context, id string, usage kit.Usage) (kit.Usage, bool) {
	sess, err := s.sessionStore.Get(ctx, id)
	if err != nil {
		return kit.Usage{}, false
	}

	sess.Usage.Add(usage)
	_ = s.sessionStore.Save(ctx, sess)

	return sess.Usage, true
}

func (s *Session) buildAgentInput(ctx context.Context, req Request) (kit.Message, session.Event, error) {
	text := req.Text

	if req.Skill != nil {
		skill, err := s.skillRegistry.GetSkill(req.Skill.Name)
		if err != nil {
			return kit.Message{}, session.Event{}, fmt.Errorf("load skill %q: %w", req.Skill.Name, err)
		}

		text = skills.FormatLoaded(skill, strings.Join(req.Skill.Args, " "))
	}

	if req.MCPPrompt != nil {
		if s.mcpManager == nil {
			return kit.Message{}, session.Event{}, fmt.Errorf("load mcp prompt %q from %q: mcp manager is not configured", req.MCPPrompt.Name, req.MCPPrompt.Server)
		}

		result, err := s.mcpManager.GetPrompt(ctx, req.MCPPrompt.Server, req.MCPPrompt.Name, req.MCPPrompt.Args)
		if err != nil {
			return kit.Message{}, session.Event{}, fmt.Errorf("load mcp prompt %q from %q: %w", req.MCPPrompt.Name, req.MCPPrompt.Server, err)
		}

		text = formatMCPPromptResult(result)
		if strings.TrimSpace(text) == "" {
			return kit.Message{}, session.Event{}, fmt.Errorf("load mcp prompt %q from %q: returned no text", req.MCPPrompt.Name, req.MCPPrompt.Server)
		}
	}

	msg := kit.NewUserMessage(kit.NewTextContent(text))

	event := session.NewMessageEvent(s.id, msg)
	if req.Skill != nil {
		event = session.NewSkillMessageEvent(s.id, msg, session.SkillInvocation{
			Name: req.Skill.Name,
			Args: req.Skill.Args,
		})
	}

	if req.MCPPrompt != nil {
		event = session.NewMCPPromptMessageEvent(s.id, msg, session.MCPPromptInvocation{
			Server: req.MCPPrompt.Server,
			Name:   req.MCPPrompt.Name,
			Args:   req.MCPPrompt.Args,
		})
	}

	return msg, event, nil
}

func formatMCPPromptResult(result mcp.PromptResult) string {
	var parts []string
	for _, message := range result.Messages {
		for _, content := range message.Content {
			if text := mcpPromptContentText(content); text != "" {
				parts = append(parts, text)
			}
		}
	}

	return strings.Join(parts, "\n\n")
}

func mcpPromptContentText(content mcp.PromptContent) string {
	switch content.Type {
	case "text":
		return content.Text
	case "resource":
		if content.Resource != nil && content.Resource.Text != "" {
			return content.Resource.Text
		}

		if content.Text != "" {
			return content.Text
		}

		return fmt.Sprintf("[resource: %s, %s]", content.URI, content.MIMEType)
	case "resource_link":
		return fmt.Sprintf("[resource: %s]", content.URI)
	case "image":
		return fmt.Sprintf("[image: %s, %d bytes]", content.MIMEType, len(content.Data))
	case "audio":
		return fmt.Sprintf("[audio: %s, %d bytes]", content.MIMEType, len(content.Data))
	default:
		return ""
	}
}

func (s *Session) buildAgent(ctx context.Context, sessionID string, cfg config.Config, model kit.Model, mem kit.Memory) (*adk.Agent, error) {
	return appagent.Build(ctx, appagent.Request{
		SessionID: sessionID,
		Config:    cfg,
		Model:     model,
		Memory:    mem,
		Asker:     s,
	},
		coreContributor{
			background: s.backgroundManager,
		},
		permissionsContributor{
			s.permissions,
		},
		compactionContributor{},
		subagentsContributor{
			session:    s,
			background: s.backgroundManager,
		},
		bgext.New(s.backgroundManager),
		skillsext.New(s.skillRegistry),
		planningext.New(s.sessionStore),
		mcpext.New(s.mcpManager),
	)
}

func (s *Session) clearCancel() {
	s.mu.Lock()
	s.cancel = nil
	s.mu.Unlock()
}
