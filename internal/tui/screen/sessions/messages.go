package sessions

import "github.com/vitaliiPsl/crappy-ai/internal/session"

type OpenSessionMsg struct {
	SessionID string
}

type OpenDraftSessionMsg struct{}

type ClosedMsg struct{}

type sessionsLoadedMsg struct {
	sessions []*session.Session
	err      error
	cursor   int
}
