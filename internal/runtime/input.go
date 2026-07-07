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

	if req.Skill != nil {
		resolved, err := p.resolveSkill(*req.Skill)
		if err != nil {
			return kit.Message{}, session.Event{}, err
		}

		text = resolved
	}

	if req.MCPPrompt != nil {
		resolved, err := p.resolveMCPPrompt(ctx, *req.MCPPrompt)
		if err != nil {
			return kit.Message{}, session.Event{}, err
		}

		text = resolved
	}

	msg := kit.NewUserMessage(kit.NewTextContent(text))

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

func (p *InputProcessor) resolveMCPPrompt(ctx context.Context, prompt MCPPromptInvocation) (string, error) {
	if p.mcpManager == nil {
		return "", fmt.Errorf("load mcp prompt %q from %q: mcp manager is not configured", prompt.Name, prompt.Server)
	}

	result, err := p.mcpManager.GetPrompt(ctx, prompt.Server, prompt.Name, prompt.Args)
	if err != nil {
		return "", fmt.Errorf("load mcp prompt %q from %q: %w", prompt.Name, prompt.Server, err)
	}

	text := formatMCPPromptResult(result)
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("load mcp prompt %q from %q: returned no text", prompt.Name, prompt.Server)
	}

	return text, nil
}

func formatMCPPromptResult(result mcpcore.PromptResult) string {
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

func mcpPromptContentText(content mcpcore.PromptContent) string {
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
