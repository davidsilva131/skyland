package wallets

import (
	"context"
	"fmt"
)

// Monto entero en la mínima unidad de la moneda (2 decimales en VES:
// 12345 = 123,45). Nunca float.
type Monto int64

// MonedaISO código ISO 4217 ("VES" hoy).
type MonedaISO string

// Entrada de una partida de doble entrada (debito o credito en una cuenta
// dentro de libros contables).
type Entrada struct {
	// Debe y Haber: uno de los dos es positivo, el otro 0. Suma de la
	// entrada = 0.
	Debe, Haber Monto
	// Jugador a quien pertenece el asiento.
	JugadorID int64
}

// Asiento de doble entrada: Debe total == Haber total == transacción.
type Asiento struct {
	// Moneda del asiento (una sola por asiento; multi-moneda desde el día 1,
	// pero una transacción toca una moneda a la vez).
	Moneda MonedaISO
	// Descripción de la operación (ej: "apuesta animalitos ronda 12").
	Detalle string
	// Entradas debe+haber que cierran a 0.
	Entradas []Entrada
}

// Saldo es lo que el Jugador tiene disponible para apostar.
type Saldo struct {
	Monto  Monto
	Moneda MonedaISO
}

// ErrSaldoInsuficiente: la apuesta/caída de saldo no procede.
type ErrSaldoInsuficiente struct{ Disp, Ped Monto }

func (e ErrSaldoInsuficiente) Error() string {
	return fmt.Sprintf("saldo insuficiente: disponible %d, pedido %d", e.Disp, e.Ped)
}

// Backend es el único punto de contacto con el almacenamiento de Saldo.
// Toda mutación de dinero corre dentro de una transacción ACID local
// (ADR-0001 Rev. 4: "camino del dinero solo en Postgres").
//
// Un solo método: la interface es una transacción.
type Backend interface {
	// Debito_ID: dentro de una tx de Postgres, debita el saldo del
	// jugador y escribe el asiento de doble entrada. Devuelve el saldo
	// resultante.
	Debito(ctx context.Context, jugadorID int64, a Asiento) (Saldo, error)
	// Acredita_ID: idem, acredita (recarga, payout).
	Acredita(ctx context.Context, jugadorID int64, a Asiento) (Saldo, error)
	// Saldo: consulta puntual (fuera del camino crítico del dinero).
	Saldo(ctx context.Context, jugadorID int64) (Saldo, error)
}
