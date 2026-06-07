package command

import (
	"fmt"
	"sort"
)

type Registry struct {
	commands map[string]Command
}

func NewRegistry(skillSource SkillSource) *Registry {
	r := &Registry{commands: make(map[string]Command)}

	r.Register(NewNewCommand())
	r.Register(NewSessionsCommand())
	r.Register(NewSettingsCommand())
	r.Register(NewMCPCommand())
	r.Register(NewJobsCommand())
	r.Register(NewCompactCommand())
	r.Register(NewHelpCommand(r))

	if skillSource != nil {
		for _, sk := range skillSource.GetSkills() {
			if _, exists := r.commands[sk.Name]; exists {
				continue
			}

			r.Register(NewSkillCommand(sk))
		}
	}

	return r
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
