package config

import (
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type (
	Config struct {
		App      `yaml:"app"`
		HTTP     `yaml:"http"`
		Log      `yaml:"logger"`
		Postgres `yaml:"postgres"`
		Kafka    `yaml:"kafka"`
	}

	App struct {
		Name    string `env-required:"true" yaml:"name"    env:"APP_NAME"`
		Version string `env-required:"true" yaml:"version" env:"APP_VERSION"`
	}

	HTTP struct {
		Port                     string `env-default:"8080" yaml:"port" env:"HTTP_PORT"`
		ReadHeaderTimeoutSeconds int    `env-default:"5" yaml:"read_header_timeout_seconds" env:"HTTP_READ_HEADER_TIMEOUT_SECONDS"`
		ReadTimeoutSeconds       int    `env-default:"10" yaml:"read_timeout_seconds" env:"HTTP_READ_TIMEOUT_SECONDS"`
		WriteTimeoutSeconds      int    `env-default:"15" yaml:"write_timeout_seconds" env:"HTTP_WRITE_TIMEOUT_SECONDS"`
		IdleTimeoutSeconds       int    `env-default:"60" yaml:"idle_timeout_seconds" env:"HTTP_IDLE_TIMEOUT_SECONDS"`
	}

	Log struct {
		Level string `env-default:"debug" yaml:"log_level" env:"LOG_LEVEL"`
	}

	Postgres struct {
		PoolMaxOpen            int    `yaml:"pool_max_open" env:"PG_POOL_MAX_OPEN"`
		PoolMaxIdle            int    `yaml:"pool_max_idle" env:"PG_POOL_MAX_IDLE"`
		PoolMaxLegacy          int    `yaml:"pool_max" env:"PG_POOL_MAX"`
		ConnMaxLifetimeMinutes int    `yaml:"conn_max_lifetime_minutes" env:"PG_CONN_MAX_LIFETIME_MINUTES"`
		URL                    string `env-required:"true" env:"PG_URL"`
	}

	Kafka struct {
		Brokers []string `env-required:"true" env:"KAFKA_BROKERS"`
		Topic   string   `env-required:"true" env:"KAFKA_TOPIC"`
		GroupID string   `env-default:"processor-group" env:"KAFKA_GROUP_ID"`
	}
)

var (
	instance *Config
	mu       sync.Mutex
)

// NewConfig reads config from file (if exists) and then applies environment overrides.
func NewConfig() (*Config, error) {
	mu.Lock()
	defer mu.Unlock()

	if instance != nil {
		return instance, nil
	}

	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}
	instance = cfg

	return instance, nil
}

// ResetConfig is for testing purposes only, to reset the singleton config instance.
func ResetConfig() {
	mu.Lock()
	defer mu.Unlock()
	instance = nil
}

func loadConfig() (*Config, error) {
	cfg := &Config{}

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}

	if _, statErr := os.Stat(configPath); statErr == nil {
		if readErr := cleanenv.ReadConfig(configPath, cfg); readErr != nil {
			return nil, fmt.Errorf("read config file error: %w", readErr)
		}
	} else {
		slog.Info("Config file not found, using defaults and environment variables", "path", configPath)
	}

	if readErr := cleanenv.ReadEnv(cfg); readErr != nil {
		return nil, fmt.Errorf("read env error: %w", readErr)
	}

	if cfg.Postgres.ConnMaxLifetimeMinutes <= 0 {
		cfg.Postgres.ConnMaxLifetimeMinutes = int((5 * time.Minute) / time.Minute)
	}
	if cfg.Postgres.PoolMaxOpen <= 0 {
		if cfg.Postgres.PoolMaxLegacy > 0 {
			cfg.Postgres.PoolMaxOpen = cfg.Postgres.PoolMaxLegacy
		} else {
			cfg.Postgres.PoolMaxOpen = 25
		}
	}
	if cfg.Postgres.PoolMaxIdle <= 0 {
		cfg.Postgres.PoolMaxIdle = 5
	}

	return cfg, nil
}
