package server

import (
	"context"

	"github.com/vitaliiPsl/crappy-ai/internal/session"
)

const eventBuffer = 64

type subscriber struct {
	ch chan session.Event
}

func newSubscriber() *subscriber {
	return &subscriber{
		ch: make(chan session.Event, eventBuffer),
	}
}

func (s *subscriber) events() <-chan session.Event {
	return s.ch
}

func (s *subscriber) notify(ctx context.Context, ev session.Event) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = nil
		}
	}()

	select {
	case s.ch <- ev:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *subscriber) close() {
	close(s.ch)
}
