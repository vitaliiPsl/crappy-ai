package summarization

import (
	"encoding/json"
	"math"

	"github.com/vitaliiPsl/crappy-adk/kit"
	xsummarization "github.com/vitaliiPsl/crappy-adk/x/summarization"
)

func whenEstimatedContextExceeds(config kit.ModelConfig, ratio float64) xsummarization.Trigger {
	limit := inputLimit(config)
	if limit <= 0 {
		return func(*kit.RunContext) bool { return false }
	}

	if ratio <= 0 || ratio > 1 {
		ratio = thresholdRatio
	}

	threshold := max(int(math.Floor(float64(limit)*ratio)), 1)

	return func(rc *kit.RunContext) bool {
		messages, err := rc.Memory.Context(rc.Context)
		if err != nil {
			return false
		}

		return estimateMessagesTokens(messages) >= threshold
	}
}

func inputLimit(config kit.ModelConfig) int {
	if config.InputLimit > 0 {
		return config.InputLimit
	}

	if config.ContextWindow <= 0 {
		return 0
	}

	if config.OutputLimit > 0 && config.OutputLimit < config.ContextWindow {
		return config.ContextWindow - config.OutputLimit
	}

	return config.ContextWindow
}

func estimateMessagesTokens(messages []kit.Message) int {
	total := 0
	for _, msg := range messages {
		total += messageTokenOverhead
		for _, content := range msg.Content {
			total += estimateContentTokens(content)
		}
	}

	return total
}

func estimateContentTokens(content kit.Content) int {
	bytes := 0

	switch content.Type {
	case kit.ContentTypeText:
		if content.Text != nil {
			bytes = len(content.Text.Text)
		}
	case kit.ContentTypeThinking:
		if t := content.Thinking; t != nil {
			bytes = len(t.ID) + len(t.Text) + len(t.Signature)
		}
	case kit.ContentTypeSummary:
		if content.Summary != nil {
			bytes = len(content.Summary.Text)
		}
	case kit.ContentTypeToolCall:
		if c := content.ToolCall; c != nil {
			bytes = len(c.ID) + len(c.Name) + jsonLen(c.Arguments) + len(c.Signature)
		}
	case kit.ContentTypeToolResult:
		if r := content.ToolResult; r != nil {
			bytes = len(r.Call.ID) + len(r.Call.Name) + len(r.Output) + len(r.Error)
		}
	}

	if bytes == 0 {
		return 0
	}

	return int(math.Ceil(float64(bytes) / estimatedCharsPerToken))
}

func jsonLen(value any) int {
	if value == nil {
		return 0
	}

	data, err := json.Marshal(value)
	if err != nil {
		return 0
	}

	return len(data)
}
