package extension

import (
	"context"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/assistant/spec"
	"github.com/vitaliiPsl/crappy-ai/internal/config"
)

type Context struct {
	Ctx       context.Context
	SessionID string
	Config    config.Config
	Model     kit.Model
}

type Extension interface {
	Name() string
	Context(ctx Context) ([]spec.ContextPiece, error)
	Tools(ctx Context) ([]spec.ToolSpec, error)
	Hooks(ctx Context) ([]spec.HookSpec, error)
}
