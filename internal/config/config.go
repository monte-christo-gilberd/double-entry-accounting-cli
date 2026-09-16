package config

import (
	"errors"
	"os"
	"strconv"
	"strings"

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
	if _, err := strconv.Atoi(cfg.DatabasePort); err != nil {
		return nil, errors.New("DATABASE_PORT must be numeric")
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
	switch strings.ToLower(cfg.DatabaseSSLMode) {
	case "disable", "allow", "prefer", "require", "verify-ca", "verify-full":
	default:
		return nil, errors.New("DATABASE_SSLMODE must be one of disable, allow, prefer, require, verify-ca, verify-full")
	}
	return cfg, nil
}
