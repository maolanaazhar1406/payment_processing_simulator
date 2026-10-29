package simulator

import (
	"context"
	"time"
)

// Response is a simulated issuer decision.
type Response struct {
	Code string
}

// Issuer resolves a test PAN into a simulated issuer response.
type Issuer interface {
	Authorize(context.Context, string, bool) (Response, error)
}

// SimulatedIssuer applies deterministic test-card rules and configurable latency.
type SimulatedIssuer struct {
	Latency time.Duration
}

// Authorize returns a response based on the PAN's last four digits.
func (s SimulatedIssuer) Authorize(ctx context.Context, last4 string, panValid bool) (Response, error) {
	if s.Latency > 0 {
		timer := time.NewTimer(s.Latency)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return Response{}, ctx.Err()
		case <-timer.C:
		}
	}
	if !panValid {
		return Response{Code: "14"}, nil
	}
	switch last4 {
	case "0005":
		return Response{Code: "05"}, nil
	case "0051":
		return Response{Code: "51"}, nil
	case "0054":
		return Response{Code: "54"}, nil
	case "0091":
		return Response{Code: "91"}, nil
	default:
		return Response{Code: "00"}, nil
	}
}

var _ Issuer = SimulatedIssuer{}
