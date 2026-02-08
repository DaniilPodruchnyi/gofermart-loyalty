package config

import (
	"flag"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	RunAddress           string
	DatabaseURI          string
	AccrualSystemAddress string
	JWTSecret            string
}

func Load() *Config {
	// Попытка загрузить .env файл (игнорируем ошибку если файла нет)
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found or error loading it: %v", err)
	}

	cfg := &Config{
		JWTSecret: "your-secret-key-change-in-production",
	}

	flag.StringVar(&cfg.RunAddress, "a", ":8080", "server run address")
	flag.StringVar(&cfg.DatabaseURI, "d", "", "database connection string")
	flag.StringVar(&cfg.AccrualSystemAddress, "r", "", "accrual system address")
	flag.Parse()

	// Приоритет: флаги -> переменные окружения -> значения по умолчанию
	if cfg.DatabaseURI == "" {
		if envDatabaseURI := os.Getenv("DATABASE_URI"); envDatabaseURI != "" {
			cfg.DatabaseURI = envDatabaseURI
		} else if envDatabaseURL := os.Getenv("DATABASE_URL"); envDatabaseURL != "" {
			// Поддержка DATABASE_URL (часто используется в Heroku и других платформах)
			cfg.DatabaseURI = envDatabaseURL
		}
	}

	if envRunAddress := os.Getenv("RUN_ADDRESS"); envRunAddress != "" {
		cfg.RunAddress = envRunAddress
	}

	if envAccrualAddress := os.Getenv("ACCRUAL_SYSTEM_ADDRESS"); envAccrualAddress != "" {
		cfg.AccrualSystemAddress = envAccrualAddress
	}

	return cfg
}
