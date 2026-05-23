package summarization

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/vitaliiPsl/crappy-adk/kit"
	"github.com/vitaliiPsl/crappy-adk/kittest"
)

func TestSummarize_ReturnsModelTextAndUsage(t *testing.T) {
	model := kittest.NewModel(t, kittest.ModelResult{
		Response: kit.ModelResponse{
			Message: kit.NewModelMessage(kit.NewTextContent("compact summary")),
			Usage:   kit.Usage{InputTokens: 12, OutputTokens: 4},
		},
	})

	text, usage, err := generateSummary(
		context.Background(),
		model,
		[]kit.Message{kit.NewUserMessage(kit.NewTextContent("hi"))},
	)
	if err != nil {
		t.Fatalf("summarize: %v", err)
	}

	if text != "compact summary" {
		t.Fatalf("text = %q, want compact summary", text)
	}

	if usage.InputTokens != 12 || usage.OutputTokens != 4 {
		t.Fatalf("usage = %+v, want input=12 output=4", usage)
	}
}

func TestSummarize_SendsPromptAndSingleUserMessage(t *testing.T) {
	model := kittest.NewModel(t, kittest.ModelResult{
		Response: kit.ModelResponse{
			Message: kit.NewModelMessage(kit.NewTextContent("ok")),
		},
	})

	messages := []kit.Message{
		kit.NewUserMessage(kit.NewTextContent("first")),
		kit.NewModelMessage(kit.NewTextContent("second")),
	}

	if _, _, err := generateSummary(context.Background(), model, messages); err != nil {
		t.Fatalf("summarize: %v", err)
	}

	req := model.CallAt(0)
	if req.Instructions != Prompt {
		t.Fatalf("Instructions = %q, want package Prompt", req.Instructions)
	}

	if len(req.Messages) != 1 {
		t.Fatalf("len(Messages) = %d, want 1 (transcript should be sent as a single user message)", len(req.Messages))
	}

	if req.Messages[0].Role != kit.RoleUser {
		t.Fatalf("Messages[0].Role = %q, want %q", req.Messages[0].Role, kit.RoleUser)
	}

	if req.Messages[0].Content[0].Type != kit.ContentTypeText {
		t.Fatalf("Messages[0].Content[0].Type = %q, want %q", req.Messages[0].Content[0].Type, kit.ContentTypeText)
	}
}

func TestSummarize_FlattensRolesIntoTranscript(t *testing.T) {
	model := kittest.NewModel(t, kittest.ModelResult{
		Response: kit.ModelResponse{Message: kit.NewModelMessage(kit.NewTextContent("ok"))},
	})

	messages := []kit.Message{
		kit.NewUserMessage(kit.NewTextContent("hello")),
		kit.NewModelMessage(kit.NewTextContent("hi back")),
	}

	if _, _, err := generateSummary(context.Background(), model, messages); err != nil {
		t.Fatalf("summarize: %v", err)
	}

	transcript := model.CallAt(0).Messages[0].Content[0].Text.Text

	if !strings.Contains(transcript, "User:\nhello") {
		t.Fatalf("transcript missing user turn:\n%s", transcript)
	}

	if !strings.Contains(transcript, "Assistant:\nhi back") {
		t.Fatalf("transcript missing assistant turn:\n%s", transcript)
	}
}

func TestSummarize_OmitsThinkingFromTranscript(t *testing.T) {
	model := kittest.NewModel(t, kittest.ModelResult{
		Response: kit.ModelResponse{Message: kit.NewModelMessage(kit.NewTextContent("ok"))},
	})

	messages := []kit.Message{
		kit.NewUserMessage(kit.NewTextContent("hello")),
		kit.NewModelMessage(
			kit.NewThinkingContent("t1", "internal reasoning blob", "sig"),
			kit.NewTextContent("hi back"),
		),
	}

	if _, _, err := generateSummary(context.Background(), model, messages); err != nil {
		t.Fatalf("summarize: %v", err)
	}

	transcript := model.CallAt(0).Messages[0].Content[0].Text.Text

	if strings.Contains(transcript, "internal reasoning blob") {
		t.Fatalf("transcript includes thinking text:\n%s", transcript)
	}
}

func TestSummarize_RendersToolCallsAndResults(t *testing.T) {
	model := kittest.NewModel(t, kittest.ModelResult{
		Response: kit.ModelResponse{Message: kit.NewModelMessage(kit.NewTextContent("ok"))},
	})

	call := kit.ToolCall{
		ID:        "call-1",
		Name:      "bash",
		Arguments: map[string]any{"command": "ls"},
		Signature: "sig-blob",
	}

	messages := []kit.Message{
		kit.NewModelMessage(kit.NewToolCallContent(call)),
		kit.NewToolMessage(kit.NewToolResultContent(kit.NewToolResult(call, "file.go", nil))),
	}

	if _, _, err := generateSummary(context.Background(), model, messages); err != nil {
		t.Fatalf("summarize: %v", err)
	}

	transcript := model.CallAt(0).Messages[0].Content[0].Text.Text

	if !strings.Contains(transcript, "[Tool call: bash(") || !strings.Contains(transcript, `"command":"ls"`) {
		t.Fatalf("transcript missing tool call:\n%s", transcript)
	}

	if !strings.Contains(transcript, "[Tool result from bash: file.go]") {
		t.Fatalf("transcript missing tool result:\n%s", transcript)
	}

	if strings.Contains(transcript, "sig-blob") {
		t.Fatalf("transcript leaked tool signature:\n%s", transcript)
	}
}

func TestSummarize_RendersToolError(t *testing.T) {
	model := kittest.NewModel(t, kittest.ModelResult{
		Response: kit.ModelResponse{Message: kit.NewModelMessage(kit.NewTextContent("ok"))},
	})

	call := kit.ToolCall{ID: "call-1", Name: "bash"}
	result := kit.ToolResult{Call: call, Error: "command not found"}

	messages := []kit.Message{
		kit.NewToolMessage(kit.NewToolResultContent(result)),
	}

	if _, _, err := generateSummary(context.Background(), model, messages); err != nil {
		t.Fatalf("summarize: %v", err)
	}

	transcript := model.CallAt(0).Messages[0].Content[0].Text.Text

	if !strings.Contains(transcript, "[Tool error from bash: command not found]") {
		t.Fatalf("transcript missing tool error:\n%s", transcript)
	}
}

func TestSummarize_RendersPriorSummary(t *testing.T) {
	model := kittest.NewModel(t, kittest.ModelResult{
		Response: kit.ModelResponse{Message: kit.NewModelMessage(kit.NewTextContent("ok"))},
	})

	messages := []kit.Message{
		kit.NewUserMessage(kit.NewSummaryContent("previous recap")),
		kit.NewUserMessage(kit.NewTextContent("new question")),
	}

	if _, _, err := generateSummary(context.Background(), model, messages); err != nil {
		t.Fatalf("summarize: %v", err)
	}

	transcript := model.CallAt(0).Messages[0].Content[0].Text.Text

	if !strings.Contains(transcript, "[Previous summary]\nprevious recap") {
		t.Fatalf("transcript missing prior summary:\n%s", transcript)
	}

	if !strings.Contains(transcript, "new question") {
		t.Fatalf("transcript missing later user turn:\n%s", transcript)
	}
}

func TestSummarize_PropagatesModelError(t *testing.T) {
	wantErr := errors.New("model down")

	model := kittest.NewModel(t, kittest.ModelResult{Error: wantErr})

	_, _, err := generateSummary(
		context.Background(),
		model,
		[]kit.Message{kit.NewUserMessage(kit.NewTextContent("hi"))},
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want wraps %v", err, wantErr)
	}
}

func TestSummarize_ErrorsOnMissingTextContent(t *testing.T) {
	model := kittest.NewModel(t, kittest.ModelResult{
		Response: kit.ModelResponse{Message: kit.NewModelMessage()},
	})

	_, _, err := generateSummary(
		context.Background(),
		model,
		[]kit.Message{kit.NewUserMessage(kit.NewTextContent("hi"))},
	)
	if err == nil {
		t.Fatal("expected error for missing text content, got nil")
	}
}

func TestSummarize_ErrorsOnEmptyText(t *testing.T) {
	model := kittest.NewModel(t, kittest.ModelResult{
		Response: kit.ModelResponse{
			Message: kit.NewModelMessage(kit.NewTextContent("")),
		},
	})

	_, _, err := generateSummary(
		context.Background(),
		model,
		[]kit.Message{kit.NewUserMessage(kit.NewTextContent("hi"))},
	)
	if err == nil {
		t.Fatal("expected error for empty text, got nil")
	}
}
