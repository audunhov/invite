package config

import (
	"fmt"
	"github.com/caarlos0/env/v11"
)

type Config struct {
	DatabaseURL string `env:"DB_URL,required"`
	Port        int    `env:"PORT" envDefault:"8080"`
	LogLevel    string `env:"LOG_LEVEL" envDefault:"info"`
	LogFormat   string `env:"LOG_FORMAT" envDefault:"text"` // text or json
	SMTPHost    string `env:"SMTP_HOST"`
	SMTPPort    int    `env:"SMTP_PORT" envDefault:"587"`
	SMTPUser    string `env:"SMTP_USER"`
	SMTPPass    string `env:"SMTP_PASS"`
	BaseURL     string `env:"BASE_URL" envDefault:"http://localhost:8080"`
}

func Load() (*Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}
	return &cfg, nil
}
