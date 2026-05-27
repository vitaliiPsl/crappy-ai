package command

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/vitaliiPsl/crappy-ai/internal/skills"
)

type SkillSource interface {
	GetSkills() []skills.Skill
}

type SkillCommand struct {
	skill skills.Skill
}

func NewSkillCommand(skill skills.Skill) *SkillCommand {
	return &SkillCommand{skill: skill}
}

func (c *SkillCommand) Definition() Definition {
	return Definition{Name: c.skill.Name, Description: c.skill.Description, Kind: KindSkill}
}

func (c *SkillCommand) Execute(_ context.Context, req Request) tea.Cmd {
	return func() tea.Msg {
		return SubmitSkillMsg{Text: req.Raw, Name: c.skill.Name, Args: req.Args}
	}
}
