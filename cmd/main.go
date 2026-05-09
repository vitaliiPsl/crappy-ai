package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/vitaliiPsl/crappy-ai/internal/assistant"
	"github.com/vitaliiPsl/crappy-ai/internal/cli"
	"github.com/vitaliiPsl/crappy-ai/internal/config"
	"github.com/vitaliiPsl/crappy-ai/internal/models"
	"github.com/vitaliiPsl/crappy-ai/internal/server"
	sessionstore "github.com/vitaliiPsl/crappy-ai/internal/session/store"
	"github.com/vitaliiPsl/crappy-ai/internal/settings"
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
		prompt   = flag.String("prompt", "", "if set, run a single turn with this prompt and exit")
	)

	flag.Parse()

	settingsStore, err := settings.Load()
	if err != nil {
		return fmt.Errorf("load settings: %w", err)
	}

	s := settingsStore.Get()

	configStore, err := config.Load(s.ConfigPath, config.Flags{
		Provider: *provider,
		Model:    *model,
		Thinking: *thinking,
	})
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	sessStore, err := sessionstore.NewFileStore(s.SessionsDir)
	if err != nil {
		return fmt.Errorf("init session store: %w", err)
	}

	registry := models.NewRegistry(settingsStore)
	asst := assistant.New(configStore, sessStore, registry)
	srv := server.New(asst, settingsStore, configStore, sessStore, registry)

	if *prompt != "" {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
		defer cancel()

		srv.AddTransport(cli.NewTransport(srv, *prompt))

		return srv.Run(ctx)
	}

	c := srv.GetConfig()

	sessions, err := srv.ListSessions(context.Background())
	if err != nil {
		return fmt.Errorf("list sessions: %w", err)
	}

	fmt.Printf("settings: config_path=%s sessions_dir=%s providers=%d\n", s.ConfigPath, s.SessionsDir, len(s.Providers))
	fmt.Printf("config:   provider=%s model=%s thinking=%s\n", c.Provider, c.Model, c.Thinking)
	fmt.Printf("sessions: count=%d\n", len(sessions))

	return nil
}
