package server

import (
	"atlas-financial-core/internal/middleware"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"
)

func Run(port string, mux *http.ServeMux, logger *slog.Logger) {

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           middleware.DetailLog(mux, logger),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("Server start on :" + port)
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Server listen error", slog.String("error", err.Error()))
		}
	}()
	<-ctx.Done()
	logger.Info("Shutdown Signal Received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := server.Shutdown(shutdownCtx)
	if err != nil {
		logger.Error("Server shutdown failed", slog.String("error", err.Error()))
	}

	logger.Info("Server Has Shutdown Gracefully")
}
