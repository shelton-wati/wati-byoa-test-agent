package server

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/wati/wati-byoa-test-agent/internal/agent"
	"github.com/wati/wati-byoa-test-agent/internal/config"
	"github.com/wati/wati-byoa-test-agent/internal/webhook"
)

type Server struct {
	cfg    config.Config
	agent  *agent.Service
	server *http.Server
}

func New(cfg config.Config) (*Server, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	agentService, err := agent.NewService(agent.Config{
		WatiBaseURL:      cfg.WatiBaseURL,
		WatiTenantID:     cfg.WatiTenantID,
		WatiAPIToken:     cfg.WatiAPIToken,
		SourceType:       cfg.SourceType,
		LLMBaseURL:       cfg.LLMBaseURL,
		LLMAPIKey:        cfg.LLMAPIKey,
		LLMModel:         cfg.LLMModel,
		LLMTimeout:       cfg.LLMTimeout,
		MaxSteps:         cfg.MaxSteps,
		MaxOutputTokens:  cfg.MaxOutputTokens,
		MaxContextTokens: cfg.MaxContextTokens,
	})
	if err != nil {
		return nil, err
	}

	handler := webhook.Handler{AuthToken: cfg.WebhookToken, Agent: agentService}
	mux := http.NewServeMux()
	mux.HandleFunc("/livez", okHandler)
	mux.HandleFunc("/healthz", okHandler)
	mux.Handle("/wati/webhook", handler)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		okHandler(w, r)
	})

	return &Server{
		cfg:   cfg,
		agent: agentService,
		server: &http.Server{
			Addr:              cfg.HTTPAddr,
			Handler:           mux,
			ReadHeaderTimeout: 10 * time.Second,
			ReadTimeout:       30 * time.Second,
			IdleTimeout:       120 * time.Second,
		},
	}, nil
}

func (s *Server) Run(ctx context.Context) error {
	if err := os.MkdirAll(config.StateDir, 0o700); err != nil {
		return err
	}
	errCh := make(chan error, 1)
	go func() {
		log.Printf("wati-byoa-test-agent listening on %s (webhook: POST /wati/webhook, state: %s)",
			s.cfg.HTTPAddr, filepath.Clean(config.StateDir))
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return s.server.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

func okHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
