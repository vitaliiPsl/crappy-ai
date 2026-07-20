package command

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	provideroauth "github.com/vitaliiPsl/crappy-ai/internal/providers/oauth"
)

func TestUsageProviderRegistersForProvider(t *testing.T) {
	registry := NewRegistry(context.Background(), NewUsageProvider(&usageSource{}, "openai-codex"))

	if _, ok := registry.Get("usage"); !ok {
		t.Fatal("usage command was not registered")
	}
}

func TestUsageCommandFormatsLimits(t *testing.T) {
	reset := time.Now().Add(2*time.Hour + 10*time.Minute)
	source := &usageSource{
		limits: provideroauth.Limits{
			Plan: "plus",
			Snapshots: []provideroauth.LimitSnapshot{{
				Windows: []provideroauth.LimitWindow{{
					UsedPercent: 42,
					Duration:    5 * time.Hour,
					ResetsAt:    reset,
				}},
			}, {
				Name: "Other model",
				Windows: []provideroauth.LimitWindow{{
					UsedPercent: 70,
					Duration:    7 * 24 * time.Hour,
				}},
			}},
		},
	}

	msg := (&UsageCommand{source: source, providerID: "openai-codex"}).Execute(
		context.Background(),
		Request{},
	)().(SystemMsg)

	for _, want := range []string{"Plus plan", "5 hours: 42% used", "Other model", "7 days: 70% used"} {
		if !strings.Contains(msg.Text, want) {
			t.Errorf("output %q does not contain %q", msg.Text, want)
		}
	}
}

func TestUsageCommandReportsFetchError(t *testing.T) {
	source := &usageSource{err: errors.New("request failed")}
	msg := (&UsageCommand{source: source, providerID: "openai-codex"}).Execute(
		context.Background(),
		Request{},
	)().(SystemMsg)

	if msg.Text != "Usage unavailable: request failed" {
		t.Fatalf("output = %q", msg.Text)
	}
}

type usageSource struct {
	limits provideroauth.Limits
	err    error
}

func (s *usageSource) ProviderLimits(context.Context, string) (provideroauth.Limits, error) {
	return s.limits, s.err
}
