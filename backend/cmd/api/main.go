package main

import (
	"log/slog"
	"net/http"
	"os"

	"atlas-financial-core/internal/config"
	"atlas-financial-core/internal/database"
	"atlas-financial-core/internal/handler"
	"atlas-financial-core/internal/server"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := config.Load()

	pool, err := database.DBConn(cfg.ConnStr)
	if err != nil {
		slog.Error("Connection Unsucceed", "error", err)
		os.Exit(1)
	}

	defer pool.Close()

	mux := http.NewServeMux()
	handler.HealthRoute(mux)

	server.Run(cfg.Port, mux, logger)

}
