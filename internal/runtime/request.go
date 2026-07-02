package runtime

type Request struct {
	Text  string
	Skill *SkillInvocation
}

type SkillInvocation struct {
	Name string
	Args []string
}
