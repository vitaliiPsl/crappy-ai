package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/assistant"
	"github.com/vitaliiPsl/crappy-ai/internal/config"
	"github.com/vitaliiPsl/crappy-ai/internal/models"
	"github.com/vitaliiPsl/crappy-ai/internal/session"
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

	if *prompt != "" {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
		defer cancel()

		return oneShot(ctx, asst, sessStore, *prompt)
	}

	c := configStore.Get()

	sessions, err := sessStore.List(context.Background())
	if err != nil {
		return fmt.Errorf("list sessions: %w", err)
	}

	fmt.Printf("settings: config_path=%s sessions_dir=%s providers=%d\n", s.ConfigPath, s.SessionsDir, len(s.Providers))
	fmt.Printf("config:   provider=%s model=%s thinking=%s\n", c.Provider, c.Model, c.Thinking)
	fmt.Printf("sessions: count=%d\n", len(sessions))

	return nil
}

func oneShot(ctx context.Context, asst *assistant.Assistant, sessStore session.Store, prompt string) error {
	workDir, _ := os.Getwd()

	sess, err := sessStore.Create(ctx, "cli", workDir)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	stream, err := asst.Run(ctx, sess.ID, prompt)
	if err != nil {
		return fmt.Errorf("run: %w", err)
	}

	for ev := range stream.Iter() {
		switch ev.Type {
		case session.EventContentDelta:
			if ev.Content != nil && ev.Content.Type == kit.ContentTypeText && ev.Content.Text != nil {
				fmt.Print(ev.Content.Text.Text)
			}
		case session.EventTurnComplete:
			fmt.Println()

			if ev.Stats != nil {
				fmt.Fprintf(os.Stderr, "[usage: in=%d out=%d]\n", ev.Stats.Usage.InputTokens, ev.Stats.Usage.OutputTokens)
			}
		case session.EventTurnCancelled:
			fmt.Fprintln(os.Stderr, "\n[cancelled]")
		case session.EventError:
			fmt.Fprintf(os.Stderr, "\n[error] %s\n", ev.Error)
		}
	}

	if _, err := stream.Result(); err != nil {
		return err
	}

	return nil
}
