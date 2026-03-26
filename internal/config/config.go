// Package config loads application configuration from an optional YAML file plus environment variables.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/ilyakaznacheev/cleanenv"
)

const (
	defaultConfigPath                   = "config.yaml"
	defaultPostgresConnLifetimeMinutes  = 5
	defaultPostgresPoolMaxOpen          = 25
	defaultPostgresPoolMaxIdle          = 5
	defaultKafkaProcessTimeoutMs        = 5000
	defaultKafkaRetryBaseDelayMs        = 100
	defaultKafkaRetryJitterMs           = 300
	defaultKafkaMaxProcessRetries       = 3
	defaultKafkaMetricsFlushSec         = 15
	defaultKafkaDLQTopicSuffix          = ".dlq"
	defaultKafkaShutdownDrainTimeoutSec = 60
	defaultKafkaBatchSize               = 100
	defaultKafkaBatchFlushIntervalSec   = 5
)

// Config aggregates all runtime settings for API, logging, Postgres, and Kafka.
type Config struct {
	App      `yaml:"app"`
	HTTP     `yaml:"http"`
	Log      `yaml:"logger"`
	Postgres `yaml:"postgres"`
	Kafka    `yaml:"kafka"`
}

// App identifies the running service instance (metadata only).
type App struct {
	Name    string `env-required:"true" yaml:"name"    env:"APP_NAME"`
	Version string `env-required:"true" yaml:"version" env:"APP_VERSION"`
}

// HTTP configures the API listen port and server timeouts.
type HTTP struct {
	Port                     string `env-default:"8080" yaml:"port" env:"HTTP_PORT"`
	ReadHeaderTimeoutSeconds int    `env-default:"5" yaml:"read_header_timeout_seconds" env:"HTTP_READ_HEADER_TIMEOUT_SECONDS"`
	ReadTimeoutSeconds       int    `env-default:"10" yaml:"read_timeout_seconds" env:"HTTP_READ_TIMEOUT_SECONDS"`
	WriteTimeoutSeconds      int    `env-default:"15" yaml:"write_timeout_seconds" env:"HTTP_WRITE_TIMEOUT_SECONDS"`
	IdleTimeoutSeconds       int    `env-default:"60" yaml:"idle_timeout_seconds" env:"HTTP_IDLE_TIMEOUT_SECONDS"`
}

// Log selects the global slog level (debug, info, warn, error).
type Log struct {
	Level string `env-default:"debug" yaml:"log_level" env:"LOG_LEVEL"`
}

// Postgres holds the DSN and connection pool tuning knobs.
type Postgres struct {
	PoolMaxOpen            int    `yaml:"pool_max_open" env:"PG_POOL_MAX_OPEN"`
	PoolMaxIdle            int    `yaml:"pool_max_idle" env:"PG_POOL_MAX_IDLE"`
	PoolMaxLegacy          int    `yaml:"pool_max" env:"PG_POOL_MAX"`
	ConnMaxLifetimeMinutes int    `yaml:"conn_max_lifetime_minutes" env:"PG_CONN_MAX_LIFETIME_MINUTES"`
	URL                    string `env-required:"true" env:"PG_URL"`
}

// Kafka configures brokers, consumer group, retry timing, and per-partition batch flush.
type Kafka struct {
	Brokers                 []string `yaml:"brokers" env-required:"true" env:"KAFKA_BROKERS"`
	Topic                   string   `yaml:"topic" env-required:"true" env:"KAFKA_TOPIC"`
	GroupID                 string   `yaml:"group_id" env-default:"processor-group" env:"KAFKA_GROUP_ID"`
	ProcessTimeoutMs        int      `yaml:"process_timeout_ms" env-default:"5000" env:"KAFKA_PROCESS_TIMEOUT_MS"`
	RetryBaseDelayMs        int      `yaml:"retry_base_delay_ms" env-default:"100" env:"KAFKA_RETRY_BASE_DELAY_MS"`
	RetryJitterMs           int      `yaml:"retry_jitter_ms" env-default:"300" env:"KAFKA_RETRY_JITTER_MS"`
	MaxProcessRetries       int      `yaml:"max_process_retries" env-default:"3" env:"KAFKA_MAX_PROCESS_RETRIES"`
	DLQTopic                string   `yaml:"dlq_topic" env:"KAFKA_DLQ_TOPIC"`
	MetricsFlushSec         int      `yaml:"metrics_flush_sec" env-default:"15" env:"KAFKA_METRICS_FLUSH_SEC"`
	ShutdownDrainTimeoutSec int      `yaml:"shutdown_drain_timeout_sec" env-default:"60" env:"KAFKA_SHUTDOWN_DRAIN_TIMEOUT_SEC"`
	BatchSize               int      `yaml:"batch_size" env-default:"100" env:"KAFKA_BATCH_SIZE"`
	BatchFlushIntervalSec   int      `yaml:"batch_flush_interval_sec" env-default:"5" env:"KAFKA_BATCH_FLUSH_INTERVAL_SEC"`
}

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

// loadConfig reads optional CONFIG_PATH file, merges env (cleanenv), and applies safe defaults.
func loadConfig() (*Config, error) {
	cfg := &Config{}

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = defaultConfigPath
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
		cfg.Postgres.ConnMaxLifetimeMinutes = defaultPostgresConnLifetimeMinutes
	}
	if cfg.Postgres.PoolMaxOpen <= 0 {
		if cfg.Postgres.PoolMaxLegacy > 0 {
			cfg.Postgres.PoolMaxOpen = cfg.Postgres.PoolMaxLegacy
		} else {
			cfg.Postgres.PoolMaxOpen = defaultPostgresPoolMaxOpen
		}
	}
	if cfg.Postgres.PoolMaxIdle <= 0 {
		cfg.Postgres.PoolMaxIdle = defaultPostgresPoolMaxIdle
	}
	if cfg.Kafka.ProcessTimeoutMs <= 0 {
		cfg.Kafka.ProcessTimeoutMs = defaultKafkaProcessTimeoutMs
	}
	if cfg.Kafka.RetryBaseDelayMs <= 0 {
		cfg.Kafka.RetryBaseDelayMs = defaultKafkaRetryBaseDelayMs
	}
	if cfg.Kafka.RetryJitterMs <= 0 {
		cfg.Kafka.RetryJitterMs = defaultKafkaRetryJitterMs
	}
	if cfg.Kafka.MaxProcessRetries < 0 {
		cfg.Kafka.MaxProcessRetries = defaultKafkaMaxProcessRetries
	}
	if cfg.Kafka.MetricsFlushSec <= 0 {
		cfg.Kafka.MetricsFlushSec = defaultKafkaMetricsFlushSec
	}
	if cfg.Kafka.ShutdownDrainTimeoutSec <= 0 {
		cfg.Kafka.ShutdownDrainTimeoutSec = defaultKafkaShutdownDrainTimeoutSec
	}
	if cfg.Kafka.DLQTopic == "" && cfg.Kafka.Topic != "" {
		cfg.Kafka.DLQTopic = cfg.Kafka.Topic + defaultKafkaDLQTopicSuffix
	}
	if cfg.Kafka.BatchSize <= 0 {
		cfg.Kafka.BatchSize = defaultKafkaBatchSize
	}
	if cfg.Kafka.BatchFlushIntervalSec <= 0 {
		cfg.Kafka.BatchFlushIntervalSec = defaultKafkaBatchFlushIntervalSec
	}

	return cfg, nil
}
