package contextstore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

type Message struct {
	Role       Role   `json:"role"`
	Content    string `json:"content,omitempty"`
	Name       string `json:"name,omitempty"`
	ToolCallID string `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type Session struct {
	ConversationID string    `json:"conversation_id"`
	Messages       []Message `json:"messages"`
}

type Store struct {
	dir string
	mu  sync.Mutex
}

func New(dir string) (*Store, error) {
	if dir == "" {
		return nil, fmt.Errorf("context store dir is required")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	return &Store{dir: dir}, nil
}

func (s *Store) Reset(conversationID string) error {
	conversationID = sanitizeID(conversationID)
	if conversationID == "" {
		return fmt.Errorf("conversation id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	session := Session{ConversationID: conversationID, Messages: nil}
	return s.saveLocked(session)
}

func (s *Store) Append(conversationID string, msg Message) error {
	conversationID = sanitizeID(conversationID)
	if conversationID == "" {
		return fmt.Errorf("conversation id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	session, err := s.loadLocked(conversationID)
	if err != nil {
		return err
	}
	session.Messages = append(session.Messages, msg)
	return s.saveLocked(session)
}

func (s *Store) Get(conversationID string) ([]Message, error) {
	conversationID = sanitizeID(conversationID)
	if conversationID == "" {
		return nil, fmt.Errorf("conversation id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	session, err := s.loadLocked(conversationID)
	if err != nil {
		return nil, err
	}
	out := make([]Message, len(session.Messages))
	copy(out, session.Messages)
	return out, nil
}

func (s *Store) Truncate(conversationID string, maxTokens int) error {
	conversationID = sanitizeID(conversationID)
	if conversationID == "" {
		return fmt.Errorf("conversation id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	session, err := s.loadLocked(conversationID)
	if err != nil {
		return err
	}
	session.Messages = truncateMessages(session.Messages, maxTokens)
	return s.saveLocked(session)
}

func truncateMessages(messages []Message, maxTokens int) []Message {
	if maxTokens <= 0 || len(messages) == 0 {
		return messages
	}
	for estimateTokens(messages) > maxTokens && len(messages) > 1 {
		dropIdx := -1
		for i, msg := range messages {
			if msg.Role == RoleSystem {
				continue
			}
			dropIdx = i
			break
		}
		if dropIdx < 0 {
			break
		}
		messages = append(messages[:dropIdx], messages[dropIdx+1:]...)
	}
	return messages
}

func estimateTokens(messages []Message) int {
	total := 0
	for _, msg := range messages {
		total += len(msg.Content)/4 + 4
		for _, call := range msg.ToolCalls {
			total += len(call.Function.Arguments)/4 + 8
		}
	}
	return total
}

func (s *Store) pathFor(conversationID string) string {
	return filepath.Join(s.dir, conversationID+".json")
}

func (s *Store) loadLocked(conversationID string) (Session, error) {
	path := s.pathFor(conversationID)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Session{ConversationID: conversationID}, nil
	}
	if err != nil {
		return Session{}, err
	}
	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return Session{}, err
	}
	if session.ConversationID == "" {
		session.ConversationID = conversationID
	}
	return session, nil
}

func (s *Store) saveLocked(session Session) error {
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.pathFor(session.ConversationID), data, 0o600)
}

func sanitizeID(raw string) string {
	raw = strings.TrimSpace(raw)
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
			return r
		default:
			return '_'
		}
	}, raw)
}
