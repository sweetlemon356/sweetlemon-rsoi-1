package config_test

import (
	"testing"
	"time"

	"github.com/sweetlemon356/sweetlemon-rsoi-1/src/internal/config"
)

func TestLoadUsesOperationalDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	for _, name := range []string{
		"PORT",
		"DB_MAX_CONNECTIONS",
		"DB_CONNECT_TIMEOUT",
		"HTTP_READ_HEADER_TIMEOUT",
		"HTTP_READ_TIMEOUT",
		"HTTP_WRITE_TIMEOUT",
		"HTTP_IDLE_TIMEOUT",
		"HTTP_SHUTDOWN_TIMEOUT",
	} {
		t.Setenv(name, "")
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load configuration: %v", err)
	}
	if cfg.HTTP.Address != ":8080" {
		t.Fatalf("address: got %q, want :8080", cfg.HTTP.Address)
	}
	if cfg.Database.MaxConnections != 10 {
		t.Fatalf("max connections: got %d, want 10", cfg.Database.MaxConnections)
	}
	if cfg.Database.ConnectTimeout != 5*time.Second {
		t.Fatalf("connect timeout: got %s, want 5s", cfg.Database.ConnectTimeout)
	}
}

func TestLoadRejectsInvalidPort(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("PORT", "70000")

	if _, err := config.Load(); err == nil {
		t.Fatal("expected invalid port error")
	}
}

func TestLoadRejectsInvalidOperationalValues(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "DB_MAX_CONNECTIONS", value: "2147483648"},
		{name: "DB_CONNECT_TIMEOUT", value: "0s"},
		{name: "HTTP_READ_TIMEOUT", value: "-1s"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("DATABASE_URL", "postgres://example")
			t.Setenv(test.name, test.value)
			if _, err := config.Load(); err == nil {
				t.Fatalf("expected error for %s=%s", test.name, test.value)
			}
		})
	}
}
