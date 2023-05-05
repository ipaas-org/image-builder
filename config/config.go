package config

import (
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
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
		RequestQueue  string `env-required:"true" yaml:"requestQueue" env:"RABBITMQ_REQUEST_QUEUE"`
		ResponseQueue string `env-required:"true" yaml:"responseQueue" env:"RABBITMQ_REPONSE_QUEUE"`
	}

	Database struct {
		Driver string `env-required:"true"  yaml:"driver" env:"DATABASE_DRIVER"`
		URI    string `                                   env:"DATABASE_URI"`
	}

	Services struct {
		Connectors []Connector `yaml:"connectors,flow"`
		Builders   []Builder   `yaml:"builders,flow"`
		Registries []Registry  `yaml:"registries,flow"`
	}

	Connector struct {
		Name              string `yaml:"name"`
		DownloadDirectory string `yaml:"downloadDirectory"`
	}

	Builder struct {
		Name string `yaml:"name"`
	}

	Registry struct {
		Name          string `yaml:"name"`
		ServerAddress string `env-required:"true" yaml:"serverAddress"` //env:"REGISTRY_SERVER_ADDRESS"
	}
)

func NewConfig() (*Config, error) {
	cfg := &Config{}

	if err := godotenv.Load("./config/.env"); err != nil {
		if err.Error() != "open ./config/.env: no such file or directory" {
			return nil, err
		} else {
			logrus.Warn(".env file not found, using env variables")
		}
	}

	if err := cleanenv.ReadConfig("./config/config.yml", cfg); err != nil {
		return nil, fmt.Errorf("config error: %w", err)
	}

	if err := cleanenv.ReadEnv(cfg); err != nil {
		return nil, err
	}

	if os.Getenv("REGISTRY_DOCKER_USERNAME") == "" || os.Getenv("REGISTRY_DOCKER_PASSWORD") == "" {
		logrus.Warn("REGISTRY_DOCKER_USERNAME or REGISTRY_DOCKER_PASSWORD not set, using anonymous access")
	}

	if os.Getenv("RABBITMQ_URI") == "" {
		logrus.Fatalln("RABBITMQ_URI not set, this env variable is required")
	}

	return cfg, nil
}
