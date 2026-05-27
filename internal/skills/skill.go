package skills

type Skill struct {
	Name        string
	Description string
	Path        string
	Body        string
}

func cloneSkills(skills []Skill) []Skill {
	if len(skills) == 0 {
		return nil
	}

	cloned := make([]Skill, len(skills))
	copy(cloned, skills)

	return cloned
}

func findSkill(skills []Skill, name string) (Skill, bool) {
	for _, skill := range skills {
		if skill.Name == name {
			return skill, true
		}
	}

	return Skill{}, false
}
