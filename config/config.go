package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
)

type (
	Config struct {
		App      `yaml:"app"`
		Log      `yaml:"logger"`
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

	Database struct {
		Driver string `env-required:"true"  yaml:"driver" env:"DATABASE_DRIVER"`
		URI    string `                                   env:"DATABASE_URI"`
	}

	Services struct {
		Downloaders []Downloader `yaml:"downloaders,flow"`
		Builders    []Builder    `yaml:"builders,flow"`
	}

	Downloader struct {
		Name              string `yaml:"name"`
		DownloadDirectory string `yaml:"downloadDirectory"`
	}

	Builder struct {
		Name        string `yaml:"name"`
		RegistryUri string `yaml:"registryUri"`
	}
)

func NewConfig() (*Config, error) {
	cfg := &Config{}

	err := cleanenv.ReadConfig("./config/config.yml", cfg)
	if err != nil {
		return nil, fmt.Errorf("config error: %w", err)
	}

	err = cleanenv.ReadEnv(cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
