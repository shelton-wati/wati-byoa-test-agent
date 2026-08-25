package dedup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type Store struct {
	path string
	mu   sync.Mutex
	seen map[string]struct{}
}

func New(path string) (*Store, error) {
	if path == "" {
		return nil, os.ErrInvalid
	}
	store := &Store{path: path, seen: make(map[string]struct{})}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) MarkIfNew(id string) (bool, error) {
	if id == "" {
		return true, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.seen[id]; ok {
		return false, nil
	}
	s.seen[id] = struct{}{}
	return true, s.persistLocked()
}

func (s *Store) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var ids []string
	if err := json.Unmarshal(data, &ids); err != nil {
		return err
	}
	for _, id := range ids {
		if id != "" {
			s.seen[id] = struct{}{}
		}
	}
	return nil
}

func (s *Store) persistLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	ids := make([]string, 0, len(s.seen))
	for id := range s.seen {
		ids = append(ids, id)
	}
	data, err := json.Marshal(ids)
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o600)
}
