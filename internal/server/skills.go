package server

import "github.com/vitaliiPsl/crappy-ai/internal/skills"

func (s *Server) GetSkills() []skills.Skill {
	return s.skillRegistry.GetSkills()
}

func (s *Server) GetSkill(name string) (skills.Skill, error) {
	return s.skillRegistry.GetSkill(name)
}
