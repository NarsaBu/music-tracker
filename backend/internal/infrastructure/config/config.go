// Package config loads application configuration from environment variables.
package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	HTTP        HTTP
	DatabaseURL string `env:"DATABASE_URL,required,notEmpty"`
	// MigrateOnStart applies pending migrations when `serve` starts.
	MigrateOnStart bool   `env:"MIGRATE_ON_START" envDefault:"true"`
	PublicBaseURL  string `env:"PUBLIC_BASE_URL" envDefault:"http://localhost:8080"`
	// StaticDir is the built SPA served on non-API paths; empty = API only (dev uses Vite).
	StaticDir   string `env:"STATIC_DIR"`
	Log         Log
	MusicBrainz MusicBrainz
	Sync        Sync
	SMTP        SMTP
	Worker      Worker
}

type HTTP struct {
	Addr            string        `env:"HTTP_ADDR" envDefault:":8080"`
	ReadTimeout     time.Duration `env:"HTTP_READ_TIMEOUT" envDefault:"10s"`
	WriteTimeout    time.Duration `env:"HTTP_WRITE_TIMEOUT" envDefault:"30s"`
	ShutdownTimeout time.Duration `env:"HTTP_SHUTDOWN_TIMEOUT" envDefault:"15s"`
}

type Log struct {
	Level  string `env:"LOG_LEVEL" envDefault:"info"`
	Format string `env:"LOG_FORMAT" envDefault:"json"` // json | text
}

type MusicBrainz struct {
	BaseURL   string  `env:"MB_BASE_URL" envDefault:"https://musicbrainz.org/ws/2"`
	UserAgent string  `env:"MB_USER_AGENT,required,notEmpty"`
	RateLimit float64 `env:"MB_RATE_LIMIT" envDefault:"1"`
}

type Sync struct {
	Cron string `env:"SYNC_CRON" envDefault:"0 6 * * *"`
}

type SMTP struct {
	Host     string `env:"SMTP_HOST"`
	Port     int    `env:"SMTP_PORT" envDefault:"587"`
	User     string `env:"SMTP_USER"`
	Password string `env:"SMTP_PASSWORD"`
	From     string `env:"SMTP_FROM"`
}

type Worker struct {
	Concurrency int `env:"WORKER_CONCURRENCY" envDefault:"4"`
}

// Load parses the environment and validates cross-field constraints.
func Load() (Config, error) {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return Config{}, fmt.Errorf("parse env: %w", err)
	}
	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) validate() error {
	if c.MusicBrainz.RateLimit <= 0 || c.MusicBrainz.RateLimit > 1 {
		return fmt.Errorf("MB_RATE_LIMIT must be in (0, 1], MusicBrainz allows 1 request per second")
	}
	if c.Log.Format != "json" && c.Log.Format != "text" {
		return fmt.Errorf("LOG_FORMAT must be json or text")
	}
	return nil
}
