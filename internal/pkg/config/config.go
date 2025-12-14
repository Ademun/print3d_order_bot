package config

import (
	"fmt"
	"os"

	"github.com/caarlos0/env/v10"
	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type Config struct {
	DB          DBConfig
	FileService FileServiceCfg `yaml:"file_service"`
	TelegramCfg TelegramCfg    `yaml:"telegram"`
	MTProtoCfg  MTProtoCfg
}

type DBConfig struct {
	ConnString string `env:"DB_CONN_STRING"`
}

type FileServiceCfg struct {
	DirPath string `yaml:"dir_path"`
}

type TelegramCfg struct {
	Token string `env:"TOKEN,required"`
}

type MTProtoCfg struct {
	AppID   int    `env:"APP_ID,required"`
	AppHash string `env:"APP_HASH,required"`
	Token   string `env:"TOKEN,required"`
}

func Load(configPath string) (*Config, error) {
	if err := godotenv.Load(".env"); err != nil {
		return nil, fmt.Errorf("failed to load .env file: %w", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse environment variables: %w", err)
	}

	return &cfg, nil
}
