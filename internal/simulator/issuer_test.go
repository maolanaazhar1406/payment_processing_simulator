package simulator

import (
	"context"
	"testing"
)

func TestIssuerResponses(t *testing.T) {
	tests := []struct {
		last4 string
		valid bool
		want  string
	}{
		{"0000", true, "00"}, {"0005", true, "05"}, {"0051", true, "51"},
		{"0054", true, "54"}, {"0091", true, "91"}, {"1234", true, "00"},
		{"0000", false, "14"},
	}
	issuer := SimulatedIssuer{}
	for _, tt := range tests {
		response, err := issuer.Authorize(context.Background(), tt.last4, tt.valid)
		if err != nil || response.Code != tt.want {
			t.Errorf("Authorize(%q, %t)=(%q, %v), want %q", tt.last4, tt.valid, response.Code, err, tt.want)
		}
	}
}
