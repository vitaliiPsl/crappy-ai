package session

import (
	"sort"
	"strings"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/ask"
	sessiondata "github.com/vitaliiPsl/crappy-ai/internal/session"
)

func Reduce(s State, ev sessiondata.Event) State {
	if !ev.Timestamp.IsZero() {
		s.LastEventAt = ev.Timestamp
	}

	switch ev.Type {
	case sessiondata.EventContentStarted:
		return reduceContentStarted(s, ev.Content)
	case sessiondata.EventContentDelta:
		return reduceContentDelta(s, ev.Content)
	case sessiondata.EventContentDone:
		return reduceContentDone(s, ev.Content)
	case sessiondata.EventMessage:
		return reduceMessage(s, ev.Message, ev.Skill, ev.MCPPrompt)
	case sessiondata.EventAsk:
		return reduceAsk(s, ev.Ask)
	case sessiondata.EventTurnComplete:
		return reduceTurnComplete(s, ev.Stats)
	case sessiondata.EventTurnCancelled:
		return reduceTurnCancelled(s)
	case sessiondata.EventQueueChanged:
		s.Pending = append([]sessiondata.QueuedRequest(nil), ev.Queue...)

		return s
	case sessiondata.EventError:
		return reduceErrorEvent(s, ev.Error)
	}

	return s
}

func reduceContentStarted(s State, c *kit.Content) State {
	if c == nil {
		return s
	}

	if c.Type == kit.ContentTypeSummary {
		s.Phase = PhaseCompacting
	} else if s.Phase == PhaseIdle || s.Phase == PhaseAwaitingPermission {
		s.Phase = PhaseRunning
	}

	s.Activity = activityFor(c.Type)
	s.Prompt = nil

	return s
}

func reduceContentDelta(s State, c *kit.Content) State {
	if c == nil {
		return s
	}

	s.Prompt = nil

	if s.Phase != PhaseCompacting {
		s.Phase = PhaseRunning
	}

	s.Activity = activityFor(c.Type)

	draft := ensureDraft(&s)

	switch c.Type {
	case kit.ContentTypeText:
		if c.Text != nil {
			draft.Text += c.Text.Text
		}
	case kit.ContentTypeThinking:
		if c.Thinking != nil {
			draft.Thinking += c.Thinking.Text
		}
	}

	return s
}

func reduceContentDone(s State, c *kit.Content) State {
	if c == nil {
		return s
	}

	s.Prompt = nil

	switch c.Type {
	case kit.ContentTypeSummary:
		text := ""
		if c.Summary != nil {
			text = c.Summary.Text
		}

		s.Streaming = nil
		s.Phase = PhaseRunning

		return appendSummary(s, text)

	case kit.ContentTypeText:
		if c.Text == nil {
			return s
		}

		draft := ensureDraft(&s)
		draft.Text = c.Text.Text

		return s

	case kit.ContentTypeThinking:
		if c.Thinking == nil {
			return s
		}

		draft := ensureDraft(&s)
		draft.Thinking = c.Thinking.Text

		return s

	case kit.ContentTypeToolCall:
		if c.ToolCall == nil {
			return s
		}

		s.Activity = ActivityToolCall

		return addOrUpdateDraftTool(s, *c.ToolCall)

	case kit.ContentTypeToolResult:
		if c.ToolResult == nil {
			return s
		}

		s = mergeToolResult(s, *c.ToolResult)
		if s.ActiveTool() == nil {
			s.Activity = ActivityNone
		}

		return s
	}

	return s
}

func reduceMessage(
	s State,
	msg *kit.Message,
	skill *sessiondata.SkillInvocation,
	mcpPrompt *sessiondata.MCPPromptInvocation,
) State {
	if msg == nil {
		return s
	}

	if containsSummary(*msg) {
		s.Streaming = nil

		return appendSummary(s, kitMessageText(*msg))
	}

	if msg.Role == kit.RoleTool {
		return mergeToolMessage(s, *msg)
	}

	rendered := kitToMessage(*msg)
	if skill != nil && msg.Role == kit.RoleUser {
		rendered.Text = skillInvocationText(*skill)
	}

	if mcpPrompt != nil && msg.Role == kit.RoleUser {
		rendered.Text = mcpPromptInvocationText(*mcpPrompt)
	}

	s.Messages = append(cloneMessages(s.Messages), rendered)
	s.Streaming = nil

	return s
}

func skillInvocationText(skill sessiondata.SkillInvocation) string {
	if len(skill.Args) == 0 {
		return "/" + skill.Name
	}

	return "/" + skill.Name + " " + strings.Join(skill.Args, " ")
}

func mcpPromptInvocationText(prompt sessiondata.MCPPromptInvocation) string {
	text := "/mcp:" + prompt.Server + ":" + prompt.Name
	if len(prompt.Args) == 0 {
		return text
	}

	args := make([]string, 0, len(prompt.Args))
	for name, value := range prompt.Args {
		args = append(args, name+"="+value)
	}

	sort.Strings(args)

	return text + " " + strings.Join(args, " ")
}

func reduceAsk(s State, req *ask.Request) State {
	if req == nil {
		return s
	}

	snapshot := *req
	s.Prompt = &snapshot
	s.Phase = PhaseAwaitingPermission

	return s
}

func reduceTurnComplete(s State, stats *sessiondata.TurnStats) State {
	if stats != nil {
		snapshot := *stats
		s.Stats = &snapshot
	}

	return clearTurn(s)
}

func reduceTurnCancelled(s State) State {
	return clearTurn(s)
}

func reduceErrorEvent(s State, errText string) State {
	s = clearTurn(s)
	s.LastError = errText

	return s
}

func clearTurn(s State) State {
	s.Phase = PhaseIdle
	s.Activity = ActivityNone
	s.Streaming = nil
	s.Prompt = nil

	return s
}

func activityFor(t kit.ContentType) Activity {
	switch t {
	case kit.ContentTypeThinking:
		return ActivityThinking
	case kit.ContentTypeText:
		return ActivityGenerating
	case kit.ContentTypeSummary:
		return ActivityCompacting
	case kit.ContentTypeToolCall:
		return ActivityToolCall
	}

	return ActivityNone
}

func ensureDraft(s *State) *Message {
	if s.Streaming == nil {
		s.Streaming = &Message{
			Role: RoleModel,
		}
	}

	return s.Streaming
}

func addOrUpdateDraftTool(s State, call kit.ToolCall) State {
	draft := ensureDraft(&s)

	for i := range draft.Tools {
		if draft.Tools[i].ID == call.ID {
			draft.Tools[i].Name = call.Name
			draft.Tools[i].Arguments = call.Arguments

			return s
		}
	}

	draft.Tools = append(draft.Tools, ToolUse{
		ID:        call.ID,
		Name:      call.Name,
		Arguments: call.Arguments,
	})

	return s
}

func mergeToolResult(s State, result kit.ToolResult) State {
	if applyToolResult(s.Streaming, result) {
		return s
	}

	if msg, ok := lastModelMessage(s.Messages); ok && applyToolResult(msg, result) {
		return s
	}

	draft := ensureDraft(&s)
	draft.Tools = append(draft.Tools, ToolUse{
		ID:        result.Call.ID,
		Name:      result.Call.Name,
		Arguments: result.Call.Arguments,
		Result:    toolResultText(result),
		Error:     result.Error,
		Done:      true,
	})

	return s
}

func applyToolResult(target *Message, result kit.ToolResult) bool {
	if target == nil {
		return false
	}

	for i := range target.Tools {
		if target.Tools[i].ID == result.Call.ID {
			target.Tools[i].Result = toolResultText(result)
			target.Tools[i].Error = result.Error
			target.Tools[i].Done = true

			return true
		}
	}

	return false
}

func toolResultText(result kit.ToolResult) string {
	return kit.ContentsText(result.Output.Content)
}

func mergeToolMessage(s State, msg kit.Message) State {
	results := msg.ToolResults()
	if len(results) == 0 {
		if text := kitMessageText(msg); text != "" {
			s.Messages = append(cloneMessages(s.Messages), Message{Role: RoleTool, Text: text})
		}

		return s
	}

	for _, result := range results {
		s = mergeToolResult(s, result)
	}

	return s
}

func lastModelMessage(msgs []Message) (*Message, bool) {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == RoleModel {
			return &msgs[i], true
		}
	}

	return nil, false
}

func appendSummary(s State, text string) State {
	if text == "" {
		return s
	}

	if n := len(s.Messages); n > 0 {
		last := s.Messages[n-1]
		if last.Role == RoleSystem && last.Text == text {
			return s
		}
	}

	s.Messages = append(cloneMessages(s.Messages), Message{Role: RoleSystem, Text: text})

	return s
}

func containsSummary(msg kit.Message) bool {
	for _, c := range msg.Content {
		if c.Type == kit.ContentTypeSummary {
			return true
		}
	}

	return false
}

func kitToMessage(msg kit.Message) Message {
	role := RoleModel
	if msg.Role == kit.RoleUser {
		role = RoleUser
	}

	var tools []ToolUse
	for _, call := range msg.ToolCalls() {
		tools = append(tools, ToolUse{
			ID:        call.ID,
			Name:      call.Name,
			Arguments: call.Arguments,
		})
	}

	text := kitMessageText(msg)
	if msg.Role == kit.RoleUser {
		text = userContentText(msg.Content)
	}

	return Message{
		Role:     role,
		Text:     text,
		Thinking: kitMessageThinking(msg),
		Tools:    tools,
	}
}

func kitMessageText(msg kit.Message) string {
	var out strings.Builder
	for _, c := range msg.Content {
		switch c.Type {
		case kit.ContentTypeText:
			if c.Text != nil {
				out.WriteString(c.Text.Text)
			}
		case kit.ContentTypeSummary:
			if c.Summary != nil {
				out.WriteString(c.Summary.Text)
			}
		}
	}

	return out.String()
}

func kitMessageThinking(msg kit.Message) string {
	var out strings.Builder
	for _, c := range msg.Content {
		if c.Type == kit.ContentTypeThinking && c.Thinking != nil {
			out.WriteString(c.Thinking.Text)
		}
	}

	return out.String()
}
