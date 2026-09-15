package wallets

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Postgres es el adapter pgx de Backend. Un backend = hc de dinero real.
//
// Esquema esperado (migrations/000002_wallets.up.sql):
//
//	balances(jugador_id bigint, currency text, amount bigint,
//	         primary key (jugador_id, currency))
//	ledger_txns(id bigserial pk, moneda text, detalle text)
//	ledger_entries(txn_id bigint, jugador_id bigint, debe bigint, haber bigint)
type postgresBackend struct{ pool *pgxpool.Pool }

func NewPostgres(pool *pgxpool.Pool) Backend { return postgresBackend{pool} }

// Debito corrida del camino del dinero: una única transacción local ACID
// (pgx) que debita el balance y escribe la partida de doble entrada.
func (b postgresBackend) Debito(ctx context.Context, jugadorID int64, a Asiento) (Saldo, error) {
	return b.apply(ctx, jugadorID, a, -1)
}

func (b postgresBackend) Acredita(ctx context.Context, jugadorID int64, a Asiento) (Saldo, error) {
	return b.apply(ctx, jugadorID, a, +1)
}

func (b postgresBackend) apply(ctx context.Context, jugadorID int64, a Asiento, sign int) (Saldo, error) {
	// assertFalse quickly: asiento cierra a 0, moneda no vacía
	if a.Moneda == "" {
		return Saldo{}, fmt.Errorf("asiento sin moneda")
	}
	var d, h Monto
	for _, e := range a.Entradas {
		d += e.Debe
		h += e.Haber
	}
	if d != h {
		return Saldo{}, fmt.Errorf("asiento descuadrado: debe %d != haber %d", d, h)
	}

	tx, err := b.pool.Begin(ctx)
	if err != nil {
		return Saldo{}, err
	}
	defer tx.Rollback(ctx)

	// Delta del balance del jugador dentro de la transacción: lo que toca
	// al jugador en el asiento, con el signo de la operación.
	op := int64(sign) * int64(montoAsiento(a))

	// Lock de fila primero (evita carreras), validar saldo en el código
	// (el CHECK amount >= 0 in-DB es la última defensa, no el ponytail del
	// error amable) y solo entonces UPDATE.
	var amount int64
	q := `SELECT amount FROM balances WHERE jugador_id = $1 AND currency = $2 FOR UPDATE`
	err = tx.QueryRow(ctx, q, jugadorID, string(a.Moneda)).Scan(&amount)
	if err == pgx.ErrNoRows {
		// Fila no existe: inicial en cero. Un débito sobre cero cayará en
		// ErrSaldoInsuficiente abajo (correcto: no hay saldo).
		if _, e2 := tx.Exec(ctx, `
			INSERT INTO balances (jugador_id, currency, amount)
			VALUES ($1, $2, 0)`, jugadorID, string(a.Moneda)); e2 != nil {
			return Saldo{}, e2
		}
	} else if err != nil {
		return Saldo{}, err
	}
	nuevo := Monto(amount) + Monto(op)
	if nuevo < 0 {
		return Saldo{}, ErrSaldoInsuficiente{Disp: Monto(amount), Ped: Monto(-op)}
	}
	if _, err = tx.Exec(ctx,
		`UPDATE balances SET amount = $1 WHERE jugador_id = $2 AND currency = $3`,
		int64(nuevo), jugadorID, string(a.Moneda)); err != nil {
		return Saldo{}, err
	}

	// Escribir el asiento.
	var txnID int64
	if err = tx.QueryRow(ctx, `
		INSERT INTO ledger_txns (currency, detalle) VALUES ($1, $2) RETURNING id`,
		string(a.Moneda), a.Detalle).Scan(&txnID); err != nil {
		return Saldo{}, err
	}
	for _, e := range a.Entradas {
		var debe, haber int64
		if e.Debe > 0 {
			debe = int64(e.Debe)
		} else {
			haber = int64(e.Haber)
		}
		if _, err = tx.Exec(ctx, `
			INSERT INTO ledger_entries (txn_id, jugador_id, debe, haber)
			VALUES ($1, $2, $3, $4)`, txnID, e.JugadorID, debe, haber); err != nil {
			return Saldo{}, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return Saldo{}, err
	}
	return Saldo{Monto: nuevo, Moneda: a.Moneda}, nil
}

func (b postgresBackend) Saldo(ctx context.Context, jugadorID int64) (Saldo, error) {
	// ponytail: hoy solo VES; multi-moneda real llega con el catálogo de
	// monedas en la tabla balances. Upgrade path: devolver []Saldo.
	var m MonedaISO = "VES"
	var amount int64
	err := b.pool.QueryRow(ctx,
		`SELECT amount FROM balances WHERE jugador_id = $1 AND currency = $2`,
		jugadorID, string(m)).Scan(&amount)
	if err != nil {
		return Saldo{}, err
	}
	return Saldo{Monto: Monto(amount), Moneda: m}, nil
}

var _ Backend = postgresBackend{}
