package command

import (
	"context"
	"sort"
)

type Registry struct {
	commands map[string]Command
}

type Provider interface {
	Commands(ctx context.Context) []Command
}

func NewRegistry(ctx context.Context, providers ...Provider) *Registry {
	if ctx == nil {
		ctx = context.Background()
	}

	r := &Registry{commands: make(map[string]Command)}

	r.Register(NewNewCommand())
	r.Register(NewSessionsCommand())
	r.Register(NewSettingsCommand())
	r.Register(NewMCPCommand())
	r.Register(NewJobsCommand())
	r.Register(NewCompactCommand())
	r.Register(NewForkCommand())
	r.Register(NewAttachCommand())
	r.Register(NewHelpCommand(r))

	for _, provider := range providers {
		if provider == nil {
			continue
		}

		for _, cmd := range provider.Commands(ctx) {
			r.Register(cmd)
		}
	}

	return r
}

func (r *Registry) Register(cmd Command) {
	name := cmd.Definition().Name
	if _, exists := r.commands[name]; exists {
		return
	}

	r.commands[name] = cmd
}

func (r *Registry) Definitions() []Definition {
	defs := make([]Definition, 0, len(r.commands))
	for _, cmd := range r.commands {
		defs = append(defs, cmd.Definition())
	}

	sort.Slice(defs, func(i, j int) bool {
		if defs[i].Kind != defs[j].Kind {
			return defs[i].Kind < defs[j].Kind
		}

		return defs[i].Name < defs[j].Name
	})

	return defs
}

func (r *Registry) Get(name string) (Command, bool) {
	cmd, ok := r.commands[name]

	return cmd, ok
}
