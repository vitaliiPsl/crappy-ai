package jobs

import "github.com/vitaliiPsl/crappy-ai/internal/background"

type ClosedMsg struct{}

type jobsLoadedMsg struct {
	jobs []background.Job
}
