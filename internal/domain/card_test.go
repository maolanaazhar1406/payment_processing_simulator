package domain

import "testing"

func TestValidPAN(t *testing.T) {
	tests := []struct {
		pan  string
		want bool
	}{
		{"4111111111111111", true},
		{"5555555555554444", true},
		{"4111111111111112", false},
		{"123", false},
		{"12345678901234567890", false},
		{"4111 1111 1111 1111", false},
	}
	for _, tt := range tests {
		if got := ValidPAN(tt.pan); got != tt.want {
			t.Errorf("ValidPAN(%q)=%t, want %t", tt.pan, got, tt.want)
		}
	}
}

func TestMaskPAN(t *testing.T) {
	if got := MaskPAN("4111111111111111"); got != "************1111" {
		t.Fatalf("MaskPAN()=%q", got)
	}
	if got := MaskPAN("12"); got != "****************" {
		t.Fatalf("MaskPAN(short)=%q", got)
	}
}
