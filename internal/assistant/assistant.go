package assistant

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/vitaliiPsl/crappy-adk/agent"
	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/assistant/extension"
	"github.com/vitaliiPsl/crappy-ai/internal/assistant/extension/planning"
	"github.com/vitaliiPsl/crappy-ai/internal/assistant/extension/summarization"
	"github.com/vitaliiPsl/crappy-ai/internal/assistant/memory"
	"github.com/vitaliiPsl/crappy-ai/internal/assistant/spec"
	"github.com/vitaliiPsl/crappy-ai/internal/background"
	"github.com/vitaliiPsl/crappy-ai/internal/config"
	"github.com/vitaliiPsl/crappy-ai/internal/mcp"
	"github.com/vitaliiPsl/crappy-ai/internal/models"
	"github.com/vitaliiPsl/crappy-ai/internal/permission"
	"github.com/vitaliiPsl/crappy-ai/internal/session"
	"github.com/vitaliiPsl/crappy-ai/internal/settings"
	"github.com/vitaliiPsl/crappy-ai/internal/skills"

	bgext "github.com/vitaliiPsl/crappy-ai/internal/assistant/extension/background"
	mcpext "github.com/vitaliiPsl/crappy-ai/internal/assistant/extension/mcp"
	skillsext "github.com/vitaliiPsl/crappy-ai/internal/assistant/extension/skills"
)

type Assistant struct {
	configStore   *config.Store
	settingsStore *settings.Store
	sessionStore  session.Store
	modelRegistry *models.Registry
	skillRegistry *skills.Registry
	permissions   *permission.Service
	background    *background.Manager

	extensions []extension.Extension
}

func New(
	configStore *config.Store,
	settingsStore *settings.Store,
	sessionStore session.Store,
	artifactStore session.ArtifactStore,
	modelRegistry *models.Registry,
	skillRegistry *skills.Registry,
	permissions *permission.Service,
	backgroundManager *background.Manager,
	mcpManager *mcp.Manager,
) *Assistant {
	return &Assistant{
		configStore:   configStore,
		settingsStore: settingsStore,
		sessionStore:  sessionStore,
		modelRegistry: modelRegistry,
		skillRegistry: skillRegistry,
		permissions:   permissions,
		background:    backgroundManager,
		extensions: []extension.Extension{
			summarization.New(),
			bgext.New(backgroundManager),
			planning.New(artifactStore),
			skillsext.New(skillRegistry),
			mcpext.New(mcpManager),
		},
	}
}

func (a *Assistant) Run(ctx context.Context, sessionID string, req RunRequest) (*kit.Stream[session.Event, struct{}], error) {
	cfg := a.configStore.Get()

	mem := memory.New(a.sessionStore, sessionID)

	model, err := a.modelRegistry.Build(cfg)
	if err != nil {
		return nil, fmt.Errorf("build model: %w", err)
	}

	runSpec, err := a.buildAgentSpec(extension.Context{Ctx: ctx, SessionID: sessionID, Config: cfg, Model: model})
	if err != nil {
		return nil, fmt.Errorf("build agent spec: %w", err)
	}

	compiled, err := spec.Compile(runSpec)
	if err != nil {
		return nil, fmt.Errorf("compile agent spec: %w", err)
	}

	if cfg.Thinking != "" {
		compiled.Options = append(compiled.Options, agent.WithThinking(kit.ThinkingLevel(cfg.Thinking)))
	}

	ag, err := agent.New(model, mem, compiled.Tools, compiled.Options...)
	if err != nil {
		return nil, fmt.Errorf("build agent: %w", err)
	}

	userMsg, userEvent, err := a.buildUserInput(sessionID, req)
	if err != nil {
		return nil, err
	}

	return kit.NewStream(func(emit kit.Emitter[session.Event]) (struct{}, error) {
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

func (a *Assistant) buildUserInput(sessionID string, req RunRequest) (kit.Message, session.Event, error) {
	text := req.Text
	if req.Skill != nil {
		skill, err := a.skillRegistry.GetSkill(req.Skill.Name)
		if err != nil {
			return kit.Message{}, session.Event{}, fmt.Errorf("load skill %q: %w", req.Skill.Name, err)
		}

		text = skills.FormatLoaded(skill, strings.Join(req.Skill.Args, " "))
	}

	msg := kit.NewUserMessage(kit.NewTextContent(text))

	event := session.NewMessageEvent(sessionID, msg)
	if req.Skill != nil {
		event = session.NewSkillMessageEvent(sessionID, msg, session.SkillInvocation{
			Name: req.Skill.Name,
			Args: req.Skill.Args,
		})
	}

	return msg, event, nil
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
