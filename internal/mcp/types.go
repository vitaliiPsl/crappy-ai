package mcp

import (
	"context"

	"github.com/vitaliiPsl/crappy-adk/kit"
)

type Client interface {
	Config() Config

	Connect(ctx context.Context) error
	Close() error

	ListTools(ctx context.Context) ([]kit.ToolDefinition, error)
	CallTool(ctx context.Context, call kit.ToolCall) (kit.ToolResult, error)
}

type Config struct {
	Name    string
	Command string
	Args    []string
	Env     []string
	URL     string
}
