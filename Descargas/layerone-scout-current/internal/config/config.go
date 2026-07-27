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
	App        AppConfig        `yaml:"app"`
	Server     ServerConfig     `yaml:"server"`
	Fetcher    FetcherConfig    `yaml:"fetcher"`
	Storage    StorageConfig    `yaml:"storage"`
	Log        LogConfig        `yaml:"log"`
	Behavioral BehavioralConfig `yaml:"behavioral"`
	Traits     TraitsConfig     `yaml:"traits"`
	Database   DatabaseConfig   `yaml:"database"`
	Math       MathConfig       `yaml:"math"`
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
	CacheDir     string `yaml:"cache_dir"`
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

type BehavioralConfig struct {
	Model BehavioralModelConfig `yaml:"model"`
}

type BehavioralModelConfig struct {
	Type               string  `yaml:"type"`
	EmbeddingsModel     string  `yaml:"embeddings_model"`
	PCAComponents      int     `yaml:"pca_components"`
	LDATopics          int     `yaml:"lda_topics"`
	HMMStates          int     `yaml:"hmm_states"`
	NumClusters        int     `yaml:"num_clusters"`
	LearningRate       float64 `yaml:"learning_rate"`
	EpochsFineTune     int     `yaml:"epochs_fine_tune"`
	UncertaintyDropout float64 `yaml:"uncertainty_dropout"`
}

type TraitsConfig struct {
	Observable []string `yaml:"observable"`
	Hidden     []string `yaml:"hidden"`
}

type DatabaseConfig struct {
	Driver         string `yaml:"driver"`
	Host           string `yaml:"host"`
	Port           int    `yaml:"port"`
	Name           string `yaml:"name"`
	MaxConnections int    `yaml:"max_connections"`
	QueryTimeout   string `yaml:"query_timeout"`
	CacheTTL       string `yaml:"cache_ttl"`
}

type MathConfig struct {
	MCMCSamples   int  `yaml:"mcmc_samples"`
	KalmanFilters bool `yaml:"kalman_filters"`
	CausalGraph   bool `yaml:"causal_graph"`
	NeuralFineTune bool `yaml:"neural_finetune"`
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
			CacheDir:     "/tmp/layerone_cache",
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
		Behavioral: BehavioralConfig{
			Model: BehavioralModelConfig{
				Type:               "transformer-mcmc-kmeans-kalman",
				EmbeddingsModel:     "sentence-transformer-all-MiniLM-L6-v2",
				PCAComponents:      32,
				LDATopics:          12,
				HMMStates:          8,
				NumClusters:        5,
				LearningRate:       0.001,
				EpochsFineTune:     5,
				UncertaintyDropout: 0.3,
			},
		},
		Traits: TraitsConfig{
			Observable: []string{"posting_frequency", "reply_rate", "mention_density", "emoji_ratio", "url_ratio", "mention_frequency", "hashtag_frequency", "weekday_pattern", "hour_pattern", "timezone", "network_size", "mutual_friends_ratio", "follower_growth_rate", "verification_status", "bio_sentiment", "link_quality_score", "tfidf_top_words", "bigfive_raw", "interests_clusters"},
			Hidden:     []string{"intelligence_score", "emotional_stability_index", "risk_tolerance_index", "social_power_index", "cultural_alignment_score", "causal_influence_score", "q_learning_value", "hmm_latent_state", "kurtosis_outlier_score"},
		},
		Database: DatabaseConfig{
			Driver:         "postgres",
			Host:           "",
			Port:           5432,
			Name:           "layerone_scout_v3",
			MaxConnections: 1000,
			QueryTimeout:   "30s",
			CacheTTL:       "7d",
		},
		Math: MathConfig{
			MCMCSamples:   10000,
			KalmanFilters: true,
			CausalGraph:   true,
			NeuralFineTune: true,
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
		"LAYERONE_TOKEN":   "server.token",
		"SCOUT_RATE_LIMIT": "server.rate_limit",
		"SCOUT_DB_PATH":    "storage.path",
		"SCOUT_LOG_LEVEL":  "log.level",
		"SCOUT_USER_AGENT": "fetcher.user_agent",
		"SCOUT_TIMEOUT":    "fetcher.timeout",
		"DB_HOST":          "database.host",
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
		case "cache_dir":
			cfg.Server.CacheDir = val
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
	case "database":
		switch parts[1] {
		case "driver":
			cfg.Database.Driver = val
		case "host":
			cfg.Database.Host = val
		case "port":
			n, err := strconv.Atoi(val)
			if err != nil {
				return err
			}
			cfg.Database.Port = n
		case "name":
			cfg.Database.Name = val
		case "max_connections":
			n, err := strconv.Atoi(val)
			if err != nil {
				return err
			}
			cfg.Database.MaxConnections = n
		case "query_timeout":
			if _, err := time.ParseDuration(val); err != nil {
				return err
			}
			cfg.Database.QueryTimeout = val
		case "cache_ttl":
			if _, err := time.ParseDuration(val); err != nil {
				return err
			}
			cfg.Database.CacheTTL = val
		}
	case "behavioral":
		if parts[1] == "model" && len(parts) == 3 {
			switch parts[2] {
			case "type":
				cfg.Behavioral.Model.Type = val
			case "embeddings_model":
				cfg.Behavioral.Model.EmbeddingsModel = val
			case "pca_components":
				n, err := strconv.Atoi(val)
				if err != nil {
					return err
				}
				cfg.Behavioral.Model.PCAComponents = n
			case "lda_topics":
				n, err := strconv.Atoi(val)
				if err != nil {
					return err
				}
				cfg.Behavioral.Model.LDATopics = n
			case "hmm_states":
				n, err := strconv.Atoi(val)
				if err != nil {
					return err
				}
				cfg.Behavioral.Model.HMMStates = n
			case "num_clusters":
				n, err := strconv.Atoi(val)
				if err != nil {
					return err
				}
				cfg.Behavioral.Model.NumClusters = n
			case "learning_rate":
				f, err := strconv.ParseFloat(val, 64)
				if err != nil {
					return err
				}
				cfg.Behavioral.Model.LearningRate = f
			case "epochs_fine_tune":
				n, err := strconv.Atoi(val)
				if err != nil {
					return err
				}
				cfg.Behavioral.Model.EpochsFineTune = n
			case "uncertainty_dropout":
				f, err := strconv.ParseFloat(val, 64)
				if err != nil {
					return err
				}
				cfg.Behavioral.Model.UncertaintyDropout = f
			}
		}
	case "math":
		switch parts[1] {
		case "mcmc_samples":
			n, err := strconv.Atoi(val)
			if err != nil {
				return err
			}
			cfg.Math.MCMCSamples = n
		case "kalman_filters":
			cfg.Math.KalmanFilters = val == "true"
		case "causal_graph":
			cfg.Math.CausalGraph = val == "true"
		case "neural_finetune":
			cfg.Math.NeuralFineTune = val == "true"
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
