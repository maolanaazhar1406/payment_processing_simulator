package config

import (
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	for _, key := range []string{"PORT", "SIM_LATENCY_MS", "IDEMPOTENCY_TTL_HOURS", "MERCHANTS", "LOG_LEVEL"} {
		t.Setenv(key, "")
	}
	config, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if config.Port != "8080" || config.SimulatorLatency.Milliseconds() != 50 ||
		config.IdempotencyTTL.Hours() != 24 || len(config.Merchants) != 1 || config.LogLevel != "info" {
		t.Fatalf("unexpected defaults: %+v", config)
	}
}

func TestLoadRejectsInvalidLatency(t *testing.T) {
	t.Setenv("SIM_LATENCY_MS", "-1")
	if _, err := Load(); err == nil {
		t.Fatal("Load() accepted negative latency")
	}
}
