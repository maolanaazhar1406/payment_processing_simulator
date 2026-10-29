package domain

import (
	"errors"
	"testing"
)

func TestMoneyOperations(t *testing.T) {
	tests := []struct {
		name    string
		left    Money
		right   Money
		add     int64
		sub     int64
		compare int
		wantErr error
	}{
		{"same currency", Money{100, "IDR"}, Money{40, "IDR"}, 140, 60, 1, nil},
		{"different currency", Money{100, "IDR"}, Money{40, "USD"}, 0, 0, 0, ErrCurrencyMismatch},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			add, addErr := tt.left.Add(tt.right)
			sub, subErr := tt.left.Sub(tt.right)
			cmp, cmpErr := tt.left.Compare(tt.right)
			if tt.wantErr != nil {
				for _, err := range []error{addErr, subErr, cmpErr} {
					if !errors.Is(err, tt.wantErr) {
						t.Fatalf("got error %v, want %v", err, tt.wantErr)
					}
				}
				return
			}
			if addErr != nil || subErr != nil || cmpErr != nil {
				t.Fatalf("unexpected operation errors: add=%v sub=%v compare=%v", addErr, subErr, cmpErr)
			}
			if add.Amount != tt.add || sub.Amount != tt.sub || cmp != tt.compare {
				t.Fatalf("got add=%d sub=%d compare=%d", add.Amount, sub.Amount, cmp)
			}
		})
	}
}

func TestMoneySubRejectsNegativeResult(t *testing.T) {
	_, err := (Money{Amount: 10, Currency: "IDR"}).Sub(Money{Amount: 40, Currency: "IDR"})
	if !errors.Is(err, ErrInvalidMoney) {
		t.Fatalf("Sub() error=%v, want ErrInvalidMoney", err)
	}
}

func TestNewMoney(t *testing.T) {
	tests := []struct {
		amount   int64
		currency string
		wantErr  bool
	}{
		{100, "IDR", false}, {100, "usd", false}, {-1, "IDR", true}, {1, "EUR", true},
	}
	for _, tt := range tests {
		_, err := NewMoney(tt.amount, tt.currency)
		if (err != nil) != tt.wantErr {
			t.Errorf("NewMoney(%d, %q) error=%v, wantErr=%t", tt.amount, tt.currency, err, tt.wantErr)
		}
	}
}
