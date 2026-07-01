package memory

import (
	"context"
	"errors"
	"testing"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/session"
)

const testSessionID = "session-1"

type fakeStore struct {
	events      []session.Event
	loadErr     error
	appendErr   error
	appendCalls int
}

func (s *fakeStore) LoadEvents(context.Context, string) ([]session.Event, error) {
	if s.loadErr != nil {
		return nil, s.loadErr
	}

	return s.events, nil
}

func (s *fakeStore) AppendEvents(_ context.Context, _ string, events ...session.Event) error {
	s.appendCalls++

	if s.appendErr != nil {
		return s.appendErr
	}

	s.events = append(s.events, events...)

	return nil
}

func (s *fakeStore) Create(context.Context, session.CreateParams) (*session.Session, error) {
	panic("fakeStore.Create not implemented")
}

func (s *fakeStore) Get(context.Context, string) (*session.Session, error) {
	panic("fakeStore.Get not implemented")
}

func (s *fakeStore) List(context.Context) ([]*session.Session, error) {
	panic("fakeStore.List not implemented")
}

func (s *fakeStore) Delete(context.Context, string) error {
	panic("fakeStore.Delete not implemented")
}

func (s *fakeStore) Save(context.Context, *session.Session) error {
	panic("fakeStore.Save not implemented")
}

func (s *fakeStore) SaveArtifact(context.Context, string, string, any) error {
	panic("fakeStore.SaveArtifact not implemented")
}

func (s *fakeStore) LoadArtifact(context.Context, string, string, any) (bool, error) {
	panic("fakeStore.LoadArtifact not implemented")
}

func (s *fakeStore) ListArtifacts(context.Context, string) ([]string, error) {
	panic("fakeStore.ListArtifacts not implemented")
}

func (s *fakeStore) DeleteArtifact(context.Context, string, string) error {
	panic("fakeStore.DeleteArtifact not implemented")
}

func newStoreWithMessages(messages ...kit.Message) *fakeStore {
	store := &fakeStore{}
	for _, msg := range messages {
		store.events = append(store.events, session.NewMessageEvent(testSessionID, msg))
	}

	return store
}

func TestMemoryRecordPersistsMessages(t *testing.T) {
	store := &fakeStore{}
	mem := New(store, testSessionID)

	first := kit.NewUserMessage(kit.NewTextContent("first"))
	second := kit.NewModelMessage(kit.NewTextContent("second"))

	if err := mem.Record(context.Background(), first, second); err != nil {
		t.Fatalf("Record: %v", err)
	}

	if len(store.events) != 2 {
		t.Fatalf("len(events) = %d, want 2", len(store.events))
	}

	for i, want := range []string{"first", "second"} {
		ev := store.events[i]
		if ev.Type != session.EventMessage {
			t.Fatalf("events[%d].Type = %q, want %q", i, ev.Type, session.EventMessage)
		}

		if ev.SessionID != testSessionID {
			t.Fatalf("events[%d].SessionID = %q, want %q", i, ev.SessionID, testSessionID)
		}

		if ev.Message == nil || ev.Message.TextContent().Text != want {
			t.Fatalf("events[%d] text = %+v, want %q", i, ev.Message, want)
		}
	}
}

func TestMemoryRecordEmptySkipsStore(t *testing.T) {
	store := &fakeStore{}
	mem := New(store, testSessionID)

	if err := mem.Record(context.Background()); err != nil {
		t.Fatalf("Record: %v", err)
	}

	if store.appendCalls != 0 {
		t.Fatalf("appendCalls = %d, want 0", store.appendCalls)
	}
}

func TestMemoryRecordReturnsStoreError(t *testing.T) {
	wantErr := errors.New("append failed")
	store := &fakeStore{appendErr: wantErr}
	mem := New(store, testSessionID)

	err := mem.Record(context.Background(), kit.NewUserMessage(kit.NewTextContent("x")))
	if !errors.Is(err, wantErr) {
		t.Fatalf("Record error = %v, want %v", err, wantErr)
	}
}

func TestMemoryHistoryReturnsRecordedMessages(t *testing.T) {
	first := kit.NewUserMessage(kit.NewTextContent("first"))
	second := kit.NewModelMessage(kit.NewTextContent("second"))

	store := newStoreWithMessages(first, second)
	mem := New(store, testSessionID)

	got, err := mem.History(context.Background())
	if err != nil {
		t.Fatalf("History: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("len(History) = %d, want 2", len(got))
	}

	if got[0].TextContent().Text != "first" || got[1].TextContent().Text != "second" {
		t.Fatalf("History = %+v, want first then second", got)
	}
}

func TestMemoryHistoryIgnoresNonMessageEvents(t *testing.T) {
	kept := kit.NewUserMessage(kit.NewTextContent("kept"))

	store := &fakeStore{events: []session.Event{
		session.NewMessageEvent(testSessionID, kept),
		session.NewErrorEvent(testSessionID, errors.New("ignored")),
		session.NewTurnCompleteEvent(testSessionID, session.TurnStats{}),
		session.NewTurnCancelledEvent(testSessionID),
	}}

	mem := New(store, testSessionID)

	got, err := mem.History(context.Background())
	if err != nil {
		t.Fatalf("History: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("len(History) = %d, want 1", len(got))
	}

	if got[0].TextContent().Text != "kept" {
		t.Fatalf("History[0] text = %q, want kept", got[0].TextContent().Text)
	}
}

func TestMemoryHistoryIgnoresMessageEventsWithoutPayload(t *testing.T) {
	kept := kit.NewUserMessage(kit.NewTextContent("kept"))

	store := &fakeStore{events: []session.Event{
		{ID: "x", SessionID: testSessionID, Type: session.EventMessage, Message: nil},
		session.NewMessageEvent(testSessionID, kept),
	}}

	mem := New(store, testSessionID)

	got, err := mem.History(context.Background())
	if err != nil {
		t.Fatalf("History: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("len(History) = %d, want 1", len(got))
	}

	if got[0].TextContent().Text != "kept" {
		t.Fatalf("History[0] text = %q, want kept", got[0].TextContent().Text)
	}
}

func TestMemoryHistoryReturnsStoreError(t *testing.T) {
	wantErr := errors.New("load failed")
	store := &fakeStore{loadErr: wantErr}
	mem := New(store, testSessionID)

	if _, err := mem.History(context.Background()); !errors.Is(err, wantErr) {
		t.Fatalf("History error = %v, want %v", err, wantErr)
	}
}

func TestMemoryContextReturnsFullHistoryWhenNoSummary(t *testing.T) {
	first := kit.NewUserMessage(kit.NewTextContent("first"))
	second := kit.NewModelMessage(kit.NewTextContent("second"))

	store := newStoreWithMessages(first, second)
	mem := New(store, testSessionID)

	got, err := mem.Context(context.Background())
	if err != nil {
		t.Fatalf("Context: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("len(Context) = %d, want 2", len(got))
	}

	if got[0].TextContent().Text != "first" || got[1].TextContent().Text != "second" {
		t.Fatalf("Context = %+v, want first then second", got)
	}
}

func TestMemoryContextStartsAtLatestSummary(t *testing.T) {
	old := kit.NewUserMessage(kit.NewTextContent("old"))
	firstSummary := kit.NewUserMessage(kit.NewSummaryContent("first summary"))
	middle := kit.NewUserMessage(kit.NewTextContent("middle"))
	latestSummary := kit.NewUserMessage(kit.NewSummaryContent("latest summary"))
	recent := kit.NewUserMessage(kit.NewTextContent("recent"))

	store := newStoreWithMessages(old, firstSummary, middle, latestSummary, recent)
	mem := New(store, testSessionID)

	got, err := mem.Context(context.Background())
	if err != nil {
		t.Fatalf("Context: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("len(Context) = %d, want latest summary + recent", len(got))
	}

	if got[0].Content[0].Summary == nil || got[0].Content[0].Summary.Text != "latest summary" {
		t.Fatalf("Context[0] summary = %+v, want latest summary", got[0].Content[0].Summary)
	}

	if got[1].TextContent().Text != "recent" {
		t.Fatalf("Context[1] = %q, want recent", got[1].TextContent().Text)
	}

	history, err := mem.History(context.Background())
	if err != nil {
		t.Fatalf("History: %v", err)
	}

	if len(history) != 5 {
		t.Fatalf("len(History) = %d, want full history length 5", len(history))
	}
}

func TestMemoryContextIncludesSummaryWhenLast(t *testing.T) {
	old := kit.NewUserMessage(kit.NewTextContent("old"))
	summary := kit.NewUserMessage(kit.NewSummaryContent("recap"))

	store := newStoreWithMessages(old, summary)
	mem := New(store, testSessionID)

	got, err := mem.Context(context.Background())
	if err != nil {
		t.Fatalf("Context: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("len(Context) = %d, want 1", len(got))
	}

	if got[0].Content[0].Summary == nil || got[0].Content[0].Summary.Text != "recap" {
		t.Fatalf("Context[0] summary = %+v, want recap", got[0].Content[0].Summary)
	}
}

func TestMemoryContextReturnsStoreError(t *testing.T) {
	wantErr := errors.New("load failed")
	store := &fakeStore{loadErr: wantErr}
	mem := New(store, testSessionID)

	if _, err := mem.Context(context.Background()); !errors.Is(err, wantErr) {
		t.Fatalf("Context error = %v, want %v", err, wantErr)
	}
}
