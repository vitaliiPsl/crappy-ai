package planning

import (
	"context"
	"fmt"
	"strings"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/session"
)

func injectPlan(sessionID string, store session.ArtifactStore) func(*kit.RunContext, kit.ModelRequest) (kit.ModelRequest, error) {
	return func(rc *kit.RunContext, req kit.ModelRequest) (kit.ModelRequest, error) {
		if store == nil {
			return req, nil
		}

		text, err := currentPlanText(rc.Context, sessionID, store)
		if err != nil {
			return req, err
		}

		if text == "" {
			return req, nil
		}

		req.Instructions = appendInstruction(req.Instructions, text)

		return req, nil
	}
}

func currentPlanText(ctx context.Context, sessionID string, store session.ArtifactStore) (string, error) {
	var plan Plan

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

func appendInstruction(existing, addition string) string {
	if existing == "" {
		return addition
	}

	return existing + "\n\n" + addition
}
