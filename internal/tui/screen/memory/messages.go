package memory

import corememory "github.com/vitaliiPsl/crappy-ai/internal/memory"

type ClosedMsg struct{}

type memoriesLoadedMsg struct {
	memories []corememory.Memory
	selectID string
	editing  bool
	err      error
}
