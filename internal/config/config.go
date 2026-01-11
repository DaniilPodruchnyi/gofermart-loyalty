package config

import (
	"flag"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	RunAddress           string
	DatabaseURI          string
	AccrualSystemAddress string
	JWTSecret            string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	runAddress := flag.String("a", "", "server run address")
	databaseURI := flag.String("d", "", "database connection URI")
	accrualAddress := flag.String("r", "", "accrual system address")
	flag.Parse()

	cfg := &Config{
		RunAddress:           getEnvOrFlag("RUN_ADDRESS", *runAddress, ":8080"),
		DatabaseURI:          getEnvOrFlag("DATABASE_URI", *databaseURI, ""),
		AccrualSystemAddress: getEnvOrFlag("ACCRUAL_SYSTEM_ADDRESS", *accrualAddress, ""),
		JWTSecret:            os.Getenv("JWT_SECRET"),
	}

	if cfg.DatabaseURI == "" {
		return nil, fmt.Errorf("DATABASE_URI is required")
	}

	if cfg.JWTSecret == "" {
		cfg.JWTSecret = "default-secret-key"
	}

	return cfg, nil
}

func getEnvOrFlag(envKey, flagValue, defaultValue string) string {
	if flagValue != "" {
		return flagValue
	}
	if envValue := os.Getenv(envKey); envValue != "" {
		return envValue
	}
	return defaultValue
}
