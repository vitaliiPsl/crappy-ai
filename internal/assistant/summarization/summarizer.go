package summarization

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/vitaliiPsl/crappy-adk/kit"
)

type Summarizer struct {
	model kit.Model
}

type Result struct {
	Text  string
	Usage kit.Usage
}

func NewSummarizer(model kit.Model) *Summarizer {
	return &Summarizer{model: model}
}

func (s *Summarizer) Summarize(ctx context.Context, messages []kit.Message) (Result, error) {
	resp, err := s.model.Generate(ctx, kit.ModelRequest{
		Instructions: Prompt,
		Messages: []kit.Message{
			kit.NewUserMessage(kit.NewTextContent(flatten(messages))),
		},
	})
	if err != nil {
		return Result{}, fmt.Errorf("summarization: model call failed: %w", err)
	}

	text := responseText(resp.Message)
	if text == "" {
		return Result{}, emptyResponseError(resp)
	}

	return Result{Text: text, Usage: resp.Usage}, nil
}

func emptyResponseError(resp kit.ModelResponse) error {
	return fmt.Errorf(
		"summarization: model returned no text (finish=%q, content=[%s])",
		resp.FinishReason,
		contentTypes(resp.Message),
	)
}

func responseText(msg kit.Message) string {
	var b strings.Builder
	for _, c := range msg.Content {
		if c.Type == kit.ContentTypeText && c.Text != nil {
			b.WriteString(c.Text.Text)
		}
	}

	return b.String()
}

func contentTypes(msg kit.Message) string {
	types := make([]string, 0, len(msg.Content))
	for _, c := range msg.Content {
		types = append(types, string(c.Type))
	}

	return strings.Join(types, ",")
}

// flatten renders the conversation as a single text transcript so the
// summarizer model receives plain text instead of structured turns. This
// avoids provider-specific constraints on thought signatures and function
// call parts (notably Gemini), and is sufficient for summarization where
// conversation structure does not need to be preserved on the wire.
func flatten(messages []kit.Message) string {
	var b strings.Builder
	for i, msg := range messages {
		if i > 0 {
			b.WriteByte('\n')
		}

		writeTurn(&b, msg)
	}

	return strings.TrimRight(b.String(), "\n")
}

func writeTurn(b *strings.Builder, msg kit.Message) {
	b.WriteString(roleLabel(msg.Role))
	b.WriteString(":\n")

	for _, c := range msg.Content {
		writeContent(b, c)
	}
}

func writeContent(b *strings.Builder, c kit.Content) {
	switch c.Type {
	case kit.ContentTypeText:
		writeText(b, c.Text)
	case kit.ContentTypeSummary:
		writeSummary(b, c.Summary)
	case kit.ContentTypeToolCall:
		writeToolCall(b, c.ToolCall)
	case kit.ContentTypeToolResult:
		writeToolResult(b, c.ToolResult)
	}
}

func writeText(b *strings.Builder, t *kit.Text) {
	if t == nil {
		return
	}

	b.WriteString(t.Text)
	b.WriteByte('\n')
}

func writeSummary(b *strings.Builder, s *kit.Summary) {
	if s == nil {
		return
	}

	b.WriteString("[Previous summary]\n")
	b.WriteString(s.Text)
	b.WriteByte('\n')
}

func writeToolCall(b *strings.Builder, c *kit.ToolCall) {
	if c == nil {
		return
	}

	args, _ := json.Marshal(c.Arguments)
	fmt.Fprintf(b, "[Tool call: %s(%s)]\n", c.Name, args)
}

func writeToolResult(b *strings.Builder, r *kit.ToolResult) {
	if r == nil {
		return
	}

	if r.Error != "" {
		fmt.Fprintf(b, "[Tool error from %s: %s]\n", r.Call.Name, r.Error)

		return
	}

	fmt.Fprintf(b, "[Tool result from %s: %s]\n", r.Call.Name, r.Output)
}

func roleLabel(r kit.Role) string {
	switch r {
	case kit.RoleUser:
		return "User"
	case kit.RoleModel:
		return "Assistant"
	case kit.RoleTool:
		return "Tool"
	default:
		return string(r)
	}
}
