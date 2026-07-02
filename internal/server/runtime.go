package server

import (
	"context"

	"github.com/vitaliiPsl/crappy-ai/internal/ask"
	"github.com/vitaliiPsl/crappy-ai/internal/runtime"
)

func (s *Server) Run(ctx context.Context, sessionID string, req runtime.Request) error {
	return s.runtime.Run(ctx, sessionID, req)
}

func (s *Server) Compact(ctx context.Context, sessionID string) error {
	return s.runtime.Compact(ctx, sessionID)
}

func (s *Server) Cancel(sessionID string) {
	s.runtime.Cancel(sessionID)
}

func (s *Server) Respond(sessionID string, resp ask.Response) error {
	return s.runtime.Respond(sessionID, resp)
}
