package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	DatabaseURL string `env:"DATABASE_URL,required"`
	Port        string `env:"PORT" envDefault:"8080"`
	Environment string `env:"ENVIRONMENT" envDefault:"development"`
}

func Load() (*Config, error) {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}
