package runtime

import (
	"context"
	"fmt"
	"strings"

	"github.com/vitaliiPsl/crappy-adk/kit"

	mcpcore "github.com/vitaliiPsl/crappy-ai/internal/mcp"
	"github.com/vitaliiPsl/crappy-ai/internal/session"
	"github.com/vitaliiPsl/crappy-ai/internal/skills"
)

type InputProcessor struct {
	sessionID     string
	skillRegistry *skills.Registry
	mcpManager    *mcpcore.Manager
}

func NewInputProcessor(sessionID string, skillRegistry *skills.Registry, mcpManager *mcpcore.Manager) *InputProcessor {
	return &InputProcessor{
		sessionID:     sessionID,
		skillRegistry: skillRegistry,
		mcpManager:    mcpManager,
	}
}

func (p *InputProcessor) Process(ctx context.Context, req Request) (kit.Message, session.Event, error) {
	text := req.Text
	content := []kit.Content{kit.NewTextContent(text)}

	if req.Skill != nil {
		resolved, err := p.resolveSkill(*req.Skill)
		if err != nil {
			return kit.Message{}, session.Event{}, err
		}

		text = resolved
		content = []kit.Content{kit.NewTextContent(text)}
	}

	if req.MCPPrompt != nil {
		resolved, err := p.resolveMCPPrompt(ctx, *req.MCPPrompt)
		if err != nil {
			return kit.Message{}, session.Event{}, err
		}

		content = resolved
	}

	msg := kit.NewUserMessage(content...)

	event := session.NewMessageEvent(p.sessionID, msg)
	if req.Skill != nil {
		event = session.NewSkillMessageEvent(p.sessionID, msg, session.SkillInvocation{
			Name: req.Skill.Name,
			Args: req.Skill.Args,
		})
	}

	if req.MCPPrompt != nil {
		event = session.NewMCPPromptMessageEvent(p.sessionID, msg, session.MCPPromptInvocation{
			Server: req.MCPPrompt.Server,
			Name:   req.MCPPrompt.Name,
			Args:   req.MCPPrompt.Args,
		})
	}

	return msg, event, nil
}

func (p *InputProcessor) resolveSkill(invocation SkillInvocation) (string, error) {
	if p.skillRegistry == nil {
		return "", fmt.Errorf("load skill %q: skill registry is not configured", invocation.Name)
	}

	skill, err := p.skillRegistry.GetSkill(invocation.Name)
	if err != nil {
		return "", fmt.Errorf("load skill %q: %w", invocation.Name, err)
	}

	return skills.FormatLoaded(skill, strings.Join(invocation.Args, " ")), nil
}

func (p *InputProcessor) resolveMCPPrompt(ctx context.Context, prompt MCPPromptInvocation) ([]kit.Content, error) {
	if p.mcpManager == nil {
		return nil, fmt.Errorf("load mcp prompt %q from %q: mcp manager is not configured", prompt.Name, prompt.Server)
	}

	messages, err := p.mcpManager.GetPrompt(ctx, prompt.Server, prompt.Name, prompt.Args)
	if err != nil {
		return nil, fmt.Errorf("load mcp prompt %q from %q: %w", prompt.Name, prompt.Server, err)
	}

	content := mcpPromptContent(messages)
	if len(content) == 0 {
		return nil, fmt.Errorf("load mcp prompt %q from %q: returned no content", prompt.Name, prompt.Server)
	}

	return content, nil
}

func mcpPromptContent(messages []kit.Message) []kit.Content {
	var out []kit.Content
	for _, message := range messages {
		for _, item := range message.Content {
			if item.Type == "" {
				continue
			}

			out = append(out, item)
		}
	}

	return out
}
