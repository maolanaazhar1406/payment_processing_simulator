package domain

import (
	"errors"
	"fmt"
	"math"
	"strings"
)

var ErrCurrencyMismatch = errors.New("currency mismatch")
var ErrInvalidMoney = errors.New("invalid money")

// Money stores an amount in the smallest unit of its currency.
type Money struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

// NewMoney validates and returns a monetary amount.
func NewMoney(amount int64, currency string) (Money, error) {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if amount < 0 || (currency != "IDR" && currency != "USD") {
		return Money{}, ErrInvalidMoney
	}
	return Money{Amount: amount, Currency: currency}, nil
}

// Validate checks that the amount has a supported currency and non-negative value.
func (m Money) Validate() error {
	if m.Amount < 0 || (m.Currency != "IDR" && m.Currency != "USD") {
		return ErrInvalidMoney
	}
	return nil
}

// Add returns the sum of two amounts in the same currency.
func (m Money) Add(other Money) (Money, error) {
	if err := m.sameCurrency(other); err != nil {
		return Money{}, err
	}
	if other.Amount > math.MaxInt64-m.Amount {
		return Money{}, fmt.Errorf("add money: %w", ErrInvalidMoney)
	}
	return Money{Amount: m.Amount + other.Amount, Currency: m.Currency}, nil
}

// Sub returns the difference of two amounts in the same currency.
func (m Money) Sub(other Money) (Money, error) {
	if err := m.sameCurrency(other); err != nil {
		return Money{}, err
	}
	if other.Amount > m.Amount {
		return Money{}, fmt.Errorf("subtract money: %w", ErrInvalidMoney)
	}
	return Money{Amount: m.Amount - other.Amount, Currency: m.Currency}, nil
}

// Compare compares two amounts, returning -1, 0, or 1.
func (m Money) Compare(other Money) (int, error) {
	if err := m.sameCurrency(other); err != nil {
		return 0, err
	}
	switch {
	case m.Amount < other.Amount:
		return -1, nil
	case m.Amount > other.Amount:
		return 1, nil
	default:
		return 0, nil
	}
}

func (m Money) sameCurrency(other Money) error {
	if err := m.Validate(); err != nil {
		return err
	}
	if err := other.Validate(); err != nil {
		return err
	}
	if m.Currency != other.Currency {
		return ErrCurrencyMismatch
	}
	return nil
}
