package skills

import (
	"context"
	"fmt"
	"strings"

	"github.com/vitaliiPsl/crappy-adk/kit"
	"github.com/vitaliiPsl/crappy-adk/x/tool"

	coreskills "github.com/vitaliiPsl/crappy-ai/internal/skills"
)

const (
	toolName        = "use_skill"
	toolDescription = "Load skill instructions into the conversation. Use this before doing a task that matches an available skill or when the user references one."
)

type useSkillInput struct {
	Skill string `json:"skill" jsonschema:"Skill name to load, without the leading slash. Example: review"`
	Args  string `json:"args,omitempty" jsonschema:"Optional arguments from the user request or slash command"`
}

func newTool(registry *coreskills.Registry) kit.Tool {
	return tool.MustNew(
		toolName,
		toolDescription,
		func(_ context.Context, input useSkillInput) (string, error) {
			name := strings.TrimPrefix(strings.TrimSpace(input.Skill), "/")
			if name == "" {
				return "", fmt.Errorf("skill name is required")
			}

			skill, err := registry.GetSkill(name)
			if err != nil {
				return "", err
			}

			return coreskills.FormatLoaded(skill, input.Args), nil
		},
	)
}
