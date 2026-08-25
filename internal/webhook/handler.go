package webhook

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
)

type Agent interface {
	HandleMessageReceived(ctx context.Context, event MessageReceived)
	HandleChatAssigned(ctx context.Context, event ChatAssigned)
}

type Handler struct {
	AuthToken string
	Agent     Agent
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !h.authorize(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	eventType := peekEventType(body)
	log.Printf("byoa-test-agent: webhook eventType=%q", eventType)

	switch eventType {
	case "message", "":
		var event MessageReceived
		if err := json.Unmarshal(body, &event); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if event.EventType == "" {
			event.EventType = "message"
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
		if h.Agent != nil {
			go h.Agent.HandleMessageReceived(context.Background(), event)
		}
	case "chatAssigned":
		var event ChatAssigned
		if err := json.Unmarshal(body, &event); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if event.EventType == "" {
			event.EventType = "chatAssigned"
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
		if h.Agent != nil {
			go h.Agent.HandleChatAssigned(context.Background(), event)
		}
	default:
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
		log.Printf("byoa-test-agent: ignored webhook eventType=%q", eventType)
	}
}

func (h Handler) authorize(r *http.Request) bool {
	expected := strings.TrimSpace(h.AuthToken)
	if expected == "" {
		return true
	}
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if auth == "" {
		return false
	}
	if strings.EqualFold(auth, "Bearer "+expected) {
		return true
	}
	return auth == expected
}

func peekEventType(body []byte) string {
	var probe struct {
		EventType string `json:"eventType"`
	}
	if err := json.Unmarshal(body, &probe); err != nil {
		return ""
	}
	return probe.EventType
}
