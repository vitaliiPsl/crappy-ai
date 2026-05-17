package server

import (
	"context"
	"errors"
	"testing"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/session"
)

func TestSubscriber_NotifyDeliversEvent(t *testing.T) {
	sub := newSubscriber()
	ev := session.NewMessageEvent("s", kit.NewUserMessage(kit.NewTextContent("hi")))

	if err := sub.notify(context.Background(), ev); err != nil {
		t.Fatalf("notify: %v", err)
	}

	got := <-sub.events()
	if got.ID != ev.ID {
		t.Fatalf("event ID = %q, want %q", got.ID, ev.ID)
	}
}

func TestSubscriber_NotifyReturnsCtxErrorWhenBufferFull(t *testing.T) {
	sub := newSubscriber()

	for i := 0; i < cap(sub.ch); i++ {
		sub.ch <- session.Event{}
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := sub.notify(ctx, session.Event{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("notify err = %v, want context.Canceled", err)
	}
}

func TestSubscriber_NotifyRecoversFromClosedChannel(t *testing.T) {
	sub := newSubscriber()
	sub.close()

	if err := sub.notify(context.Background(), session.Event{}); err != nil {
		t.Fatalf("notify after close = %v, want nil", err)
	}
}

func TestSubscriber_CloseSignalsConsumer(t *testing.T) {
	sub := newSubscriber()
	sub.close()

	if _, ok := <-sub.events(); ok {
		t.Fatal("events channel should be closed")
	}
}
