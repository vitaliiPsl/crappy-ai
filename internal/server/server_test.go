package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/session"
	sessionstore "github.com/vitaliiPsl/crappy-ai/internal/session/store"
)

type fakeAssistant struct {
	stream *kit.Stream[session.Event, struct{}]
	err    error
}

func (a *fakeAssistant) Run(context.Context, string, string) (*kit.Stream[session.Event, struct{}], error) {
	return a.stream, a.err
}

func (a *fakeAssistant) Compact(context.Context, string) (*kit.Stream[session.Event, struct{}], error) {
	return a.stream, a.err
}

func newTestServer(t *testing.T, asst Assistant) (*Server, *session.Session) {
	t.Helper()

	store, err := sessionstore.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	sess, err := store.Create(context.Background(), "test", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	return New(asst, nil, nil, store, nil), sess
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

func TestRunTurn_FansOutStreamEvents(t *testing.T) {
	var (
		sessionID string
		delta     = kit.NewTextContent("hello")
	)

	asst := &fakeAssistant{
		stream: kit.NewStream(func(emit kit.Emitter[session.Event]) (struct{}, error) {
			if err := emit.Emit(session.NewContentDeltaEvent(sessionID, delta)); err != nil {
				return struct{}{}, err
			}

			return struct{}{}, nil
		}),
	}

	srv, sess := newTestServer(t, asst)
	sessionID = sess.ID

	ch, err := srv.Attach(context.Background(), sess.ID)
	if err != nil {
		t.Fatalf("Attach: %v", err)
	}

	defer srv.Detach(sess.ID, ch)

	if err := srv.RunTurn(context.Background(), sess.ID, "hi"); err != nil {
		t.Fatalf("RunTurn: %v", err)
	}

	ev := readEvent(t, ch)
	if ev.Type != session.EventContentDelta {
		t.Fatalf("event type = %q, want %q", ev.Type, session.EventContentDelta)
	}

	if ev.Content == nil || ev.Content.Text == nil || ev.Content.Text.Text != "hello" {
		t.Fatalf("event content = %+v, want text delta hello", ev.Content)
	}
}

func TestRunTurn_FansOutStreamResultError(t *testing.T) {
	wantErr := errors.New("stream failed")

	asst := &fakeAssistant{
		stream: kit.NewStream(func(kit.Emitter[session.Event]) (struct{}, error) {
			return struct{}{}, wantErr
		}),
	}

	srv, sess := newTestServer(t, asst)

	ch, err := srv.Attach(context.Background(), sess.ID)
	if err != nil {
		t.Fatalf("Attach: %v", err)
	}

	defer srv.Detach(sess.ID, ch)

	if err := srv.RunTurn(context.Background(), sess.ID, "hi"); err != nil {
		t.Fatalf("RunTurn: %v", err)
	}

	ev := readEvent(t, ch)
	if ev.Type != session.EventError {
		t.Fatalf("event type = %q, want %q", ev.Type, session.EventError)
	}

	if ev.Error != "stream failed" {
		t.Fatalf("event error = %q, want stream failed", ev.Error)
	}
}

func TestRunTurn_FansOutStreamResultCancellation(t *testing.T) {
	asst := &fakeAssistant{
		stream: kit.NewStream(func(kit.Emitter[session.Event]) (struct{}, error) {
			return struct{}{}, context.Canceled
		}),
	}

	srv, sess := newTestServer(t, asst)

	ch, err := srv.Attach(context.Background(), sess.ID)
	if err != nil {
		t.Fatalf("Attach: %v", err)
	}

	defer srv.Detach(sess.ID, ch)

	if err := srv.RunTurn(context.Background(), sess.ID, "hi"); err != nil {
		t.Fatalf("RunTurn: %v", err)
	}

	ev := readEvent(t, ch)
	if ev.Type != session.EventTurnCancelled {
		t.Fatalf("event type = %q, want %q", ev.Type, session.EventTurnCancelled)
	}
}
