package store

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	ErrIdempotencyKeyReused = errors.New("idempotency key reused with a different request")
	ErrRequestInProgress    = errors.New("idempotent request is in progress")
)

// IdempotencyRecord contains the cached response for one merchant request key.
type IdempotencyRecord struct {
	RequestHash    string
	ResponseStatus int
	ResponseBody   []byte
	CreatedAt      time.Time
	ExpiresAt      time.Time
	InProgress     bool
}

// MemoryIdempotency stores idempotency records with lazy expiration.
type MemoryIdempotency struct {
	mu      sync.Mutex
	ttl     time.Duration
	records map[string]IdempotencyRecord
	now     func() time.Time
}

// NewMemoryIdempotency creates an in-memory idempotency store.
func NewMemoryIdempotency(ttl time.Duration) *MemoryIdempotency {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &MemoryIdempotency{ttl: ttl, records: make(map[string]IdempotencyRecord), now: time.Now}
}

// Start atomically reserves a key, or returns its cached response.
func (s *MemoryIdempotency) Start(ctx context.Context, merchantID, key, hash string) (IdempotencyRecord, bool, error) {
	if err := ctx.Err(); err != nil {
		return IdempotencyRecord{}, false, err
	}
	now := s.now()
	index := merchantID + "\x00" + key
	s.mu.Lock()
	defer s.mu.Unlock()
	record, exists := s.records[index]
	if exists && !now.Before(record.ExpiresAt) {
		delete(s.records, index)
		exists = false
	}
	if exists {
		if record.RequestHash != hash {
			return IdempotencyRecord{}, false, ErrIdempotencyKeyReused
		}
		if record.InProgress {
			return IdempotencyRecord{}, false, ErrRequestInProgress
		}
		return cloneRecord(record), true, nil
	}
	s.records[index] = IdempotencyRecord{
		RequestHash: hash, CreatedAt: now, ExpiresAt: now.Add(s.ttl), InProgress: true,
	}
	return IdempotencyRecord{}, false, nil
}

// Complete saves the response for a previously reserved key.
func (s *MemoryIdempotency) Complete(ctx context.Context, merchantID, key, hash string, status int, body []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	index := merchantID + "\x00" + key
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.records[index]
	if !ok || !record.InProgress || record.RequestHash != hash {
		return errors.New("idempotency reservation was lost")
	}
	record.ResponseStatus = status
	record.ResponseBody = append([]byte(nil), body...)
	record.InProgress = false
	s.records[index] = record
	return nil
}

// Abort removes an unfinished reservation after a request fails before responding.
func (s *MemoryIdempotency) Abort(ctx context.Context, merchantID, key, hash string) {
	index := merchantID + "\x00" + key
	s.mu.Lock()
	defer s.mu.Unlock()
	if record, ok := s.records[index]; ok && record.InProgress && record.RequestHash == hash {
		delete(s.records, index)
	}
}

func cloneRecord(record IdempotencyRecord) IdempotencyRecord {
	record.ResponseBody = append([]byte(nil), record.ResponseBody...)
	return record
}
