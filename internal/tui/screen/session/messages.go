package session

import (
	"github.com/vitaliiPsl/crappy-ai/internal/config"
	sessiondata "github.com/vitaliiPsl/crappy-ai/internal/session"
)

type CreatedMsg struct {
	SessionID string
}

type ForkedMsg struct {
	SessionID string
}

type sessionEventMsg struct {
	event sessiondata.Event
}

type historyLoadedMsg struct {
	events []sessiondata.Event
	err    error
}

type submitMsg struct {
	Text string
}

type commandMsg struct {
	Name string
	Args []string
	Raw  string
}

type effectErrorMsg struct {
	err error
}

type modeUpdatedMsg struct {
	mode config.Mode
	err  error
}

type forkedMsg struct {
	session *sessiondata.Session
	err     error
}
