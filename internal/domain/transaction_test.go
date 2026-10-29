package domain

import (
	"errors"
	"testing"
)

func TestTransitionTable(t *testing.T) {
	tests := []struct {
		name       string
		from       Status
		op         Operation
		code       string
		fullRefund bool
		want       Status
		wantErr    bool
	}{
		{"approved authorize", "", OperationAuthorize, "00", false, StatusAuthorized, false},
		{"declined authorize", "", OperationAuthorize, "05", false, StatusDeclined, false},
		{"timeout authorize", "", OperationAuthorize, "91", false, StatusDeclined, false},
		{"capture", StatusAuthorized, OperationCapture, "", false, StatusCaptured, false},
		{"void", StatusAuthorized, OperationVoid, "", false, StatusVoided, false},
		{"partial refund", StatusCaptured, OperationRefund, "", false, StatusPartiallyRefunded, false},
		{"full refund", StatusCaptured, OperationRefund, "", true, StatusRefunded, false},
		{"partial refund remaining", StatusPartiallyRefunded, OperationRefund, "", false, StatusPartiallyRefunded, false},
		{"complete refund remaining", StatusPartiallyRefunded, OperationRefund, "", true, StatusRefunded, false},
		{"capture after void", StatusVoided, OperationCapture, "", false, "", true},
		{"refund after decline", StatusDeclined, OperationRefund, "", false, "", true},
		{"second void", StatusVoided, OperationVoid, "", false, "", true},
		{"refund after refunded", StatusRefunded, OperationRefund, "", false, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Next(tt.from, tt.op, tt.code, tt.fullRefund)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Next() error=%v, wantErr=%t", err, tt.wantErr)
			}
			if tt.wantErr && !errors.Is(err, ErrInvalidTransition) {
				t.Fatalf("error=%v, want ErrInvalidTransition", err)
			}
			if got != tt.want {
				t.Fatalf("Next()=%q, want %q", got, tt.want)
			}
		})
	}
}

func TestCanTransition(t *testing.T) {
	if !CanTransition("", OperationAuthorize, "00", false) {
		t.Fatal("approved authorization should be allowed")
	}
	if CanTransition(StatusRefunded, OperationCapture, "", false) {
		t.Fatal("capture after refund should be rejected")
	}
}

func TestCardBrand(t *testing.T) {
	tests := []struct {
		pan  string
		want string
	}{
		{"4111111111111111", "visa"},
		{"5555555555554444", "mastercard"},
		{"371449635398431", "amex"},
		{"6011111111111117", "unknown"},
	}
	for _, tt := range tests {
		if got := CardBrand(tt.pan); got != tt.want {
			t.Errorf("CardBrand(%q)=%q, want %q", tt.pan, got, tt.want)
		}
	}
}
