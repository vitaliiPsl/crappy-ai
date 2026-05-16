package settings

import (
	"fmt"
	"strings"

	"github.com/vitaliiPsl/crappy-adk/kit"
)

const (
	maxModelSuggestions     = 7
	modelSuggestPrompt      = "> "
	modelSuggestPlaceholder = "Filter models..."
	modelSuggestEmpty       = "No matching models"
	modelSuggestSep         = "  "
)

type modelSuggestions struct {
	models   []kit.ModelConfig
	matches  []kit.ModelConfig
	selected int
}

func newModelSuggestions(models []kit.ModelConfig) modelSuggestions {
	s := modelSuggestions{models: models}
	s.Update("")

	return s
}

func (s *modelSuggestions) SetModels(models []kit.ModelConfig, current string) {
	s.models = models
	s.selected = 0
	s.Update("")
	s.selectModel(current)
}

func (s *modelSuggestions) Update(value string) {
	query := strings.ToLower(strings.TrimSpace(value))

	s.matches = s.matches[:0]
	for _, model := range s.models {
		if query == "" || strings.Contains(strings.ToLower(model.ID), query) {
			s.matches = append(s.matches, model)
		}
	}

	if s.selected >= len(s.matches) {
		s.selected = 0
	}
}

func (s *modelSuggestions) Previous() bool {
	if len(s.matches) == 0 {
		return false
	}

	s.selected = (s.selected - 1 + len(s.matches)) % len(s.matches)

	return true
}

func (s *modelSuggestions) Next() bool {
	if len(s.matches) == 0 {
		return false
	}

	s.selected = (s.selected + 1) % len(s.matches)

	return true
}

func (s modelSuggestions) Selected() (kit.ModelConfig, bool) {
	if len(s.matches) == 0 || s.selected < 0 || s.selected >= len(s.matches) {
		return kit.ModelConfig{}, false
	}

	return s.matches[s.selected], true
}

func (s modelSuggestions) View() string {
	if len(s.matches) == 0 {
		return mutedStyle.Render(modelSuggestEmpty)
	}

	var lines []string

	start, end := s.visibleRange()
	for idx := start; idx < end; idx++ {
		model := s.matches[idx]
		nameStyle := valueStyle

		if idx == s.selected {
			nameStyle = selectedStyle
		}

		line := nameStyle.Render(model.ID)
		if summary := modelSummary(model); summary != "" {
			line += modelSuggestSep + mutedStyle.Render(summary)
		}

		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

func (s *modelSuggestions) selectModel(id string) {
	for i, model := range s.matches {
		if model.ID == id {
			s.selected = i

			return
		}
	}
}

func (s modelSuggestions) visibleRange() (int, int) {
	if len(s.matches) <= maxModelSuggestions {
		return 0, len(s.matches)
	}

	start := s.selected - maxModelSuggestions/2
	start = max(start, 0)

	end := start + maxModelSuggestions
	if end > len(s.matches) {
		end = len(s.matches)
		start = end - maxModelSuggestions
	}

	return start, end
}

func modelSummary(model kit.ModelConfig) string {
	parts := make([]string, 0, 3)
	if limits := modelLimits(model); limits != "" {
		parts = append(parts, limits)
	}

	if cost := modelCost(model.Cost); cost != "" {
		parts = append(parts, cost)
	}

	return strings.Join(parts, " | ")
}

func modelLimits(model kit.ModelConfig) string {
	var parts []string
	if model.InputLimit > 0 {
		parts = append(parts, fmt.Sprintf("%s in", formatLimit(model.InputLimit)))
	} else if model.ContextWindow > 0 {
		parts = append(parts, fmt.Sprintf("%s ctx", formatLimit(model.ContextWindow)))
	}

	if model.OutputLimit > 0 {
		parts = append(parts, fmt.Sprintf("%s out", formatLimit(model.OutputLimit)))
	}

	return strings.Join(parts, " / ")
}

func modelCost(cost kit.ModelCost) string {
	if cost.Input == 0 && cost.Output == 0 {
		return ""
	}

	return fmt.Sprintf("$%s/$%s", formatCost(cost.Input), formatCost(cost.Output))
}

func formatLimit(n int) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}

	if n >= 1_000 {
		return fmt.Sprintf("%dk", n/1_000)
	}

	return fmt.Sprintf("%d", n)
}

func formatCost(v float64) string {
	if v == 0 {
		return "0"
	}

	if v < 1 {
		return fmt.Sprintf("%.3g", v)
	}

	return fmt.Sprintf("%.2f", v)
}
