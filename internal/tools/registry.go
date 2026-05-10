package tools

import (
	"fmt"
	"sort"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/tools/bash"
	filesystem "github.com/vitaliiPsl/crappy-ai/internal/tools/fs"
	"github.com/vitaliiPsl/crappy-ai/internal/tools/web"
)

type Registry struct {
	entries map[string]kit.Tool
}

func NewRegistry() *Registry {
	r := &Registry{entries: make(map[string]kit.Tool)}

	registerTools(r.entries,
		bash.NewBash(),
		web.NewFetch(),
		filesystem.NewReadFile(),
		filesystem.NewWriteFile(),
		filesystem.NewEditFile(),
		filesystem.NewListDirectory(),
	)

	return r
}

func (r *Registry) GetTool(name string) (kit.Tool, error) {
	t, ok := r.entries[name]
	if !ok {
		return nil, fmt.Errorf("unknown tool %q — available: %v", name, r.names())
	}

	return t, nil
}

func (r *Registry) GetTools() []kit.Tool {
	names := r.names()

	tools := make([]kit.Tool, 0, len(names))
	for _, name := range names {
		tools = append(tools, r.entries[name])
	}

	return tools
}

func registerTools(entries map[string]kit.Tool, tools ...kit.Tool) {
	for _, tool := range tools {
		name := tool.Definition().Name
		if _, exists := entries[name]; exists {
			panic(fmt.Sprintf("tool %q already registered", name))
		}

		entries[name] = tool
	}
}

func (r *Registry) names() []string {
	names := make([]string, 0, len(r.entries))
	for name := range r.entries {
		names = append(names, name)
	}

	sort.Strings(names)

	return names
}
