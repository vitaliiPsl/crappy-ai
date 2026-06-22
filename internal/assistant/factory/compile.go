package factory

import (
	"strings"

	"github.com/vitaliiPsl/crappy-adk/agent"
	"github.com/vitaliiPsl/crappy-adk/kit"
	"github.com/vitaliiPsl/crappy-adk/x/tool"
)

type Compiled struct {
	Tools   kit.ToolSet
	Options []agent.Option
}

func Compile(s AgentSpec) (Compiled, error) {
	compiled := Compiled{
		Tools: tool.NewSet(),
	}

	if instructions := staticContext(s.Context); instructions != "" {
		compiled.Options = append(compiled.Options, agent.WithInstructions(instructions))
	}

	if pieces := dynamicContextPieces(s.Context); len(pieces) > 0 {
		compiled.Options = append(compiled.Options, agent.WithOnModelRequest(resolveDynamicContext(pieces)))
	}

	if len(s.Tools) > 0 {
		tools := make([]kit.Tool, 0, len(s.Tools))
		for _, toolSpec := range s.Tools {
			tools = append(tools, toolSpec.Tool)
		}

		compiled.Tools = tool.NewSet(tools...)
	}

	for _, hook := range s.Hooks {
		compiled.Options = append(compiled.Options, hook.Option)
	}

	return compiled, nil
}

func staticContext(pieces []ContextPiece) string {
	var out []string
	for _, piece := range pieces {
		if piece.Resolve != nil {
			continue
		}

		content := strings.TrimSpace(piece.Content)
		if content != "" {
			out = append(out, content)
		}
	}

	return strings.Join(out, "\n\n")
}

func dynamicContextPieces(pieces []ContextPiece) []ContextPiece {
	var out []ContextPiece
	for _, piece := range pieces {
		if piece.Resolve != nil {
			out = append(out, piece)
		}
	}

	return out
}

func resolveDynamicContext(pieces []ContextPiece) kit.OnModelRequest {
	return func(rc *kit.RunContext, req kit.ModelRequest) (kit.ModelRequest, error) {
		for _, piece := range pieces {
			content, err := piece.ResolveContent(rc.Context)
			if err != nil {
				return kit.ModelRequest{}, err
			}

			req.Instructions = appendInstruction(req.Instructions, content)
		}

		return req, nil
	}
}

func appendInstruction(existing, addition string) string {
	addition = strings.TrimSpace(addition)
	if addition == "" {
		return existing
	}

	if existing == "" {
		return addition
	}

	return existing + "\n\n" + addition
}
