package planning

import (
	"context"
	"fmt"

	"github.com/vitaliiPsl/crappy-adk/kit"
	"github.com/vitaliiPsl/crappy-adk/x/tool"

	"github.com/vitaliiPsl/crappy-ai/internal/session"
)

const (
	toolName        = "write_plan"
	toolDescription = "Update the current visible task plan. Use this for multi-step work to record pending, in-progress, and completed steps."
	toolResult      = "Plan updated"
)

type updateInput struct {
	Explanation string      `json:"explanation,omitempty" jsonschema:"Optional short note explaining why the plan changed"`
	Items       []itemInput `json:"items" jsonschema:"Current plan items. Use statuses pending, in_progress, or completed. Keep at most one item in_progress."`
}

type itemInput struct {
	Step   string `json:"step" jsonschema:"Short description of the step"`
	Status string `json:"status" jsonschema:"One of pending, in_progress, or completed"`
}

func newTool(sessionID string, store session.ArtifactStore) kit.Tool {
	return tool.MustNew(
		toolName,
		toolDescription,
		func(ctx context.Context, input updateInput) (string, error) {
			plan := input.plan()
			if err := plan.validate(); err != nil {
				return "", err
			}

			if store == nil {
				return "", fmt.Errorf("artifact store is not configured")
			}

			if err := store.SaveArtifact(ctx, sessionID, ArtifactName, plan); err != nil {
				return "", fmt.Errorf("save plan: %w", err)
			}

			return toolResult, nil
		},
	)
}

func (i updateInput) plan() Plan {
	items := make([]Item, 0, len(i.Items))
	for _, item := range i.Items {
		items = append(items, Item{
			Step:   item.Step,
			Status: Status(item.Status),
		})
	}

	return Plan{
		Explanation: i.Explanation,
		Items:       items,
	}
}
