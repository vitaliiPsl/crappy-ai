package assistant

type RunRequest struct {
	Text  string
	Skill *SkillInvocation
}

type SkillInvocation struct {
	Name string
	Args []string
}
