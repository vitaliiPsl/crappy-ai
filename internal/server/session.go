package server

import (
	"context"

	"github.com/vitaliiPsl/crappy-ai/internal/session"
)

func (s *Server) CreateSession(ctx context.Context, title string) (*session.Session, error) {
	config := s.configStore.Get()

	return s.sessionStore.Create(ctx, title, config.Cwd)
}

func (s *Server) GetSession(ctx context.Context, sessionID string) (*session.Session, error) {
	return s.sessionStore.Get(ctx, sessionID)
}

func (s *Server) ListSessions(ctx context.Context) ([]*session.Session, error) {
	return s.sessionStore.List(ctx)
}

func (s *Server) DeleteSession(ctx context.Context, id string) error {
	return s.sessionStore.Delete(ctx, id)
}

func (s *Server) LoadEvents(ctx context.Context, sessionID string) ([]session.Event, error) {
	return s.sessionStore.LoadEvents(ctx, sessionID)
}

func (s *Server) getOrCreateSessionState(sessionID string) *sessionState {
	s.mu.RLock()
	st, ok := s.sessions[sessionID]
	s.mu.RUnlock()

	if ok {
		return st
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if st, ok = s.sessions[sessionID]; ok {
		return st
	}

	st = &sessionState{}
	s.sessions[sessionID] = st

	return st
}
