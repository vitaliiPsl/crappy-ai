package skills

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/vitaliiPsl/crappy-ai/internal/settings"
)

var ErrUnknownSkill = errors.New("unknown skill")

const (
	maxSkillFileBytes = 64 * 1024

	skillFileName = "SKILL.md"
)

type Registry struct {
	list []Skill
}

func NewRegistry(settingsStore *settings.Store) *Registry {
	return &Registry{
		list: loadSkills(settingsStore.Get().SkillsPath),
	}
}

func (r *Registry) GetSkills() []Skill {
	return cloneSkills(r.list)
}

func (r *Registry) GetSkill(name string) (Skill, error) {
	metadata, ok := findSkill(r.list, name)
	if !ok {
		return Skill{}, fmt.Errorf("%w %q", ErrUnknownSkill, name)
	}

	loaded, err := loadSkill(metadata.Path)
	if err != nil {
		return Skill{}, fmt.Errorf("load skill %q: %w", name, err)
	}

	if strings.TrimSpace(loaded.Body) == "" {
		return Skill{}, fmt.Errorf("skill %q has no instructions", name)
	}

	return loaded, nil
}

func loadSkills(root string) []Skill {
	byName := make(map[string]Skill)

	for _, path := range skillPaths(root) {
		skill, err := loadSkill(path)
		if err != nil {
			continue
		}

		skill.Body = ""
		if skill.Name != "" {
			byName[skill.Name] = skill
		}
	}

	skills := make([]Skill, 0, len(byName))
	for _, skill := range byName {
		skills = append(skills, skill)
	}

	slices.SortFunc(skills, func(a, b Skill) int {
		return strings.Compare(a.Name, b.Name)
	})

	return skills
}

func loadSkill(path string) (Skill, error) {
	data, err := readSkillFile(path)
	if err != nil {
		return Skill{}, err
	}

	loaded, err := parse(path, data)
	if err != nil {
		return Skill{}, fmt.Errorf("parse %s: %w", path, err)
	}

	return loaded, nil
}

func skillPaths(root string) []string {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}

	var paths []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		paths = append(paths, filepath.Join(root, entry.Name(), skillFileName))
	}

	slices.Sort(paths)

	return paths
}

func readSkillFile(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if info.IsDir() {
		return nil, fmt.Errorf("%s is a directory", path)
	}

	if info.Size() > maxSkillFileBytes {
		return nil, fmt.Errorf("%s is larger than %d bytes", path, maxSkillFileBytes)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return data, nil
}
