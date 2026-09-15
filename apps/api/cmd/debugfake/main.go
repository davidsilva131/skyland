// Comando dev-only: API corriendo sobre el fake in-memory (sin Postgres).
package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/davidsilva131/skyland/apps/api/internal/httpapi"
)

func main() {
	mux := http.NewServeMux()
	httpapi.RegisterFake(mux)
	slog.New(slog.NewTextHandler(os.Stdout, nil)).Info("fake api en :8081")
	_ = http.ListenAndServe(":8081", mux)
}
