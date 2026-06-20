package summarization

import (
	"github.com/vitaliiPsl/crappy-adk/kit"
	xsummarization "github.com/vitaliiPsl/crappy-adk/x/summarization"

	"github.com/vitaliiPsl/crappy-ai/internal/assistant/factory"
	"github.com/vitaliiPsl/crappy-ai/internal/assistant/spec"
)

const thresholdRatio = 0.75

type ext struct{}

func New() factory.Extension {
	return &ext{}
}

func (e *ext) Name() string {
	return "summarization"
}

func (e *ext) Spec(ctx factory.Context) (spec.AgentSpec, error) {
	return spec.AgentSpec{
		Hooks: []spec.HookSpec{
			{
				Name:   "Summarize when context is large",
				Source: e.Name(),
				Kind:   spec.HookTurnStart,
				Option: xsummarization.WithSummarization(
					whenLastInputExceeds(ctx.Model.Config(), thresholdRatio),
					strategy(NewSummarizer(ctx.Model)),
				),
			},
		},
	}, nil
}

func strategy(summarizer *Summarizer) func(*kit.RunContext) error {
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
