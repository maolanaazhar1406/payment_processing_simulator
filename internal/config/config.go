package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config contains runtime settings loaded from environment variables.
type Config struct {
	Port             string
	SimulatorLatency time.Duration
	IdempotencyTTL   time.Duration
	Merchants        []string
	LogLevel         string
}

// Load reads and validates configuration from the process environment.
func Load() (Config, error) {
	port := envOrDefault("PORT", "8080")
	portValue, err := strconv.Atoi(port)
	if err != nil || portValue < 1 || portValue > 65535 {
		return Config{}, fmt.Errorf("PORT must be an integer between 1 and 65535")
	}
	latencyMS, err := envInt("SIM_LATENCY_MS", 50)
	if err != nil || latencyMS < 0 {
		return Config{}, fmt.Errorf("SIM_LATENCY_MS must be a non-negative integer")
	}
	ttlHours, err := envInt("IDEMPOTENCY_TTL_HOURS", 24)
	if err != nil || ttlHours < 1 {
		return Config{}, fmt.Errorf("IDEMPOTENCY_TTL_HOURS must be a positive integer")
	}
	merchants := splitNonEmpty(envOrDefault("MERCHANTS", "merchant_demo"))
	if len(merchants) == 0 {
		return Config{}, fmt.Errorf("MERCHANTS must contain at least one merchant")
	}
	logLevel := strings.ToLower(envOrDefault("LOG_LEVEL", "info"))
	if logLevel != "debug" && logLevel != "info" && logLevel != "warn" && logLevel != "error" {
		return Config{}, fmt.Errorf("LOG_LEVEL must be debug, info, warn, or error")
	}
	return Config{
		Port: port, SimulatorLatency: time.Duration(latencyMS) * time.Millisecond,
		IdempotencyTTL: time.Duration(ttlHours) * time.Hour, Merchants: merchants, LogLevel: logLevel,
	}, nil
}

func envOrDefault(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return fallback
}

func envInt(key string, fallback int) (int, error) {
	value := envOrDefault(key, strconv.Itoa(fallback))
	return strconv.Atoi(value)
}

func splitNonEmpty(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if merchant := strings.TrimSpace(part); merchant != "" {
			result = append(result, merchant)
		}
	}
	return result
}
