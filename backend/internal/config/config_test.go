package config

import "testing"

func TestApplyDefaults_LogFormatDefaultsToJSON(t *testing.T) {
	var cfg Config
	applyDefaults(&cfg)
	if cfg.Log.Format != "json" {
		t.Fatalf("expected log.format default json, got %q", cfg.Log.Format)
	}
}

func TestValidateLog_ValidValues(t *testing.T) {
	tests := []struct {
		name   string
		level  string
		format string
	}{
		{name: "json info", level: "info", format: "json"},
		{name: "development debug", level: "debug", format: "development"},
		{name: "warning alias", level: "warning", format: "json"},
		{name: "empty level allowed", level: "", format: "json"},
		{name: "trim lower normalize", level: " WARN ", format: " Development "},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Config{}
			cfg.Log.Level = tc.level
			cfg.Log.Format = tc.format
			if err := validateLog(&cfg); err != nil {
				t.Fatalf("validateLog returned error: %v", err)
			}
		})
	}
}

func TestValidateLog_InvalidFormat(t *testing.T) {
	cfg := Config{}
	cfg.Log.Level = "info"
	cfg.Log.Format = "pretty"

	err := validateLog(&cfg)
	if err == nil {
		t.Fatalf("expected error for invalid format")
	}
	if err.Error() != "log.format must be one of: json, development" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateLog_InvalidLevel(t *testing.T) {
	cfg := Config{}
	cfg.Log.Level = "trace"
	cfg.Log.Format = "json"

	err := validateLog(&cfg)
	if err == nil {
		t.Fatalf("expected error for invalid level")
	}
	if err.Error() != "log.level must be one of: debug, info, warn, warning, error" {
		t.Fatalf("unexpected error: %v", err)
	}
}
