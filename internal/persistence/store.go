// Package persistence stores small, user-scoped BioScope library records.
package persistence

import (
	"context"
	"encoding/json"
	"sort"
	"sync"
	"time"
)

// Record is a saved search or reproducible workspace snapshot. OwnerID is a
// browser-generated UUID used as a high-entropy capability; it is not an auth
// identity. Deployments that need authenticated users should bind it to auth.
type Record struct {
	OwnerID string          `json:"owner_id"`
	Key     string          `json:"record_key"`
	Kind    string          `json:"kind"`
	Payload json.RawMessage `json:"payload"`
	Updated time.Time       `json:"updated_at"`
}

// Store is the persistence boundary used by HTTP handlers.
type Store interface {
	List(context.Context, string, string) ([]Record, error)
	Put(context.Context, Record) error
	Delete(context.Context, string, string) error
}

// MemoryStore keeps the application useful in local/offline deployments.
// Durable browser storage remains available independently through localStorage.
type MemoryStore struct {
	mu      sync.RWMutex
	records map[string]Record
}

func NewMemoryStore() *MemoryStore { return &MemoryStore{records: make(map[string]Record)} }

func storageKey(owner, key string) string { return owner + ":" + key }

func (m *MemoryStore) List(ctx context.Context, owner, kind string) ([]Record, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Record, 0)
	for _, r := range m.records {
		if r.OwnerID == owner && (kind == "" || r.Kind == kind) {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Updated.After(out[j].Updated) })
	return out, nil
}

func (m *MemoryStore) Put(ctx context.Context, r Record) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if r.Updated.IsZero() {
		r.Updated = time.Now().UTC()
	}
	m.records[storageKey(r.OwnerID, r.Key)] = r
	return nil
}

func (m *MemoryStore) Delete(ctx context.Context, owner, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.records, storageKey(owner, key))
	return nil
}
