package background

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/vitaliiPsl/crappy-adk/kit"
)

type wrappedTool struct {
	def kit.ToolDefinition
	run func(rc *kit.RunContext, input map[string]any) (string, error)
}

func (t wrappedTool) Definition() kit.ToolDefinition {
	return t.def
}

func (t wrappedTool) Execute(rc *kit.RunContext, input map[string]any) (string, error) {
	return t.run(rc, input)
}

func Wrap(tool kit.Tool, manager *Manager) (kit.Tool, error) {
	if manager == nil {
		return nil, errors.New("background manager is required")
	}

	def := tool.Definition()

	schema, err := addBackgroundArgument(def.Schema)
	if err != nil {
		return nil, fmt.Errorf("wrap %q: %w", def.Name, err)
	}

	def.Schema = schema

	return wrappedTool{
		def: def,
		run: func(rc *kit.RunContext, input map[string]any) (string, error) {
			background, args, err := splitBackground(input)
			if err != nil {
				return "", err
			}

			if !background {
				return tool.Execute(rc, args)
			}

			if err := rc.Err(); err != nil {
				return "", err
			}

			job, err := manager.Start(tool.Definition().Name, func(ctx context.Context) (string, error) {
				jobRC := *rc
				jobRC.Context = ctx
				jobRC.Events = kit.NoopEmitter[kit.AgentEvent]{}

				return tool.Execute(&jobRC, args)
			})
			if err != nil {
				return "", err
			}

			data, err := json.Marshal(job)
			if err != nil {
				return "", err
			}

			return string(data), nil
		},
	}, nil
}

func splitBackground(input map[string]any) (bool, map[string]any, error) {
	args := make(map[string]any, len(input))

	var background bool
	for key, value := range input {
		if key != ArgName {
			args[key] = value

			continue
		}

		boolValue, ok := value.(bool)
		if !ok {
			return false, nil, fmt.Errorf("%q must be a boolean", ArgName)
		}

		background = boolValue
	}

	return background, args, nil
}

func addBackgroundArgument(schema map[string]any) (map[string]any, error) {
	out := cloneSchema(schema)
	if out == nil {
		out = make(map[string]any)
	}

	if _, ok := out["type"]; !ok {
		out["type"] = "object"
	}

	props, ok := out["properties"].(map[string]any)
	if !ok {
		props = make(map[string]any)
		out["properties"] = props
	}

	if _, exists := props[ArgName]; exists {
		return nil, fmt.Errorf("tool schema already defines %q", ArgName)
	}

	props[ArgName] = map[string]any{
		"type":        "boolean",
		"description": "Set to true to run this tool as a background job and return immediately with a job ID.",
		"default":     false,
	}

	return out, nil
}

func cloneSchema(schema map[string]any) map[string]any {
	if schema == nil {
		return nil
	}

	data, err := json.Marshal(schema)
	if err != nil {
		return nil
	}

	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil
	}

	return out
}
