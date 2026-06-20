package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/vitaliiPsl/crappy-ai/internal/assistant"
	"github.com/vitaliiPsl/crappy-ai/internal/assistant/factory"
	"github.com/vitaliiPsl/crappy-ai/internal/background"
	"github.com/vitaliiPsl/crappy-ai/internal/cli"
	"github.com/vitaliiPsl/crappy-ai/internal/config"
	"github.com/vitaliiPsl/crappy-ai/internal/mcp"
	oauthstore "github.com/vitaliiPsl/crappy-ai/internal/mcp/oauth/store"
	"github.com/vitaliiPsl/crappy-ai/internal/models"
	"github.com/vitaliiPsl/crappy-ai/internal/permission"
	"github.com/vitaliiPsl/crappy-ai/internal/server"
	sessionstore "github.com/vitaliiPsl/crappy-ai/internal/session/store"
	"github.com/vitaliiPsl/crappy-ai/internal/settings"
	"github.com/vitaliiPsl/crappy-ai/internal/skills"
	"github.com/vitaliiPsl/crappy-ai/internal/tui"

	bgext "github.com/vitaliiPsl/crappy-ai/internal/extensions/background"
	mcpext "github.com/vitaliiPsl/crappy-ai/internal/extensions/mcp"
	"github.com/vitaliiPsl/crappy-ai/internal/extensions/planning"
	skillsext "github.com/vitaliiPsl/crappy-ai/internal/extensions/skills"
	"github.com/vitaliiPsl/crappy-ai/internal/extensions/subagents"
	"github.com/vitaliiPsl/crappy-ai/internal/extensions/summarization"
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

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("load cwd: %w", err)
	}

	settingsStore, err := settings.Load()
	if err != nil {
		return fmt.Errorf("load settings: %w", err)
	}

	go func() {
		if err := settingsStore.RefreshModels(context.Background()); err != nil {
			fmt.Fprintf(os.Stderr, "warning: refresh models from remote: %v\n", err)
		}
	}()

	configStore, err := config.Load(settingsStore.Get().ConfigPath, config.Flags{
		Cwd:      cwd,
		Provider: *provider,
		Model:    *model,
		Thinking: *thinking,
		Mode:     *mode,
	})
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	sessStore, err := sessionstore.NewFileStore(settingsStore.Get().SessionsDir)
	if err != nil {
		return fmt.Errorf("init session store: %w", err)
	}

	oauthStore, err := oauthstore.NewFileStore(settingsStore.Get().OAuthPath)
	if err != nil {
		return fmt.Errorf("init oauth store: %w", err)
	}

	modelRegistry := models.NewRegistry(settingsStore)
	skillRegistry := skills.NewRegistry(settingsStore)

	backgroundManager := background.NewManager(ctx)
	defer backgroundManager.Close()

	permissionService := permission.NewService(configStore, nil)

	mcpManager := mcp.NewManager(
		ctx,
		settingsStore.Get().MCPClients,
		oauthStore,
		mcp.NewBrowserCallback(),
	)
	defer mcpManager.Close()

	agentFactory := factory.New(permissionService, backgroundManager)

	baseExtensions := []factory.Extension{
		summarization.New(),
		bgext.New(backgroundManager),
		skillsext.New(skillRegistry),
		mcpext.New(mcpManager),
	}

	rootExtensions := append(
		append([]factory.Extension{}, baseExtensions...),
		planning.New(sessStore),
		subagents.New(agentFactory, baseExtensions, modelRegistry, backgroundManager),
	)

	asst := assistant.New(
		configStore,
		sessStore,
		modelRegistry,
		skillRegistry,
		agentFactory,
		rootExtensions,
	)

	srv := server.New(
		asst,
		settingsStore,
		configStore,
		sessStore,
		modelRegistry,
		skillRegistry,
		backgroundManager,
		mcpManager,
	)

	permissionService.SetHandler(srv)

	if *prompt != "" {
		srv.AddTransport(cli.NewTransport(srv, *prompt))
	} else {
		srv.AddTransport(tui.NewTransport(ctx, srv))
	}

	return srv.Run(ctx)
}
