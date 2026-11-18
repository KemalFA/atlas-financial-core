package main

import (
	"log/slog"
	"net/http"
	"os"

	"atlas-financial-core/internal/account"
	"atlas-financial-core/internal/config"
	"atlas-financial-core/internal/database"
	"atlas-financial-core/internal/health"
	"atlas-financial-core/internal/server"
)

func main() {
	//Initiate Logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	//Load Configurations of .env
	cfg := config.Load()

	//Get DB Connection string and make Pool Connection
	pool, err := database.DBConn(cfg.ConnStr)
	if err != nil {
		slog.Error("database connection failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	//Initiate mux to make endpoint
	mux := http.NewServeMux()

	//Initiate Account Repo, Service and Handler
	accountRepo := account.NewRepository(pool)
	accountService := account.NewService(accountRepo)
	accountHandler := account.NewHandler(accountService)

	//Endpoint
	health.HealthRoute(mux)           //Server health endpoint
	accountHandler.RegisterRoute(mux) //Account Endpoint

	//Run Server
	server.Run(cfg.Port, mux, logger)

}
