package domain

import (
	"errors"
	"time"
)

var (
	ErrInvalidTransition = errors.New("invalid transaction transition")
	ErrNotFound          = errors.New("transaction not found")
	ErrVersionConflict   = errors.New("transaction version conflict")
	ErrForbidden         = errors.New("merchant does not own transaction")
	ErrInvalidInput      = errors.New("invalid input")
	ErrInvalidAmount     = errors.New("invalid transaction amount")
)

type Status string

const (
	StatusAuthorized        Status = "AUTHORIZED"
	StatusCaptured          Status = "CAPTURED"
	StatusPartiallyRefunded Status = "PARTIALLY_REFUNDED"
	StatusRefunded          Status = "REFUNDED"
	StatusVoided            Status = "VOIDED"
	StatusDeclined          Status = "DECLINED"
)

type Operation string

const (
	OperationAuthorize Operation = "authorize"
	OperationCapture   Operation = "capture"
	OperationVoid      Operation = "void"
	OperationRefund    Operation = "refund"
)

// CardInfo contains only non-sensitive card metadata.
type CardInfo struct {
	Brand    string `json:"brand"`
	Last4    string `json:"last4"`
	ExpMonth int    `json:"exp_month"`
	ExpYear  int    `json:"exp_year"`
}

// Transaction is a safe, persisted representation of a payment transaction.
type Transaction struct {
	ID             string    `json:"id"`
	Type           string    `json:"type"`
	Status         Status    `json:"status"`
	Amount         Money     `json:"amount"`
	CapturedAmount Money     `json:"captured_amount"`
	RefundedAmount Money     `json:"refunded_amount"`
	Card           CardInfo  `json:"card"`
	MerchantID     string    `json:"merchant_id"`
	ResponseCode   string    `json:"response_code"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	Version        int64     `json:"-"`
}

// CanTransition reports whether an operation is valid for the current status.
func CanTransition(from Status, operation Operation, responseCode string, completesRefund bool) bool {
	_, err := Next(from, operation, responseCode, completesRefund)
	return err == nil
}

// Next returns the status resulting from a valid operation.
func Next(from Status, operation Operation, responseCode string, completesRefund bool) (Status, error) {
	switch {
	case from == "" && operation == OperationAuthorize:
		if responseCode == "00" {
			return StatusAuthorized, nil
		}
		return StatusDeclined, nil
	case from == StatusAuthorized && operation == OperationCapture:
		return StatusCaptured, nil
	case from == StatusAuthorized && operation == OperationVoid:
		return StatusVoided, nil
	case from == StatusCaptured && operation == OperationRefund:
		if completesRefund {
			return StatusRefunded, nil
		}
		return StatusPartiallyRefunded, nil
	case from == StatusPartiallyRefunded && operation == OperationRefund:
		if completesRefund {
			return StatusRefunded, nil
		}
		return StatusPartiallyRefunded, nil
	default:
		return "", ErrInvalidTransition
	}
}
