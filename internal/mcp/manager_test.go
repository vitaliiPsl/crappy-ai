package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/vitaliiPsl/crappy-adk/kit"
)

type fakeClient struct {
	config            Config
	status            ClientStatus
	tools             []kit.Tool
	prompts           []Prompt
	resources         []Resource
	resourceTemplates []ResourceTemplate
	err               error

	connects int
	closes   int
	called   kit.ToolCall
	result   kit.ToolResult
	prompt   []kit.Message
	resource []kit.Content
}

func (c *fakeClient) Config() Config {
	return c.config
}

func (c *fakeClient) State() ClientState {
	status := c.status
	if status == "" {
		status = ClientConnected
	}

	state := ClientState{Status: status}
	if c.err != nil {
		state.Error = c.err.Error()
	}

	return state
}

func (c *fakeClient) Connect(context.Context) error {
	c.connects++

	return c.err
}

func (c *fakeClient) Close() error {
	c.closes++

	return c.err
}

func (c *fakeClient) ListTools(context.Context) ([]kit.Tool, error) {
	return c.tools, c.err
}

func (c *fakeClient) ListPrompts(context.Context) ([]Prompt, error) {
	return c.prompts, c.err
}

func (c *fakeClient) ListResources(context.Context) ([]Resource, error) {
	return c.resources, c.err
}

func (c *fakeClient) ListResourceTemplates(context.Context) ([]ResourceTemplate, error) {
	return c.resourceTemplates, c.err
}

func (c *fakeClient) CallTool(_ context.Context, call kit.ToolCall) (kit.ToolResult, error) {
	c.called = call

	return c.result, c.err
}

func (c *fakeClient) GetPrompt(context.Context, string, map[string]string) ([]kit.Message, error) {
	return c.prompt, c.err
}

func (c *fakeClient) ReadResource(context.Context, string) ([]kit.Content, error) {
	return c.resource, c.err
}

type fakeAuthenticator struct {
	config Config
	err    error
	calls  int
}

func (a *fakeAuthenticator) Authenticate(_ context.Context, cfg Config) error {
	a.config = cfg
	a.calls++

	return a.err
}

func newManager(clients ...Client) *Manager {
	byName := make(map[string]Client, len(clients))
	for _, client := range clients {
		byName[client.Config().Name] = client
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Manager{
		ctx:     ctx,
		cancel:  cancel,
		clients: byName,
		newClient: func(cfg Config) Client {
			return &fakeClient{config: cfg, status: ClientDisconnected}
		},
		authenticator: &fakeAuthenticator{},
	}
}

func TestManagerConnectConnectsClients(t *testing.T) {
	first := &fakeClient{config: Config{Name: "first"}}
	second := &fakeClient{config: Config{Name: "second"}}

	if err := newManager(first, second).Connect(); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	if first.connects != 1 || second.connects != 1 {
		t.Fatalf("connects = %d/%d, want 1/1", first.connects, second.connects)
	}
}

func TestManagerConnectReturnsErrors(t *testing.T) {
	want := errors.New("boom")

	err := newManager(&fakeClient{err: want}).Connect()
	if !errors.Is(err, want) {
		t.Fatalf("Connect error = %v, want %v", err, want)
	}
}

func TestManagerConnectSkipsDisabledClients(t *testing.T) {
	disabled := false
	client := &fakeClient{config: Config{Name: "github", Enabled: &disabled}}

	if err := newManager(client).Connect(); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	if client.connects != 0 {
		t.Fatalf("connects = %d, want 0", client.connects)
	}
}

func TestManagerSnapshotsReturnsClientConfigAndState(t *testing.T) {
	client := &fakeClient{
		config: Config{Name: "github"},
		status: ClientFailed,
		err:    errors.New("boom"),
	}

	snapshots := newManager(client).Snapshots()
	if len(snapshots) != 1 {
		t.Fatalf("len(snapshots) = %d, want 1", len(snapshots))
	}

	if snapshots[0].Config.Name != "github" {
		t.Fatalf("config = %+v, want github", snapshots[0].Config)
	}

	if snapshots[0].State.Status != ClientFailed || snapshots[0].State.Error != "boom" {
		t.Fatalf("state = %+v, want failed boom", snapshots[0].State)
	}
}

func TestManagerApplyConfigDisablesByReplacingClient(t *testing.T) {
	client := &fakeClient{config: Config{Name: "github"}}
	manager := newManager(client)
	cfg := client.Config()
	disabled := false
	cfg.Enabled = &disabled

	if err := manager.ApplyConfig(context.Background(), cfg); err != nil {
		t.Fatalf("ApplyConfig: %v", err)
	}

	if client.closes != 1 {
		t.Fatalf("old client closes = %d, want 1", client.closes)
	}

	snapshots := manager.Snapshots()
	if len(snapshots) != 1 || snapshots[0].Config.IsEnabled() {
		t.Fatalf("snapshots = %+v, want disabled github", snapshots)
	}

	if snapshots[0].State.Status != ClientDisconnected {
		t.Fatalf("state = %+v, want disconnected replacement", snapshots[0].State)
	}
}

func TestManagerApplyConfigEnablesByReplacingAndConnectingClient(t *testing.T) {
	disabled := false
	old := &fakeClient{config: Config{Name: "github", Enabled: &disabled}}
	next := &fakeClient{status: ClientDisconnected}
	manager := newManager(old)
	manager.newClient = func(cfg Config) Client {
		next.config = cfg

		return next
	}
	cfg := old.Config()
	cfg.Enabled = nil

	if err := manager.ApplyConfig(context.Background(), cfg); err != nil {
		t.Fatalf("ApplyConfig: %v", err)
	}

	if old.closes != 1 {
		t.Fatalf("old client closes = %d, want 1", old.closes)
	}

	if next.connects != 1 {
		t.Fatalf("replacement connects = %d, want 1", next.connects)
	}

	snapshots := manager.Snapshots()
	if len(snapshots) != 1 || !snapshots[0].Config.IsEnabled() {
		t.Fatalf("snapshots = %+v, want enabled github", snapshots)
	}
}

func TestManagerApplyConfigReturnsConnectErrorFromReplacement(t *testing.T) {
	disabled := false
	want := errors.New("connect boom")
	old := &fakeClient{config: Config{Name: "github", Enabled: &disabled}}
	next := &fakeClient{status: ClientDisconnected, err: want}
	manager := newManager(old)
	manager.newClient = func(cfg Config) Client {
		next.config = cfg

		return next
	}
	cfg := old.Config()
	cfg.Enabled = nil

	err := manager.ApplyConfig(context.Background(), cfg)
	if !errors.Is(err, want) {
		t.Fatalf("ApplyConfig error = %v, want %v", err, want)
	}

	snapshots := manager.Snapshots()
	if len(snapshots) != 1 || snapshots[0].State.Status != ClientDisconnected || snapshots[0].State.Error != want.Error() {
		t.Fatalf("snapshots = %+v, want replacement state with connect error", snapshots)
	}
}

func TestManagerApplyConfigUnknownClient(t *testing.T) {
	err := newManager(&fakeClient{config: Config{Name: "github"}}).ApplyConfig(context.Background(), Config{Name: "missing"})
	if err == nil || err.Error() != `mcp: unknown client "missing"` {
		t.Fatalf("ApplyConfig error = %v, want unknown client", err)
	}
}

func TestManagerReconnectClosesThenConnects(t *testing.T) {
	client := &fakeClient{config: Config{Name: "github"}}

	if err := newManager(client).Reconnect(context.Background(), "github"); err != nil {
		t.Fatalf("Reconnect: %v", err)
	}

	if client.closes != 1 || client.connects != 1 {
		t.Fatalf("closes/connects = %d/%d, want 1/1", client.closes, client.connects)
	}
}

func TestManagerReconnectUnknownClient(t *testing.T) {
	err := newManager(&fakeClient{config: Config{Name: "github"}}).Reconnect(context.Background(), "missing")
	if err == nil || err.Error() != `mcp: unknown client "missing"` {
		t.Fatalf("Reconnect error = %v, want unknown client", err)
	}
}

func TestManagerAuthenticateReplacesAndConnectsClient(t *testing.T) {
	client := &fakeClient{config: Config{Name: "github"}}
	next := &fakeClient{status: ClientDisconnected}
	authenticator := &fakeAuthenticator{}
	manager := newManager(client)
	manager.authenticator = authenticator
	manager.newClient = func(cfg Config) Client {
		next.config = cfg

		return next
	}

	if err := manager.Authenticate(context.Background(), "github"); err != nil {
		t.Fatalf("Authenticate: %v", err)
	}

	if authenticator.calls != 1 || authenticator.config.Name != "github" {
		t.Fatalf("authenticator = %d/%+v, want github call", authenticator.calls, authenticator.config)
	}

	if client.closes != 1 {
		t.Fatalf("old client closes = %d, want 1", client.closes)
	}

	if next.connects != 1 {
		t.Fatalf("replacement connects = %d, want 1", next.connects)
	}
}

func TestManagerAuthenticateReturnsAuthenticatorError(t *testing.T) {
	want := errors.New("auth boom")
	client := &fakeClient{config: Config{Name: "github"}}
	manager := newManager(client)
	manager.authenticator = &fakeAuthenticator{err: want}

	err := manager.Authenticate(context.Background(), "github")
	if !errors.Is(err, want) {
		t.Fatalf("Authenticate error = %v, want %v", err, want)
	}

	if client.closes != 0 || client.connects != 0 {
		t.Fatalf("closes/connects = %d/%d, want 0/0", client.closes, client.connects)
	}
}
