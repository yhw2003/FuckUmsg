package main

import (
	"testing"

	"chat-assist-backend/internal/config"
)

func TestResolveLogOptions_ConfigOverridesEnv(t *testing.T) {
	cfg := config.Config{}
	cfg.Log.Level = "debug"
	cfg.Log.Format = "development"

	level, format := resolveLogOptions(cfg, "error")
	if level != "debug" {
		t.Fatalf("expected level debug, got %q", level)
	}
	if format != "development" {
		t.Fatalf("expected format development, got %q", format)
	}
}

func TestResolveLogOptions_EnvFallback(t *testing.T) {
	cfg := config.Config{}
	cfg.Log.Format = "json"

	level, format := resolveLogOptions(cfg, "warn")
	if level != "warn" {
		t.Fatalf("expected level warn, got %q", level)
	}
	if format != "json" {
		t.Fatalf("expected format json, got %q", format)
	}
}

func TestResolveLogOptions_DefaultLevel(t *testing.T) {
	cfg := config.Config{}
	cfg.Log.Format = "json"

	level, format := resolveLogOptions(cfg, "")
	if level != "info" {
		t.Fatalf("expected level info, got %q", level)
	}
	if format != "json" {
		t.Fatalf("expected format json, got %q", format)
	}
}
