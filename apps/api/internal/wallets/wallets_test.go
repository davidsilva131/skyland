package wallets

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestTableWallets corre el mismo estándar de doble entrada y saldo
// insuficiente contra cualquier adapter de Backend (fake hoy; pgx cuando
// SKYLAND_TEST_DATABASE_URL apunta a Postgres).
func TestTableWallets(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name  string
		setup func(t *testing.T) Backend
	}{
		{"fake", func(t *testing.T) Backend { return NewFake() }},
	}
	if dsn := pgDADS(); dsn != "" {
		cases = append(cases, pgCase(t, dsn))
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := tc.setup(t)
			// Acredita recarga: saldo 100,00 VES (10000 mínima unidad).
			s, err := b.Acredita(ctx, 1, recarga(10000))
			if err != nil || s.Monto != 10000 {
				t.Fatalf("acredita: %v %v", s, err)
			}
			// Apuesta 40,00: saldo 60,00 y el asiento cierra a 0.
			s, err = b.Debito(ctx, 1, apuesta(4000, 1))
			if err != nil || s.Monto != 6000 {
				t.Fatalf("debito: %v %v", s, err)
			}
			// Saldo consulta lo mismo.
			if s, _ := b.Saldo(ctx, 1); s.Monto != 6000 {
				t.Fatalf("saldo %v", s)
			}
			// Apuesta mayor al saldo: ErrSaldoInsuficiente, sin tocar nada.
			_, err = b.Debito(ctx, 1, apuesta(7000, 1))
			var e ErrSaldoInsuficiente
			if !errors.As(err, &e) || e.Disp != 6000 || e.Ped != 7000 {
				t.Fatalf("esperaba ErrSaldoInsuficiente{6000,7000}, got %v", err)
			}
			if s, _ := b.Saldo(ctx, 1); s.Monto != 6000 {
				t.Fatalf("saldo tocado tras fallo: %v", s)
			}
		})
	}
}

func pgDADS() string {
	dsn := os.Getenv("SKYLAND_TEST_DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("SKYLAND_DATABASE_URL")
	}
	return dsn
}

func pgCase(t *testing.T, dsn string) (out struct {
	name  string
	setup func(t *testing.T) Backend
}) {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Skipf("pg no disponible: %v", err)
	}
	t.Cleanup(pool.Close)
	return struct {
		name  string
		setup func(t *testing.T) Backend
	}{"pg", func(t *testing.T) Backend {
		// Estado limpio: los saldos del test no pueden heredar corridas previas.
		if _, err := pool.Exec(context.Background(),
			`TRUNCATE ledger_entries, ledger_txns, balances`); err != nil {
			t.Fatalf("truncate: %v", err)
		}
		return NewPostgres(pool)
	}}
}

func recarga(m Monto) Asiento {
	return Asiento{
		Moneda:   "VES",
		Detalle:  "recarga pago móvil",
		Entradas: []Entrada{{Debe: m, JugadorID: 0}, {Haber: m, JugadorID: 1}},
	}
}

func apuesta(m Monto, jugador int64) Asiento {
	return Asiento{
		Moneda:   "VES",
		Detalle:  "apuesta",
		Entradas: []Entrada{{Debe: m, JugadorID: jugador}, {Haber: m, JugadorID: 0}},
	}
}
