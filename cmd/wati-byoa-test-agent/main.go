package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/wati/wati-byoa-test-agent/internal/config"
	"github.com/wati/wati-byoa-test-agent/internal/server"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr, "Set WEBHOOK_TOKEN, WATI_BASE_URL, WATI_API_TOKEN, LLM_API_KEY, and LLM_MODEL.")
		os.Exit(1)
	}

	srv, err := server.New(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := srv.Run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
