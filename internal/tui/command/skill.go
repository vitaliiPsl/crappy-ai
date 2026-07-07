package command

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/vitaliiPsl/crappy-ai/internal/skills"
)

type SkillSource interface {
	GetSkills() []skills.Skill
}

type skillProvider struct {
	source SkillSource
}

func NewSkillProvider(source SkillSource) Provider {
	if source == nil {
		return nil
	}

	return skillProvider{source: source}
}

func (p skillProvider) Commands(_ context.Context) []Command {
	skills := p.source.GetSkills()

	cmds := make([]Command, 0, len(skills))
	for _, sk := range skills {
		cmds = append(cmds, NewSkillCommand(sk))
	}

	return cmds
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
