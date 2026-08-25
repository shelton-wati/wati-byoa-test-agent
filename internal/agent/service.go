package agent

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/wati/wati-byoa-test-agent/internal/config"
	contextstore "github.com/wati/wati-byoa-test-agent/internal/context"
	"github.com/wati/wati-byoa-test-agent/internal/dedup"
	"github.com/wati/wati-byoa-test-agent/internal/llm"
	"github.com/wati/wati-byoa-test-agent/internal/outbound"
	"github.com/wati/wati-byoa-test-agent/internal/tools"
	"github.com/wati/wati-byoa-test-agent/internal/webhook"
)

type Service struct {
	loop    Loop
	wati    outbound.Client
	dedup   *dedup.Store
	context *contextstore.Store
	convMu  sync.Map
}

type Config struct {
	WatiBaseURL  string
	WatiTenantID string
	WatiAPIToken string
	SourceType   string

	LLMBaseURL string
	LLMAPIKey  string
	LLMModel   string
	LLMTimeout time.Duration

	MaxSteps         int
	MaxOutputTokens  int
	MaxContextTokens int
}

func NewService(cfg Config) (*Service, error) {
	contextStore, err := contextstore.New(config.StateDir + "/sessions")
	if err != nil {
		return nil, fmt.Errorf("init context store: %w", err)
	}
	dedupStore, err := dedup.New(config.StateDir + "/processed-event-ids.json")
	if err != nil {
		return nil, fmt.Errorf("init dedup store: %w", err)
	}

	watiClient := outbound.Client{
		BaseURL:    cfg.WatiBaseURL,
		TenantID:   cfg.WatiTenantID,
		APIToken:   cfg.WatiAPIToken,
		SourceType: cfg.SourceType,
	}

	return &Service{
		loop: Loop{
			LLM: llm.Client{
				BaseURL: cfg.LLMBaseURL,
				APIKey:  cfg.LLMAPIKey,
				Model:   cfg.LLMModel,
			},
			Tools: tools.NewRegistry(),
			Config: LoopConfig{
				MaxSteps:         cfg.MaxSteps,
				MaxOutputTokens:  cfg.MaxOutputTokens,
				MaxContextTokens: cfg.MaxContextTokens,
			},
		},
		wati:    watiClient,
		dedup:   dedupStore,
		context: contextStore,
	}, nil
}

func (s *Service) HandleMessageReceived(ctx context.Context, event webhook.MessageReceived) {
	_ = ctx
	if event.EventType != "" && event.EventType != "message" {
		return
	}
	if event.Owner {
		return
	}
	userText := strings.TrimSpace(event.UserText())
	if userText == "" {
		log.Printf("byoa-test-agent: skip empty inbound message id=%s conversation=%s", event.ID, event.ConversationID)
		return
	}
	if event.ConversationID == "" || event.WaID == "" {
		log.Printf("byoa-test-agent: skip message missing conversationId/waId id=%s", event.ID)
		return
	}
	if event.SenderName != "" {
		userText = fmt.Sprintf("[%s] %s", event.SenderName, userText)
	}

	s.processTurn(
		event.Context(),
		"message:"+event.ID,
		"message="+event.ID,
		userText,
		false,
		event.ReplyContextID,
	)
}

func (s *Service) HandleChatAssigned(ctx context.Context, event webhook.ChatAssigned) {
	_ = ctx
	if event.EventType != "" && event.EventType != "chatAssigned" {
		return
	}
	if event.ConversationID == "" || event.WaID == "" {
		log.Printf("byoa-test-agent: skip chatAssigned missing conversationId/waId ticket=%s", event.TicketID)
		return
	}

	dedupKey := strings.TrimSpace(event.TicketID)
	if dedupKey == "" {
		dedupKey = event.ConversationID
	}

	s.processTurn(
		event.Context(),
		"chatAssigned:"+dedupKey,
		"chatAssigned ticket="+dedupKey,
		ChatAssignedUserPrompt,
		true,
		"",
	)
}

func (s *Service) processTurn(
	conv webhook.ConversationContext,
	dedupID string,
	logLabel string,
	userText string,
	resetSession bool,
	replyContextID string,
) {
	newEvent, err := s.dedup.MarkIfNew(dedupID)
	if err != nil {
		log.Printf("byoa-test-agent: dedup error id=%s: %v", dedupID, err)
		return
	}
	if !newEvent {
		log.Printf("byoa-test-agent: duplicate %s, skipping", dedupID)
		return
	}

	unlock := s.lockConversation(conv.ConversationID)
	defer unlock()

	runCtx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	start := time.Now()
	log.Printf(
		"byoa-test-agent: agent start %s conversation=%s waId=%s channel=%s reset=%t",
		logLabel, conv.ConversationID, conv.WaID, conv.ChannelType.String(), resetSession,
	)

	result, err := s.loop.Run(runCtx, s.context, TurnInput{
		ConversationID: conv.ConversationID,
		Scope:          ScopeFromContext(conv),
		UserText:       userText,
		ResetSession:   resetSession,
	})
	elapsed := time.Since(start)
	if err != nil {
		log.Printf(
			"byoa-test-agent: agent failed %s conversation=%s after %s: %v",
			logLabel, conv.ConversationID, elapsed.Round(time.Millisecond), err,
		)
		return
	}

	reply := strings.TrimSpace(result.Reply)
	if reply == "" {
		log.Printf("byoa-test-agent: empty reply %s conversation=%s after %s", logLabel, conv.ConversationID, elapsed.Round(time.Millisecond))
		return
	}
	log.Printf(
		"byoa-test-agent: agent done %s conversation=%s after %s reply_len=%d",
		logLabel, conv.ConversationID, elapsed.Round(time.Millisecond), len(reply),
	)

	sendCtx, sendCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer sendCancel()
	sendResult, err := s.wati.SendText(sendCtx, sendRequestFor(conv, reply, replyContextID))
	if err != nil {
		log.Printf(
			"byoa-test-agent: send failed %s conversation=%s waId=%s channel=%s: %v",
			logLabel, conv.ConversationID, conv.WaID, conv.ChannelType, err,
		)
		return
	}
	log.Printf(
		"byoa-test-agent: replied %s conversation=%s waId=%s channel=%s outbound=%s",
		logLabel, conv.ConversationID, conv.WaID, conv.ChannelType, sendResult.PlatformMessageID,
	)
}

func (s *Service) lockConversation(conversationID string) func() {
	value, _ := s.convMu.LoadOrStore(conversationID, &sync.Mutex{})
	mutex := value.(*sync.Mutex)
	mutex.Lock()
	return mutex.Unlock
}
