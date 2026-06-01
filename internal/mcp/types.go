package mcp

import (
	"context"

	"github.com/vitaliiPsl/crappy-adk/kit"
)

type Client interface {
	State() ClientState

	Connect(ctx context.Context) error
	Close() error

	ListTools(ctx context.Context) ([]kit.Tool, error)
	CallTool(ctx context.Context, call kit.ToolCall) (kit.ToolResult, error)
}

type TransportType string

const (
	TransportStdio TransportType = "stdio"
	TransportHTTP  TransportType = "http"
)

type ClientStatus string

const (
	ClientDisconnected ClientStatus = "disconnected"
	ClientConnecting   ClientStatus = "connecting"
	ClientConnected    ClientStatus = "connected"
	ClientFailed       ClientStatus = "failed"
)

type ClientState struct {
	Config Config
	Status ClientStatus
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
