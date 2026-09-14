// Comando api: punto de entrada del servidor HTTP de Skyland.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/davidsilva131/skyland/apps/api/internal/httpapi"
	"github.com/davidsilva131/skyland/apps/api/internal/platform/config"
	"github.com/davidsilva131/skyland/apps/api/internal/platform/db"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuración inválida", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("postgres no disponible", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	logger.Info("postgres conectado", "env", cfg.Environment)

	srv := &http.Server{
		Addr:              ":" + getenvOrDefault("PORT", "8080"),
		Handler:           httpapi.New(cfg, pool, logger),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("api escuchando", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("servidor http", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("apagando…")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

func getenvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
