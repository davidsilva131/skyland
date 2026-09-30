// Package httpapi arma el servidor HTTP de la API: middleware CORS + routers de los módulos.
package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/davidsilva131/skyland/apps/api/internal/auth"
	"github.com/davidsilva131/skyland/apps/api/internal/platform/config"
	"github.com/davidsilva131/skyland/apps/api/internal/ratelimit"
	"github.com/davidsilva131/skyland/apps/api/internal/wallets"
	"github.com/jackc/pgx/v5/pgxpool"
)

// New construye el http.Handler raíz de la API.
func New(cfg *config.Config, pool *pgxpool.Pool, logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{
			"status": "ok",
			"app":    "skyland-api",
			"env":    cfg.Environment,
		})
	})

	// Módulos del monolito modular (cada uno registra sus rutas y servicios).
	wallets.Register(mux, pool)
	// auth: Postgres store + Upstash Redis limiter (fake si no hay URL).
	auth.Register(mux, auth.NewPostgres(pool), newLimiter(cfg, logger), logger, cfg.CORSOrigins)
	// TODO(games): motor de apuestas + contratos BetMarket/ResultSource.
	// TODO(notifications): SSE contador de sorteo y resultados.
	// TODO(admin): backoffice RBAC.

	registerStatus(mux, "GET /api/v1/games", "games")
	registerStatus(mux, "GET /api/v1/payments/status", "payments")
	registerStatus(mux, "GET /api/v1/notifications/status", "notifications")
	registerStatus(mux, "GET /api/v1/admin/status", "admin")

	_ = logger // los módulos realicen su logging en su construcción real

	return logRequests(logger)(cors(cfg.CORSOrigins)(mux))
}

// newLimiter: Upstash Redis si hay URL; fail-open (sin límite efectivo) si
// no — en dev sin Redis la API sigue sirviendo (spec §4).
func newLimiter(cfg *config.Config, logger *slog.Logger) ratelimit.Limiter {
	if cfg.UpstashRedisURL == "" {
		logger.Warn("ratelimit: SKYLAND_UPSTASH_REDIS_URL vacía — sin límite (fail-open)")
		return ratelimit.NewFake()
	}
	return ratelimit.NewRedis(cfg.UpstashRedisURL, logger)
}

// RegisterFake arma los módulos sobre el fake in-memory (dev sin DB).
// ponytail: no pasa por cors/logRequests; bórralo si el fake en runtime
// deja de ser necesario (auth + Postgres local siempre disponible).
func RegisterFake(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "backend": "fake"})
	})
	wallets.RegisterTest(mux)
	auth.RegisterTest(mux)
}
