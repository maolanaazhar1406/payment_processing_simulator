package store

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestIdempotencyStartCompleteAndReplay(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryIdempotency(time.Hour)
	record, replay, err := store.Start(ctx, "merchant", "key", "hash")
	if err != nil || replay || record.ResponseStatus != 0 {
		t.Fatalf("initial Start()=(%+v, %t, %v)", record, replay, err)
	}
	if _, _, err := store.Start(ctx, "merchant", "key", "hash"); !errors.Is(err, ErrRequestInProgress) {
		t.Fatalf("second Start() error=%v", err)
	}
	if _, _, err := store.Start(ctx, "merchant", "key", "other-hash"); !errors.Is(err, ErrIdempotencyKeyReused) {
		t.Fatalf("different hash error=%v", err)
	}
	if err := store.Complete(ctx, "merchant", "key", "hash", 201, []byte("response")); err != nil {
		t.Fatal(err)
	}
	record, replay, err = store.Start(ctx, "merchant", "key", "hash")
	if err != nil || !replay || record.ResponseStatus != 201 || string(record.ResponseBody) != "response" {
		t.Fatalf("replay Start()=(%+v, %t, %v)", record, replay, err)
	}
}

func TestIdempotencyExpiresLazily(t *testing.T) {
	store := NewMemoryIdempotency(time.Hour)
	now := time.Now()
	store.now = func() time.Time { return now }
	if _, _, err := store.Start(context.Background(), "m", "k", "h"); err != nil {
		t.Fatal(err)
	}
	now = now.Add(2 * time.Hour)
	if _, replay, err := store.Start(context.Background(), "m", "k", "h"); err != nil || replay {
		t.Fatalf("expired key was not reserved again: replay=%t err=%v", replay, err)
	}
}
