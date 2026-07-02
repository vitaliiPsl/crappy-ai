package session

import (
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/ask"
)

type EventType string

const (
	EventContentStarted EventType = "content_started"
	EventContentDelta   EventType = "content_delta"
	EventContentDone    EventType = "content_done"
	EventMessage        EventType = "message"
	EventError          EventType = "error"
	EventTurnComplete   EventType = "turn_complete"
	EventTurnCancelled  EventType = "turn_cancelled"
	EventAsk            EventType = "ask"
)

type SkillInvocation struct {
	Name string   `json:"name"`
	Args []string `json:"args,omitempty"`
}

func (s SkillInvocation) String() string {
	if len(s.Args) == 0 {
		return "/" + s.Name
	}

	return "/" + s.Name + " " + strings.Join(s.Args, " ")
}

type TurnStats struct {
	Usage         kit.Usage `json:"usage"`
	ContextUsed   int64     `json:"context_used"`
	ContextWindow int64     `json:"context_window,omitempty"`
}

type Event struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Type      EventType `json:"type"`
	Timestamp time.Time `json:"timestamp"`

	Content *kit.Content `json:"content,omitempty"`
	Message *kit.Message `json:"message,omitempty"`

	Error string `json:"error,omitempty"`

	Stats *TurnStats       `json:"stats,omitempty"`
	Skill *SkillInvocation `json:"skill,omitempty"`
	Ask   *ask.Request     `json:"ask,omitempty"`
}

func newEvent(sessionID string, t EventType) Event {
	return Event{
		ID:        uuid.NewString(),
		SessionID: sessionID,
		Type:      t,
		Timestamp: time.Now(),
	}
}

func NewContentStartedEvent(sessionID string, content kit.Content) Event {
	e := newEvent(sessionID, EventContentStarted)
	e.Content = &content

	return e
}

func NewContentDeltaEvent(sessionID string, content kit.Content) Event {
	e := newEvent(sessionID, EventContentDelta)
	e.Content = &content

	return e
}

func NewContentDoneEvent(sessionID string, content kit.Content) Event {
	e := newEvent(sessionID, EventContentDone)
	e.Content = &content

	return e
}

func NewMessageEvent(sessionID string, msg kit.Message) Event {
	e := newEvent(sessionID, EventMessage)
	e.Message = &msg

	return e
}

func NewSkillMessageEvent(sessionID string, msg kit.Message, skill SkillInvocation) Event {
	e := NewMessageEvent(sessionID, msg)
	e.Skill = &skill

	return e
}

func NewErrorEvent(sessionID string, err error) Event {
	e := newEvent(sessionID, EventError)
	e.Error = err.Error()

	return e
}

func NewTurnCompleteEvent(sessionID string, stats TurnStats) Event {
	e := newEvent(sessionID, EventTurnComplete)
	e.Stats = &stats

	return e
}

func NewTurnCancelledEvent(sessionID string) Event {
	return newEvent(sessionID, EventTurnCancelled)
}

func NewAskEvent(sessionID string, request ask.Request) Event {
	e := newEvent(sessionID, EventAsk)
	e.Ask = &request

	return e
}

func FromKitEvent(sessionID string, e kit.AgentEvent) (Event, bool) {
	switch e.Type {
	case kit.EventContentStarted:
		if e.Content == nil {
			return Event{}, false
		}

		return NewContentStartedEvent(sessionID, *e.Content), true
	case kit.EventContentDelta:
		if e.Content == nil {
			return Event{}, false
		}

		return NewContentDeltaEvent(sessionID, *e.Content), true
	case kit.EventContentDone:
		if e.Content == nil {
			return Event{}, false
		}

		return NewContentDoneEvent(sessionID, *e.Content), true
	case kit.EventMessage:
		if e.Message == nil {
			return Event{}, false
		}

		return NewMessageEvent(sessionID, *e.Message), true
	default:
		return Event{}, false
	}
}
