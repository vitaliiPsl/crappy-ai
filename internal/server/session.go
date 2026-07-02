package server

import (
	"context"

	"github.com/vitaliiPsl/crappy-ai/internal/session"
)

func (s *Server) CreateSession(ctx context.Context, title string) (*session.Session, error) {
	return s.runtime.CreateSession(ctx, title)
}

func (s *Server) GetSession(ctx context.Context, sessionID string) (*session.Session, error) {
	return s.runtime.GetSession(ctx, sessionID)
}

func (s *Server) ListSessions(ctx context.Context) ([]*session.Session, error) {
	return s.runtime.ListSessions(ctx)
}

func (s *Server) DeleteSession(ctx context.Context, id string) error {
	return s.runtime.DeleteSession(ctx, id)
}

func (s *Server) LoadEvents(ctx context.Context, sessionID string) ([]session.Event, error) {
	return s.runtime.LoadEvents(ctx, sessionID)
}
