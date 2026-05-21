package server

import (
	"context"
	"fmt"

	"github.com/vitaliiPsl/crappy-ai/internal/permission/model"
	"github.com/vitaliiPsl/crappy-ai/internal/session"
)

func (s *Server) Ask(ctx context.Context, sessionID string, request model.AskRequest) (model.AskResponse, error) {
	respCh := make(chan model.AskResponse, 1)
	event := session.NewPermissionPromptEvent(sessionID, request)

	st := s.getOrCreateSessionState(sessionID)

	st.mu.Lock()
	st.pending[request.Call.ID] = &pendingPrompt{event: event, response: respCh}
	st.mu.Unlock()

	defer s.removePending(sessionID, request.Call.ID)

	if err := s.broadcast(ctx, sessionID, event); err != nil {
		return model.AskResponse{}, err
	}

	select {
	case resp := <-respCh:
		return resp, nil
	case <-ctx.Done():
		return model.AskResponse{}, ctx.Err()
	}
}

func (s *Server) RespondPrompt(sessionID, toolCallID string, resp model.AskResponse) error {
	st, ok := s.getSessionState(sessionID)
	if !ok {
		return fmt.Errorf("no pending prompts for session %q", sessionID)
	}

	st.mu.Lock()

	p, ok := st.pending[toolCallID]
	if ok {
		delete(st.pending, toolCallID)
	}
	st.mu.Unlock()

	if !ok {
		return fmt.Errorf("no pending prompt %q", toolCallID)
	}

	p.response <- resp

	return nil
}

func (s *Server) removePending(sessionID, toolCallID string) {
	st, ok := s.getSessionState(sessionID)
	if !ok {
		return
	}

	st.mu.Lock()
	delete(st.pending, toolCallID)
	st.mu.Unlock()
}
