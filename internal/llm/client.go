package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	contextstore "github.com/wati/wati-byoa-test-agent/internal/context"
)

// Client calls an OpenAI-compatible /chat/completions endpoint.
// Request shaping follows the OpenAI tool-calling format and works for
// OpenAI, OpenCode Go, and other compatible providers — no provider forks.
type Client struct {
	BaseURL string
	APIKey  string
	Model   string
	HTTP    *http.Client
}

type ToolDef struct {
	Type     string      `json:"type"`
	Function FunctionDef `json:"function"`
}

type FunctionDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type ChatRequest struct {
	Model       string
	Messages    []contextstore.Message
	Tools       []ToolDef
	MaxTokens   int
	Temperature *float64
}

type ChatResponse struct {
	Choices []struct {
		Message struct {
			Role      string
			Content   string
			ToolCalls []contextstore.ToolCall
		}
		FinishReason string
	}
	Error *struct {
		Message string
		Type    string
	}
}

func (c Client) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	body := chatBody(req)
	payload, err := json.Marshal(body)
	if err != nil {
		return ChatResponse{}, err
	}

	endpoint := strings.TrimRight(c.BaseURL, "/") + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return ChatResponse{}, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+strings.TrimSpace(c.APIKey))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 2 * time.Minute}
	}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return ChatResponse{}, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return ChatResponse{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ChatResponse{}, fmt.Errorf("llm http %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Role      string                  `json:"role"`
				Content   string                  `json:"content"`
				ToolCalls []contextstore.ToolCall `json:"tool_calls"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return ChatResponse{}, fmt.Errorf("decode llm response: %w", err)
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return ChatResponse{}, fmt.Errorf("llm error: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return ChatResponse{}, fmt.Errorf("llm returned no choices")
	}

	out := ChatResponse{}
	out.Choices = make([]struct {
		Message struct {
			Role      string
			Content   string
			ToolCalls []contextstore.ToolCall
		}
		FinishReason string
	}, len(parsed.Choices))
	for i, choice := range parsed.Choices {
		out.Choices[i].FinishReason = choice.FinishReason
		out.Choices[i].Message.Role = choice.Message.Role
		out.Choices[i].Message.Content = choice.Message.Content
		out.Choices[i].Message.ToolCalls = choice.Message.ToolCalls
	}
	return out, nil
}

func chatBody(req ChatRequest) map[string]any {
	body := map[string]any{
		"model":    req.Model,
		"messages": apiMessages(req.Messages),
	}
	if len(req.Tools) > 0 {
		body["tools"] = req.Tools
	}
	if req.Temperature != nil {
		body["temperature"] = *req.Temperature
	}
	if req.MaxTokens > 0 {
		body["max_tokens"] = req.MaxTokens
	}
	return body
}

func apiMessages(messages []contextstore.Message) []map[string]any {
	out := make([]map[string]any, 0, len(messages))
	for _, msg := range messages {
		item := map[string]any{
			"role": string(msg.Role),
		}
		switch msg.Role {
		case contextstore.RoleAssistant:
			if len(msg.ToolCalls) > 0 {
				if strings.TrimSpace(msg.Content) == "" {
					item["content"] = nil
				} else {
					item["content"] = msg.Content
				}
				item["tool_calls"] = apiToolCalls(msg.ToolCalls)
			} else {
				item["content"] = msg.Content
			}
		case contextstore.RoleTool:
			item["content"] = msg.Content
			item["tool_call_id"] = msg.ToolCallID
		default:
			item["content"] = msg.Content
		}
		out = append(out, item)
	}
	return out
}

func apiToolCalls(calls []contextstore.ToolCall) []map[string]any {
	out := make([]map[string]any, 0, len(calls))
	for _, call := range calls {
		toolType := strings.TrimSpace(call.Type)
		if toolType == "" {
			toolType = "function"
		}
		out = append(out, map[string]any{
			"id":   call.ID,
			"type": toolType,
			"function": map[string]any{
				"name":      call.Function.Name,
				"arguments": call.Function.Arguments,
			},
		})
	}
	return out
}

func ObjectSchema(required []string, properties map[string]any) map[string]any {
	if properties == nil {
		properties = map[string]any{}
	}
	schema := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}
