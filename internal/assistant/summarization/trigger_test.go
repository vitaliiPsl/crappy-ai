package summarization

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/vitaliiPsl/crappy-adk/kit"
	"github.com/vitaliiPsl/crappy-adk/x/memory"
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

func TestEstimateContentTokens_Text(t *testing.T) {
	got := estimateContentTokens(kit.NewTextContent(strings.Repeat("a", 16)))
	if got != 4 {
		t.Fatalf("estimateContentTokens = %d, want 4", got)
	}
}

func TestEstimateContentTokens_ThinkingIncludesSignature(t *testing.T) {
	got := estimateContentTokens(kit.NewThinkingContent("id", "thought text", strings.Repeat("s", 100)))

	// len("id") + len("thought text") + 100 = 114 bytes, ceil(114/4) = 29
	if got != 29 {
		t.Fatalf("estimateContentTokens = %d, want 29", got)
	}
}

func TestEstimateContentTokens_Summary(t *testing.T) {
	got := estimateContentTokens(kit.NewSummaryContent(strings.Repeat("x", 40)))
	if got != 10 {
		t.Fatalf("estimateContentTokens = %d, want 10", got)
	}
}

func TestEstimateContentTokens_ToolCallSumsFields(t *testing.T) {
	call := kit.NewToolCall("call-id", "tool-name", map[string]any{"arg": "value"})

	got := estimateContentTokens(kit.NewToolCallContent(call))

	// id=7 + name=9 + args JSON `{"arg":"value"}`=15 + signature=0 = 31 bytes, ceil(31/4) = 8
	if got != 8 {
		t.Fatalf("estimateContentTokens = %d, want 8", got)
	}
}

func TestEstimateContentTokens_ToolResultIgnoresCallArguments(t *testing.T) {
	bigArgs := map[string]any{"command": strings.Repeat("x", 1000)}
	result := kit.NewToolResult(kit.NewToolCall("call-id", "tool-name", bigArgs), "ok", nil)

	got := estimateContentTokens(kit.NewToolResultContent(result))

	// Must NOT include the 1000-byte command. Only id=7 + name=9 + output=2 + error=0 = 18 bytes, ceil(18/4) = 5
	if got != 5 {
		t.Fatalf("estimateContentTokens = %d, want 5 (must not double-count Call.Arguments)", got)
	}
}

func TestEstimateMessagesTokens_AddsPerMessageOverhead(t *testing.T) {
	messages := []kit.Message{
		kit.NewUserMessage(kit.NewTextContent("hello")),
		kit.NewModelMessage(kit.NewTextContent("hi")),
	}

	// msg 1: overhead 4 + ceil(5/4)=2 = 6
	// msg 2: overhead 4 + ceil(2/4)=1 = 5
	if got := estimateMessagesTokens(messages); got != 11 {
		t.Fatalf("estimateMessagesTokens = %d, want 11", got)
	}
}

func TestEstimateMessagesTokens_Empty(t *testing.T) {
	if got := estimateMessagesTokens(nil); got != 0 {
		t.Fatalf("estimateMessagesTokens(nil) = %d, want 0", got)
	}
}

func TestWhenEstimatedContextExceeds_DoesNotFireWhenLimitZero(t *testing.T) {
	trigger := whenEstimatedContextExceeds(kit.ModelConfig{}, thresholdRatio)

	rc := &kit.RunContext{
		Context: context.Background(),
		Memory:  memory.NewHistory(kit.NewUserMessage(kit.NewTextContent(strings.Repeat("a", 10000)))),
	}

	if trigger(rc) {
		t.Fatal("trigger fired when limit was zero")
	}
}

func TestWhenEstimatedContextExceeds_FiresAboveThreshold(t *testing.T) {
	// Threshold = floor(100 * 0.75) = 75 tokens.
	trigger := whenEstimatedContextExceeds(kit.ModelConfig{InputLimit: 100}, thresholdRatio)

	// One message: overhead 4 + ceil(400/4)=100 = 104 tokens, above 75.
	rc := &kit.RunContext{
		Context: context.Background(),
		Memory:  memory.NewHistory(kit.NewUserMessage(kit.NewTextContent(strings.Repeat("a", 400)))),
	}

	if !trigger(rc) {
		t.Fatal("trigger did not fire above threshold")
	}
}

func TestWhenEstimatedContextExceeds_DoesNotFireBelowThreshold(t *testing.T) {
	trigger := whenEstimatedContextExceeds(kit.ModelConfig{InputLimit: 100}, thresholdRatio)

	// Tiny message: overhead 4 + ceil(8/4)=2 = 6 tokens, well below 75.
	rc := &kit.RunContext{
		Context: context.Background(),
		Memory:  memory.NewHistory(kit.NewUserMessage(kit.NewTextContent("eight!!!"))),
	}

	if trigger(rc) {
		t.Fatal("trigger fired below threshold")
	}
}

func TestWhenEstimatedContextExceeds_ReturnsFalseOnMemoryError(t *testing.T) {
	trigger := whenEstimatedContextExceeds(kit.ModelConfig{InputLimit: 100}, thresholdRatio)

	rc := &kit.RunContext{
		Context: context.Background(),
		Memory:  errMemory{err: errors.New("boom")},
	}

	if trigger(rc) {
		t.Fatal("trigger fired despite memory error")
	}
}

type errMemory struct {
	err error
}

func (m errMemory) Context(context.Context) ([]kit.Message, error) { return nil, m.err }
func (m errMemory) History(context.Context) ([]kit.Message, error) { return nil, m.err }
func (errMemory) Record(context.Context, ...kit.Message) error     { return nil }
