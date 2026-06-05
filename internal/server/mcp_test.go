package server

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/vitaliiPsl/crappy-ai/internal/mcp"
	"github.com/vitaliiPsl/crappy-ai/internal/settings"
)

func TestSetMCPClientEnabledPersistsAndUpdatesManager(t *testing.T) {
	settingsPath := filepath.Join(t.TempDir(), "settings.yaml")
	settingsStore := settings.NewStore(settings.Settings{
		MCPClients: []mcp.Config{{Name: "github"}},
	}, settingsPath)
	manager := mcp.New(settingsStore.Get().MCPClients)
	srv := New(nil, settingsStore, nil, nil, nil, nil, manager)

	if err := srv.SetMCPClientEnabled(context.Background(), "github", false); err != nil {
		t.Fatalf("SetMCPClientEnabled: %v", err)
	}

	st := settingsStore.Get()
	if len(st.MCPClients) != 1 || st.MCPClients[0].IsEnabled() {
		t.Fatalf("settings MCPClients = %+v, want disabled github", st.MCPClients)
	}

	configs := manager.Configs()
	if len(configs) != 1 || configs[0].IsEnabled() {
		t.Fatalf("manager configs = %+v, want disabled github", configs)
	}
}

func TestSetMCPClientEnabledUnknownClient(t *testing.T) {
	settingsStore := settings.NewStore(settings.Settings{
		MCPClients: []mcp.Config{{Name: "github"}},
	}, filepath.Join(t.TempDir(), "settings.yaml"))
	srv := New(nil, settingsStore, nil, nil, nil, nil, mcp.New(settingsStore.Get().MCPClients))

	err := srv.SetMCPClientEnabled(context.Background(), "missing", false)
	if err == nil || err.Error() != `mcp: unknown client "missing"` {
		t.Fatalf("SetMCPClientEnabled error = %v, want unknown client", err)
	}
}
