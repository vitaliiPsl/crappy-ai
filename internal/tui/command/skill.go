package command

import (
	"context"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/vitaliiPsl/crappy-ai/internal/skills"
)

type SkillSource interface {
	GetSkills() []skills.Skill
	GetSkill(name string) (skills.Skill, error)
}

type SkillCommand struct {
	source SkillSource
	skill  skills.Skill
}

func NewSkillCommand(source SkillSource, skill skills.Skill) *SkillCommand {
	return &SkillCommand{source: source, skill: skill}
}

func (c *SkillCommand) Definition() Definition {
	return Definition{Name: c.skill.Name, Description: c.skill.Description, Kind: KindSkill}
}

func (c *SkillCommand) Execute(_ context.Context, req Request) tea.Cmd {
	return func() tea.Msg {
		loaded, err := c.source.GetSkill(c.skill.Name)
		if err != nil {
			return SystemMsg{Text: "skill error: " + err.Error()}
		}

		return SubmitTextMsg{Text: skills.FormatLoaded(loaded, strings.Join(req.Args, " "))}
	}
}
