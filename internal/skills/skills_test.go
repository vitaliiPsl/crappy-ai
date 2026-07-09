package skills

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/vitaliiPsl/crappy-ai/internal/settings"
)

func TestParseSkillWithFrontMatter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "review")

	got, err := parse(path, []byte(strings.Join([]string{
		"---",
		"name: review",
		"description: Review code changes",
		"---",
		"",
		"# Review",
		"",
		"Find bugs first.",
	}, "\n")))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if got.Name != "review" {
		t.Fatalf("Name = %q, want review", got.Name)
	}

	if got.Description != "Review code changes" {
		t.Fatalf("Description = %q, want Review code changes", got.Description)
	}

	if got.Body != "# Review\n\nFind bugs first." {
		t.Fatalf("Body = %q", got.Body)
	}

	if got.Path != path {
		t.Fatalf("Path = %q, want %q", got.Path, path)
	}
}

func TestParseSkillDefaultsNameFromDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "explain")

	got, err := parse(path, []byte("Explain the selected code."))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if got.Name != "explain" {
		t.Fatalf("Name = %q, want explain", got.Name)
	}
}

func TestLoadSkillListLoadsMetadata(t *testing.T) {
	userDir := filepath.Join(t.TempDir(), "skills")

	writeSkillFile(t, filepath.Join(userDir, "review", "SKILL.md"), "review", "review skill", "user review")
	writeSkillFile(t, filepath.Join(userDir, "explain", "SKILL.md"), "explain", "explain skill", "user explain")

	list := loadSkills(userDir)

	review, ok := findSkill(list, "review")
	if !ok {
		t.Fatal("review skill missing")
	}

	if review.Description != "review skill" {
		t.Fatalf("review description = %q, want review skill", review.Description)
	}

	if review.Body != "" {
		t.Fatalf("review body = %q, want metadata without body", review.Body)
	}

	if review.Path != filepath.Join(userDir, "review") {
		t.Fatalf("review path = %q, want skill directory", review.Path)
	}

	explain, ok := findSkill(list, "explain")
	if !ok {
		t.Fatal("explain skill missing")
	}

	if explain.Description != "explain skill" {
		t.Fatalf("explain description = %q, want explain skill", explain.Description)
	}

	names := []string{review.Name, explain.Name}
	slices.Sort(names)

	if !slices.Equal(names, []string{"explain", "review"}) {
		t.Fatalf("Names = %#v, want explain/review", names)
	}
}

func TestRegistryGetSkillsLoadsMetadataOnly(t *testing.T) {
	userDir := filepath.Join(t.TempDir(), "skills")
	writeSkillFile(t, filepath.Join(userDir, "review", "SKILL.md"), "review", "review skill", "Find bugs first.")
	registry := newTestRegistry(userDir)

	list := registry.GetSkills()

	skill, ok := findSkill(list, "review")
	if !ok {
		t.Fatal("review skill missing")
	}

	if skill.Description != "review skill" {
		t.Fatalf("Description = %q, want review skill", skill.Description)
	}

	if skill.Body != "" {
		t.Fatalf("metadata body = %q, want empty", skill.Body)
	}
}

func TestRegistryGetSkillsCachesMetadata(t *testing.T) {
	userDir := filepath.Join(t.TempDir(), "skills")
	writeSkillFile(t, filepath.Join(userDir, "review", "SKILL.md"), "review", "review skill", "Find bugs first.")
	registry := newTestRegistry(userDir)

	first := registry.GetSkills()

	writeSkillFile(t, filepath.Join(userDir, "explain", "SKILL.md"), "explain", "explain skill", "Explain code.")

	second := registry.GetSkills()

	if len(first) != 1 || first[0].Name != "review" {
		t.Fatalf("first = %#v, want one review entry", first)
	}

	if len(second) != 1 || second[0].Name != "review" {
		t.Fatalf("second = %#v, want cached review entry", second)
	}
}

func TestRegistryGetSkillsReturnsSnapshot(t *testing.T) {
	userDir := filepath.Join(t.TempDir(), "skills")
	writeSkillFile(t, filepath.Join(userDir, "review", "SKILL.md"), "review", "review skill", "Find bugs first.")
	registry := newTestRegistry(userDir)

	list := registry.GetSkills()
	list[0].Name = "changed"

	next := registry.GetSkills()

	review, ok := findSkill(next, "review")
	if !ok {
		t.Fatal("review skill missing")
	}

	if review.Name != "review" {
		t.Fatalf("Name = %q, want cached snapshot to remain review", review.Name)
	}
}

func TestRegistryGetSkillReadsOnlyNamedSkill(t *testing.T) {
	userDir := filepath.Join(t.TempDir(), "skills")
	writeSkillFile(t, filepath.Join(userDir, "review", "SKILL.md"), "review", "review skill", "Find bugs first.")
	writeSkillFile(t, filepath.Join(userDir, "broken", "SKILL.md"), "broken", "broken skill", "!!!")
	registry := newTestRegistry(userDir)

	got, err := registry.GetSkill("review")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got.Body != "Find bugs first." {
		t.Fatalf("Body = %q, want selected skill body", got.Body)
	}
}

func TestRegistryGetSkillRereadsSelectedSkill(t *testing.T) {
	userDir := filepath.Join(t.TempDir(), "skills")
	path := filepath.Join(userDir, "review", "SKILL.md")

	writeSkillFile(t, path, "review", "review skill", "old body")

	registry := newTestRegistry(userDir)

	_ = registry.GetSkills()

	writeSkillFile(t, path, "review", "review skill", "new body")

	got, err := registry.GetSkill("review")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got.Body != "new body" {
		t.Fatalf("Body = %q, want new body", got.Body)
	}
}

func TestRegistryGetSkillUnknownSkill(t *testing.T) {
	userDir := filepath.Join(t.TempDir(), "skills")
	writeSkillFile(t, filepath.Join(userDir, "review", "SKILL.md"), "review", "review skill", "Review changes.")
	registry := newTestRegistry(userDir)

	_, err := registry.GetSkill("missing")
	if err == nil {
		t.Fatal("Load error = nil, want unknown skill")
	}

	if !errors.Is(err, ErrUnknownSkill) {
		t.Fatalf("error = %v, want ErrUnknownSkill", err)
	}

	if strings.Contains(err.Error(), "review") {
		t.Fatalf("error = %q, should not list available skills", err)
	}
}

func newTestRegistry(userDir string) *Registry {
	return NewRegistry(settings.NewStore(settings.Settings{SkillsPath: userDir}, ""))
}

func writeSkillFile(t *testing.T, path, name, description, body string) {
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
