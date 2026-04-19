package config

import (
	"fmt"
	"github.com/caarlos0/env/v11"
)

type Config struct {
	DatabaseURL string `env:"DB_URL,required"`
	Port        int    `env:"PORT" envDefault:"8080"`
}

func Load() (*Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}
	return &cfg, nil
}
