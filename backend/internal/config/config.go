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
		StaticEmbed     bool   `toml:"static_embed"`
		TokenTTLMinutes int    `toml:"token_ttl_minutes"`
	} `toml:"server"`
	OneBot struct {
		SSEURL      string `toml:"sse_url"`
		APIURL      string `toml:"api_url"`
		AccessToken string `toml:"access_token"`
	} `toml:"onebot"`
	OpenAI struct {
		BaseURL        string `toml:"base_url"`
		APIKey         string `toml:"api_key"`
		Model          string `toml:"model"`
		TimeoutSeconds int    `toml:"timeout_seconds"`
	} `toml:"openai"`
	Calendar struct {
		WeekMode    string `toml:"week_mode"`
		Week1Monday string `toml:"week1_monday"`
	} `toml:"calendar"`
	Storage struct {
		SQLitePath string `toml:"sqlite_path"`
	} `toml:"storage"`
	Log struct {
		Level  string `toml:"level"`
		Format string `toml:"format"`
	} `toml:"log"`
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
	if err := validateCalendar(&cfg); err != nil {
		return cfg, err
	}
	if err := validateLog(&cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if strings.TrimSpace(cfg.Server.StaticDir) == "" {
		cfg.Server.StaticDir = filepath.FromSlash("internal/server/web_dist")
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
	if strings.TrimSpace(cfg.Calendar.WeekMode) == "" {
		cfg.Calendar.WeekMode = "academic"
	}
	if strings.TrimSpace(cfg.Storage.SQLitePath) == "" {
		cfg.Storage.SQLitePath = "./data.db"
	}
	if strings.TrimSpace(cfg.Log.Format) == "" {
		cfg.Log.Format = "json"
	}
}

func validateCalendar(cfg *Config) error {
	weekMode := strings.ToLower(strings.TrimSpace(cfg.Calendar.WeekMode))
	if weekMode != "academic" && weekMode != "natural" {
		return errors.New("calendar.week_mode must be one of: academic, natural")
	}

	week1Monday := strings.TrimSpace(cfg.Calendar.Week1Monday)
	if weekMode == "academic" {
		if week1Monday == "" {
			return errors.New("calendar.week1_monday is required when calendar.week_mode=academic, format: YYYY-MM-DD")
		}
	}
	if week1Monday == "" {
		return nil
	}

	date, err := time.Parse("2006-01-02", week1Monday)
	if err != nil {
		return errors.New("calendar.week1_monday must be in YYYY-MM-DD format")
	}
	if date.Weekday() != time.Monday {
		return errors.New("calendar.week1_monday must be a Monday")
	}
	return nil
}

func validateLog(cfg *Config) error {
	format := strings.ToLower(strings.TrimSpace(cfg.Log.Format))
	switch format {
	case "json", "development":
	default:
		return errors.New("log.format must be one of: json, development")
	}

	level := strings.ToLower(strings.TrimSpace(cfg.Log.Level))
	switch level {
	case "", "debug", "info", "warn", "warning", "error":
		return nil
	default:
		return errors.New("log.level must be one of: debug, info, warn, warning, error")
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
