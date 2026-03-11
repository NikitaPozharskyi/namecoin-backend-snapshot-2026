package core

import "sync"

type Store interface {
	Get(key string) []byte
	Set(key string, value []byte)
	Delete(key string)
	ForEach(func(key string, value []byte) bool)
}

type MemoryStore struct {
	mu   sync.RWMutex
	data map[string][]byte
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{data: make(map[string][]byte)}
}

func (s *MemoryStore) Get(key string) []byte {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return CloneBytes(s.data[key])
}

func (s *MemoryStore) Set(key string, value []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = CloneBytes(value)
}

func (s *MemoryStore) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
}

func (s *MemoryStore) ForEach(fn func(key string, value []byte) bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for key, value := range s.data {
		if !fn(key, CloneBytes(value)) {
			return
		}
	}
}

type overlayStore struct {
	base Store
	mu   sync.Mutex
	data map[string][]byte
	del  map[string]struct{}
}

func newOverlayStore(base Store) *overlayStore {
	return &overlayStore{
		base: base,
		data: make(map[string][]byte),
		del:  make(map[string]struct{}),
	}
}

func (s *overlayStore) Get(key string) []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, deleted := s.del[key]; deleted {
		return nil
	}
	if value, ok := s.data[key]; ok {
		return CloneBytes(value)
	}
	return s.base.Get(key)
}

func (s *overlayStore) Set(key string, value []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.del, key)
	s.data[key] = CloneBytes(value)
}

func (s *overlayStore) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
	s.del[key] = struct{}{}
}

func (s *overlayStore) ForEach(fn func(key string, value []byte) bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	seen := make(map[string]struct{}, len(s.data)+len(s.del))
	for key, value := range s.data {
		seen[key] = struct{}{}
		if !fn(key, CloneBytes(value)) {
			return
		}
	}
	s.base.ForEach(func(key string, value []byte) bool {
		if _, deleted := s.del[key]; deleted {
			return true
		}
		if _, ok := seen[key]; ok {
			return true
		}
		return fn(key, value)
	})
}

func (s *overlayStore) Commit() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for key := range s.del {
		s.base.Delete(key)
	}
	for key, value := range s.data {
		s.base.Set(key, value)
	}
	s.data = make(map[string][]byte)
	s.del = make(map[string]struct{})
}
