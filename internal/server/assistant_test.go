package server

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/session"
)

func TestSend_FansOutStreamEvents(t *testing.T) {
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

	ch, err := srv.Subscribe(context.Background(), sess.ID)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	defer srv.Unsubscribe(sess.ID, ch)

	if err := srv.Send(context.Background(), sess.ID, "hi"); err != nil {
		t.Fatalf("Send: %v", err)
	}

	ev := readEvent(t, ch)
	if ev.Type != session.EventContentDelta {
		t.Fatalf("event type = %q, want %q", ev.Type, session.EventContentDelta)
	}

	if ev.Content == nil || ev.Content.Text == nil || ev.Content.Text.Text != "hello" {
		t.Fatalf("event content = %+v, want text delta hello", ev.Content)
	}
}

func TestSend_FansOutStreamResultError(t *testing.T) {
	wantErr := errors.New("stream failed")

	asst := &fakeAssistant{
		stream: kit.NewStream(func(kit.Emitter[session.Event]) (struct{}, error) {
			return struct{}{}, wantErr
		}),
	}

	srv, sess := newTestServer(t, asst)

	ch, err := srv.Subscribe(context.Background(), sess.ID)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	defer srv.Unsubscribe(sess.ID, ch)

	if err := srv.Send(context.Background(), sess.ID, "hi"); err != nil {
		t.Fatalf("Send: %v", err)
	}

	ev := readEvent(t, ch)
	if ev.Type != session.EventError {
		t.Fatalf("event type = %q, want %q", ev.Type, session.EventError)
	}

	if ev.Error != "stream failed" {
		t.Fatalf("event error = %q, want stream failed", ev.Error)
	}
}

func TestSend_FansOutStreamResultCancellation(t *testing.T) {
	asst := &fakeAssistant{
		stream: kit.NewStream(func(kit.Emitter[session.Event]) (struct{}, error) {
			return struct{}{}, context.Canceled
		}),
	}

	srv, sess := newTestServer(t, asst)

	ch, err := srv.Subscribe(context.Background(), sess.ID)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	defer srv.Unsubscribe(sess.ID, ch)

	if err := srv.Send(context.Background(), sess.ID, "hi"); err != nil {
		t.Fatalf("Send: %v", err)
	}

	ev := readEvent(t, ch)
	if ev.Type != session.EventTurnCancelled {
		t.Fatalf("event type = %q, want %q", ev.Type, session.EventTurnCancelled)
	}
}

func TestSend_RefusesWhileRunActive(t *testing.T) {
	hold := make(chan struct{})
	defer close(hold)

	asst := &fakeAssistant{
		stream: kit.NewStream(func(kit.Emitter[session.Event]) (struct{}, error) {
			<-hold

			return struct{}{}, nil
		}),
	}

	srv, sess := newTestServer(t, asst)

	ch, err := srv.Subscribe(context.Background(), sess.ID)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	defer srv.Unsubscribe(sess.ID, ch)

	if err := srv.Send(context.Background(), sess.ID, "first"); err != nil {
		t.Fatalf("first Send: %v", err)
	}

	err = srv.Send(context.Background(), sess.ID, "second")
	if err == nil {
		t.Fatal("second Send should fail while first is active")
	}

	if !strings.Contains(err.Error(), "already has an active turn") {
		t.Fatalf("error = %q, want it to mention an active turn", err)
	}
}

func TestCompact_FansOutStreamEvents(t *testing.T) {
	var sessionID string

	summary := kit.NewSummaryContent("recap")

	asst := &fakeAssistant{
		streamFn: func(_ context.Context) *kit.Stream[session.Event, struct{}] {
			return kit.NewStream(func(emit kit.Emitter[session.Event]) (struct{}, error) {
				return struct{}{}, emit.Emit(session.NewContentDoneEvent(sessionID, summary))
			})
		},
	}

	srv, sess := newTestServer(t, asst)
	sessionID = sess.ID

	ch, err := srv.Subscribe(context.Background(), sess.ID)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	defer srv.Unsubscribe(sess.ID, ch)

	if err := srv.Compact(context.Background(), sess.ID); err != nil {
		t.Fatalf("Compact: %v", err)
	}

	ev := readEvent(t, ch)
	if ev.Type != session.EventContentDone {
		t.Fatalf("event type = %q, want %q", ev.Type, session.EventContentDone)
	}

	if ev.Content == nil || ev.Content.Type != kit.ContentTypeSummary {
		t.Fatalf("event content = %+v, want summary", ev.Content)
	}
}

func TestCancelRun_PropagatesCancellation(t *testing.T) {
	asst := &fakeAssistant{
		streamFn: func(ctx context.Context) *kit.Stream[session.Event, struct{}] {
			return kit.NewStream(func(kit.Emitter[session.Event]) (struct{}, error) {
				<-ctx.Done()

				return struct{}{}, ctx.Err()
			})
		},
	}

	srv, sess := newTestServer(t, asst)

	ch, err := srv.Subscribe(context.Background(), sess.ID)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	defer srv.Unsubscribe(sess.ID, ch)

	if err := srv.Send(context.Background(), sess.ID, "hi"); err != nil {
		t.Fatalf("Send: %v", err)
	}

	srv.CancelRun(sess.ID)

	ev := readEvent(t, ch)
	if ev.Type != session.EventTurnCancelled {
		t.Fatalf("event type = %q, want %q", ev.Type, session.EventTurnCancelled)
	}
}

func TestCancelRun_NoActiveRunIsNoop(t *testing.T) {
	srv, sess := newTestServer(t, &fakeAssistant{})

	// Should not panic or block when called against an unknown or idle session.
	srv.CancelRun("missing")
	srv.CancelRun(sess.ID)
}
