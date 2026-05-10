package command

import (
	"fmt"
	"sort"
)

type Registry struct {
	commands map[string]Command
}

func NewRegistry() *Registry {
	return &Registry{commands: make(map[string]Command)}
}

func (r *Registry) Register(cmd Command) {
	name := cmd.Definition().Name
	if _, exists := r.commands[name]; exists {
		panic(fmt.Sprintf("command %q already registered", name))
	}

	r.commands[name] = cmd
}

func (r *Registry) Definitions() []Definition {
	defs := make([]Definition, 0, len(r.commands))
	for _, cmd := range r.commands {
		defs = append(defs, cmd.Definition())
	}

	sort.Slice(defs, func(i, j int) bool {
		return defs[i].Name < defs[j].Name
	})

	return defs
}

func (r *Registry) Get(name string) (Command, bool) {
	cmd, ok := r.commands[name]

	return cmd, ok
}
