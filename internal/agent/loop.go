package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	contextstore "github.com/wati/wati-byoa-test-agent/internal/context"
	"github.com/wati/wati-byoa-test-agent/internal/llm"
	"github.com/wati/wati-byoa-test-agent/internal/tools"
	"github.com/wati/wati-byoa-test-agent/internal/webhook"
)

type LoopConfig struct {
	MaxSteps        int
	MaxOutputTokens int
	MaxContextTokens int
}

type Loop struct {
	LLM    llm.Client
	Tools  *tools.Registry
	Config LoopConfig
}

type TurnInput struct {
	ConversationID string
	Scope          tools.Scope
	UserText       string
	ResetSession   bool
}

type TurnResult struct {
	Reply string
}

func (l Loop) Run(ctx context.Context, store *contextstore.Store, input TurnInput) (TurnResult, error) {
	if input.ResetSession {
		if err := store.Reset(input.ConversationID); err != nil {
			return TurnResult{}, err
		}
	}

	history, err := store.Get(input.ConversationID)
	if err != nil {
		return TurnResult{}, err
	}
	if len(history) == 0 {
		if err := store.Append(input.ConversationID, contextstore.Message{
			Role:    contextstore.RoleSystem,
			Content: SystemPrompt,
		}); err != nil {
			return TurnResult{}, err
		}
	}

	userText := strings.TrimSpace(input.UserText)
	if userText == "" {
		return TurnResult{}, fmt.Errorf("user text is required")
	}
	if err := store.Append(input.ConversationID, contextstore.Message{
		Role:    contextstore.RoleUser,
		Content: userText,
	}); err != nil {
		return TurnResult{}, err
	}
	if err := store.Truncate(input.ConversationID, l.Config.MaxContextTokens); err != nil {
		return TurnResult{}, err
	}

	maxSteps := l.Config.MaxSteps
	if maxSteps <= 0 {
		maxSteps = 6
	}

	for step := 0; step < maxSteps; step++ {
		messages, err := store.Get(input.ConversationID)
		if err != nil {
			return TurnResult{}, err
		}

		resp, err := l.LLM.Chat(ctx, llm.ChatRequest{
			Model:     l.LLM.Model,
			Messages:  messages,
			Tools:     l.Tools.Definitions(),
			MaxTokens: l.Config.MaxOutputTokens,
		})
		if err != nil {
			return TurnResult{}, err
		}

		choice := resp.Choices[0].Message
		assistantMsg := contextstore.Message{
			Role:      contextstore.RoleAssistant,
			Content:   strings.TrimSpace(choice.Content),
			ToolCalls: choice.ToolCalls,
		}
		if err := store.Append(input.ConversationID, assistantMsg); err != nil {
			return TurnResult{}, err
		}

		if len(choice.ToolCalls) == 0 {
			reply := strings.TrimSpace(choice.Content)
			if reply == "" {
				return TurnResult{}, fmt.Errorf("model returned empty reply")
			}
			return TurnResult{Reply: reply}, nil
		}

		for _, call := range choice.ToolCalls {
			toolName := strings.TrimSpace(call.Function.Name)
			result, execErr := l.Tools.Execute(ctx, input.Scope, toolName, json.RawMessage(call.Function.Arguments))
			if execErr != nil {
				result = "error: " + execErr.Error()
			}
			if err := store.Append(input.ConversationID, contextstore.Message{
				Role:       contextstore.RoleTool,
				Name:       toolName,
				ToolCallID: call.ID,
				Content:    result,
			}); err != nil {
				return TurnResult{}, err
			}
		}
	}

	return TurnResult{}, fmt.Errorf("agent exceeded max steps (%d)", maxSteps)
}

func ScopeFromContext(ctx webhook.ConversationContext) tools.Scope {
	return tools.Scope{Conversation: ctx}
}
