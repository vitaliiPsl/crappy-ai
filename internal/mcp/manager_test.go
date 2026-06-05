package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/vitaliiPsl/crappy-adk/kit"
)

type fakeClient struct {
	config Config
	status ClientStatus
	tools  []kit.Tool
	err    error

	connects int
	auths    int
	closes   int
	called   kit.ToolCall
	result   kit.ToolResult
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

func (c *fakeClient) Authenticate(context.Context) error {
	c.auths++

	return c.err
}

func (c *fakeClient) Close() error {
	c.closes++

	return c.err
}

func (c *fakeClient) ListTools(context.Context) ([]kit.Tool, error) {
	return c.tools, c.err
}

func (c *fakeClient) CallTool(_ context.Context, call kit.ToolCall) (kit.ToolResult, error) {
	c.called = call

	return c.result, c.err
}

func newManager(clients ...Client) *Manager {
	byName := make(map[string]Client, len(clients))
	for _, client := range clients {
		byName[client.Config().Name] = client
	}

	return &Manager{clients: byName}
}

func TestManagerConnectConnectsClients(t *testing.T) {
	first := &fakeClient{config: Config{Name: "first"}}
	second := &fakeClient{config: Config{Name: "second"}}

	if err := newManager(first, second).Connect(context.Background()); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	if first.connects != 1 || second.connects != 1 {
		t.Fatalf("connects = %d/%d, want 1/1", first.connects, second.connects)
	}
}

func TestManagerConnectReturnsErrors(t *testing.T) {
	want := errors.New("boom")

	err := newManager(&fakeClient{err: want}).Connect(context.Background())
	if !errors.Is(err, want) {
		t.Fatalf("Connect error = %v, want %v", err, want)
	}
}

func TestManagerConnectSkipsDisabledClients(t *testing.T) {
	disabled := false
	client := &fakeClient{config: Config{Name: "github", Enabled: &disabled}}

	if err := newManager(client).Connect(context.Background()); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	if client.connects != 0 {
		t.Fatalf("connects = %d, want 0", client.connects)
	}
}

func TestManagerStatesReturnsClientStates(t *testing.T) {
	client := &fakeClient{
		status: ClientFailed,
		err:    errors.New("boom"),
	}

	states := newManager(client).States()
	if len(states) != 1 {
		t.Fatalf("len(states) = %d, want 1", len(states))
	}

	if states[0].Status != ClientFailed || states[0].Error != "boom" {
		t.Fatalf("state = %+v, want failed boom", states[0])
	}
}

func TestManagerSetEnabledDisablesByReplacingClient(t *testing.T) {
	client := &fakeClient{config: Config{Name: "github"}}
	manager := newManager(client)

	if err := manager.SetEnabled(context.Background(), "github", false); err != nil {
		t.Fatalf("SetEnabled: %v", err)
	}

	if client.closes != 1 {
		t.Fatalf("old client closes = %d, want 1", client.closes)
	}

	configs := manager.Configs()
	if len(configs) != 1 || configs[0].IsEnabled() {
		t.Fatalf("configs = %+v, want disabled github", configs)
	}

	states := manager.States()
	if len(states) != 1 || states[0].Status != ClientDisconnected {
		t.Fatalf("states = %+v, want disconnected replacement", states)
	}
}

func TestManagerSetEnabledEnablesByReplacingAndConnectingClient(t *testing.T) {
	disabled := false
	want := errors.New("transport boom")
	old := &fakeClient{config: Config{Name: "github", Enabled: &disabled}}
	factory := &fakeTransportFactory{err: want}
	manager := newManager(old)
	manager.transport = factory.New

	err := manager.SetEnabled(context.Background(), "github", true)
	if !errors.Is(err, want) {
		t.Fatalf("SetEnabled error = %v, want %v", err, want)
	}

	if old.closes != 1 {
		t.Fatalf("old client closes = %d, want 1", old.closes)
	}

	if factory.calls != 1 {
		t.Fatalf("transport calls = %d, want 1", factory.calls)
	}

	configs := manager.Configs()
	if len(configs) != 1 || !configs[0].IsEnabled() {
		t.Fatalf("configs = %+v, want enabled github", configs)
	}

	states := manager.States()
	if len(states) != 1 || states[0].Status != ClientFailed || states[0].Error != want.Error() {
		t.Fatalf("states = %+v, want failed replacement with transport error", states)
	}
}

func TestManagerSetEnabledUnknownClient(t *testing.T) {
	err := newManager(&fakeClient{config: Config{Name: "github"}}).SetEnabled(context.Background(), "missing", false)
	if err == nil || err.Error() != `mcp: unknown client "missing"` {
		t.Fatalf("SetEnabled error = %v, want unknown client", err)
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

func TestManagerAuthenticateClient(t *testing.T) {
	client := &fakeClient{config: Config{Name: "github"}}

	if err := newManager(client).Authenticate(context.Background(), "github"); err != nil {
		t.Fatalf("Authenticate: %v", err)
	}

	if client.auths != 1 {
		t.Fatalf("auths = %d, want 1", client.auths)
	}
}
