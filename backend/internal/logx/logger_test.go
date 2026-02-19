package logx

import (
	"testing"

	"go.uber.org/zap/zapcore"
)

func TestNew_JSON(t *testing.T) {
	logger, err := New("info", "json")
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	_ = logger.Sync()
}

func TestNew_Development(t *testing.T) {
	logger, err := New("debug", "development")
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	_ = logger.Sync()
}

func TestParseLevelMappings(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  zapcore.Level
	}{
		{name: "warn", input: "warn", want: zapcore.WarnLevel},
		{name: "warning", input: "warning", want: zapcore.WarnLevel},
		{name: "error", input: "error", want: zapcore.ErrorLevel},
		{name: "default", input: "unknown", want: zapcore.InfoLevel},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseLevel(tc.input)
			if got != tc.want {
				t.Fatalf("parseLevel(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}
