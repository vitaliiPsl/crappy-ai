package session

import (
	"time"

	"github.com/vitaliiPsl/crappy-ai/internal/config"
	"github.com/vitaliiPsl/crappy-ai/internal/permission/model"
	sessiondata "github.com/vitaliiPsl/crappy-ai/internal/session"
)

type Phase int

const (
	PhaseIdle Phase = iota
	PhaseRunning
	PhaseCompacting
	PhaseAwaitingPermission
)

type Role int

const (
	RoleUser Role = iota
	RoleTool
	RoleModel
	RoleSystem
)

type ToolUse struct {
	ID        string
	Name      string
	Arguments map[string]any
	Result    string
	Error     string
	Done      bool
}

type Message struct {
	Role     Role
	Text     string
	Thinking string
	Tools    []ToolUse
	Error    string
}

type State struct {
	ID       string
	Title    string
	Cwd      string
	Model    string
	Provider string

	Messages  []Message
	Streaming *Message

	Phase  Phase
	Stats  *sessiondata.TurnStats
	Prompt *model.AskRequest

	LastError   string
	LastEventAt time.Time
}

func NewState(cfg config.Config) State {
	return State{
		Cwd:      cfg.Cwd,
		Model:    cfg.Model,
		Provider: cfg.Provider,
	}
}

func (s State) WithSession(sess *sessiondata.Session) State {
	if sess == nil {
		return s
	}

	s.ID = sess.ID
	s.Title = sess.Title

	if sess.Cwd != "" {
		s.Cwd = sess.Cwd
	}

	return s
}

func (s State) Reset() State {
	s.Messages = nil
	s.Streaming = nil
	s.Phase = PhaseIdle
	s.Stats = nil
	s.Prompt = nil
	s.LastError = ""
	s.LastEventAt = time.Time{}

	return s
}

func (s State) StartTurn() State {
	s.Phase = PhaseRunning
	s.LastError = ""

	return s
}

func (s State) AnswerPrompt() State {
	s.Prompt = nil
	s.Phase = PhaseRunning

	return s
}

func (s State) SetError(err error) State {
	s.Phase = PhaseIdle
	s.Streaming = nil

	if err != nil {
		s.LastError = err.Error()
	}

	return s
}

func (s State) ClearError() State {
	s.LastError = ""

	return s
}

func (s State) AppendSystem(text string) State {
	if text == "" {
		return s
	}

	s.Messages = append(
		cloneMessages(s.Messages), Message{
			Role: RoleSystem,
			Text: text,
		},
	)

	return s
}

func (s State) HasDraft() bool {
	if s.Streaming == nil {
		return false
	}

	return s.Streaming.Text != "" ||
		s.Streaming.Thinking != "" ||
		len(s.Streaming.Tools) > 0 ||
		s.Streaming.Error != ""
}

func (s State) ActiveTool() *ToolUse {
	if s.Streaming == nil {
		return nil
	}

	for i := len(s.Streaming.Tools) - 1; i >= 0; i-- {
		if !s.Streaming.Tools[i].Done {
			return &s.Streaming.Tools[i]
		}
	}

	return nil
}

func cloneMessages(in []Message) []Message {
	out := make([]Message, len(in), len(in)+1)
	copy(out, in)

	return out
}
