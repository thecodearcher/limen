package limen

import (
	"context"
	"sync"
	"time"
)

type memoryEntry struct {
	value     []byte
	expiresAt time.Time // zero value means no expiry
}

func (e *memoryEntry) isExpired() bool {
	return !e.expiresAt.IsZero() && time.Now().After(e.expiresAt)
}

// MemoryCacheStore is the default in-process CacheAdapter implementation.
// It uses sync.RWMutex-protected maps with lazy expiry on read.
type MemoryCacheStore struct {
	mu   sync.RWMutex
	data map[string]*memoryEntry
}

func NewMemoryCacheStore() *MemoryCacheStore {
	return &MemoryCacheStore{
		data: make(map[string]*memoryEntry),
	}
}

func (m *MemoryCacheStore) Get(_ context.Context, key string) ([]byte, error) {
	m.mu.RLock()
	entry, ok := m.data[key]
	m.mu.RUnlock()

	if !ok {
		return nil, ErrRecordNotFound
	}
	if entry.isExpired() {
		m.mu.Lock()
		delete(m.data, key)
		m.mu.Unlock()
		return nil, ErrRecordNotFound
	}

	cp := make([]byte, len(entry.value))
	copy(cp, entry.value)
	return cp, nil
}

func (m *MemoryCacheStore) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	cp := make([]byte, len(value))
	copy(cp, value)

	m.data[key] = &memoryEntry{
		value:     cp,
		expiresAt: expiresAt,
	}
	return nil
}

func (m *MemoryCacheStore) Has(_ context.Context, key string) (bool, error) {
	m.mu.RLock()
	entry, ok := m.data[key]
	m.mu.RUnlock()

	if !ok {
		return false, nil
	}
	if entry.isExpired() {
		m.mu.Lock()
		delete(m.data, key)
		m.mu.Unlock()
		return false, nil
	}
	return true, nil
}

func (m *MemoryCacheStore) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	delete(m.data, key)
	m.mu.Unlock()
	return nil
}
