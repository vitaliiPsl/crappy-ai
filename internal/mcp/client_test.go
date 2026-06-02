package mcp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/vitaliiPsl/crappy-ai/internal/mcp/oauth"

	"github.com/vitaliiPsl/crappy-adk/kit"
)

func TestNewClientStartsDisconnected(t *testing.T) {
	state := NewClient(Config{Name: "docs", Transport: TransportHTTP, URL: "http://localhost:3000/mcp"}).State()

	if state.Status != ClientDisconnected {
		t.Fatalf("Status = %q, want disconnected", state.Status)
	}

	if state.Error != "" {
		t.Fatalf("Error = %q, want empty", state.Error)
	}
}

func TestConnectRejectsDisabledClient(t *testing.T) {
	disabled := false

	err := NewClient(Config{Name: "docs", Enabled: &disabled}).Connect(context.Background())
	if err == nil || err.Error() != "mcp: client is disabled" {
		t.Fatalf("Connect() error = %v, want disabled", err)
	}
}

func TestConnectValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr string
	}{
		{
			name:    "stdio requires command",
			config:  Config{Name: "test"},
			wantErr: `mcp: client "test" has no command for stdio transport`,
		},
		{
			name:    "http requires url",
			config:  Config{Name: "test", Transport: TransportHTTP},
			wantErr: `mcp: client "test" has no url for http transport`,
		},
		{
			name:    "unsupported transport",
			config:  Config{Name: "test", Transport: "websocket"},
			wantErr: `mcp: client "test" has unsupported transport "websocket"`,
		},
		{
			name: "auth env must be set",
			config: Config{
				Name:      "test",
				Transport: TransportHTTP,
				URL:       "http://example.com",
				Headers:   map[string]string{"Authorization": "Bearer static"},
				HeaderEnv: map[string]string{"Authorization": "MISSING_MCP_AUTHORIZATION"},
			},
			wantErr: `mcp: client "test" auth header "Authorization" references empty env "MISSING_MCP_AUTHORIZATION"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewClient(tt.config).Connect(context.Background())
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("Connect() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestHTTPTransportConfiguresOAuthHandler(t *testing.T) {
	transport, err := newHTTPTransport(Config{
		Name:  "test",
		URL:   "http://example.com/mcp",
		OAuth: &oauth.Config{},
	}, transportOptions{OAuthInteractive: true})
	if err != nil {
		t.Fatalf("newHTTPTransport() error = %v", err)
	}

	streamable, ok := transport.(*mcpsdk.StreamableClientTransport)
	if !ok {
		t.Fatalf("transport = %T, want streamable", transport)
	}

	if streamable.OAuthHandler == nil {
		t.Fatal("OAuthHandler is nil, want configured handler")
	}
}

func TestHTTPTransportSkipsDisabledOAuth(t *testing.T) {
	enabled := false

	transport, err := newHTTPTransport(Config{
		Name: "test",
		URL:  "http://example.com/mcp",
		OAuth: &oauth.Config{
			Enabled: &enabled,
		},
	}, transportOptions{})
	if err != nil {
		t.Fatalf("newHTTPTransport() error = %v", err)
	}

	streamable, ok := transport.(*mcpsdk.StreamableClientTransport)
	if !ok {
		t.Fatalf("transport = %T, want streamable", transport)
	}

	if streamable.OAuthHandler != nil {
		t.Fatal("OAuthHandler is configured, want nil")
	}
}

func TestConnectUsesTransportFactory(t *testing.T) {
	want := errors.New("transport boom")
	factory := &fakeTransportFactory{err: want}
	client := newSDKClient(Config{Name: "test"})
	client.newTransport = factory.New

	err := client.Connect(context.Background())
	if !errors.Is(err, want) {
		t.Fatalf("Connect() error = %v, want %v", err, want)
	}

	if factory.calls != 1 {
		t.Fatalf("transport factory calls = %d, want 1", factory.calls)
	}

	if factory.config.Name != "test" {
		t.Fatalf("transport factory config = %+v, want test config", factory.config)
	}

	if factory.options.OAuthInteractive {
		t.Fatal("transport factory OAuthInteractive = true, want false")
	}

	state := client.State()
	if state.Status != ClientFailed || state.Error != want.Error() {
		t.Fatalf("state = %+v, want failed with transport error", state)
	}
}

func TestConnectMapsOAuthAuthorizationRequiredStatus(t *testing.T) {
	factory := &fakeTransportFactory{err: fmt.Errorf("wrapped: %w", oauth.ErrAuthorizationRequired)}
	client := newSDKClient(Config{Name: "test"})
	client.newTransport = factory.New

	_ = client.Connect(context.Background())

	state := client.State()
	if state.Status != ClientAuthRequired {
		t.Fatalf("Status = %q, want auth_required", state.Status)
	}

	if state.Error != oauth.ErrAuthorizationRequired.Error() {
		t.Fatalf("Error = %q, want auth required", state.Error)
	}
}

func TestAuthenticateUsesInteractiveOAuthTransport(t *testing.T) {
	factory := &fakeTransportFactory{err: errors.New("transport boom")}
	client := newSDKClient(Config{
		Name: "test",
		OAuth: &oauth.Config{
			Enabled: nil,
		},
	})
	client.newTransport = factory.New

	_ = client.Authenticate(context.Background())

	if !factory.options.OAuthInteractive {
		t.Fatal("transport factory OAuthInteractive = false, want true")
	}
}

func TestStatusLifecycle(t *testing.T) {
	client := newClient(t, newServer(t, "greet"))

	if got := client.State().Status; got != ClientDisconnected {
		t.Fatalf("initial Status = %q, want disconnected", got)
	}

	if err := client.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	if got := client.State().Status; got != ClientConnected {
		t.Fatalf("connected Status = %q, want connected", got)
	}

	if err := client.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if got := client.State().Status; got != ClientDisconnected {
		t.Fatalf("closed Status = %q, want disconnected", got)
	}
}

func TestConnectIsIdempotent(t *testing.T) {
	client := connect(t, newServer(t, "greet"))

	session := client.session

	if err := client.Connect(context.Background()); err != nil {
		t.Fatalf("second Connect() error = %v", err)
	}

	if client.session != session {
		t.Fatal("second Connect() replaced the live session")
	}
}

func TestConnectIsConcurrencySafe(t *testing.T) {
	client := newClient(t, newServer(t, "greet"))

	var wg sync.WaitGroup

	errs := make(chan error, 8)
	for range 8 {
		wg.Go(func() {
			errs <- client.Connect(context.Background())
		})
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("Connect() error = %v", err)
		}
	}

	if got := client.State().Status; got != ClientConnected {
		t.Fatalf("Status = %q, want connected", got)
	}
}

func TestOperationsRequireConnection(t *testing.T) {
	client := NewClient(Config{Name: "test"})
	wantErr := "mcp: client is not connected"

	if _, err := client.ListTools(context.Background()); err == nil || err.Error() != wantErr {
		t.Fatalf("ListTools() error = %v, want %q", err, wantErr)
	}

	if _, err := client.CallTool(context.Background(), kit.NewToolCall("1", "greet", nil)); err == nil || err.Error() != wantErr {
		t.Fatalf("CallTool() error = %v, want %q", err, wantErr)
	}
}

func TestToolLifecycle(t *testing.T) {
	client := connect(t, newServer(t, "greet"))

	tools, err := client.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}

	if len(tools) != 1 {
		t.Fatalf("len(tools) = %d, want 1", len(tools))
	}

	if got := tools[0].Definition().Name; got != "mcp__test__greet" {
		t.Fatalf("tool name = %q, want mcp__test__greet", got)
	}

	result, err := client.CallTool(context.Background(), kit.NewToolCall("call_1", "greet", map[string]any{"name": "Ada"}))
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}

	if !strings.Contains(result.Output, "Hi Ada") {
		t.Fatalf("output = %q, want greeting", result.Output)
	}
}

func TestListToolsCachesBetweenCalls(t *testing.T) {
	client := connect(t, newServer(t, "greet"))

	first, err := client.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}

	second, err := client.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}

	if !sameTools(first, second) {
		t.Fatal("ListTools() refetched instead of returning the cache")
	}
}

func TestToolListChangedRefreshesCache(t *testing.T) {
	server := newServer(t, "greet")
	client := connect(t, server)

	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "farewell"}, greet)

	eventually(t, time.Second, func() bool {
		tools, err := client.ListTools(context.Background())
		if err != nil || len(tools) != 2 {
			return false
		}

		names := map[string]bool{}
		for _, tool := range tools {
			names[tool.Definition().Name] = true
		}

		return names["mcp__test__greet"] && names["mcp__test__farewell"]
	})
}

func TestRefetchFailureMarksFailedAndKeepsCache(t *testing.T) {
	var failCalls atomic.Bool

	handler := mcpsdk.NewStreamableHTTPHandler(func(*http.Request) *mcpsdk.Server {
		return newServer(t, "greet")
	}, nil)

	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if failCalls.Load() && r.Method == http.MethodPost {
			http.Error(w, "boom", http.StatusInternalServerError)

			return
		}

		handler.ServeHTTP(w, r)
	}))
	t.Cleanup(httpServer.Close)

	client := NewClient(Config{Name: "test", Transport: TransportHTTP, URL: httpServer.URL}).(*sdkClient)
	t.Cleanup(func() { _ = client.Close() })

	if err := client.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	before, err := client.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}

	failCalls.Store(true)

	client.refetchTools(context.Background())

	state := client.State()
	if state.Status != ClientFailed {
		t.Fatalf("Status = %q, want failed", state.Status)
	}

	if state.Error == "" {
		t.Fatal("Error is empty, want refetch error recorded")
	}

	if !sameTools(before, client.tools) {
		t.Fatal("refetch failure replaced cached tools")
	}
}

func TestRefetchCancellationKeepsConnected(t *testing.T) {
	client := connect(t, newServer(t, "greet"))

	before, err := client.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client.refetchTools(ctx)

	state := client.State()
	if state.Status != ClientConnected {
		t.Fatalf("Status = %q, want connected", state.Status)
	}

	if state.Error != "" {
		t.Fatalf("Error = %q, want empty", state.Error)
	}

	if !sameTools(before, client.tools) {
		t.Fatal("cancelled refetch replaced cached tools")
	}
}

func TestCallToolHonorsTimeout(t *testing.T) {
	server := mcpsdk.NewServer(&mcpsdk.Implementation{Name: "test", Version: "0.1.0"}, nil)
	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "slow"}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, _ greetInput) (*mcpsdk.CallToolResult, greetOutput, error) {
		<-ctx.Done()

		return nil, greetOutput{}, ctx.Err()
	})

	client := NewClient(Config{
		Name:           "test",
		Transport:      TransportHTTP,
		URL:            serve(t, server),
		RequestTimeout: 20 * time.Millisecond,
	})
	if err := client.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	t.Cleanup(func() { _ = client.Close() })

	_, err := client.CallTool(context.Background(), kit.NewToolCall("1", "slow", map[string]any{"name": "Ada"}))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("CallTool() error = %v, want context.DeadlineExceeded", err)
	}

	if got := client.State().Status; got != ClientConnected {
		t.Fatalf("Status = %q, want connected", got)
	}
}

type greetInput struct {
	Name string `json:"name" jsonschema:"Name to greet"`
}

type greetOutput struct {
	Greeting string `json:"greeting" jsonschema:"Greeting text"`
}

func greet(_ context.Context, _ *mcpsdk.CallToolRequest, in greetInput) (*mcpsdk.CallToolResult, greetOutput, error) {
	return nil, greetOutput{Greeting: "Hi " + in.Name}, nil
}

func newServer(t *testing.T, tools ...string) *mcpsdk.Server {
	t.Helper()

	server := mcpsdk.NewServer(&mcpsdk.Implementation{Name: "test", Version: "0.1.0"}, nil)
	for _, name := range tools {
		mcpsdk.AddTool(server, &mcpsdk.Tool{Name: name}, greet)
	}

	return server
}

func serve(t *testing.T, server *mcpsdk.Server) string {
	t.Helper()

	handler := mcpsdk.NewStreamableHTTPHandler(func(*http.Request) *mcpsdk.Server {
		return server
	}, nil)

	httpServer := httptest.NewServer(handler)
	t.Cleanup(httpServer.Close)

	return httpServer.URL
}

func newClient(t *testing.T, server *mcpsdk.Server) *sdkClient {
	t.Helper()

	client := NewClient(Config{Name: "test", Transport: TransportHTTP, URL: serve(t, server)}).(*sdkClient)
	t.Cleanup(func() { _ = client.Close() })

	return client
}

func connect(t *testing.T, server *mcpsdk.Server) *sdkClient {
	t.Helper()

	client := newClient(t, server)
	if err := client.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	return client
}

func sameTools(a, b []kit.Tool) bool {
	return len(a) == len(b) && (len(a) == 0 || &a[0] == &b[0])
}

func eventually(t *testing.T, timeout time.Duration, ok func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if ok() {
			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	if !ok() {
		t.Fatal("condition was not met before timeout")
	}
}

type fakeTransportFactory struct {
	config  Config
	options transportOptions
	err     error
	calls   int
}

func (f *fakeTransportFactory) New(config Config, opts transportOptions) (mcpsdk.Transport, error) {
	f.config = config
	f.options = opts
	f.calls++

	return nil, f.err
}
