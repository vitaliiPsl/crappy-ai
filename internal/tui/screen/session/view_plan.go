package session

import (
	"encoding/json"
	"strings"
)

const (
	planToolName = "write_plan"

	planStatusPending    = "pending"
	planStatusInProgress = "in_progress"
	planStatusCompleted  = "completed"

	planHeading        = "Plan"
	planExplanationSep = " — "

	planMarkPending    = "[ ]"
	planMarkInProgress = "[~]"
	planMarkCompleted  = "[x]"
)

type planView struct {
	Explanation string     `json:"explanation,omitempty"`
	Items       []planItem `json:"items"`
}

type planItem struct {
	Step   string `json:"step"`
	Status string `json:"status"`
}

func renderPlanTool(tool *ToolUse) string {
	p, ok := parsePlanArgs(tool.Arguments)
	if !ok {
		return ""
	}

	var b strings.Builder

	b.WriteString(toolNameStyle.Render(planHeading))

	if p.Explanation != "" {
		b.WriteString(subtleTextStyle.Render(planExplanationSep + p.Explanation))
	}

	for _, item := range p.Items {
		b.WriteString("\n")
		b.WriteString(renderPlanItem(item))
	}

	if tool.Error != "" {
		b.WriteString("\n" + errorStyle.Render(truncateInline(tool.Error, convMaxResultLen)))
	}

	return toolBlockStyle(tool).Render(b.String()) + "\n"
}

func renderPlanItem(item planItem) string {
	line := planMark(item.Status) + " " + item.Step

	switch item.Status {
	case planStatusCompleted:
		return subtleTextStyle.Render(line)
	case planStatusInProgress:
		return textStyle.Bold(true).Render(line)
	default:
		return textStyle.Render(line)
	}
}

func planMark(status string) string {
	switch status {
	case planStatusCompleted:
		return planMarkCompleted
	case planStatusInProgress:
		return planMarkInProgress
	default:
		return planMarkPending
	}
}

func parsePlanArgs(args map[string]any) (planView, bool) {
	if len(args) == 0 {
		return planView{}, false
	}

	data, err := json.Marshal(args)
	if err != nil {
		return planView{}, false
	}

	var p planView
	if err := json.Unmarshal(data, &p); err != nil {
		return planView{}, false
	}

	if len(p.Items) == 0 {
		return planView{}, false
	}

	for _, item := range p.Items {
		if item.Step == "" || !isValidPlanStatus(item.Status) {
			return planView{}, false
		}
	}

	return p, true
}

func isValidPlanStatus(status string) bool {
	switch status {
	case planStatusPending, planStatusInProgress, planStatusCompleted:
		return true
	}

	return false
}
