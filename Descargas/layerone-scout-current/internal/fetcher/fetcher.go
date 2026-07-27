package fetcher

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"layerone-scout/internal/model"
)

type Fetcher interface {
	Fetch(ctx context.Context, username string) (model.Person, error)
	Platform() string
}

var fetchers = map[string]Fetcher{}

func Register(f Fetcher) {
	fetchers[strings.ToLower(f.Platform())] = f
}

func FetchByPlatform(ctx context.Context, platform, username string) (model.Person, error) {
	f, ok := fetchers[strings.ToLower(strings.TrimSpace(platform))]
	if !ok {
		return model.Person{}, fmt.Errorf("plataforma no soportada: %s", platform)
	}
	return f.Fetch(ctx, username)
}

func Init(cfg Config) {
	fetchers = map[string]Fetcher{}
	Register(NewInstagramFetcher(cfg))
	Register(NewTwitterFetcher(cfg))
	if os.Getenv("SCOUT_MOCK") == "1" {
		Register(NewMockFetcher())
	}
}

type Config struct {
	Timeout      time.Duration
	UserAgent    string
	MaxRetries   int
	BackoffBase  time.Duration
	Jitter       time.Duration
	MaxBodyBytes int64
}

func DefaultConfig() Config {
	return Config{
		Timeout:      15 * time.Second,
		UserAgent:    "LayerOneScout/1.0 (+analytics; public-profile-scraper)",
		MaxRetries:   3,
		BackoffBase:  2 * time.Second,
		Jitter:       500 * time.Millisecond,
		MaxBodyBytes: 1 << 20,
	}
}
