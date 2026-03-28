package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	DatabaseURL    string `env:"DATABASE_URL,required"`
	Port           string `env:"PORT" envDefault:"8080"`
	Environment    string `env:"ENVIRONMENT" envDefault:"development"`
	AdminAPIKey    string `env:"ADMIN_API_KEY"`
	JWTSecret      string `env:"JWT_SECRET" envDefault:"dev-secret-change-in-production"`
	GeminiAPIKey   string `env:"GEMINI_API_KEY"`
	AllowedOrigins string `env:"ALLOWED_ORIGINS" envDefault:"*"`
	DBMaxConns     int32  `env:"DB_MAX_CONNS" envDefault:"25"`
	DBMinConns     int32  `env:"DB_MIN_CONNS" envDefault:"5"`
}

func (c *Config) IsProduction() bool { return c.Environment == "production" }

func Load() (*Config, error) {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}
