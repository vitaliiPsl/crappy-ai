package oauth

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"

	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
)

func TestAuthorizationCodeConfigUsesConfiguredPrompter(t *testing.T) {
	wantErr := errors.New("prompt failed")
	prompter := &recordingPrompter{err: wantErr}

	cfg, err := authorizationCodeConfig(HandlerConfig{
		Config:   &Config{},
		Prompter: prompter,
	}, localRedirectURL(t))
	if err != nil {
		t.Fatalf("authorizationCodeConfig() error = %v", err)
	}

	_, err = cfg.AuthorizationCodeFetcher(context.Background(), &mcpauth.AuthorizationArgs{
		URL: "https://auth.example.com/authorize",
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("AuthorizationCodeFetcher() error = %v, want %v", err, wantErr)
	}

	if prompter.url != "https://auth.example.com/authorize" {
		t.Fatalf("prompter URL = %q, want auth URL", prompter.url)
	}
}

type recordingPrompter struct {
	url string
	err error
}

func (p *recordingPrompter) Prompt(authURL string) error {
	p.url = authURL

	return p.err
}

func localRedirectURL(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen on local port: %v", err)
	}

	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("close local listener: %v", err)
	}

	if !strings.HasPrefix(addr, "127.0.0.1:") {
		t.Fatalf("listener address = %q, want 127.0.0.1", addr)
	}

	return "http://" + addr + "/oauth/callback"
}
