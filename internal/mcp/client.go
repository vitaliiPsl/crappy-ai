package mcp

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/vitaliiPsl/crappy-adk/kit"
)

const (
	clientName    = "crappy"
	clientVersion = "0.1.0"
)

type sdkClient struct {
	config Config

	ctx    context.Context
	cancel context.CancelFunc

	mu      sync.Mutex
	session *mcpsdk.ClientSession
}

func NewClient(config Config) Client {
	ctx, cancel := context.WithCancel(context.Background())

	return &sdkClient{
		config: config,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (c *sdkClient) Config() Config {
	return c.config
}

func (c *sdkClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.session != nil {
		return nil
	}

	if c.config.Command == "" {
		return fmt.Errorf("mcp: client %q has no command", c.config.Name)
	}

	cmd := exec.CommandContext(c.ctx, c.config.Command, c.config.Args...)
	cmd.Env = append(os.Environ(), c.config.Env...)

	sdk := mcpsdk.NewClient(&mcpsdk.Implementation{Name: clientName, Version: clientVersion}, nil)

	session, err := sdk.Connect(ctx, &mcpsdk.CommandTransport{Command: cmd}, nil)
	if err != nil {
		return fmt.Errorf("mcp: connect %q: %w", c.config.Name, err)
	}

	c.session = session

	return nil
}

func (c *sdkClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	defer c.cancel()

	if c.session == nil {
		return nil
	}

	err := c.session.Close()
	c.session = nil

	return err
}

func (c *sdkClient) ListTools(ctx context.Context) ([]kit.ToolDefinition, error) {
	session, err := c.sessionOrErr()
	if err != nil {
		return nil, err
	}

	res, err := session.ListTools(ctx, nil)
	if err != nil {
		return nil, err
	}

	return convertTools(res.Tools)
}

func (c *sdkClient) CallTool(ctx context.Context, call kit.ToolCall) (kit.ToolResult, error) {
	session, err := c.sessionOrErr()
	if err != nil {
		return kit.ToolResult{}, err
	}

	res, err := session.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      call.Name,
		Arguments: call.Arguments,
	})
	if err != nil {
		return kit.ToolResult{}, err
	}

	return convertToolResult(call, res), nil
}

func (c *sdkClient) sessionOrErr() (*mcpsdk.ClientSession, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.session != nil {
		return c.session, nil
	}

	return nil, fmt.Errorf("mcp: client %q is not connected", c.config.Name)
}
