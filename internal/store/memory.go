package store

import (
	"context"
	"sync"

	"payment-processing-simulator/internal/domain"
)

// MemoryTransactions stores transactions safely in process memory.
type MemoryTransactions struct {
	mu           sync.RWMutex
	transactions map[string]domain.Transaction
}

// NewMemoryTransactions constructs an empty in-memory transaction store.
func NewMemoryTransactions() *MemoryTransactions {
	return &MemoryTransactions{transactions: make(map[string]domain.Transaction)}
}

// Create stores a transaction with its initial version.
func (s *MemoryTransactions) Create(ctx context.Context, tx domain.Transaction) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.transactions[tx.ID]; exists {
		return domain.ErrVersionConflict
	}
	tx.Version = 1
	s.transactions[tx.ID] = tx
	return nil
}

// Get returns a transaction by ID.
func (s *MemoryTransactions) Get(ctx context.Context, id string) (domain.Transaction, error) {
	if err := ctx.Err(); err != nil {
		return domain.Transaction{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	tx, ok := s.transactions[id]
	if !ok {
		return domain.Transaction{}, domain.ErrNotFound
	}
	return tx, nil
}

// Update replaces a transaction only if its version still matches expectedVersion.
func (s *MemoryTransactions) Update(ctx context.Context, tx domain.Transaction, expectedVersion int64) (domain.Transaction, error) {
	if err := ctx.Err(); err != nil {
		return domain.Transaction{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.transactions[tx.ID]
	if !ok {
		return domain.Transaction{}, domain.ErrNotFound
	}
	if current.Version != expectedVersion {
		return domain.Transaction{}, domain.ErrVersionConflict
	}
	tx.Version = expectedVersion + 1
	s.transactions[tx.ID] = tx
	return tx, nil
}
