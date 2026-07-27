package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"layerone-scout/internal/fetcher"
)

type Config struct {
	App     AppConfig     `yaml:"app"`
	Server  ServerConfig  `yaml:"server"`
	Fetcher FetcherConfig `yaml:"fetcher"`
	Storage StorageConfig `yaml:"storage"`
	Log     LogConfig     `yaml:"log"`
}

type AppConfig struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

type ServerConfig struct {
	Addr         string `yaml:"addr"`
	Token        string `yaml:"token"`
	RateLimit    int    `yaml:"rate_limit"`
	ReadTimeout  string `yaml:"read_timeout"`
	WriteTimeout string `yaml:"write_timeout"`
	IdleTimeout  string `yaml:"idle_timeout"`
}

type FetcherConfig struct {
	Timeout      string `yaml:"timeout"`
	UserAgent    string `yaml:"user_agent"`
	MaxRetries   int    `yaml:"max_retries"`
	BackoffBase  string `yaml:"backoff_base"`
	MaxBodyBytes int64  `yaml:"max_body_bytes"`
}

type StorageConfig struct {
	Path      string `yaml:"path"`
	BackupDir string `yaml:"backup_dir"`
}

type LogConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

func Default() Config {
	return Config{
		App: AppConfig{Name: "LayerOne Scout", Version: "1.0.0"},
		Server: ServerConfig{
			Addr:         ":8787",
			Token:        "",
			RateLimit:    120,
			ReadTimeout:  "10s",
			WriteTimeout: "10s",
			IdleTimeout:  "30s",
		},
		Fetcher: FetcherConfig{
			Timeout:      "15s",
			UserAgent:    "LayerOneScout/1.0 (+analytics; public-profile-scraper)",
			MaxRetries:   3,
			BackoffBase:  "2s",
			MaxBodyBytes: 1 << 20,
		},
		Storage: StorageConfig{
			Path:      "profiles.json",
			BackupDir: "backups",
		},
		Log: LogConfig{
			Level:  "info",
			Format: "text",
		},
	}
}

func Load(path string, flags map[string]string) (Config, error) {
	cfg := Default()
	if path != "" {
		data, err := os.ReadFile(path)
		if err == nil {
			if err := yaml.Unmarshal(data, &cfg); err != nil {
				return cfg, err
			}
		}
	}
	// Env overrides
	envMap := map[string]string{
		"SCOUT_ADDR":       "server.addr",
		"SCOUT_TOKEN":      "server.token",
		"SCOUT_RATE_LIMIT": "server.rate_limit",
		"SCOUT_DB_PATH":    "storage.path",
		"SCOUT_LOG_LEVEL":  "log.level",
		"SCOUT_USER_AGENT": "fetcher.user_agent",
		"SCOUT_TIMEOUT":    "fetcher.timeout",
	}
	for env, field := range envMap {
		if val := os.Getenv(env); val != "" {
			if err := setField(&cfg, field, val); err != nil {
				return cfg, fmt.Errorf("error asignando %s: %w", env, err)
			}
		}
	}
	for flag, val := range flags {
		if err := setField(&cfg, flag, val); err != nil {
			return cfg, fmt.Errorf("error asignando flag %s: %w", flag, err)
		}
	}
	if err := validate(cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func (c Config) FetcherRuntime() fetcher.Config {
	return fetcher.Config{
		Timeout:      parseDuration(c.Fetcher.Timeout, 15*time.Second),
		UserAgent:    c.Fetcher.UserAgent,
		MaxRetries:   c.Fetcher.MaxRetries,
		BackoffBase:  parseDuration(c.Fetcher.BackoffBase, 2*time.Second),
		Jitter:       500 * time.Millisecond,
		MaxBodyBytes: c.Fetcher.MaxBodyBytes,
	}
}

func parseDuration(s string, def time.Duration) time.Duration {
	d, err := time.ParseDuration(strings.TrimSpace(s))
	if err != nil {
		return def
	}
	return d
}

func setField(cfg *Config, field, val string) error {
	parts := strings.Split(field, ".")
	if len(parts) < 2 {
		return fmt.Errorf("campo incompleto: %s", field)
	}
	switch parts[0] {
	case "server":
		switch parts[1] {
		case "addr":
			cfg.Server.Addr = val
		case "token":
			cfg.Server.Token = val
		case "rate_limit":
			n, err := strconv.Atoi(val)
			if err != nil {
				return err
			}
			cfg.Server.RateLimit = n
		case "read_timeout", "write_timeout", "idle_timeout":
			_, err := time.ParseDuration(val)
			if err != nil {
				return err
			}
			switch parts[1] {
			case "read_timeout":
				cfg.Server.ReadTimeout = val
			case "write_timeout":
				cfg.Server.WriteTimeout = val
			case "idle_timeout":
				cfg.Server.IdleTimeout = val
			}
		}
	case "storage":
		switch parts[1] {
		case "path":
			cfg.Storage.Path = val
		case "backup_dir":
			cfg.Storage.BackupDir = val
		}
	case "log":
		switch parts[1] {
		case "level":
			cfg.Log.Level = val
		case "format":
			cfg.Log.Format = val
		}
	case "fetcher":
		switch parts[1] {
		case "timeout":
			if _, err := time.ParseDuration(val); err != nil {
				return err
			}
			cfg.Fetcher.Timeout = val
		case "user_agent":
			cfg.Fetcher.UserAgent = val
		case "max_retries":
			n, err := strconv.Atoi(val)
			if err != nil {
				return err
			}
			cfg.Fetcher.MaxRetries = n
		case "backoff_base":
			if _, err := time.ParseDuration(val); err != nil {
				return err
			}
			cfg.Fetcher.BackoffBase = val
		case "max_body_bytes":
			n, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return err
			}
			cfg.Fetcher.MaxBodyBytes = n
		}
	default:
		return fmt.Errorf("campo desconocido: %s", field)
	}
	return nil
}

func validate(cfg Config) error {
	if cfg.Server.Addr == "" {
		return fmt.Errorf("server.addr no puede estar vacío")
	}
	if cfg.Storage.Path == "" {
		return fmt.Errorf("storage.path no puede estar vacío")
	}
	return nil
}
