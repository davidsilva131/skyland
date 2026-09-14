// Package httpapi arma el servidor HTTP de la API: middleware CORS + routers de los módulos.
package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/davidsilva131/skyland/apps/api/internal/platform/config"
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
	// TODO(auth): sesiones httpOnly + RBAC (Admin / Soporte).
	// TODO(wallets): débito atómico + ledger de doble entrada (transacción ACID local).
	// TODO(games): motor de apuestas + contratos BetMarket/ResultSource.
	// TODO(payments): recargas + PagoVerifier (manual hoy, API bancaria mañana).
	// TODO(notifications): SSE contador de sorteo y resultados.
	// TODO(admin): backoffice RBAC.
	registerStatus(mux, "GET /api/v1/auth/status", "auth")
	registerStatus(mux, "GET /api/v1/wallets/status", "wallets")
	registerStatus(mux, "GET /api/v1/games", "games")
	registerStatus(mux, "GET /api/v1/payments/status", "payments")
	registerStatus(mux, "GET /api/v1/notifications/status", "notifications")
	registerStatus(mux, "GET /api/v1/admin/status", "admin")

	_ = pool // los módulos recibirán el pool en su construcción real

	return logRequests(logger)(cors(cfg.CORSOrigins)(mux))
}
