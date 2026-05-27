package skills

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var skillNamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)

type frontMatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

func parse(path string, data []byte) (Skill, error) {
	meta, body, err := splitFrontMatter(string(data))
	if err != nil {
		return Skill{}, err
	}

	name := strings.TrimSpace(meta.Name)
	if name == "" {
		name = skillNameFromPath(path)
	}

	if !skillNamePattern.MatchString(name) {
		return Skill{}, fmt.Errorf("invalid skill name %q", name)
	}

	return Skill{
		Name:        name,
		Description: strings.TrimSpace(meta.Description),
		Path:        path,
		Body:        strings.TrimSpace(body),
	}, nil
}

func splitFrontMatter(text string) (frontMatter, string, error) {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	if !strings.HasPrefix(text, "---\n") {
		return frontMatter{}, text, nil
	}

	rest := strings.TrimPrefix(text, "---\n")

	before, after, ok := strings.Cut(rest, "\n---")
	if !ok {
		return frontMatter{}, "", fmt.Errorf("unterminated frontmatter")
	}

	raw := before
	body := strings.TrimPrefix(after, "\n")

	var meta frontMatter
	if err := yaml.Unmarshal([]byte(raw), &meta); err != nil {
		return frontMatter{}, "", fmt.Errorf("parse frontmatter: %w", err)
	}

	return meta, body, nil
}

func skillNameFromPath(path string) string {
	return filepath.Base(filepath.Dir(path))
}
