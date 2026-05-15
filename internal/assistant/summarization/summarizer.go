package summarization

import (
	"context"
	"fmt"
	"strings"

	"github.com/vitaliiPsl/crappy-adk/kit"
	xsummarization "github.com/vitaliiPsl/crappy-adk/x/summarization"
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
			kit.NewUserMessage(kit.NewTextContent(xsummarization.Flatten(messages))),
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
