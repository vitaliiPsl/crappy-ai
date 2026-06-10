package tools

import (
	"fmt"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/background"
	"github.com/vitaliiPsl/crappy-ai/internal/tools/bash"
	filesystem "github.com/vitaliiPsl/crappy-ai/internal/tools/fs"
	"github.com/vitaliiPsl/crappy-ai/internal/tools/web"
)

func Core(backgroundManager *background.Manager) []kit.Tool {
	return []kit.Tool{
		wrapBackground(bash.NewBash(), backgroundManager),
		web.NewFetch(),
		filesystem.NewReadFile(),
		filesystem.NewWriteFile(),
		filesystem.NewEditFile(),
		filesystem.NewListDirectory(),
	}
}

func wrapBackground(t kit.Tool, manager *background.Manager) kit.Tool {
	wrapped, err := background.Wrap(t, manager)
	if err != nil {
		panic(fmt.Sprintf("wrap tool %q for background: %v", t.Definition().Name, err))
	}

	return wrapped
}
