package store

import (
	"context"
	"errors"
	"testing"

	"payment-processing-simulator/internal/domain"
)

func TestMemoryTransactionsVersioning(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryTransactions()
	tx := domain.Transaction{ID: "tx1", Amount: domain.Money{Amount: 100, Currency: "IDR"}}
	if err := store.Create(ctx, tx); err != nil {
		t.Fatal(err)
	}
	saved, err := store.Get(ctx, tx.ID)
	if err != nil || saved.Version != 1 {
		t.Fatalf("Get()=(%+v, %v)", saved, err)
	}
	saved.Status = domain.StatusAuthorized
	saved, err = store.Update(ctx, saved, 1)
	if err != nil || saved.Version != 2 {
		t.Fatalf("Update()=(%+v, %v)", saved, err)
	}
	_, err = store.Update(ctx, saved, 1)
	if !errors.Is(err, domain.ErrVersionConflict) {
		t.Fatalf("stale update error=%v", err)
	}
}
