package config

import (
	"fmt"
	"os"
	"sync"

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
		Port string `env-default:"8080" yaml:"port" env:"HTTP_PORT"`
	}

	Log struct {
		Level string `env-default:"debug" yaml:"log_level" env:"LOG_LEVEL"`
	}

	Postgres struct {
		PoolMax int    `env-default:"2" yaml:"pool_max" env:"PG_POOL_MAX"`
		URL     string `env-required:"true"               env:"PG_URL"`
	}

	Kafka struct {
		Brokers []string `env-required:"true" env:"KAFKA_BROKERS"`
		Topic   string   `env-required:"true" env:"KAFKA_TOPIC"`
		GroupID string   `env-default:"processor-group" env:"KAFKA_GROUP_ID"`
	}
)

var (
	instance *Config
	once     sync.Once
)

// NewConfig читает конфиг из файла (если есть) и перекрывает его переменными окружения
func NewConfig() (*Config, error) {
	var err error
	once.Do(func() {
		instance = &Config{}
		
		// 1. Берем путь из переменной окружения или используем дефолт
		configPath := os.Getenv("CONFIG_PATH")
		if configPath == "" {
			configPath = "config.yaml" // Дефолт в корне
		}

		// Проверяем, существует ли файл
		if _, statErr := os.Stat(configPath); statErr == nil {
			// Файл есть, читаем его
			if readErr := cleanenv.ReadConfig(configPath, instance); readErr != nil {
				err = fmt.Errorf("read config file error: %w", readErr)
				return
			}
		} else {
			// Файла нет, сообщаем и идем дальше читать ENV
			fmt.Printf("Config file not found at %s, using defaults and ENV variables\n", configPath)
			if readErr := cleanenv.ReadEnv(instance); readErr != nil {
				err = fmt.Errorf("read env error: %w", readErr)
				return
			}
		}
	})

	if err != nil {
		return nil, err
	}

	return instance, nil
}
