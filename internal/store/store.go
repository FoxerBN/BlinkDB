package store

import "sync"

type Store struct {
	mu   sync.RWMutex
	data map[string]string
}

//* Creates a new instance of Store with an empty data map.
func NewStore() *Store {
	return &Store{
		data: make(map[string]string),
	}
}

//* Sets the value for the given key in the store. If the key already exists, its value is overwritten.
func (s *Store) Set(key string, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[key] = value
}

//* Retrieves the value associated with the given key. Returns the value and a boolean indicating whether the key exists.
func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	value, exists := s.data[key]
	return value, exists
}

//* Deletes the key-value pair associated with the given key. Returns true if the key was found and deleted, false otherwise.
func (s *Store) Delete(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.data[key]; !exists {
		return false
	}

	delete(s.data, key)
	return true
}

//* Returns the number of key-value pairs in the store.
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.data)
}

//* Returns true if the key exists in the store, false otherwise.
func (s *Store) Exists(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.data[key]
	return exists
}

//* Clears all key-value pairs from the store.
func (s *Store) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = make(map[string]string)
}
