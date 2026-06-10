package planning

import (
	"context"
	"fmt"
	"strings"

	"github.com/vitaliiPsl/crappy-ai/internal/session"
)

func currentPlanText(ctx context.Context, sessionID string, store session.ArtifactStore) (string, error) {
	var plan Plan

	if store == nil {
		return "", nil
	}

	ok, err := store.LoadArtifact(ctx, sessionID, ArtifactName, &plan)
	if err != nil {
		return "", fmt.Errorf("load plan: %w", err)
	}

	if !ok || len(plan.Items) == 0 {
		return "", nil
	}

	if err := plan.validate(); err != nil {
		return "", fmt.Errorf("invalid saved plan: %w", err)
	}

	return formatPlan(plan), nil
}

func formatPlan(plan Plan) string {
	var b strings.Builder

	b.WriteString("Current plan:")

	if plan.Explanation != "" {
		b.WriteString("\n")
		b.WriteString(plan.Explanation)
	}

	for _, item := range plan.Items {
		fmt.Fprintf(&b, "\n- [%s] %s", item.Status, item.Step)
	}

	return b.String()
}
