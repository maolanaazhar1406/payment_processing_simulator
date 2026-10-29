package service

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"payment-processing-simulator/internal/domain"
	"payment-processing-simulator/internal/simulator"
)

// TransactionRepository is the transaction persistence contract used by Service.
type TransactionRepository interface {
	Create(context.Context, domain.Transaction) error
	Get(context.Context, string) (domain.Transaction, error)
	Update(context.Context, domain.Transaction, int64) (domain.Transaction, error)
}

// Issuer is the simulated issuer contract used by Service.
type Issuer interface {
	Authorize(context.Context, string, bool) (simulator.Response, error)
}

// Service implements payment transaction use cases.
type Service struct {
	transactions TransactionRepository
	issuer       Issuer
	now          func() time.Time
}

// New creates a service with its required dependencies.
func New(transactions TransactionRepository, issuer Issuer) *Service {
	return &Service{transactions: transactions, issuer: issuer, now: time.Now}
}

// AuthorizeInput contains transient card data used only during authorization.
type AuthorizeInput struct {
	MerchantID string
	Amount     domain.Money
	PAN        string
	ExpMonth   int
	ExpYear    int
	CVV        string
}

// Authorize validates input, simulates issuer authorization, and persists the result.
func (s *Service) Authorize(ctx context.Context, input AuthorizeInput) (domain.Transaction, error) {
	if strings.TrimSpace(input.MerchantID) == "" || input.Amount.Validate() != nil || input.Amount.Amount <= 0 {
		return domain.Transaction{}, domain.ErrInvalidInput
	}
	if len(input.PAN) < 12 || len(input.PAN) > 19 || !asciiDigits(input.PAN) ||
		input.ExpMonth < 1 || input.ExpMonth > 12 || input.ExpYear < 2000 ||
		len(input.CVV) < 3 || len(input.CVV) > 4 || !asciiDigits(input.CVV) {
		return domain.Transaction{}, domain.ErrInvalidInput
	}
	now := s.now().UTC()
	responseCode := ""
	expired := input.ExpYear < now.Year() || (input.ExpYear == now.Year() && input.ExpMonth < int(now.Month()))
	if expired {
		responseCode = "54"
	} else {
		response, err := s.issuer.Authorize(ctx, input.PAN[len(input.PAN)-4:], domain.ValidPAN(input.PAN))
		if err != nil {
			return domain.Transaction{}, fmt.Errorf("authorize with issuer simulator: %w", err)
		}
		responseCode = response.Code
	}
	status, err := domain.Next("", domain.OperationAuthorize, responseCode, false)
	if err != nil {
		return domain.Transaction{}, err
	}
	last4 := input.PAN[len(input.PAN)-4:]
	tx := domain.Transaction{
		ID: uuid.NewString(), Type: "AUTHORIZATION", Status: status,
		Amount:         input.Amount,
		CapturedAmount: domain.Money{Currency: input.Amount.Currency},
		RefundedAmount: domain.Money{Currency: input.Amount.Currency},
		Card:           domain.CardInfo{Brand: domain.CardBrand(input.PAN), Last4: last4, ExpMonth: input.ExpMonth, ExpYear: input.ExpYear},
		MerchantID:     input.MerchantID, ResponseCode: responseCode, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.transactions.Create(ctx, tx); err != nil {
		return domain.Transaction{}, fmt.Errorf("save authorized transaction: %w", err)
	}
	return tx, nil
}

// Capture captures an authorized transaction in full or in part.
func (s *Service) Capture(ctx context.Context, merchantID, id string, amount domain.Money) (domain.Transaction, error) {
	tx, err := s.ownedTransaction(ctx, merchantID, id)
	if err != nil {
		return domain.Transaction{}, err
	}
	if tx.Status != domain.StatusAuthorized {
		return domain.Transaction{}, domain.ErrInvalidTransition
	}
	if err := validatePositiveAmount(amount, tx.Amount.Currency); err != nil {
		return domain.Transaction{}, err
	}
	if amount.Amount > tx.Amount.Amount {
		return domain.Transaction{}, domain.ErrInvalidAmount
	}
	next, err := domain.Next(tx.Status, domain.OperationCapture, "", false)
	if err != nil {
		return domain.Transaction{}, err
	}
	tx.Status, tx.CapturedAmount, tx.UpdatedAt = next, amount, s.now().UTC()
	return s.transactions.Update(ctx, tx, tx.Version)
}

// Void cancels an authorized transaction that has not been captured.
func (s *Service) Void(ctx context.Context, merchantID, id string) (domain.Transaction, error) {
	tx, err := s.ownedTransaction(ctx, merchantID, id)
	if err != nil {
		return domain.Transaction{}, err
	}
	next, err := domain.Next(tx.Status, domain.OperationVoid, "", false)
	if err != nil {
		return domain.Transaction{}, err
	}
	tx.Status, tx.UpdatedAt = next, s.now().UTC()
	return s.transactions.Update(ctx, tx, tx.Version)
}

// Refund refunds part or all of the amount captured for a transaction.
func (s *Service) Refund(ctx context.Context, merchantID, id string, amount domain.Money) (domain.Transaction, error) {
	tx, err := s.ownedTransaction(ctx, merchantID, id)
	if err != nil {
		return domain.Transaction{}, err
	}
	if tx.Status != domain.StatusCaptured && tx.Status != domain.StatusPartiallyRefunded {
		return domain.Transaction{}, domain.ErrInvalidTransition
	}
	if err := validatePositiveAmount(amount, tx.CapturedAmount.Currency); err != nil {
		return domain.Transaction{}, err
	}
	remaining, err := tx.CapturedAmount.Sub(tx.RefundedAmount)
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("calculate refundable balance: %w", err)
	}
	if amount.Amount > remaining.Amount {
		return domain.Transaction{}, domain.ErrInvalidAmount
	}
	refunded, err := tx.RefundedAmount.Add(amount)
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("add refunded amount: %w", err)
	}
	next, err := domain.Next(tx.Status, domain.OperationRefund, "", amount.Amount == remaining.Amount)
	if err != nil {
		return domain.Transaction{}, err
	}
	tx.Status, tx.RefundedAmount, tx.UpdatedAt = next, refunded, s.now().UTC()
	return s.transactions.Update(ctx, tx, tx.Version)
}

// GetTransaction retrieves a transaction only for its owning merchant.
func (s *Service) GetTransaction(ctx context.Context, merchantID, id string) (domain.Transaction, error) {
	return s.ownedTransaction(ctx, merchantID, id)
}

func (s *Service) ownedTransaction(ctx context.Context, merchantID, id string) (domain.Transaction, error) {
	tx, err := s.transactions.Get(ctx, id)
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("get transaction: %w", err)
	}
	if tx.MerchantID != merchantID {
		return domain.Transaction{}, domain.ErrForbidden
	}
	return tx, nil
}

func validatePositiveAmount(amount domain.Money, currency string) error {
	if err := amount.Validate(); err != nil || amount.Amount <= 0 {
		return domain.ErrInvalidAmount
	}
	if amount.Currency != currency {
		return domain.ErrCurrencyMismatch
	}
	return nil
}

func asciiDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, char := range value {
		if !unicode.IsDigit(char) || char > unicode.MaxASCII {
			return false
		}
	}
	return true
}
