package summarization

import (
	"github.com/vitaliiPsl/crappy-adk/agent"
	"github.com/vitaliiPsl/crappy-adk/kit"
	xsummarization "github.com/vitaliiPsl/crappy-adk/x/summarization"

	"github.com/vitaliiPsl/crappy-ai/internal/assistant/extension"
)

const thresholdRatio = 0.75

type ext struct{}

func New() extension.Extension {
	return &ext{}
}

func (e *ext) Name() string {
	return "summarization"
}

func (e *ext) Options(ctx extension.Context) (agent.Option, error) {
	return xsummarization.WithSummarization(
		whenLastInputExceeds(ctx.Model.Config(), thresholdRatio),
		strategy(NewSummarizer(ctx.Model)),
	), nil
}

func strategy(summarizer *Summarizer) xsummarization.Strategy {
	return func(rc *kit.RunContext) error {
		result, err := summarizer.Summarize(rc.Context, rc.Memory, rc.Emit)
		if err != nil || result == nil {
			return err
		}

		rc.Usage.Add(result.Usage)
		rc.Append(result.Summary)

		return nil
	}
}
