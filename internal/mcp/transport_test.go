package mcp

import (
	"context"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/vitaliiPsl/crappy-ai/internal/mcp/oauth"
)

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
			err := NewClient(tt.config, testTransport).Connect(context.Background())
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
	}, nil, noopCallback{})
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
	}, nil, nil)
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

type noopCallback struct{}

func (noopCallback) Wait(context.Context, string, string) (string, string, error) {
	return "", "", nil
}
