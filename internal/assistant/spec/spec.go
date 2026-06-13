package spec

import (
	"context"
	"strings"

	"github.com/vitaliiPsl/crappy-adk/agent"
	"github.com/vitaliiPsl/crappy-adk/kit"
)

type AgentSpec struct {
	Context []ContextPiece
	Tools   []ToolSpec
	Hooks   []HookSpec
}

func (s *AgentSpec) Merge(other AgentSpec) {
	s.Context = append(s.Context, other.Context...)
	s.Tools = append(s.Tools, other.Tools...)
	s.Hooks = append(s.Hooks, other.Hooks...)
}

type ContextKind string

const (
	ContextSystemPrompt ContextKind = "system_prompt"
	ContextEnvironment  ContextKind = "environment"
	ContextInstructions ContextKind = "instructions"
	ContextMemory       ContextKind = "memory"
	ContextArtifact     ContextKind = "artifact"
	ContextExtension    ContextKind = "extension"
)

type ContextPiece struct {
	Name   string
	Source string
	Kind   ContextKind

	Content string
	Resolve func(context.Context) (string, error)
}

type ToolSpec struct {
	Source string
	Tool   kit.Tool
}

func (t ToolSpec) Name() string {
	if t.Tool == nil {
		return ""
	}

	return t.Tool.Definition().Name
}

type HookKind string

const (
	HookTurnStart     HookKind = "turn_start"
	HookTurnEnd       HookKind = "turn_end"
	HookModelRequest  HookKind = "model_request"
	HookModelResponse HookKind = "model_response"
	HookToolCall      HookKind = "tool_call"
	HookToolResult    HookKind = "tool_result"
)

type HookSpec struct {
	Name   string
	Source string
	Kind   HookKind

	Option agent.Option
}

func (p ContextPiece) ResolveContent(ctx context.Context) (string, error) {
	if p.Resolve == nil {
		return strings.TrimSpace(p.Content), nil
	}

	content, err := p.Resolve(ctx)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(content), nil
}
