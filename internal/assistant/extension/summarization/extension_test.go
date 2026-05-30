package summarization

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/vitaliiPsl/crappy-adk/kit"
	"github.com/vitaliiPsl/crappy-adk/kittest"
	"github.com/vitaliiPsl/crappy-adk/x/memory"
)

type recordAgentEvents struct {
	events []kit.AgentEvent
}

func (r *recordAgentEvents) Emit(event kit.AgentEvent) error {
	r.events = append(r.events, event)

	return nil
}

func TestStrategy_RecordsSummaryAndEmitsEvents(t *testing.T) {
	msgs := []kit.Message{
		kit.NewUserMessage(kit.NewTextContent("first")),
		kit.NewModelMessage(kit.NewTextContent("second")),
	}

	model := kittest.NewModel(t, kittest.ModelResult{
		Response: kit.ModelResponse{
			Message: kit.NewModelMessage(kit.NewTextContent("recap text")),
			Usage:   kit.Usage{InputTokens: 5, OutputTokens: 2},
		},
	})

	mem := memory.NewHistory(msgs...)
	events := &recordAgentEvents{}
	rc := &kit.RunContext{
		Context: context.Background(),
		Memory:  mem,
		Events:  events,
	}

	if err := strategy(NewSummarizer(model))(rc); err != nil {
		t.Fatalf("strategy: %v", err)
	}

	if rc.Usage.InputTokens != 5 || rc.Usage.OutputTokens != 2 {
		t.Fatalf("rc.Usage = %+v, want input=5 output=2", rc.Usage)
	}

	history, err := mem.History(context.Background())
	if err != nil {
		t.Fatalf("History: %v", err)
	}

	if got := len(history); got != len(msgs)+1 {
		t.Fatalf("len(History) = %d, want original history + summary", got)
	}

	summaryMsg := history[len(history)-1]

	if summaryMsg.Role != kit.RoleUser {
		t.Fatalf("summary role = %q, want %q", summaryMsg.Role, kit.RoleUser)
	}

	if summaryMsg.Content[0].Type != kit.ContentTypeSummary {
		t.Fatalf("summary content type = %q, want %q", summaryMsg.Content[0].Type, kit.ContentTypeSummary)
	}

	if summaryMsg.Content[0].Summary.Text != "recap text" {
		t.Fatalf("summary text = %q, want recap text", summaryMsg.Content[0].Summary.Text)
	}

	if !reflect.DeepEqual(rc.Messages, []kit.Message{summaryMsg}) {
		t.Fatalf("rc.Messages = %+v, want %+v", rc.Messages, []kit.Message{summaryMsg})
	}

	if len(events.events) != 3 {
		t.Fatalf("len(events) = %d, want 3", len(events.events))
	}

	if events.events[0].Type != kit.EventContentStarted {
		t.Fatalf("event[0] type = %q, want %q", events.events[0].Type, kit.EventContentStarted)
	}

	if events.events[0].Content == nil || events.events[0].Content.Type != kit.ContentTypeSummary {
		t.Fatalf("event[0] content = %+v, want summary content", events.events[0].Content)
	}

	if events.events[1].Type != kit.EventContentDone {
		t.Fatalf("event[1] type = %q, want %q", events.events[1].Type, kit.EventContentDone)
	}

	if events.events[1].Content == nil || !reflect.DeepEqual(*events.events[1].Content, summaryMsg.Content[0]) {
		t.Fatalf("event[1] content = %+v, want summary content", events.events[1].Content)
	}

	if events.events[2].Type != kit.EventMessage {
		t.Fatalf("event[2] type = %q, want %q", events.events[2].Type, kit.EventMessage)
	}

	if events.events[2].Message == nil || !reflect.DeepEqual(*events.events[2].Message, summaryMsg) {
		t.Fatalf("event[2] message = %+v, want %+v", events.events[2].Message, summaryMsg)
	}

	model.AssertCallCount(t, 1)
}

func TestStrategy_NoOpWhenContextIsEmpty(t *testing.T) {
	model := kittest.NewModel(t)

	mem := memory.NewHistory()
	events := &recordAgentEvents{}
	rc := &kit.RunContext{
		Context: context.Background(),
		Memory:  mem,
		Events:  events,
	}

	if err := strategy(NewSummarizer(model))(rc); err != nil {
		t.Fatalf("strategy: %v", err)
	}

	model.AssertCallCount(t, 0)

	if len(events.events) != 0 {
		t.Fatalf("len(events) = %d, want 0", len(events.events))
	}

	history, err := mem.History(context.Background())
	if err != nil {
		t.Fatalf("History: %v", err)
	}

	if len(history) != 0 {
		t.Fatalf("len(History) = %d, want 0", len(history))
	}
}

func TestStrategy_PropagatesSummarizerError(t *testing.T) {
	wantErr := errors.New("model failed")

	model := kittest.NewModel(t, kittest.ModelResult{Error: wantErr})

	mem := memory.NewHistory(kit.NewUserMessage(kit.NewTextContent("msg")))
	rc := &kit.RunContext{
		Context: context.Background(),
		Memory:  mem,
		Events:  &recordAgentEvents{},
	}

	err := strategy(NewSummarizer(model))(rc)
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want wraps %v", err, wantErr)
	}

	history, _ := mem.History(context.Background())
	if len(history) != 1 {
		t.Fatalf("len(History) = %d, want 1 (no summary recorded on failure)", len(history))
	}
}

func TestStrategy_PropagatesMemoryReadError(t *testing.T) {
	wantErr := errors.New("read failed")

	model := kittest.NewModel(t)

	rc := &kit.RunContext{
		Context: context.Background(),
		Memory:  errMemory{err: wantErr},
		Events:  &recordAgentEvents{},
	}

	err := strategy(NewSummarizer(model))(rc)
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want wraps %v", err, wantErr)
	}

	model.AssertCallCount(t, 0)
}

type errMemory struct {
	err error
}

func (m errMemory) Context(context.Context) ([]kit.Message, error) { return nil, m.err }
func (m errMemory) History(context.Context) ([]kit.Message, error) { return nil, m.err }
func (errMemory) Record(context.Context, ...kit.Message) error     { return nil }
