package runtime

import (
	"context"
	"testing"
	"time"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/ask"
	"github.com/vitaliiPsl/crappy-ai/internal/session"
)

func bareSession() *Session {
	return newSession("s1", nil, nil, nil, nil, nil, nil, nil)
}

func textRequest(text string) Request {
	return Request{Content: []kit.Content{kit.NewTextContent(text)}}
}

func recv(t *testing.T, ch <-chan session.Event) session.Event {
	t.Helper()

	select {
	case ev := <-ch:
		return ev
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")

		return session.Event{}
	}
}

func TestBroadcastReachesSubscribers(t *testing.T) {
	s := bareSession()

	sub := s.Subscribe()

	want := session.NewMessageEvent("s1", kit.NewModelMessage(kit.NewTextContent("hi")))
	s.events.Publish(want)

	if got := recv(t, sub.Events()); got.Type != session.EventMessage {
		t.Fatalf("event type = %q, want %q", got.Type, session.EventMessage)
	}
}

func TestAskRoundTrip(t *testing.T) {
	s := bareSession()
	sub := s.Subscribe()

	req := ask.Request{ID: "r1", Title: "Allow bash?", Options: []ask.Option{{ID: "allow", Label: "Allow"}}}

	answered := make(chan ask.Response, 1)
	go func() {
		resp, _ := s.Ask(context.Background(), req)
		answered <- resp
	}()

	got := recv(t, sub.Events())
	if got.Type != session.EventAsk || got.Ask == nil || got.Ask.ID != "r1" {
		t.Fatalf("event = %+v, want an ask event for r1", got)
	}

	if err := s.Respond(ask.Response{RequestID: "r1", OptionID: "allow"}); err != nil {
		t.Fatalf("Respond: %v", err)
	}

	select {
	case resp := <-answered:
		if resp.OptionID != "allow" {
			t.Fatalf("Ask returned %q, want allow", resp.OptionID)
		}
	case <-time.After(time.Second):
		t.Fatal("Ask did not return after Respond")
	}
}

func TestCompactRejectedWhileTurnIsActive(t *testing.T) {
	s := bareSession()

	s.mu.Lock()
	s.cancel = func() {}
	s.mu.Unlock()

	if err := s.Compact(context.Background()); err == nil {
		t.Fatal("compaction during an active turn should fail")
	}
}

func TestRunQueuesFollowUpWhileTurnIsActive(t *testing.T) {
	s := bareSession()
	s.cancel = func() {}
	sub := s.Subscribe()

	if err := s.Run(context.Background(), textRequest("next")); err != nil {
		t.Fatalf("Run: %v", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.pending) != 1 {
		t.Fatalf("pending turns = %d, want 1", len(s.pending))
	}

	if text := kit.ContentsText(s.pending[0].Request.Content); text != "next" {
		t.Fatalf("pending request text = %q, want next", text)
	}

	event := recv(t, sub.Events())
	if event.Type != session.EventQueueChanged || len(event.Queue) != 1 {
		t.Fatalf("event = %+v, want queued turn", event)
	}

	if kit.ContentsText(event.Queue[0].Request.Content) != "next" {
		t.Fatalf("queued request = %+v, want text next", event.Queue[0])
	}
}

func TestUpdateQueuedPublishesQueueSnapshot(t *testing.T) {
	s := bareSession()
	s.pending = []QueuedRequest{{
		ID:      "queued",
		Request: textRequest("before"),
	}}
	sub := s.Subscribe()

	if err := s.UpdateQueued("queued", textRequest("after")); err != nil {
		t.Fatalf("UpdateQueued: %v", err)
	}

	event := recv(t, sub.Events())
	if len(event.Queue) != 1 || kit.ContentsText(event.Queue[0].Request.Content) != "after" {
		t.Fatalf("queue snapshot = %+v, want updated request", event.Queue)
	}
}

func TestRemoveQueuedPublishesQueueSnapshot(t *testing.T) {
	s := bareSession()
	s.pending = []QueuedRequest{{ID: "queued", Request: textRequest("remove me")}}
	sub := s.Subscribe()

	if err := s.RemoveQueued("queued"); err != nil {
		t.Fatalf("RemoveQueued: %v", err)
	}

	event := recv(t, sub.Events())
	if event.Type != session.EventQueueChanged || len(event.Queue) != 0 {
		t.Fatalf("event = %+v, want empty queue snapshot", event)
	}
}
