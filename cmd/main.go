package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/vitaliiPsl/crappy-ai/internal/background"
	"github.com/vitaliiPsl/crappy-ai/internal/cli"
	"github.com/vitaliiPsl/crappy-ai/internal/config"
	"github.com/vitaliiPsl/crappy-ai/internal/mcp"
	oauthstore "github.com/vitaliiPsl/crappy-ai/internal/mcp/oauth/store"
	"github.com/vitaliiPsl/crappy-ai/internal/models"
	appoauth "github.com/vitaliiPsl/crappy-ai/internal/oauth"
	appproviders "github.com/vitaliiPsl/crappy-ai/internal/providers"
	provideroauthstore "github.com/vitaliiPsl/crappy-ai/internal/providers/oauth/store"
	"github.com/vitaliiPsl/crappy-ai/internal/providers/openai"
	"github.com/vitaliiPsl/crappy-ai/internal/runtime"
	"github.com/vitaliiPsl/crappy-ai/internal/server"
	sessionstore "github.com/vitaliiPsl/crappy-ai/internal/session/store"
	"github.com/vitaliiPsl/crappy-ai/internal/settings"
	"github.com/vitaliiPsl/crappy-ai/internal/skills"
	"github.com/vitaliiPsl/crappy-ai/internal/tui"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		provider = flag.String("provider", "", "active provider id")
		model    = flag.String("model", "", "active model id")
		thinking = flag.String("thinking", "", "thinking level (disabled|low|medium|high)")
		mode     = flag.String("mode", "", "permission mode (default|yolo)")
		prompt   = flag.String("prompt", "", "if set, run a single cli turn with this prompt and exit")
	)

	flag.Parse()

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("load cwd: %w", err)
	}

	flags := config.Flags{
		Cwd:      cwd,
		Provider: *provider,
		Model:    *model,
		Thinking: *thinking,
		Mode:     *mode,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	settingsStore, err := settings.Load()
	if err != nil {
		return fmt.Errorf("load settings: %w", err)
	}

	configStore, err := config.Load(settingsStore.Get().ConfigPath, flags)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	sessStore, err := sessionstore.NewFileStore(settingsStore.Get().SessionsDir)
	if err != nil {
		return fmt.Errorf("init session store: %w", err)
	}

	go func() {
		if err := settingsStore.RefreshModels(context.Background()); err != nil {
			fmt.Fprintf(os.Stderr, "warning: refresh models from remote: %v\n", err)
		}
	}()

	skillRegistry := skills.NewRegistry(settingsStore)
	oauthCallback := appoauth.NewBrowserCallback()

	mcpOauthStore, err := oauthstore.NewFileStore(settingsStore.Get().MCPOAuthPath)
	if err != nil {
		return fmt.Errorf("init oauth store: %w", err)
	}

	mcpManager := mcp.NewManager(
		ctx,
		settingsStore.Get().MCPClients,
		mcpOauthStore,
		oauthCallback,
	)
	defer mcpManager.Close()

	providerOauthStore, err := provideroauthstore.NewFileStore(settingsStore.Get().OAuthPath)
	if err != nil {
		return fmt.Errorf("init provider oauth store: %w", err)
	}

	providerManager := appproviders.NewManager(
		providerOauthStore,
		oauthCallback,
		openai.New(),
	)
	modelRegistry := models.NewRegistry(settingsStore, providerManager)

	backgroundManager := background.NewManager(ctx)
	defer backgroundManager.Close()

	runtimeManager := runtime.NewManager(
		configStore,
		sessStore,
		modelRegistry,
		skillRegistry,
		mcpManager,
		backgroundManager,
	)
	defer runtimeManager.Close()

	srv := server.New(
		runtimeManager,
		settingsStore,
		configStore,
		modelRegistry,
		skillRegistry,
		mcpManager,
		backgroundManager,
	)

	if *prompt != "" {
		srv.AddTransport(cli.NewTransport(srv, *prompt))
	} else {
		srv.AddTransport(tui.NewTransport(ctx, srv))
	}

	return srv.Start(ctx)
}
