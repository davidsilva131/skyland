package wallets

import (
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Register arma el módulo wallets en prod con el adapter pgx.
func Register(mux *http.ServeMux, pool *pgxpool.Pool) {
	RegisterRoutes(mux, NewPostgres(pool))
}

// RegisterTest arma el módulo con el adapter in-memory (tests / dev sin DB).
func RegisterTest(mux *http.ServeMux) { RegisterRoutes(mux, NewFake()) }

// RegisterRoutes registra las rutas HTTP del módulo sobre cualquier Backend.
//
// ponytail: sin sesiones aún (TODO auth) — jugador_id llega hardcodeado=1;
// upgrade path: inyectarlo del módulo auth cuando exista.
func RegisterRoutes(mux *http.ServeMux, b Backend) {
	mux.HandleFunc("GET /api/v1/wallets/saldo", func(w http.ResponseWriter, r *http.Request) {
		s, err := b.Saldo(r.Context(), jugadorDePrueba)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"monto": int64(s.Monto), "moneda": string(s.Moneda)})
	})

	mux.HandleFunc("POST /api/v1/wallets/acredita", func(w http.ResponseWriter, r *http.Request) {
		movimiento(w, r, b, +1)
	})
	mux.HandleFunc("POST /api/v1/wallets/debita", func(w http.ResponseWriter, r *http.Request) {
		movimiento(w, r, b, -1)
	})
}

func movimiento(w http.ResponseWriter, r *http.Request, b Backend, op int) {
	var req struct {
		Monto   int64  `json:"monto"`
		Moneda  string `json:"moneda"`
		Detalle string `json:"detalle"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if req.Monto <= 0 {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("monto debe ser positivo"))
		return
	}
	if req.Moneda == "" {
		req.Moneda = "VES"
	}
	a := Asiento{
		Moneda:  MonedaISO(req.Moneda),
		Detalle: req.Detalle,
		// Entrada del jugador contra la Casa (jugador 0 = contraparte).
		Entradas: []Entrada{
			{Debe: Monto(req.Monto), JugadorID: 1},
			{Haber: Monto(req.Monto), JugadorID: idCasa},
		},
	}
	if op < 0 {
		// Invertir el asiento para un débito: el jugador debe.
		a.Entradas = []Entrada{
			{Debe: Monto(req.Monto), JugadorID: 1},
			{Haber: Monto(req.Monto), JugadorID: idCasa},
		}
	}
	var (
		s   Saldo
		err error
	)
	if op > 0 {
		s, err = b.Acredita(r.Context(), jugadorDePrueba, a)
	} else {
		s, err = b.Debito(r.Context(), jugadorDePrueba, a)
	}
	if err != nil {
		if me, ok := err.(ErrSaldoInsuficiente); ok {
			writeJSON(w, http.StatusPaymentRequired, map[string]any{"error": me.Error()})
			return
		}
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"monto": int64(s.Monto), "moneda": string(s.Moneda)})
}

// contrapartidaID: la Casa (jugador 0 en el ledger).
const idCasa int64 = 0

// jugadorDePrueba: placeholder mientras no haya sesiones (TODO auth #9).
const jugadorDePrueba int64 = 1
