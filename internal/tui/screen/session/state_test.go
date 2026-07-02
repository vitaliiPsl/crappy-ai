package session

import (
	"errors"
	"testing"
	"time"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/ask"
	"github.com/vitaliiPsl/crappy-ai/internal/config"
	sessiondata "github.com/vitaliiPsl/crappy-ai/internal/session"
)

func TestReduce_ContentDeltaText_AppendsToStreaming(t *testing.T) {
	var s State

	s = Reduce(s, sessiondata.NewContentDeltaEvent("sess", kit.NewTextContent("hello ")))
	s = Reduce(s, sessiondata.NewContentDeltaEvent("sess", kit.NewTextContent("world")))

	if s.Streaming == nil {
		t.Fatal("Streaming is nil; expected a draft message")
	}

	if s.Streaming.Text != "hello world" {
		t.Fatalf("Streaming.Text = %q, want %q", s.Streaming.Text, "hello world")
	}

	if s.Phase != PhaseRunning {
		t.Fatalf("Phase = %v, want PhaseRunning", s.Phase)
	}
}

func TestReduce_ContentDeltaThinking_AppendsToStreaming(t *testing.T) {
	s := Reduce(State{}, sessiondata.NewContentDeltaEvent("sess", kit.NewThinkingContent("t-1", "considering...", "")))

	if s.Streaming == nil || s.Streaming.Thinking != "considering..." {
		t.Fatalf("Streaming.Thinking = %q, want %q", streamingThinking(s), "considering...")
	}
}

func TestReduce_ContentDoneToolCall_AddsTool(t *testing.T) {
	call := kit.NewToolCall("call-1", "bash", map[string]any{"command": "ls"})

	s := Reduce(State{}, sessiondata.NewContentDoneEvent("sess", kit.NewToolCallContent(call)))

	if s.Streaming == nil || len(s.Streaming.Tools) != 1 {
		t.Fatalf("expected one streaming tool, got %+v", s.Streaming)
	}

	tu := s.Streaming.Tools[0]
	if tu.ID != "call-1" || tu.Name != "bash" || tu.Done {
		t.Fatalf("tool = %+v, want id=call-1 name=bash done=false", tu)
	}

	if active := s.ActiveTool(); active == nil || active.ID != "call-1" {
		t.Fatalf("ActiveTool = %+v, want call-1", active)
	}
}

func TestReduce_ContentDoneToolResult_MergesIntoDraft(t *testing.T) {
	call := kit.NewToolCall("call-1", "bash", map[string]any{"command": "ls"})

	var s State

	s = Reduce(s, sessiondata.NewContentDoneEvent("sess", kit.NewToolCallContent(call)))
	s = Reduce(s, sessiondata.NewContentDoneEvent("sess", kit.NewToolResultContent(kit.NewToolResult(call, "a\nb", nil))))

	if len(s.Streaming.Tools) != 1 {
		t.Fatalf("expected one tool after merge, got %d", len(s.Streaming.Tools))
	}

	tu := s.Streaming.Tools[0]
	if !tu.Done || tu.Result != "a\nb" || tu.Error != "" {
		t.Fatalf("tool = %+v, want done with result", tu)
	}

	if s.ActiveTool() != nil {
		t.Fatalf("ActiveTool should be nil when all tools done")
	}
}

func TestReduce_ToolResultMessage_MergesIntoCommittedAssistant(t *testing.T) {
	call := kit.NewToolCall("call-1", "bash", map[string]any{"command": "ls"})

	s := State{
		Messages: []Message{{
			Role:  RoleModel,
			Tools: []ToolUse{{ID: call.ID, Name: call.Name, Arguments: call.Arguments}},
		}},
	}

	toolMsg := kit.NewToolMessage(kit.NewToolResultContent(kit.NewToolResult(call, "ok", nil)))
	s = Reduce(s, sessiondata.NewMessageEvent("sess", toolMsg))

	if len(s.Messages) != 1 {
		t.Fatalf("expected message count unchanged, got %d", len(s.Messages))
	}

	got := s.Messages[0].Tools[0]
	if !got.Done || got.Result != "ok" {
		t.Fatalf("committed tool = %+v, want done with result", got)
	}
}

func TestReduce_MessageEvent_CommitsAndClearsStreaming(t *testing.T) {
	var s State

	s = Reduce(s, sessiondata.NewContentDeltaEvent("sess", kit.NewTextContent("hi")))
	s = Reduce(s, sessiondata.NewMessageEvent("sess", kit.NewModelMessage(kit.NewTextContent("hi"))))

	if s.Streaming != nil {
		t.Fatalf("Streaming = %+v, want nil after commit", s.Streaming)
	}

	if len(s.Messages) != 1 || s.Messages[0].Role != RoleModel || s.Messages[0].Text != "hi" {
		t.Fatalf("Messages = %+v, want one assistant message 'hi'", s.Messages)
	}
}

func TestReduce_TurnComplete_SetsStatsAndIdle(t *testing.T) {
	stats := sessiondata.TurnStats{
		Usage:         kit.Usage{InputTokens: 10, OutputTokens: 20},
		ContextUsed:   1000,
		ContextWindow: 8000,
	}

	s := State{Phase: PhaseRunning, Streaming: &Message{Role: RoleModel, Text: "x"}}
	s = Reduce(s, sessiondata.NewTurnCompleteEvent("sess", stats))

	if s.Phase != PhaseIdle {
		t.Fatalf("Phase = %v, want PhaseIdle", s.Phase)
	}

	if s.Streaming != nil {
		t.Fatalf("Streaming = %+v, want nil after turn complete", s.Streaming)
	}

	if s.Stats == nil || s.Stats.Usage.InputTokens != 10 {
		t.Fatalf("Stats = %+v, want recorded stats", s.Stats)
	}
}

func TestReduce_TurnCancelled_ClearsRunState(t *testing.T) {
	s := State{Phase: PhaseRunning, Streaming: &Message{Role: RoleModel, Text: "x"}}
	s = Reduce(s, sessiondata.NewTurnCancelledEvent("sess"))

	if s.Phase != PhaseIdle || s.Streaming != nil {
		t.Fatalf("state after cancel = %+v, want idle with no streaming", s)
	}
}

func TestReduce_ErrorEvent_RecordsErrorAndIdles(t *testing.T) {
	s := State{Phase: PhaseRunning, Streaming: &Message{Role: RoleModel}}

	ev := sessiondata.NewErrorEvent("sess", errors.New("boom"))
	s = Reduce(s, ev)

	if s.LastError != "boom" {
		t.Fatalf("LastError = %q, want %q", s.LastError, "boom")
	}

	if s.Phase != PhaseIdle || s.Streaming != nil {
		t.Fatalf("state after error = %+v, want idle with no streaming", s)
	}
}

func TestReduce_Ask_SetsAwaiting(t *testing.T) {
	req := ask.Request{ID: "call-1", Title: "Allow bash?", Options: []ask.Option{{ID: "allow_once", Label: "Allow once"}}}

	s := Reduce(State{Phase: PhaseRunning}, sessiondata.NewAskEvent("sess", req))

	if s.Phase != PhaseAwaitingPermission {
		t.Fatalf("Phase = %v, want PhaseAwaitingPermission", s.Phase)
	}

	if s.Prompt == nil || s.Prompt.ID != "call-1" {
		t.Fatalf("Prompt = %+v, want one for call-1", s.Prompt)
	}
}

func TestReduce_ContentDeltaAfterPrompt_ResumesRunning(t *testing.T) {
	req := ask.Request{ID: "call-1", Title: "Allow bash?"}

	var s State

	s = Reduce(s, sessiondata.NewAskEvent("sess", req))
	s = Reduce(s, sessiondata.NewContentDeltaEvent("sess", kit.NewTextContent("ok")))

	if s.Phase != PhaseRunning {
		t.Fatalf("Phase = %v, want PhaseRunning after content resumes", s.Phase)
	}

	if s.Prompt != nil {
		t.Fatalf("Prompt = %+v, want nil after resume", s.Prompt)
	}
}

func TestReduce_SummaryFlow_CompactingThenSystemMessage(t *testing.T) {
	var s State

	s = Reduce(s, sessiondata.NewContentStartedEvent("sess", kit.NewSummaryContent("")))
	if s.Phase != PhaseCompacting {
		t.Fatalf("Phase after summary start = %v, want PhaseCompacting", s.Phase)
	}

	s = Reduce(s, sessiondata.NewContentDoneEvent("sess", kit.NewSummaryContent("summary text")))
	if s.Phase != PhaseRunning {
		t.Fatalf("Phase after summary done = %v, want PhaseRunning", s.Phase)
	}

	if len(s.Messages) != 1 || s.Messages[0].Role != RoleSystem || s.Messages[0].Text != "summary text" {
		t.Fatalf("Messages = %+v, want one system message with summary", s.Messages)
	}
}

func TestReduce_RecordsLastEventAt(t *testing.T) {
	ts := time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC)
	ev := sessiondata.NewContentDeltaEvent("sess", kit.NewTextContent("x"))
	ev.Timestamp = ts

	s := Reduce(State{}, ev)
	if !s.LastEventAt.Equal(ts) {
		t.Fatalf("LastEventAt = %v, want %v", s.LastEventAt, ts)
	}
}

func TestStartTurn_SetsRunningAndClearsError(t *testing.T) {
	s := State{Phase: PhaseIdle, LastError: "previous"}
	s = s.StartTurn()

	if s.Phase != PhaseRunning {
		t.Fatalf("Phase = %v, want PhaseRunning", s.Phase)
	}

	if s.LastError != "" {
		t.Fatalf("LastError = %q, want empty", s.LastError)
	}
}

func TestNewState_RecordsMode(t *testing.T) {
	s := NewState(config.Config{Mode: config.ModeYolo})

	if s.Mode != config.ModeYolo {
		t.Fatalf("Mode = %q, want %q", s.Mode, config.ModeYolo)
	}
}

func TestModeMetaLabel_IncludesModelAndMode(t *testing.T) {
	got := modeMetaLabel(&State{Model: "gpt-test", Mode: config.ModeYolo})
	want := "gpt-test · yolo"

	if got != want {
		t.Fatalf("modeMetaLabel = %q, want %q", got, want)
	}
}

func TestHasDraft(t *testing.T) {
	if (State{}).HasDraft() {
		t.Fatal("empty state should not have draft")
	}

	s := State{Streaming: &Message{Role: RoleModel}}
	if s.HasDraft() {
		t.Fatal("empty streaming message should not count as draft")
	}

	s.Streaming.Text = "x"
	if !s.HasDraft() {
		t.Fatal("streaming text should count as draft")
	}
}

func streamingThinking(s State) string {
	if s.Streaming == nil {
		return ""
	}

	return s.Streaming.Thinking
}
