package skillstest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vitaliiPsl/crappy-ai/internal/settings"
	"github.com/vitaliiPsl/crappy-ai/internal/skills"
)

func WriteSkill(t *testing.T, path, name, description, body string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("MkdirAll(%q): %v", filepath.Dir(path), err)
	}

	data := strings.Join([]string{
		"---",
		"name: " + name,
		"description: " + description,
		"---",
		"",
		body,
	}, "\n")

	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}

func NewRegistry(userDir string) *skills.Registry {
	return skills.NewRegistry(settings.NewStore(settings.Settings{SkillsPath: userDir}, ""))
}
