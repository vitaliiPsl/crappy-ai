package mcp

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/vitaliiPsl/crappy-adk/kit"
)

func TestStateStartsDisconnected(t *testing.T) {
	cfg := Config{Name: "docs", Transport: TransportHTTP, URL: "http://localhost:3000/mcp"}

	state := NewClient(cfg).State()
	if !reflect.DeepEqual(state.Config, cfg) {
		t.Fatalf("Config = %+v, want %+v", state.Config, cfg)
	}

	if state.Status != ClientDisconnected {
		t.Fatalf("Status = %q, want disconnected", state.Status)
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
				Auth:      AuthConfig{HeaderEnv: map[string]string{"Authorization": "MISSING_MCP_AUTHORIZATION"}},
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

func TestStatusLifecycle(t *testing.T) {
	client := NewClient(Config{Name: "test", Transport: TransportHTTP, URL: serve(t, newServer(t, "greet"))})

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

func TestOperationsRequireConnection(t *testing.T) {
	client := NewClient(Config{Name: "test"})
	wantErr := `mcp: client "test" is not connected`

	if _, err := client.ListTools(context.Background()); err == nil || err.Error() != wantErr {
		t.Fatalf("ListTools() error = %v, want %q", err, wantErr)
	}

	if _, err := client.CallTool(context.Background(), kit.NewToolCall("1", "greet", nil)); err == nil || err.Error() != wantErr {
		t.Fatalf("CallTool() error = %v, want %q", err, wantErr)
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

func TestConnectIsIdempotent(t *testing.T) {
	client := NewClient(Config{Name: "test", Transport: TransportHTTP, URL: serve(t, newServer(t, "greet"))}).(*sdkClient)
	t.Cleanup(func() { _ = client.Close() })

	if err := client.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	session := client.session

	if err := client.Connect(context.Background()); err != nil {
		t.Fatalf("second Connect() error = %v", err)
	}

	if client.session != session {
		t.Fatal("second Connect() replaced the live session")
	}
}

func TestConnectIsConcurrencySafe(t *testing.T) {
	client := NewClient(Config{Name: "test", Transport: TransportHTTP, URL: serve(t, newServer(t, "greet"))})
	t.Cleanup(func() { _ = client.Close() })

	var wg sync.WaitGroup

	errs := make(chan error, 8)
	for range 8 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			errs <- client.Connect(context.Background())
		}()
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

func TestFailSession(t *testing.T) {
	t.Run("tears down a live session", func(t *testing.T) {
		client := connect(t, newServer(t, "greet")).(*sdkClient)

		client.failSession(errors.New("boom"))

		state := client.State()
		if state.Status != ClientFailed {
			t.Fatalf("Status = %q, want failed", state.Status)
		}

		if state.Error != "boom" {
			t.Fatalf("Error = %q, want boom", state.Error)
		}

		if client.session != nil {
			t.Fatal("session was not cleared")
		}
	})

	t.Run("is a no-op without a live session", func(t *testing.T) {
		client := NewClient(Config{Name: "test"}).(*sdkClient)

		client.failSession(errors.New("boom"))

		if got := client.State().Status; got != ClientDisconnected {
			t.Fatalf("Status = %q, want disconnected", got)
		}
	})
}

func TestAuthHeaders(t *testing.T) {
	tests := []struct {
		name string
		auth AuthConfig
		env  map[string]string
		want map[string]string
	}{
		{
			name: "static headers",
			auth: AuthConfig{Headers: map[string]string{"Authorization": "Bearer static", "X-MCP-Tenant": "acme"}},
			want: map[string]string{"Authorization": "Bearer static", "X-MCP-Tenant": "acme"},
		},
		{
			name: "env header overrides static",
			auth: AuthConfig{
				Headers:   map[string]string{"Authorization": "Bearer static"},
				HeaderEnv: map[string]string{"Authorization": "TEST_MCP_AUTHORIZATION"},
			},
			env:  map[string]string{"TEST_MCP_AUTHORIZATION": "Bearer env"},
			want: map[string]string{"Authorization": "Bearer env"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for name, value := range tt.env {
				t.Setenv(name, value)
			}

			// The server rejects any request missing the expected headers, so a
			// successful Connect proves authTransport applied them.
			url := serveRequiring(t, newServer(t, "greet"), tt.want)
			client := NewClient(Config{Name: "test", Transport: TransportHTTP, URL: url, Auth: tt.auth})
			t.Cleanup(func() { _ = client.Close() })

			if err := client.Connect(context.Background()); err != nil {
				t.Fatalf("Connect() error = %v", err)
			}
		})
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

	return serveRequiring(t, server, nil)
}

func serveRequiring(t *testing.T, server *mcpsdk.Server, headers map[string]string) string {
	t.Helper()

	handler := mcpsdk.NewStreamableHTTPHandler(func(*http.Request) *mcpsdk.Server {
		return server
	}, nil)

	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for name, value := range headers {
			if r.Header.Get(name) != value {
				http.Error(w, "unauthorized", http.StatusUnauthorized)

				return
			}
		}

		handler.ServeHTTP(w, r)
	}))
	t.Cleanup(httpServer.Close)

	return httpServer.URL
}

func connect(t *testing.T, server *mcpsdk.Server) Client {
	t.Helper()

	client := NewClient(Config{Name: "test", Transport: TransportHTTP, URL: serve(t, server)})
	if err := client.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	t.Cleanup(func() { _ = client.Close() })

	return client
}

func sameTools(a, b []kit.Tool) bool {
	return len(a) == len(b) && (len(a) == 0 || &a[0] == &b[0])
}
