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
}

type DBConfig struct {
	Username string `env:"DB_USERNAME,required"`
	Password string `env:"DB_PASSWORD,required"`
	Host     string `env:"DB_HOST,required"`
	Database string `env:"DB_DATABASE,required"`
}

type FileServiceCfg struct {
	DirPath             string   `yaml:"dir_path"`
	AppendModeFilenames []string `yaml:"append_mode_filenames"`
}

type TelegramCfg struct {
	Token string `env:"TOKEN,required"`
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
