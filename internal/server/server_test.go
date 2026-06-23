package server

import (
	"context"
	"testing"
	"time"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/assistant"
	"github.com/vitaliiPsl/crappy-ai/internal/session"
	sessionstore "github.com/vitaliiPsl/crappy-ai/internal/session/store"
)

type fakeAssistant struct {
	stream   *kit.Stream[session.Event, struct{}]
	streamFn func(ctx context.Context) *kit.Stream[session.Event, struct{}]
	err      error
}

func (a *fakeAssistant) Run(ctx context.Context, _ string, _ assistant.RunRequest) (*kit.Stream[session.Event, struct{}], error) {
	return a.openStream(ctx)
}

func (a *fakeAssistant) Compact(ctx context.Context, _ string) (*kit.Stream[session.Event, struct{}], error) {
	return a.openStream(ctx)
}

func (a *fakeAssistant) openStream(ctx context.Context) (*kit.Stream[session.Event, struct{}], error) {
	if a.err != nil {
		return nil, a.err
	}

	if a.streamFn != nil {
		return a.streamFn(ctx), nil
	}

	return a.stream, nil
}

func TestAttach_UnknownSessionReturnsError(t *testing.T) {
	srv, _ := newTestServer(t, &fakeAssistant{})

	if _, err := srv.Subscribe(context.Background(), "missing"); err == nil {
		t.Fatal("Subscribe for missing session should fail")
	}
}

func TestBroadcast_DeliversToAllSubscribers(t *testing.T) {
	srv, sess := newTestServer(t, &fakeAssistant{})

	ch1, err := srv.Subscribe(context.Background(), sess.ID)
	if err != nil {
		t.Fatalf("Subscribe 1: %v", err)
	}

	defer srv.Unsubscribe(sess.ID, ch1)

	ch2, err := srv.Subscribe(context.Background(), sess.ID)
	if err != nil {
		t.Fatalf("Subscribe 2: %v", err)
	}

	defer srv.Unsubscribe(sess.ID, ch2)

	ev := session.NewMessageEvent(sess.ID, kit.NewUserMessage(kit.NewTextContent("hi")))
	if err := srv.broadcast(context.Background(), sess.ID, ev); err != nil {
		t.Fatalf("broadcast: %v", err)
	}

	if got := readEvent(t, ch1); got.ID != ev.ID {
		t.Fatalf("ch1 event ID = %q, want %q", got.ID, ev.ID)
	}

	if got := readEvent(t, ch2); got.ID != ev.ID {
		t.Fatalf("ch2 event ID = %q, want %q", got.ID, ev.ID)
	}
}

func TestDetach_StopsDeliveryAndRetiresWhenIdle(t *testing.T) {
	srv, sess := newTestServer(t, &fakeAssistant{})

	ch1, err := srv.Subscribe(context.Background(), sess.ID)
	if err != nil {
		t.Fatalf("Subscribe 1: %v", err)
	}

	ch2, err := srv.Subscribe(context.Background(), sess.ID)
	if err != nil {
		t.Fatalf("Subscribe 2: %v", err)
	}

	srv.Unsubscribe(sess.ID, ch1)

	if _, ok := <-ch1; ok {
		t.Fatal("detached channel should be closed")
	}

	ev := session.NewMessageEvent(sess.ID, kit.NewUserMessage(kit.NewTextContent("only ch2")))
	if err := srv.broadcast(context.Background(), sess.ID, ev); err != nil {
		t.Fatalf("broadcast: %v", err)
	}

	if got := readEvent(t, ch2); got.ID != ev.ID {
		t.Fatalf("ch2 event ID = %q, want %q", got.ID, ev.ID)
	}

	srv.Unsubscribe(sess.ID, ch2)

	if _, ok := srv.getSessionState(sess.ID); ok {
		t.Fatal("session state should be retired once idle")
	}
}

func newTestServer(t *testing.T, asst Assistant) (*Server, *session.Session) {
	t.Helper()

	store, err := sessionstore.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	sess, err := store.Create(context.Background(), session.CreateParams{Title: "test"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	return New(asst, nil, nil, store, nil, nil, nil, nil), sess
}

func readEvent(t *testing.T, ch <-chan session.Event) session.Event {
	t.Helper()

	select {
	case ev := <-ch:
		return ev
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}

	return session.Event{}
}
