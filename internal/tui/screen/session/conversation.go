package session

import (
	"fmt"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/vitaliiPsl/crappy-adk/kit"

	sessiondata "github.com/vitaliiPsl/crappy-ai/internal/session"
)

const bottomThreshold = 2

type messageRole string

const (
	messageRoleUser      messageRole = "user"
	messageRoleAssistant messageRole = "assistant"
	messageRoleSystem    messageRole = "system"
	messageRoleTool      messageRole = "tool"
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
	messages       []chatMessage
	showThinking   bool
	showToolResult bool

	streaming         bool
	streamingText     string
	streamingThinking string
	streamingTools    []toolUse

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

	case sessionEventMsg:
		conv.handleEvent(msg.event)

		return conv, nil

	case streamStartedMsg:
		conv.startStreaming()

		return conv, nil

	case turnStoppedMsg:
		conv.stopStreaming()

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

func (conv *conversation) loadEvents(events []sessiondata.Event) {
	conv.messages = nil
	conv.streaming = false
	conv.streamingText = ""
	conv.streamingThinking = ""
	conv.streamingTools = nil

	for _, ev := range events {
		conv.handleEvent(ev)
	}
}

func (conv *conversation) handleEvent(ev sessiondata.Event) {
	switch ev.Type {
	case sessiondata.EventContentDelta:
		conv.handleContentDelta(ev.Content)

	case sessiondata.EventContentDone:
		conv.handleContentDone(ev.Content, ev.ToolResult)

	case sessiondata.EventMessage:
		if ev.Message != nil {
			conv.appendMessage(*ev.Message)
			conv.streamingText = ""
			conv.streamingThinking = ""
			conv.streamingTools = nil
		}

	case sessiondata.EventTurnComplete, sessiondata.EventTurnCancelled:
		conv.stopStreaming()

	case sessiondata.EventError:
		conv.stopStreaming()

		if ev.Error != "" {
			conv.messages = append(conv.messages, chatMessage{error: ev.Error})
		}
	}

	conv.refreshContentPreservingFollow()
}

func (conv *conversation) handleContentDelta(content *kit.Content) {
	if content == nil {
		return
	}

	switch content.Type {
	case kit.ContentTypeText:
		if content.Text != nil {
			conv.streamingText += content.Text.Text
		}
	case kit.ContentTypeThinking:
		if content.Thinking != nil {
			conv.streamingThinking += content.Thinking.Text
		}
	case kit.ContentTypeToolResult:
		if content.ToolResult != nil {
			conv.mergeStreamingToolResult(*content.ToolResult)
		}
	}
}

func (conv *conversation) handleContentDone(content *kit.Content, result *kit.ToolResult) {
	if content == nil {
		return
	}

	switch content.Type {
	case kit.ContentTypeText:
		if content.Text != nil {
			conv.streamingText = content.Text.Text
		}
	case kit.ContentTypeThinking:
		if content.Thinking != nil {
			conv.streamingThinking = content.Thinking.Text
		}
	case kit.ContentTypeToolCall:
		if content.ToolCall != nil {
			conv.addStreamingTool(*content.ToolCall)
		}
	case kit.ContentTypeToolResult:
		if result != nil {
			conv.mergeStreamingToolResult(*result)
		} else if content.ToolResult != nil {
			conv.mergeStreamingToolResult(*content.ToolResult)
		}
	case kit.ContentTypeSummary:
		if content.Summary != nil {
			conv.messages = append(conv.messages, chatMessage{role: messageRoleSystem, text: content.Summary.Text})
		}
	}
}

func (conv *conversation) addStreamingTool(call kit.ToolCall) {
	for i := range conv.streamingTools {
		if conv.streamingTools[i].ID == call.ID {
			conv.streamingTools[i].Name = call.Name
			conv.streamingTools[i].Arguments = call.Arguments

			return
		}
	}

	conv.streamingTools = append(conv.streamingTools, toolUse{
		ID:        call.ID,
		Name:      call.Name,
		Arguments: call.Arguments,
	})
}

func (conv *conversation) mergeStreamingToolResult(result kit.ToolResult) {
	if conv.setStreamingToolResult(result) {
		return
	}

	if conv.setMessageToolResult(result) {
		return
	}

	conv.streamingTools = append(conv.streamingTools, toolUse{
		ID:        result.Call.ID,
		Name:      result.Call.Name,
		Arguments: result.Call.Arguments,
		Result:    result.Output,
		Error:     result.Error,
		Done:      true,
	})
}

func (conv *conversation) setStreamingToolResult(result kit.ToolResult) bool {
	for i := range conv.streamingTools {
		if conv.streamingTools[i].ID == result.Call.ID {
			conv.streamingTools[i].Result = result.Output
			conv.streamingTools[i].Error = result.Error
			conv.streamingTools[i].Done = true

			return true
		}
	}

	return false
}

func (conv *conversation) setMessageToolResult(result kit.ToolResult) bool {
	for i := len(conv.messages) - 1; i >= 0; i-- {
		if conv.messages[i].role != messageRoleAssistant {
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

func (conv *conversation) appendMessage(msg kit.Message) {
	if isSummaryMessage(msg) {
		conv.messages = append(conv.messages, chatMessage{role: messageRoleSystem, text: messageText(msg)})

		return
	}

	if msg.Role == kit.RoleTool {
		conv.mergeToolMessage(msg)

		return
	}

	conv.messages = append(conv.messages, toChatMessage(msg))
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

func (conv *conversation) startStreaming() {
	conv.streaming = true
	conv.streamingText = ""
	conv.streamingThinking = ""
	conv.streamingTools = nil
	conv.refreshContentPreservingFollow()
}

func (conv *conversation) stopStreaming() {
	conv.streaming = false
	conv.streamingText = ""
	conv.streamingThinking = ""
	conv.streamingTools = nil
	conv.refreshContent()
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
	role := messageRoleAssistant
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
	out := ""
	for _, content := range msg.Content {
		switch content.Type {
		case kit.ContentTypeText:
			if content.Text != nil {
				out += content.Text.Text
			}
		case kit.ContentTypeSummary:
			if content.Summary != nil {
				out += content.Summary.Text
			}
		}
	}

	return out
}

func messageThinking(msg kit.Message) string {
	out := ""
	for _, content := range msg.Content {
		if content.Type == kit.ContentTypeThinking && content.Thinking != nil {
			out += content.Thinking.Text
		}
	}

	return out
}

func toolResultText(result kit.ToolResult) string {
	if result.Error != "" {
		return fmt.Sprintf("%s: %s", result.Call.Name, result.Error)
	}

	return result.Output
}
