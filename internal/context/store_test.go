package contextstore

import (
	"path/filepath"
	"testing"
)

func TestStoreResetAndAppend(t *testing.T) {
	dir := t.TempDir()
	store, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}

	if err := store.Reset("conv1"); err != nil {
		t.Fatal(err)
	}
	if err := store.Append("conv1", Message{Role: RoleSystem, Content: "system"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Append("conv1", Message{Role: RoleUser, Content: "hello"}); err != nil {
		t.Fatal(err)
	}

	msgs, err := store.Get("conv1")
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 {
		t.Fatalf("len=%d", len(msgs))
	}

	if err := store.Reset("conv1"); err != nil {
		t.Fatal(err)
	}
	msgs, err = store.Get("conv1")
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 0 {
		t.Fatalf("expected empty session after reset, got %d", len(msgs))
	}

	if _, err := filepath.Glob(filepath.Join(dir, "*.json")); err != nil {
		t.Fatal(err)
	}
}

func TestTruncateMessages(t *testing.T) {
	msgs := []Message{
		{Role: RoleSystem, Content: "system"},
		{Role: RoleUser, Content: "one"},
		{Role: RoleAssistant, Content: "two"},
		{Role: RoleUser, Content: "three"},
	}
	out := truncateMessages(msgs, 10)
	if len(out) >= len(msgs) {
		t.Fatalf("expected truncation, got len=%d", len(out))
	}
	if out[0].Role != RoleSystem {
		t.Fatalf("system message should be preserved")
	}
}
