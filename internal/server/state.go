package server

import (
	"context"
	"sync"

	"github.com/vitaliiPsl/crappy-ai/internal/permission/model"
	"github.com/vitaliiPsl/crappy-ai/internal/session"
)

type sessionState struct {
	mu      sync.Mutex
	subs    []*subscriber
	cancel  context.CancelFunc
	pending map[string]*pendingPrompt
}

type pendingPrompt struct {
	event    session.Event
	response chan model.AskResponse
}

func (st *sessionState) subscribe(ctx context.Context) (*subscriber, error) {
	st.mu.Lock()
	sub := newSubscriber()
	st.subs = append(st.subs, sub)

	pending := make([]session.Event, 0, len(st.pending))
	for _, p := range st.pending {
		pending = append(pending, p.event)
	}
	st.mu.Unlock()

	for _, ev := range pending {
		if err := sub.notify(ctx, ev); err != nil {
			st.unsubscribe(sub.ch)

			return nil, err
		}
	}

	return sub, nil
}

func (st *sessionState) unsubscribe(ch <-chan session.Event) (removed, last bool) {
	st.mu.Lock()
	defer st.mu.Unlock()

	for i, sub := range st.subs {
		if sub.ch == ch {
			st.subs = append(st.subs[:i], st.subs[i+1:]...)

			sub.close()

			removed = true

			break
		}
	}

	return removed, len(st.subs) == 0
}

func (st *sessionState) broadcast(ctx context.Context, ev session.Event) error {
	st.mu.Lock()
	snapshot := make([]*subscriber, len(st.subs))
	copy(snapshot, st.subs)
	st.mu.Unlock()

	for _, sub := range snapshot {
		if err := sub.notify(ctx, ev); err != nil {
			return err
		}
	}

	return nil
}

func (st *sessionState) idle() bool {
	return len(st.subs) == 0 && len(st.pending) == 0 && st.cancel == nil
}

func (s *Server) getSessionState(sessionID string) (*sessionState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	st, ok := s.sessions[sessionID]

	return st, ok
}

func (s *Server) getOrCreateSessionState(sessionID string) *sessionState {
	if st, ok := s.getSessionState(sessionID); ok {
		return st
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	st := &sessionState{
		pending: make(map[string]*pendingPrompt),
	}
	s.sessions[sessionID] = st

	return st
}
