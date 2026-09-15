package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/davidsilva131/skyland/apps/api/internal/wallets"
)

// TestWalletsEndpoints prueba los endpoints de wallets end-to-end vía
// httptest sobre el mux, con el adapter in-memory (sin Postgres).
func TestWalletsEndpoints(t *testing.T) {
	mux := http.NewServeMux()
	wallets.RegisterTest(mux)
	do := func(r *http.Request) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}

	w := do(httptest.NewRequest("POST", "/api/v1/wallets/acredita", strings.NewReader(`{"monto":10000,"detalle":"recarga"}`)))
	if w.Code != 200 {
		t.Fatalf("acredita: %d %s", w.Code, w.Body.String())
	}

	w = do(httptest.NewRequest("POST", "/api/v1/wallets/debita", strings.NewReader(`{"monto":4000,"detalle":"apuesta"}`)))
	if w.Code != 200 {
		t.Fatalf("debita: %d %s", w.Code, w.Body.String())
	}

	w = do(httptest.NewRequest("GET", "/api/v1/wallets/saldo", nil))
	if got, want := strings.TrimSpace(w.Body.String()), `{"moneda":"VES","monto":6000}`; got != want {
		t.Fatalf("saldo: got %s", got)
	}

	// Débito mayor al saldo → 402, saldo intacto.
	w = do(httptest.NewRequest("POST", "/api/v1/wallets/debita", strings.NewReader(`{"monto":99999}`)))
	if w.Code != http.StatusPaymentRequired {
		t.Fatalf("saldo insuficiente: esperaba 402, got %d %s", w.Code, w.Body.String())
	}
	w = do(httptest.NewRequest("GET", "/api/v1/wallets/saldo", nil))
	if got, want := strings.TrimSpace(w.Body.String()), `{"moneda":"VES","monto":6000}`; got != want {
		t.Fatalf("saldo tocado tras 402: %s", got)
	}

	// Monto negativo → 400.
	w = do(httptest.NewRequest("POST", "/api/v1/wallets/acredita", strings.NewReader(`{"monto":-5}`)))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("monto negativo: esperaba 400, got %d", w.Code)
	}
}
