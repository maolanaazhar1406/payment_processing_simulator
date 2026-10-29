package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"payment-processing-simulator/internal/domain"
	"payment-processing-simulator/internal/simulator"
	"payment-processing-simulator/internal/store"
)

func TestPaymentLifecycle(t *testing.T) {
	ctx := context.Background()
	service := New(store.NewMemoryTransactions(), simulator.SimulatedIssuer{})
	tx, err := service.Authorize(ctx, validInput())
	if err != nil || tx.Status != domain.StatusAuthorized {
		t.Fatalf("Authorize()=(%+v, %v)", tx, err)
	}
	tx, err = service.Capture(ctx, "merchant", tx.ID, domain.Money{Amount: 700, Currency: "IDR"})
	if err != nil || tx.Status != domain.StatusCaptured || tx.CapturedAmount.Amount != 700 {
		t.Fatalf("Capture()=(%+v, %v)", tx, err)
	}
	tx, err = service.Refund(ctx, "merchant", tx.ID, domain.Money{Amount: 200, Currency: "IDR"})
	if err != nil || tx.Status != domain.StatusPartiallyRefunded || tx.RefundedAmount.Amount != 200 {
		t.Fatalf("partial Refund()=(%+v, %v)", tx, err)
	}
	tx, err = service.Refund(ctx, "merchant", tx.ID, domain.Money{Amount: 500, Currency: "IDR"})
	if err != nil || tx.Status != domain.StatusRefunded || tx.RefundedAmount.Amount != 700 {
		t.Fatalf("full Refund()=(%+v, %v)", tx, err)
	}
}

func TestAuthorizeDeclinedAndCardDataNotStored(t *testing.T) {
	ctx := context.Background()
	service := New(store.NewMemoryTransactions(), simulator.SimulatedIssuer{})
	input := validInput()
	input.PAN = "4111111111111112"
	tx, err := service.Authorize(ctx, input)
	if err != nil || tx.Status != domain.StatusDeclined || tx.ResponseCode != "14" {
		t.Fatalf("Authorize(invalid Luhn)=(%+v, %v)", tx, err)
	}
	if tx.Card.Last4 != "1112" || tx.Card.Brand != "visa" {
		t.Fatalf("unsafe or incomplete card metadata: %+v", tx.Card)
	}
}

func TestCaptureRejectsExcessAndOwnership(t *testing.T) {
	ctx := context.Background()
	service := New(store.NewMemoryTransactions(), simulator.SimulatedIssuer{})
	tx, err := service.Authorize(ctx, validInput())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Capture(ctx, "other", tx.ID, domain.Money{Amount: 1, Currency: "IDR"}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("wrong merchant error=%v", err)
	}
	if _, err := service.Capture(ctx, "merchant", tx.ID, domain.Money{Amount: 1001, Currency: "IDR"}); !errors.Is(err, domain.ErrInvalidAmount) {
		t.Fatalf("excess capture error=%v", err)
	}
}

func TestVoidAndInvalidTransition(t *testing.T) {
	ctx := context.Background()
	service := New(store.NewMemoryTransactions(), simulator.SimulatedIssuer{})
	tx, err := service.Authorize(ctx, validInput())
	if err != nil {
		t.Fatal(err)
	}
	tx, err = service.Void(ctx, "merchant", tx.ID)
	if err != nil || tx.Status != domain.StatusVoided {
		t.Fatalf("Void()=(%+v, %v)", tx, err)
	}
	if _, err := service.Void(ctx, "merchant", tx.ID); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("second Void() error=%v", err)
	}
}

func TestAuthorizeRejectsMalformedInput(t *testing.T) {
	input := validInput()
	input.CVV = "12"
	if _, err := New(store.NewMemoryTransactions(), simulator.SimulatedIssuer{}).Authorize(context.Background(), input); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("Authorize() error=%v", err)
	}
}

func TestAuthorizeValidationAndIssuerOutcomes(t *testing.T) {
	tests := []struct {
		name       string
		pan        string
		code       string
		wantStatus domain.Status
	}{
		{"declined", "4111111111120005", "05", domain.StatusDeclined},
		{"insufficient funds", "4111111111150051", "51", domain.StatusDeclined},
		{"issuer unavailable", "4111111111170091", "91", domain.StatusDeclined},
		{"approved", "4111111111170000", "00", domain.StatusAuthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paymentService := New(store.NewMemoryTransactions(), simulator.SimulatedIssuer{})
			input := validInput()
			input.PAN = tt.pan
			tx, err := paymentService.Authorize(context.Background(), input)
			if err != nil || tx.Status != tt.wantStatus || tx.ResponseCode != tt.code {
				t.Fatalf("Authorize()=(%+v, %v), want status %s code %s", tx, err, tt.wantStatus, tt.code)
			}
		})
	}

	invalidInputs := []struct {
		name   string
		mutate func(*AuthorizeInput)
	}{
		{"missing merchant", func(input *AuthorizeInput) { input.MerchantID = "" }},
		{"zero amount", func(input *AuthorizeInput) { input.Amount.Amount = 0 }},
		{"invalid currency", func(input *AuthorizeInput) { input.Amount.Currency = "EUR" }},
		{"bad PAN format", func(input *AuthorizeInput) { input.PAN = "4111abc" }},
		{"bad expiry month", func(input *AuthorizeInput) { input.ExpMonth = 13 }},
		{"non digit CVV", func(input *AuthorizeInput) { input.CVV = "12x" }},
		{"non ASCII digits", func(input *AuthorizeInput) { input.CVV = "１２３" }},
	}
	for _, tt := range invalidInputs {
		t.Run(tt.name, func(t *testing.T) {
			input := validInput()
			tt.mutate(&input)
			_, err := New(store.NewMemoryTransactions(), simulator.SimulatedIssuer{}).Authorize(context.Background(), input)
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("Authorize() error=%v, want ErrInvalidInput", err)
			}
		})
	}
}

func TestExpiredCardAndTransactionLookup(t *testing.T) {
	ctx := context.Background()
	paymentService := New(store.NewMemoryTransactions(), simulator.SimulatedIssuer{})
	input := validInput()
	input.ExpYear = time.Now().Year() - 1
	tx, err := paymentService.Authorize(ctx, input)
	if err != nil || tx.Status != domain.StatusDeclined || tx.ResponseCode != "54" {
		t.Fatalf("expired Authorize()=(%+v, %v)", tx, err)
	}
	tx, err = paymentService.Authorize(ctx, validInput())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := paymentService.GetTransaction(ctx, "merchant", tx.ID); err != nil {
		t.Fatalf("GetTransaction() error=%v", err)
	}
	if _, err := paymentService.GetTransaction(ctx, "other", tx.ID); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("GetTransaction(other merchant) error=%v", err)
	}
	if _, err := paymentService.GetTransaction(ctx, "merchant", "missing"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("GetTransaction(missing) error=%v", err)
	}
}

func TestCaptureAndRefundRejectInvalidAmountsOrTransitions(t *testing.T) {
	ctx := context.Background()
	paymentService := New(store.NewMemoryTransactions(), simulator.SimulatedIssuer{})
	tx, err := paymentService.Authorize(ctx, validInput())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := paymentService.Capture(ctx, "merchant", tx.ID, domain.Money{Currency: "IDR"}); !errors.Is(err, domain.ErrInvalidAmount) {
		t.Fatalf("zero capture error=%v", err)
	}
	if _, err := paymentService.Capture(ctx, "merchant", tx.ID, domain.Money{Amount: 1, Currency: "USD"}); !errors.Is(err, domain.ErrCurrencyMismatch) {
		t.Fatalf("currency mismatch capture error=%v", err)
	}
	tx, err = paymentService.Capture(ctx, "merchant", tx.ID, domain.Money{Amount: 500, Currency: "IDR"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := paymentService.Capture(ctx, "merchant", tx.ID, domain.Money{Amount: 1, Currency: "IDR"}); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("second capture error=%v", err)
	}
	if _, err := paymentService.Refund(ctx, "merchant", tx.ID, domain.Money{Currency: "IDR"}); !errors.Is(err, domain.ErrInvalidAmount) {
		t.Fatalf("zero refund error=%v", err)
	}
	if _, err := paymentService.Refund(ctx, "merchant", tx.ID, domain.Money{Amount: 501, Currency: "IDR"}); !errors.Is(err, domain.ErrInvalidAmount) {
		t.Fatalf("excess refund error=%v", err)
	}
	if _, err := paymentService.Refund(ctx, "merchant", tx.ID, domain.Money{Amount: 1, Currency: "USD"}); !errors.Is(err, domain.ErrCurrencyMismatch) {
		t.Fatalf("currency mismatch refund error=%v", err)
	}
}

func validInput() AuthorizeInput {
	return AuthorizeInput{
		MerchantID: "merchant",
		Amount:     domain.Money{Amount: 1000, Currency: "IDR"},
		PAN:        "4111111111111111",
		ExpMonth:   12,
		ExpYear:    time.Now().Year() + 2,
		CVV:        "123",
	}
}
