package mcp

import (
	"context"

	"github.com/vitaliiPsl/crappy-adk/kit"
)

type Client interface {
	Config() Config
	Status() ClientStatus

	Connect(ctx context.Context) error
	Close() error

	ListTools(ctx context.Context) ([]kit.ToolDefinition, error)
	CallTool(ctx context.Context, call kit.ToolCall) (kit.ToolResult, error)
}

type TransportType string

const (
	TransportStdio TransportType = "stdio"
	TransportHTTP  TransportType = "http"
)

type ClientState string

const (
	ClientDisconnected ClientState = "disconnected"
	ClientConnecting   ClientState = "connecting"
	ClientConnected    ClientState = "connected"
	ClientFailed       ClientState = "failed"
)

type ClientStatus struct {
	Config Config
	State  ClientState
	Error  string
}

type Config struct {
	Name      string        `yaml:"name"`
	Transport TransportType `yaml:"type,omitempty"`
	Command   string        `yaml:"command,omitempty"`
	URL       string        `yaml:"url,omitempty"`
	Args      []string      `yaml:"args,omitempty"`
	Env       []string      `yaml:"env,omitempty"`
	Auth      AuthConfig    `yaml:"auth,omitempty"`
}

type AuthConfig struct {
	Headers   map[string]string `yaml:"headers,omitempty"`
	HeaderEnv map[string]string `yaml:"header_env,omitempty"`
}
