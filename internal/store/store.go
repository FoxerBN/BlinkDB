package store

import "sync"

type Store struct {
	mu   sync.RWMutex
	data map[string]string
}

func NewStore() *Store {
	return &Store{
		data: make(map[string]string),
	}
}

func (s *Store) Set(key string, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[key] = value
}

func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	value, exists := s.data[key]
	return value, exists
}

func (s *Store) Delete(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.data[key]; !exists {
		return false
	}

	delete(s.data, key)
	return true
}

func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.data)
}

// TODO: Wire this into protocol commands or tests when key-existence checks are needed.
func (s *Store) Exists(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.data[key]
	return exists
}

// TODO: Use this from tests or future reset/admin commands.
func (s *Store) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = make(map[string]string)
}
