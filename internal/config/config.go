package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config represents runtime configuration loaded from environment variables.
type Config struct {
	AppEnv       string
	LogLevel     string
	Port         int
	APIKey       string
	TargetDomain string
}

// Load attempts to parse the provided .env file (if any) and populate a Config
// struct from environment variables. If envPath is empty, the default loading
// behavior of godotenv applies (searching for a .env file in the working dir).
func Load(envPath string) (*Config, error) {
	if envPath != "" {
		if err := godotenv.Overload(envPath); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("config: failed to load %s: %w", envPath, err)
		}
	} else {
		// Ignore error if .env is absent; we'll rely on process env vars.
		_ = godotenv.Load()
	}

	cfg := &Config{
		AppEnv:       getString("APP_ENV", "development"),
		LogLevel:     getString("LOG_LEVEL", "info"),
		Port:         getInt("PORT", 8080),
		APIKey:       getString("API_KEY", ""),
		TargetDomain: getString("TARGET_DOMAIN", "example.com"),
	}

	if err := validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// MustLoad wraps Load and panics on error. Useful for CLI entrypoints.
func MustLoad(envPath string) *Config {
	cfg, err := Load(envPath)
	if err != nil {
		panic(err)
	}
	return cfg
}

func validate(cfg *Config) error {
	var missing []string
	if cfg.APIKey == "" {
		missing = append(missing, "API_KEY")
	}
	if strings.TrimSpace(cfg.TargetDomain) == "" {
		missing = append(missing, "TARGET_DOMAIN")
	}

	if len(missing) > 0 {
		return errors.New("config: missing required envs: " + strings.Join(missing, ", "))
	}

	return nil
}

func getString(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return fallback
}

func getInt(key string, fallback int) int {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		parsed, err := strconv.Atoi(val)
		if err == nil {
			return parsed
		}
	}
	return fallback
}
