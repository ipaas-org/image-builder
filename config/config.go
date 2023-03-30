package config

import (
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type (
	Config struct {
		App      `yaml:"app"`
		Log      `yaml:"logger"`
		RMQ      `yaml:"rabbitmq"`
		Database `yaml:"database"`
		Services `yaml:"services"`
	}

	App struct {
		Name    string `env-required:"true" yaml:"name"    env:"APP_NAME"`
		Version string `env-required:"true" yaml:"version" env:"APP_VERSION"`
	}

	Log struct {
		Level string `env-required:"true" yaml:"level" env:"LOG_LEVEL"`
		Type  string `env-required:"true" yaml:"type"  env:"LOG_TYPE"`
	}

	RMQ struct {
		URI           string `env-required:"true" yaml:"uri" env:"RABBITMQ_URI"`
		ExchangeQueue string `env-required:"true" yaml:"exchangeQueue" env:"RABBITMQ_EXCHANGE_QUEUE"`
	}

	Database struct {
		Driver string `env-required:"true"  yaml:"driver" env:"DATABASE_DRIVER"`
		URI    string `                                   env:"DATABASE_URI"`
	}

	Services struct {
		Connectors []Connector `yaml:"connector,flow"`
		Builders   []Builder   `yaml:"builders,flow"`
	}

	Connector struct {
		Name              string `yaml:"name"`
		Token             string `yaml:"token"`
		DownloadDirectory string `yaml:"downloadDirectory"`
	}

	Builder struct {
		Name        string `yaml:"name"`
		RegistryUri string `yaml:"registryUri"`
	}
)

func NewConfig() (*Config, error) {
	cfg := &Config{}

	if err := cleanenv.ReadConfig("./config/config.yml", cfg); err != nil {
		return nil, fmt.Errorf("config error: %w", err)
	}

	if err := cleanenv.ReadEnv(cfg); err != nil {
		return nil, err
	}

	if err := godotenv.Load("./config/.env"); err != nil {
		return nil, err
	}

	fmt.Println("calling config")
	fmt.Println(os.Environ())

	return cfg, nil
}
