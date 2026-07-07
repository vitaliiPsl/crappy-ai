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

	session      *mcpsdk.ClientSession
	capabilities *mcpsdk.ServerCapabilities

	tools             []kit.Tool
	prompts           []Prompt
	resources         []Resource
	resourceTemplates []ResourceTemplate

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

func (c *sdkClient) ListPrompts(_ context.Context) ([]Prompt, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.session == nil || c.status != ClientConnected {
		return nil, fmt.Errorf("mcp: client is not connected")
	}

	return append([]Prompt(nil), c.prompts...), nil
}

func (c *sdkClient) ListResources(_ context.Context) ([]Resource, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.session == nil || c.status != ClientConnected {
		return nil, fmt.Errorf("mcp: client is not connected")
	}

	return append([]Resource(nil), c.resources...), nil
}

func (c *sdkClient) ListResourceTemplates(_ context.Context) ([]ResourceTemplate, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.session == nil || c.status != ClientConnected {
		return nil, fmt.Errorf("mcp: client is not connected")
	}

	return append([]ResourceTemplate(nil), c.resourceTemplates...), nil
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

func (c *sdkClient) GetPrompt(ctx context.Context, name string, args map[string]string) (PromptResult, error) {
	ctx, cancel := withTimeout(ctx, c.config.RequestTimeout)
	defer cancel()

	session, err := c.activeSession()
	if err != nil {
		return PromptResult{}, err
	}

	res, err := session.GetPrompt(ctx, &mcpsdk.GetPromptParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		c.handleRequestError(err)

		return PromptResult{}, err
	}

	return convertPromptResult(res), nil
}

func (c *sdkClient) ReadResource(ctx context.Context, uri string) (ResourceResult, error) {
	ctx, cancel := withTimeout(ctx, c.config.RequestTimeout)
	defer cancel()

	session, err := c.activeSession()
	if err != nil {
		return ResourceResult{}, err
	}

	res, err := session.ReadResource(ctx, &mcpsdk.ReadResourceParams{URI: uri})
	if err != nil {
		c.handleRequestError(err)

		return ResourceResult{}, err
	}

	return convertResourceResult(res), nil
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

	capabilities := serverCapabilities(session)

	tools, prompts, resources, resourceTemplates, err := c.fetchLists(ctx, session, capabilities)
	if err != nil {
		_ = session.Close()

		c.handleConnectionError(err)

		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.session = session
	c.capabilities = capabilities
	c.tools = tools
	c.prompts = prompts
	c.resources = resources
	c.resourceTemplates = resourceTemplates

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
	c.capabilities = nil
	c.clearLists()

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
				go c.refetchLists(context.Background())
			},
			PromptListChangedHandler: func(context.Context, *mcpsdk.PromptListChangedRequest) {
				go c.refetchLists(context.Background())
			},
			ResourceListChangedHandler: func(context.Context, *mcpsdk.ResourceListChangedRequest) {
				go c.refetchLists(context.Background())
			},
		},
	)

	session, err := sdk.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("mcp: connect: %w", err)
	}

	return session, nil
}

func (c *sdkClient) fetchLists(ctx context.Context, session *mcpsdk.ClientSession, capabilities *mcpsdk.ServerCapabilities) ([]kit.Tool, []Prompt, []Resource, []ResourceTemplate, error) {
	ctx, cancel := withTimeout(ctx, c.config.RequestTimeout)
	defer cancel()

	var tools []kit.Tool
	if supportsTools(capabilities) {
		var err error

		tools, err = fetchTools(ctx, c.config, c, session)
		if err != nil {
			return nil, nil, nil, nil, err
		}
	}

	var prompts []Prompt
	if supportsPrompts(capabilities) {
		var err error

		prompts, err = fetchPrompts(ctx, session)
		if err != nil {
			return nil, nil, nil, nil, err
		}
	}

	var (
		resources         []Resource
		resourceTemplates []ResourceTemplate
	)

	if supportsResources(capabilities) {
		var err error

		resources, err = fetchResources(ctx, session)
		if err != nil {
			return nil, nil, nil, nil, err
		}

		resourceTemplates, err = fetchResourceTemplates(ctx, session)
		if err != nil {
			return nil, nil, nil, nil, err
		}
	}

	return tools, prompts, resources, resourceTemplates, nil
}

func (c *sdkClient) refetchLists(ctx context.Context) {
	session, capabilities, err := c.activeConnection()
	if err != nil {
		return
	}

	tools, prompts, resources, resourceTemplates, err := c.fetchLists(ctx, session, capabilities)
	if err != nil {
		c.handleRequestError(err)

		return
	}

	c.mu.Lock()
	c.tools = tools
	c.prompts = prompts
	c.resources = resources
	c.resourceTemplates = resourceTemplates
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
	c.capabilities = nil
	c.clearLists()
	c.status = ClientDisconnected
	c.err = nil
}

func (c *sdkClient) clearLists() {
	c.tools = nil
	c.prompts = nil
	c.resources = nil
	c.resourceTemplates = nil
}

func (c *sdkClient) activeSession() (*mcpsdk.ClientSession, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.session == nil || c.status != ClientConnected {
		return nil, fmt.Errorf("mcp: client is not connected")
	}

	return c.session, nil
}

func (c *sdkClient) activeConnection() (*mcpsdk.ClientSession, *mcpsdk.ServerCapabilities, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.session == nil || c.status != ClientConnected {
		return nil, nil, fmt.Errorf("mcp: client is not connected")
	}

	return c.session, c.capabilities, nil
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

func serverCapabilities(session *mcpsdk.ClientSession) *mcpsdk.ServerCapabilities {
	if session == nil || session.InitializeResult() == nil {
		return nil
	}

	return session.InitializeResult().Capabilities
}

func supportsTools(capabilities *mcpsdk.ServerCapabilities) bool {
	return capabilities == nil || capabilities.Tools != nil
}

func supportsPrompts(capabilities *mcpsdk.ServerCapabilities) bool {
	return capabilities == nil || capabilities.Prompts != nil
}

func supportsResources(capabilities *mcpsdk.ServerCapabilities) bool {
	return capabilities == nil || capabilities.Resources != nil
}
