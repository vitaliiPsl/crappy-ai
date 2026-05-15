package summarization

import (
	"context"
	"testing"

	"github.com/vitaliiPsl/crappy-adk/kit"
)

func TestInputLimit_UsesInputLimitWhenSet(t *testing.T) {
	got := inputLimit(kit.ModelConfig{InputLimit: 1000, ContextWindow: 8000, OutputLimit: 2000})
	if got != 1000 {
		t.Fatalf("inputLimit = %d, want 1000", got)
	}
}

func TestInputLimit_SubtractsOutputLimitFromContextWindow(t *testing.T) {
	got := inputLimit(kit.ModelConfig{ContextWindow: 8000, OutputLimit: 2000})
	if got != 6000 {
		t.Fatalf("inputLimit = %d, want 6000", got)
	}
}

func TestInputLimit_FallsBackToContextWindowWhenNoOutputLimit(t *testing.T) {
	got := inputLimit(kit.ModelConfig{ContextWindow: 8000})
	if got != 8000 {
		t.Fatalf("inputLimit = %d, want 8000", got)
	}
}

func TestInputLimit_FallsBackToContextWindowWhenOutputLimitNotSmaller(t *testing.T) {
	got := inputLimit(kit.ModelConfig{ContextWindow: 8000, OutputLimit: 9000})
	if got != 8000 {
		t.Fatalf("inputLimit = %d, want 8000", got)
	}
}

func TestInputLimit_ReturnsZeroWhenUnconfigured(t *testing.T) {
	if got := inputLimit(kit.ModelConfig{}); got != 0 {
		t.Fatalf("inputLimit = %d, want 0", got)
	}
}

func TestWhenLastInputExceeds_DoesNotFireWhenLimitZero(t *testing.T) {
	trigger := whenLastInputExceeds(kit.ModelConfig{}, thresholdRatio)

	rc := &kit.RunContext{
		Context:   context.Background(),
		LastUsage: kit.Usage{InputTokens: 1_000_000},
	}

	if trigger(rc) {
		t.Fatal("trigger fired when limit was zero")
	}
}

func TestWhenLastInputExceeds_FiresAtOrAboveThreshold(t *testing.T) {
	// Threshold = floor(100 * 0.75) = 75 tokens.
	trigger := whenLastInputExceeds(kit.ModelConfig{InputLimit: 100}, thresholdRatio)

	rc := &kit.RunContext{
		Context:   context.Background(),
		LastUsage: kit.Usage{InputTokens: 75},
	}

	if !trigger(rc) {
		t.Fatal("trigger did not fire at threshold")
	}
}

func TestWhenLastInputExceeds_DoesNotFireBelowThreshold(t *testing.T) {
	trigger := whenLastInputExceeds(kit.ModelConfig{InputLimit: 100}, thresholdRatio)

	rc := &kit.RunContext{
		Context:   context.Background(),
		LastUsage: kit.Usage{InputTokens: 74},
	}

	if trigger(rc) {
		t.Fatal("trigger fired below threshold")
	}
}

func TestWhenLastInputExceeds_DoesNotFireWhenLastUsageZero(t *testing.T) {
	trigger := whenLastInputExceeds(kit.ModelConfig{InputLimit: 100}, thresholdRatio)

	rc := &kit.RunContext{
		Context: context.Background(),
	}

	if trigger(rc) {
		t.Fatal("trigger fired without any prior model call")
	}
}

func TestWhenLastInputExceeds_ClampsInvalidRatio(t *testing.T) {
	// ratio out of range falls back to thresholdRatio (0.75); threshold = 75.
	trigger := whenLastInputExceeds(kit.ModelConfig{InputLimit: 100}, 1.5)

	rc := &kit.RunContext{
		Context:   context.Background(),
		LastUsage: kit.Usage{InputTokens: 75},
	}

	if !trigger(rc) {
		t.Fatal("trigger did not fire with clamped ratio")
	}
}
