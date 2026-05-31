package mcp

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/vitaliiPsl/crappy-adk/kit"
)

func TestClientReturnsConfig(t *testing.T) {
	cfg := Config{
		Name:      "docs",
		Transport: TransportHTTP,
		URL:       "http://localhost:3000/mcp",
	}

	if got := NewClient(cfg).Config(); !reflect.DeepEqual(got, cfg) {
		t.Fatalf("Config() = %+v, want %+v", got, cfg)
	}
}

func TestClientHTTPToolLifecycle(t *testing.T) {
	serverURL := newTestMCPServer(t, nil)
	client := NewClient(Config{Name: "test", Transport: TransportHTTP, URL: serverURL})

	if err := client.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer func() { _ = client.Close() }()

	tools, err := client.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}

	if len(tools) != 1 {
		t.Fatalf("len(tools) = %d, want 1", len(tools))
	}

	if tools[0].Name != "greet" {
		t.Fatalf("tool name = %q, want greet", tools[0].Name)
	}

	result, err := client.CallTool(context.Background(), kit.NewToolCall("call_1", "greet", map[string]any{
		"name": "Ada",
	}))
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}

	if !strings.Contains(result.Output, "Hi Ada") {
		t.Fatalf("tool output = %q, want greeting", result.Output)
	}
}

func TestClientHTTPAuth(t *testing.T) {
	tests := []struct {
		name       string
		auth       AuthConfig
		env        map[string]string
		wantHeader map[string]string
	}{
		{
			name: "static headers",
			auth: AuthConfig{
				Headers: map[string]string{
					"Authorization": "Bearer static",
					"X-MCP-Tenant":  "acme",
				},
			},
			wantHeader: map[string]string{
				"Authorization": "Bearer static",
				"X-MCP-Tenant":  "acme",
			},
		},
		{
			name: "env headers",
			auth: AuthConfig{
				HeaderEnv: map[string]string{
					"Authorization": "TEST_MCP_AUTHORIZATION",
				},
			},
			env: map[string]string{
				"TEST_MCP_AUTHORIZATION": "Bearer env",
			},
			wantHeader: map[string]string{
				"Authorization": "Bearer env",
			},
		},
		{
			name: "env overrides static header",
			auth: AuthConfig{
				Headers: map[string]string{
					"Authorization": "Bearer static",
				},
				HeaderEnv: map[string]string{
					"Authorization": "TEST_MCP_AUTHORIZATION",
				},
			},
			env: map[string]string{
				"TEST_MCP_AUTHORIZATION": "Bearer env",
			},
			wantHeader: map[string]string{
				"Authorization": "Bearer env",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for name, value := range tt.env {
				t.Setenv(name, value)
			}

			serverURL := newTestMCPServer(t, tt.wantHeader)
			client := NewClient(Config{
				Name:      "test",
				Transport: TransportHTTP,
				URL:       serverURL,
				Auth:      tt.auth,
			})

			if err := client.Connect(context.Background()); err != nil {
				t.Fatalf("Connect() error = %v", err)
			}
			defer func() { _ = client.Close() }()

			if _, err := client.ListTools(context.Background()); err != nil {
				t.Fatalf("ListTools() error = %v", err)
			}
		})
	}
}

func TestClientConnectValidation(t *testing.T) {
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
			name: "auth header env is required",
			config: Config{
				Name:      "test",
				Transport: TransportHTTP,
				URL:       "http://example.com",
				Auth: AuthConfig{
					HeaderEnv: map[string]string{
						"Authorization": "MISSING_MCP_AUTHORIZATION",
					},
				},
			},
			wantErr: `mcp: client "test" auth header "Authorization" references empty env "MISSING_MCP_AUTHORIZATION"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewClient(tt.config).Connect(context.Background())
			if err == nil {
				t.Fatal("Connect() error = nil, want error")
			}

			if err.Error() != tt.wantErr {
				t.Fatalf("Connect() error = %q, want %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestClientListToolsConnectsOnDemand(t *testing.T) {
	_, err := NewClient(Config{Name: "test"}).ListTools(context.Background())
	if err == nil {
		t.Fatal("ListTools() error = nil, want error")
	}

	if err.Error() != `mcp: client "test" has no command for stdio transport` {
		t.Fatalf("ListTools() error = %q, want connect failure", err.Error())
	}
}

func newTestMCPServer(t *testing.T, requiredHeaders map[string]string) string {
	t.Helper()

	server := mcpsdk.NewServer(&mcpsdk.Implementation{Name: "test", Version: "0.1.0"}, nil)
	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "greet", Description: "Greet someone"}, greet)

	handler := mcpsdk.NewStreamableHTTPHandler(func(*http.Request) *mcpsdk.Server {
		return server
	}, nil)

	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for name, value := range requiredHeaders {
			if got := r.Header.Get(name); got != value {
				http.Error(w, "unauthorized", http.StatusUnauthorized)

				return
			}
		}

		handler.ServeHTTP(w, r)
	}))
	t.Cleanup(httpServer.Close)

	return httpServer.URL
}

type greetInput struct {
	Name string `json:"name" jsonschema:"Name to greet"`
}

type greetOutput struct {
	Greeting string `json:"greeting" jsonschema:"Greeting text"`
}

func greet(_ context.Context, _ *mcpsdk.CallToolRequest, input greetInput) (*mcpsdk.CallToolResult, greetOutput, error) {
	return nil, greetOutput{Greeting: "Hi " + input.Name}, nil
}

func TestClientStatusLifecycle(t *testing.T) {
	serverURL := newTestMCPServer(t, nil)
	client := NewClient(Config{Name: "test", Transport: TransportHTTP, URL: serverURL})

	if got := client.Status().State; got != ClientDisconnected {
		t.Fatalf("initial State = %q, want disconnected", got)
	}

	if err := client.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	if got := client.Status().State; got != ClientConnected {
		t.Fatalf("connected State = %q, want connected", got)
	}

	if err := client.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if got := client.Status().State; got != ClientDisconnected {
		t.Fatalf("closed State = %q, want disconnected", got)
	}
}

func TestClientListToolsConnectsOnDemandToLiveServer(t *testing.T) {
	serverURL := newTestMCPServer(t, nil)

	client := NewClient(Config{Name: "test", Transport: TransportHTTP, URL: serverURL})
	defer func() { _ = client.Close() }()

	// No explicit Connect: ListTools must auto-connect.
	tools, err := client.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}

	if len(tools) != 1 {
		t.Fatalf("len(tools) = %d, want 1", len(tools))
	}

	if got := client.Status().State; got != ClientConnected {
		t.Fatalf("State = %q, want connected", got)
	}
}

func TestClientConnectIsIdempotent(t *testing.T) {
	serverURL := newTestMCPServer(t, nil)

	client := NewClient(Config{Name: "test", Transport: TransportHTTP, URL: serverURL}).(*sdkClient)
	defer func() { _ = client.Close() }()

	if err := client.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	first := client.session

	if err := client.Connect(context.Background()); err != nil {
		t.Fatalf("second Connect() error = %v", err)
	}

	if client.session != first {
		t.Fatal("second Connect() replaced the live session")
	}
}

func TestClientSetFailed(t *testing.T) {
	t.Run("no-op without a session", func(t *testing.T) {
		client := NewClient(Config{Name: "test"}).(*sdkClient)

		client.setFailed(errors.New("boom"))

		status := client.Status()
		if status.State != ClientDisconnected {
			t.Fatalf("State = %q, want disconnected", status.State)
		}

		if status.Error != "" {
			t.Fatalf("Error = %q, want empty", status.Error)
		}
	})

	t.Run("closes and clears a live session", func(t *testing.T) {
		serverURL := newTestMCPServer(t, nil)

		client := NewClient(Config{Name: "test", Transport: TransportHTTP, URL: serverURL}).(*sdkClient)
		defer func() { _ = client.Close() }()

		if err := client.Connect(context.Background()); err != nil {
			t.Fatalf("Connect() error = %v", err)
		}

		client.setFailed(errors.New("boom"))

		status := client.Status()
		if status.State != ClientFailed {
			t.Fatalf("State = %q, want failed", status.State)
		}

		if status.Error != "boom" {
			t.Fatalf("Error = %q, want boom", status.Error)
		}

		if client.session != nil {
			t.Fatal("session was not cleared")
		}
	})
}
