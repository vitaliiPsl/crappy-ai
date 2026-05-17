package server

import (
	"context"
	"sync"

	"github.com/vitaliiPsl/crappy-ai/internal/permission"
	"github.com/vitaliiPsl/crappy-ai/internal/session"
)

type sessionState struct {
	mu         sync.Mutex
	clients    []chan session.Event
	cancelTurn context.CancelFunc
	pending    map[string]*pendingPrompt
}

type pendingPrompt struct {
	event    session.Event
	response chan permission.Response
}

func (st *sessionState) removeClient(ch <-chan session.Event) (last bool) {
	st.mu.Lock()
	defer st.mu.Unlock()

	for i, c := range st.clients {
		if c == ch {
			st.clients = append(st.clients[:i], st.clients[i+1:]...)

			close(c)

			break
		}
	}

	return len(st.clients) == 0
}

func (st *sessionState) idle() bool {
	return len(st.clients) == 0 && len(st.pending) == 0 && st.cancelTurn == nil
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
