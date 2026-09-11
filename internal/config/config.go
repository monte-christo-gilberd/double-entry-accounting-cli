package config

import (
	"errors"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv           string
	DatabaseHost     string
	DatabasePort     string
	DatabaseUser     string
	DatabasePassword string
	DatabaseName     string
	DatabaseSSLMode  string
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	cfg := &Config{
		AppEnv:           os.Getenv("APP_ENV"),
		DatabaseHost:     os.Getenv("DATABASE_HOST"),
		DatabasePort:     os.Getenv("DATABASE_PORT"),
		DatabaseUser:     os.Getenv("DATABASE_USER"),
		DatabasePassword: os.Getenv("DATABASE_PASSWORD"),
		DatabaseName:     os.Getenv("DATABASE_NAME"),
		DatabaseSSLMode:  os.Getenv("DATABASE_SSLMODE"),
	}
	if cfg.DatabaseHost == "" {
		return nil, errors.New("DATABASE_HOST is required")
	}
	if cfg.DatabasePort == "" {
		return nil, errors.New("DATABASE_PORT is required")
	}
	if cfg.DatabaseUser == "" {
		return nil, errors.New("DATABASE_USER is required")
	}
	if cfg.DatabaseName == "" {
		return nil, errors.New("DATABASE_NAME is required")
	}
	if cfg.DatabaseSSLMode == "" {
		cfg.DatabaseSSLMode = "disable"
	}
	return cfg, nil
}
