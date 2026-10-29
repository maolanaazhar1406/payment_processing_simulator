package domain

import (
	"strings"
	"unicode"
)

// MaskPAN masks a payment card number while retaining only its last four digits.
func MaskPAN(pan string) string {
	digits := make([]rune, 0, len(pan))
	for _, r := range pan {
		if unicode.IsDigit(r) {
			digits = append(digits, r)
		}
	}
	last4 := "****"
	if len(digits) >= 4 {
		last4 = string(digits[len(digits)-4:])
	}
	return strings.Repeat("*", 12) + last4
}

// ValidPAN reports whether pan is a 12-19 digit number that passes the Luhn check.
func ValidPAN(pan string) bool {
	if len(pan) < 12 || len(pan) > 19 {
		return false
	}
	sum := 0
	double := false
	for i := len(pan) - 1; i >= 0; i-- {
		if pan[i] < '0' || pan[i] > '9' {
			return false
		}
		digit := int(pan[i] - '0')
		if double {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
		double = !double
	}
	return sum%10 == 0
}

// CardBrand returns a recognized brand name or "unknown".
func CardBrand(pan string) string {
	switch {
	case strings.HasPrefix(pan, "4"):
		return "visa"
	case len(pan) >= 2 && pan[:2] >= "51" && pan[:2] <= "55":
		return "mastercard"
	case len(pan) >= 2 && (pan[:2] == "34" || pan[:2] == "37"):
		return "amex"
	case len(pan) >= 4 && pan[:4] >= "2221" && pan[:4] <= "2720":
		return "mastercard"
	default:
		return "unknown"
	}
}
