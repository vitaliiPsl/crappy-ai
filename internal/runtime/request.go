package runtime

type Request struct {
	Text      string
	Skill     *SkillInvocation
	MCPPrompt *MCPPromptInvocation
}

type SkillInvocation struct {
	Name string
	Args []string
}

type MCPPromptInvocation struct {
	Server string
	Name   string
	Args   map[string]string
}
