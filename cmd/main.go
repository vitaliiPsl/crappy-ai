package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/vitaliiPsl/crappy-ai/internal/assistant"
	"github.com/vitaliiPsl/crappy-ai/internal/cli"
	"github.com/vitaliiPsl/crappy-ai/internal/config"
	"github.com/vitaliiPsl/crappy-ai/internal/models"
	"github.com/vitaliiPsl/crappy-ai/internal/permission"
	"github.com/vitaliiPsl/crappy-ai/internal/server"
	"github.com/vitaliiPsl/crappy-ai/internal/settings"
	"github.com/vitaliiPsl/crappy-ai/internal/tools"
	"github.com/vitaliiPsl/crappy-ai/internal/tui"

	permissionstore "github.com/vitaliiPsl/crappy-ai/internal/permission/store"
	sessionstore "github.com/vitaliiPsl/crappy-ai/internal/session/store"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		provider = flag.String("provider", "", "active provider name")
		model    = flag.String("model", "", "active model id")
		thinking = flag.String("thinking", "", "thinking level (disabled|low|medium|high)")
		cwd      = flag.String("cwd", "", "working directory for new sessions (default: current directory)")
		prompt   = flag.String("prompt", "", "if set, run a single turn with this prompt and exit")
	)

	flag.Parse()

	resolvedCwd, err := resolveCwd(*cwd)
	if err != nil {
		return fmt.Errorf("resolve cwd: %w", err)
	}

	settingsStore, err := settings.Load()
	if err != nil {
		return fmt.Errorf("load settings: %w", err)
	}

	configStore, err := config.Load(settingsStore.Get().ConfigPath, config.Flags{
		Provider: *provider,
		Model:    *model,
		Thinking: *thinking,
		Cwd:      resolvedCwd,
	})
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	sessStore, err := sessionstore.NewFileStore(settingsStore.Get().SessionsDir)
	if err != nil {
		return fmt.Errorf("init session store: %w", err)
	}

	modelRegistry := models.NewRegistry(settingsStore)
	toolRegistry := tools.NewRegistry()

	permissionService := permission.NewService(permissionstore.NewGlobal(configStore), nil)

	go func() {
		if err := settingsStore.RefreshModels(context.Background()); err != nil {
			fmt.Fprintf(os.Stderr, "warning: refresh models from remote: %v\n", err)
		}
	}()

	asst := assistant.New(configStore, sessStore, modelRegistry, toolRegistry, permissionService)
	srv := server.New(asst, settingsStore, configStore, sessStore, modelRegistry)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if *prompt != "" {
		srv.AddTransport(cli.NewTransport(srv, *prompt))
	} else {
		srv.AddTransport(tui.NewTransport(ctx, srv))
	}

	return srv.Run(ctx)
}

func resolveCwd(flagValue string) (string, error) {
	if flagValue == "" {
		return os.Getwd()
	}

	return filepath.Abs(flagValue)
}
