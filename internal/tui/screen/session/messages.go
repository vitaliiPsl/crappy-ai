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

type runStartedMsg struct{}

type runStoppedMsg struct{}

type submitMsg struct {
	Text string
}

type commandMsg struct {
	Name string
	Args []string
}

type systemMessageMsg struct {
	Text string
}

type errorMsg struct {
	err error
}
