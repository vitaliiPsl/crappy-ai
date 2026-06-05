package mcp

import (
	"context"
	"errors"
	"fmt"
	"sync"

	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/vitaliiPsl/crappy-adk/kit"
)

const (
	clientName    = "crappy"
	clientVersion = "0.1.0"
)

type sdkClient struct {
	config       Config
	newTransport TransportFactory

	connMu sync.Mutex
	mu     sync.RWMutex

	session *mcpsdk.ClientSession
	tools   []kit.Tool

	status ClientStatus
	err    error
}

func NewClient(config Config, transport TransportFactory) Client {
	return &sdkClient{
		config:       config,
		newTransport: transport,
		status:       ClientDisconnected,
	}
}

func (c *sdkClient) Config() Config {
	return c.config
}

func (c *sdkClient) State() ClientState {
	c.mu.RLock()
	defer c.mu.RUnlock()

	err := ""
	if c.err != nil {
		err = c.err.Error()
	}

	return ClientState{
		Status: c.status,
		Error:  err,
	}
}

func (c *sdkClient) Connect(ctx context.Context) error {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	if session, _ := c.activeSession(); session != nil {
		return nil
	}

	return c.connectLocked(ctx)
}

func (c *sdkClient) Close() error {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	return c.closeLocked()
}

func (c *sdkClient) ListTools(_ context.Context) ([]kit.Tool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.session == nil || c.status != ClientConnected {
		return nil, fmt.Errorf("mcp: client is not connected")
	}

	return c.tools, nil
}

func (c *sdkClient) CallTool(ctx context.Context, call kit.ToolCall) (kit.ToolResult, error) {
	ctx, cancel := withTimeout(ctx, c.config.RequestTimeout)
	defer cancel()

	session, err := c.activeSession()
	if err != nil {
		return kit.ToolResult{}, err
	}

	res, err := session.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      call.Name,
		Arguments: call.Arguments,
	})
	if err != nil {
		c.handleRequestError(err)

		return kit.ToolResult{}, err
	}

	return convertToolResult(call, res), nil
}

func (c *sdkClient) connectLocked(ctx context.Context) error {
	if !c.config.IsEnabled() {
		return fmt.Errorf("mcp: client is disabled")
	}

	c.setStatus(ClientConnecting, nil)

	session, err := c.dial(ctx)
	if err != nil {
		c.handleConnectionError(err)

		return err
	}

	tools, err := c.fetchTools(ctx, session)
	if err != nil {
		_ = session.Close()

		c.handleConnectionError(err)

		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.session = session
	c.tools = tools

	c.status = ClientConnected
	c.err = nil

	go c.watch(session)

	return nil
}

func (c *sdkClient) closeLocked() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	session := c.session
	c.session = nil
	c.tools = nil

	c.status = ClientDisconnected
	c.err = nil

	if session == nil {
		return nil
	}

	return session.Close()
}

func (c *sdkClient) dial(ctx context.Context) (*mcpsdk.ClientSession, error) {
	ctx, cancelConnect := withTimeout(ctx, c.config.ConnectTimeout)
	defer cancelConnect()

	transport, err := c.newTransport(c.config)
	if err != nil {
		return nil, err
	}

	sdk := mcpsdk.NewClient(
		&mcpsdk.Implementation{
			Name:    clientName,
			Version: clientVersion,
		},
		&mcpsdk.ClientOptions{
			ToolListChangedHandler: func(context.Context, *mcpsdk.ToolListChangedRequest) {
				go c.refetchTools(context.Background())
			},
		},
	)

	session, err := sdk.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("mcp: connect: %w", err)
	}

	return session, nil
}

func (c *sdkClient) fetchTools(ctx context.Context, session *mcpsdk.ClientSession) ([]kit.Tool, error) {
	ctx, cancel := withTimeout(ctx, c.config.RequestTimeout)
	defer cancel()

	res, err := session.ListTools(ctx, nil)
	if err != nil {
		return nil, err
	}

	defs, err := convertTools(res.Tools)
	if err != nil {
		return nil, err
	}

	tools := make([]kit.Tool, 0, len(defs))
	for _, def := range defs {
		tools = append(tools, newTool(c.config.Name, c, def))
	}

	return tools, nil
}

func (c *sdkClient) refetchTools(ctx context.Context) {
	session, err := c.activeSession()
	if err != nil {
		return
	}

	tools, err := c.fetchTools(ctx, session)
	if err != nil {
		c.handleRequestError(err)

		return
	}

	c.mu.Lock()
	c.tools = tools
	c.mu.Unlock()
}

func (c *sdkClient) watch(session *mcpsdk.ClientSession) {
	_ = session.Wait()

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.session != session {
		return
	}

	c.session = nil
	c.tools = nil
	c.status = ClientDisconnected
	c.err = nil
}

func (c *sdkClient) activeSession() (*mcpsdk.ClientSession, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.session == nil || c.status != ClientConnected {
		return nil, fmt.Errorf("mcp: client is not connected")
	}

	return c.session, nil
}

func (c *sdkClient) setStatus(status ClientStatus, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.status = status
	c.err = err
}

func (c *sdkClient) handleConnectionError(err error) {
	if errors.Is(err, mcpauth.ErrOAuth) {
		c.setStatus(ClientAuthRequired, err)

		return
	}

	c.setStatus(ClientFailed, err)
}

func (c *sdkClient) handleRequestError(err error) {
	if errors.Is(err, mcpauth.ErrOAuth) {
		c.setStatus(ClientAuthRequired, err)
	}
}
