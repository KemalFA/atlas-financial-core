package config

import (
	"log/slog"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port    string
	ConnStr string
}

func Load() Config {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	//Load .env
	err := godotenv.Load()
	if err != nil {
		logger.Error("No .env file found; using system environment variables", slog.String("error", err.Error()))
	}

	Port := os.Getenv("PORT")
	if Port == "" {
		Port = "8080"
	}

	ConnStr := os.Getenv("DATABASE_URL")
	if ConnStr == "" {
		logger.Info("DATABASE_URL is empty")
	}

	cfg := Config{
		Port:    Port,
		ConnStr: ConnStr,
	}

	return cfg

}
