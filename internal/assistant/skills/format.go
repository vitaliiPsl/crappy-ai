package skills

import (
	"strings"
	"text/template"

	coreskills "github.com/vitaliiPsl/crappy-ai/internal/skills"
)

const listingTemplate = `{{range .}}- {{.Name}}{{if .Description}}: {{.Description}}{{end}}
{{end}}`

const loadedTemplate = `Loaded skill: {{.Skill.Name}}

{{if .Args}}Arguments:
{{.Args}}

{{end}}{{template "instructions" .Skill}}`

const instructionsTemplate = `# Skill: {{.Name}}

{{if .Description}}Description: {{.Description}}

{{end}}{{if .Path}}Source: {{.Path}}

{{end}}## Instructions

{{.Body}}`

var skillTemplates = template.New("skills")

func init() {
	template.Must(skillTemplates.New("listing").Parse(listingTemplate))
	template.Must(skillTemplates.New("loaded").Parse(loadedTemplate))
	template.Must(skillTemplates.New("instructions").Parse(instructionsTemplate))
}

type loadedSkillData struct {
	Skill coreskills.Skill
	Args  string
}

func formatListing(skills []coreskills.Skill) string {
	if len(skills) == 0 {
		return ""
	}

	return renderTemplate("listing", skills)
}

func formatLoadedSkill(loaded coreskills.Skill, args string) string {
	return renderTemplate("loaded", loadedSkillData{
		Skill: loaded,
		Args:  strings.TrimSpace(args),
	})
}

func renderTemplate(name string, data any) string {
	var b strings.Builder
	if err := skillTemplates.ExecuteTemplate(&b, name, data); err != nil {
		panic(err)
	}

	return strings.TrimSpace(b.String())
}
