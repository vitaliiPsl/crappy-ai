package session

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/vitaliiPsl/crappy-adk/kit"

	sessiondata "github.com/vitaliiPsl/crappy-ai/internal/session"
)

const bottomThreshold = 2

type messageRole string

const (
	messageRoleUser   messageRole = "user"
	messageRoleModel  messageRole = "model"
	messageRoleSystem messageRole = "system"
	messageRoleTool   messageRole = "tool"
)

type toolUse struct {
	ID        string
	Name      string
	Arguments map[string]any
	Result    string
	Error     string
	Done      bool
}

type chatMessage struct {
	role     messageRole
	text     string
	thinking string
	tools    []toolUse
	error    string
}

type conversation struct {
	messages []chatMessage

	runActive  bool
	compacting bool
	draft      chatMessage

	showThinking   bool
	showToolResult bool

	viewport viewport.Model

	provider string
	model    string

	width  int
	height int
}

func newConversation(provider, model string) conversation {
	vp := viewport.New()
	vp.SoftWrap = true

	return conversation{
		viewport: vp,
		provider: provider,
		model:    model,
	}
}

func (conv conversation) Init() tea.Cmd {
	return nil
}

func (conv conversation) Update(msg tea.Msg) (conversation, tea.Cmd) {
	switch msg := msg.(type) {
	case historyLoadedMsg:
		if msg.err == nil {
			conv.loadEvents(msg.events)
		}

		return conv, nil

	case runStartedMsg:
		conv.startRun()

		return conv, nil

	case runStoppedMsg:
		conv.stopRun()

		return conv, nil

	case sessionEventMsg:
		conv.handleRunEvent(msg.event)

		return conv, nil

	case systemMessageMsg:
		conv.messages = append(conv.messages, chatMessage{
			role: messageRoleSystem,
			text: msg.Text,
		})
		conv.refreshContent()

		return conv, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+o":
			conv.showThinking = !conv.showThinking
			conv.refreshContent()

			return conv, nil

		case "ctrl+t":
			conv.showToolResult = !conv.showToolResult
			conv.refreshContent()

			return conv, nil
		}
	}

	var cmd tea.Cmd

	conv.viewport, cmd = conv.viewport.Update(msg)

	return conv, cmd
}

func (conv conversation) View() string {
	return conv.viewport.View()
}

func (conv *conversation) setSize(width, height int) {
	conv.width = width
	conv.height = height
	conv.viewport.SetWidth(width)
	conv.viewport.SetHeight(height)
	conv.refreshContent()
}

func (conv *conversation) startRun() {
	conv.runActive = true
	conv.resetDraft()
	conv.refreshContentPreservingFollow()
}

func (conv *conversation) stopRun() {
	conv.runActive = false
	conv.compacting = false
	conv.resetDraft()
	conv.refreshContent()
}

func (conv *conversation) loadEvents(events []sessiondata.Event) {
	conv.messages = nil
	conv.runActive = false
	conv.compacting = false
	conv.resetDraft()

	for _, ev := range events {
		conv.handleEvent(ev)
	}

	conv.refreshContent()
	conv.viewport.GotoBottom()
}

func (conv *conversation) handleRunEvent(ev sessiondata.Event) {
	conv.handleEvent(ev)
	conv.refreshContentPreservingFollow()
}

func (conv *conversation) handleEvent(ev sessiondata.Event) {
	switch ev.Type {
	case sessiondata.EventContentStarted:
		conv.handleContentStarted(ev.Content)

	case sessiondata.EventContentDelta:
		conv.handleContentDelta(ev.Content)

	case sessiondata.EventContentDone:
		conv.handleContentDone(ev.Content)

	case sessiondata.EventMessage:
		if ev.Message != nil {
			conv.commitMessage(*ev.Message)
		}

	case sessiondata.EventTurnComplete, sessiondata.EventTurnCancelled:
		conv.stopRun()

	case sessiondata.EventError:
		conv.stopRun()

		if ev.Error != "" {
			conv.messages = append(conv.messages, chatMessage{error: ev.Error})
		}
	}
}

func (conv *conversation) handleContentStarted(content *kit.Content) {
	if content == nil {
		return
	}

	if content.Type == kit.ContentTypeSummary {
		conv.compacting = true
	}
}

func (conv *conversation) handleContentDelta(content *kit.Content) {
	if content == nil {
		return
	}

	switch content.Type {
	case kit.ContentTypeText:
		if content.Text != nil {
			conv.ensureDraft().text += content.Text.Text
		}
	case kit.ContentTypeThinking:
		if content.Thinking != nil {
			conv.ensureDraft().thinking += content.Thinking.Text
		}
	}
}

func (conv *conversation) handleContentDone(content *kit.Content) {
	if content == nil {
		return
	}

	switch content.Type {
	case kit.ContentTypeSummary:
		conv.compacting = false
		conv.resetDraft()

		if content.Summary != nil {
			conv.appendSummaryMessage(content.Summary.Text)
		}
	case kit.ContentTypeText:
		if content.Text != nil {
			conv.ensureDraft().text = content.Text.Text
		}
	case kit.ContentTypeThinking:
		if content.Thinking != nil {
			conv.ensureDraft().thinking = content.Thinking.Text
		}
	case kit.ContentTypeToolCall:
		if content.ToolCall != nil {
			conv.addDraftTool(*content.ToolCall)
		}
	case kit.ContentTypeToolResult:
		if content.ToolResult != nil {
			conv.mergeDraftToolResult(*content.ToolResult)
		}
	}
}

func (conv *conversation) commitMessage(msg kit.Message) {
	defer conv.resetDraft()

	if isSummaryMessage(msg) {
		conv.appendSummaryMessage(messageText(msg))

		return
	}

	if msg.Role == kit.RoleTool {
		conv.mergeToolMessage(msg)

		return
	}

	conv.messages = append(conv.messages, toChatMessage(msg))
}

func (conv *conversation) appendSummaryMessage(text string) {
	if text == "" {
		return
	}

	if len(conv.messages) > 0 {
		last := conv.messages[len(conv.messages)-1]
		if last.role == messageRoleSystem && last.text == text {
			return
		}
	}

	conv.messages = append(conv.messages, chatMessage{role: messageRoleSystem, text: text})
}

func (conv *conversation) mergeToolMessage(msg kit.Message) {
	for _, result := range msg.ToolResults() {
		if !conv.setMessageToolResult(result) {
			conv.messages = append(conv.messages, chatMessage{
				role: messageRoleTool,
				text: toolResultText(result),
			})
		}
	}

	if len(msg.ToolResults()) == 0 && messageText(msg) != "" {
		conv.messages = append(conv.messages, chatMessage{role: messageRoleTool, text: messageText(msg)})
	}
}

func (conv *conversation) addDraftTool(call kit.ToolCall) {
	msg := conv.ensureDraft()
	for i := range msg.tools {
		if msg.tools[i].ID == call.ID {
			msg.tools[i].Name = call.Name
			msg.tools[i].Arguments = call.Arguments

			return
		}
	}

	msg.tools = append(msg.tools, toolUse{
		ID:        call.ID,
		Name:      call.Name,
		Arguments: call.Arguments,
	})
}

func (conv *conversation) mergeDraftToolResult(result kit.ToolResult) {
	if conv.setDraftToolResult(result) {
		return
	}

	if conv.setMessageToolResult(result) {
		return
	}

	msg := conv.ensureDraft()
	msg.tools = append(msg.tools, toolUse{
		ID:        result.Call.ID,
		Name:      result.Call.Name,
		Arguments: result.Call.Arguments,
		Result:    result.Output,
		Error:     result.Error,
		Done:      true,
	})
}

func (conv *conversation) setDraftToolResult(result kit.ToolResult) bool {
	for i := range conv.draft.tools {
		if conv.draft.tools[i].ID == result.Call.ID {
			conv.draft.tools[i].Result = result.Output
			conv.draft.tools[i].Error = result.Error
			conv.draft.tools[i].Done = true

			return true
		}
	}

	return false
}

func (conv *conversation) setMessageToolResult(result kit.ToolResult) bool {
	for i := len(conv.messages) - 1; i >= 0; i-- {
		if conv.messages[i].role != messageRoleModel {
			continue
		}

		for j := range conv.messages[i].tools {
			if conv.messages[i].tools[j].ID == result.Call.ID {
				conv.messages[i].tools[j].Result = result.Output
				conv.messages[i].tools[j].Error = result.Error
				conv.messages[i].tools[j].Done = true

				return true
			}
		}

		return false
	}

	return false
}

func (conv *conversation) ensureDraft() *chatMessage {
	if conv.draft.role == "" {
		conv.draft.role = messageRoleModel
	}

	return &conv.draft
}

func (conv *conversation) resetDraft() {
	conv.draft = chatMessage{}
}

func (conv *conversation) hasDraft() bool {
	return conv.draft.text != "" ||
		conv.draft.thinking != "" ||
		len(conv.draft.tools) > 0 ||
		conv.draft.error != ""
}

func (conv *conversation) refreshContentPreservingFollow() {
	shouldFollow := conv.isNearBottom()
	conv.refreshContent()

	if shouldFollow {
		conv.viewport.GotoBottom()
	}
}

func (conv conversation) isNearBottom() bool {
	total := conv.viewport.TotalLineCount()
	if total <= conv.viewport.Height() {
		return true
	}

	return conv.viewport.YOffset()+conv.viewport.Height() >= total-bottomThreshold
}

func toChatMessage(msg kit.Message) chatMessage {
	role := messageRoleModel
	if msg.Role == kit.RoleUser {
		role = messageRoleUser
	}

	var tools []toolUse
	for _, call := range msg.ToolCalls() {
		tools = append(tools, toolUse{
			ID:        call.ID,
			Name:      call.Name,
			Arguments: call.Arguments,
		})
	}

	return chatMessage{
		role:     role,
		text:     messageText(msg),
		thinking: messageThinking(msg),
		tools:    tools,
	}
}

func isSummaryMessage(msg kit.Message) bool {
	for _, content := range msg.Content {
		if content.Type == kit.ContentTypeSummary {
			return true
		}
	}

	return false
}

func messageText(msg kit.Message) string {
	var out strings.Builder
	for _, content := range msg.Content {
		switch content.Type {
		case kit.ContentTypeText:
			if content.Text != nil {
				out.WriteString(content.Text.Text)
			}
		case kit.ContentTypeSummary:
			if content.Summary != nil {
				out.WriteString(content.Summary.Text)
			}
		}
	}

	return out.String()
}

func messageThinking(msg kit.Message) string {
	var out strings.Builder
	for _, content := range msg.Content {
		if content.Type == kit.ContentTypeThinking && content.Thinking != nil {
			out.WriteString(content.Thinking.Text)
		}
	}

	return out.String()
}

func toolResultText(result kit.ToolResult) string {
	if result.Error != "" {
		return fmt.Sprintf("%s: %s", result.Call.Name, result.Error)
	}

	return result.Output
}
