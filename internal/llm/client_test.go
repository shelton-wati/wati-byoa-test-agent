package llm

import (
	"encoding/json"
	"testing"

	contextstore "github.com/wati/wati-byoa-test-agent/internal/context"
)

func TestObjectSchemaOmitsEmptyRequired(t *testing.T) {
	raw, err := json.Marshal(ObjectSchema(nil, map[string]any{}))
	if err != nil {
		t.Fatal(err)
	}
	if containsRequiredNull(raw) {
		t.Fatalf("schema should not include required:null: %s", raw)
	}
}

func containsRequiredNull(raw []byte) bool {
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return false
	}
	required, ok := payload["required"]
	return ok && required == nil
}

func TestChatBodyOpenAICompatible(t *testing.T) {
	body := chatBody(ChatRequest{
		Model: "gpt-4.1-mini",
		Messages: []contextstore.Message{
			{Role: contextstore.RoleUser, Content: "hello"},
		},
		MaxTokens: 2048,
		Tools: []ToolDef{{
			Type: "function",
			Function: FunctionDef{
				Name:        "get_current_time",
				Description: "time",
				Parameters:  ObjectSchema(nil, map[string]any{}),
			},
		}},
	})
	if body["max_tokens"] != 2048 {
		t.Fatalf("max_tokens = %#v, want 2048", body["max_tokens"])
	}
	tools, ok := body["tools"].([]ToolDef)
	if !ok || len(tools) != 1 {
		t.Fatalf("tools = %#v", body["tools"])
	}
	raw, err := json.Marshal(tools[0].Function.Parameters)
	if err != nil {
		t.Fatal(err)
	}
	if containsRequiredNull(raw) {
		t.Fatalf("tool parameters must not include required:null: %s", raw)
	}
}

func TestChatBodyOmitsUnsetOptionalFields(t *testing.T) {
	body := chatBody(ChatRequest{
		Model: "glm-5.2",
		Messages: []contextstore.Message{
			{Role: contextstore.RoleUser, Content: "hello"},
		},
	})
	if _, ok := body["max_tokens"]; ok {
		t.Fatalf("max_tokens should be omitted when unset: %#v", body)
	}
	if _, ok := body["tools"]; ok {
		t.Fatalf("tools should be omitted when empty: %#v", body)
	}
	if _, ok := body["temperature"]; ok {
		t.Fatalf("temperature should be omitted when unset: %#v", body)
	}
}

func TestAPIMessagesAssistantToolCallsUseNullContent(t *testing.T) {
	call := contextstore.ToolCall{ID: "call_1", Type: "function"}
	call.Function.Name = "calculate"
	call.Function.Arguments = `{"expression":"1+1"}`
	msgs := apiMessages([]contextstore.Message{{
		Role:      contextstore.RoleAssistant,
		ToolCalls: []contextstore.ToolCall{call},
	}})
	if msgs[0]["content"] != nil {
		t.Fatalf("content = %#v, want nil", msgs[0]["content"])
	}
}
