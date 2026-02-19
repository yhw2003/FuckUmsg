package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Server struct {
		Port            int    `toml:"port"`
		Password        string `toml:"password"`
		StaticDir       string `toml:"static_dir"`
		TokenTTLMinutes int    `toml:"token_ttl_minutes"`
	} `toml:"server"`
	OneBot struct {
		SSEURL      string `toml:"sse_url"`
		APIURL      string `toml:"api_url"`
		AccessToken string `toml:"access_token"`
	} `toml:"onebot"`
	SelfID int64 `toml:"self_id"`
	OpenAI struct {
		BaseURL        string `toml:"base_url"`
		APIKey         string `toml:"api_key"`
		Model          string `toml:"model"`
		TimeoutSeconds int    `toml:"timeout_seconds"`
	} `toml:"openai"`
	Storage struct {
		SQLitePath string `toml:"sqlite_path"`
	} `toml:"storage"`
}

func LoadConfig(path string) (Config, error) {
	var cfg Config
	if strings.TrimSpace(path) == "" {
		return cfg, errors.New("config path is empty")
	}
	if _, err := os.Stat(path); err != nil {
		return cfg, err
	}
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return cfg, err
	}
	applyDefaults(&cfg)
	return cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if strings.TrimSpace(cfg.Server.StaticDir) == "" {
		cfg.Server.StaticDir = filepath.FromSlash("../frontend/dist")
	}
	if cfg.Server.TokenTTLMinutes == 0 {
		cfg.Server.TokenTTLMinutes = int((24 * time.Hour).Minutes())
	}
	if strings.TrimSpace(cfg.OpenAI.BaseURL) == "" {
		cfg.OpenAI.BaseURL = "https://api.openai.com/v1"
	}
	if cfg.OpenAI.TimeoutSeconds == 0 {
		cfg.OpenAI.TimeoutSeconds = 30
	}
	if strings.TrimSpace(cfg.Storage.SQLitePath) == "" {
		cfg.Storage.SQLitePath = "./data.db"
	}
}

func ResolveConfigPath(arg string) string {
	if strings.TrimSpace(arg) != "" {
		return arg
	}
	candidates := []string{"config.toml", filepath.FromSlash("backend/config.toml")}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}
