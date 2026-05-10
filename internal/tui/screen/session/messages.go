package session

import "github.com/vitaliiPsl/crappy-ai/internal/session"

type CreatedMsg struct {
	SessionID string
}

type sessionEventMsg struct {
	event session.Event
}

type historyLoadedMsg struct {
	events []session.Event
	err    error
}

type streamStartedMsg struct{}

type turnStoppedMsg struct{}

type errorMsg struct {
	err error
}
