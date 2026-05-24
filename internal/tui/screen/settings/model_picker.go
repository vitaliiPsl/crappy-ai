package settings

import (
	"fmt"
	"strings"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/tui/component"
)

const (
	modelPickerWindow      = 7
	modelPickerPrompt      = "> "
	modelPickerPlaceholder = "Filter models..."
	modelPickerEmpty       = "No matching models. Press Enter to use the typed model ID."
	modelPickerSep         = "  "
)

type modelPicker struct {
	selector component.Selector[kit.ModelConfig]
}

func newModelPicker(models []kit.ModelConfig) modelPicker {
	return modelPicker{
		selector: component.NewSelector(component.SelectorConfig[kit.ModelConfig]{
			Items: models,
			Match: func(model kit.ModelConfig, query string) bool {
				return query == "" || strings.Contains(strings.ToLower(model.ID), query)
			},
			Render: func(model kit.ModelConfig, isSelected bool) string {
				style := valueStyle
				if isSelected {
					style = selectedStyle
				}

				line := style.Render(model.ID)
				if summary := modelSummary(model); summary != "" {
					line += modelPickerSep + mutedStyle.Render(summary)
				}

				return line
			},
			Window: modelPickerWindow,
		}),
	}
}

func (p *modelPicker) Update(value string) {
	p.selector.Filter(strings.ToLower(strings.TrimSpace(value)))
}

func (p modelPicker) View() string {
	if p.selector.Empty() {
		return mutedStyle.Render(modelPickerEmpty)
	}

	return p.selector.View()
}

func (p *modelPicker) SetModels(models []kit.ModelConfig, current string) {
	p.selector.Filter("")
	p.selector.SetItems(models)
	p.selector.SelectWhere(func(model kit.ModelConfig) bool {
		return model.ID == current
	})
}

func (p *modelPicker) Previous() bool {
	return p.selector.Previous()
}

func (p *modelPicker) Next() bool {
	return p.selector.Next()
}

func (p modelPicker) Selected() (kit.ModelConfig, bool) {
	return p.selector.Selected()
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
