package wallets

import (
	"context"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5"
)

// fakeBackend es el adapter in-memory de Backend: misma interface, sin
// Postgres. Sirve para tests de tabla y para correr la API en dev sin DB.
//
// ponytail: solo mantiene saldo VES (más simple que el mapping con currency
// del adapter prod); upgrade path: map[jugadorID]map[MonedaISO]Monto si
// multi-moneda llega antes de tocar el camino del dinero real.
type fakeBackend struct {
	mu  sync.Mutex
	sal map[int64]int64 // jugadorID -> mínima unidad (VES)
}

func NewFake() Backend { return &fakeBackend{sal: map[int64]int64{}} }

func (f *fakeBackend) apply(ctx context.Context, jugadorID int64, a Asiento, delta int64) (Saldo, error) {
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
	f.mu.Lock()
	defer f.mu.Unlock()
	nuevo := f.sal[jugadorID] + delta
	if delta < 0 && nuevo < 0 {
		return Saldo{}, ErrSaldoInsuficiente{Disp: Monto(f.sal[jugadorID]), Ped: Monto(-delta)}
	}
	f.sal[jugadorID] = nuevo
	return Saldo{Monto: Monto(nuevo), Moneda: a.Moneda}, nil
}

func (f *fakeBackend) Debito(ctx context.Context, jugadorID int64, a Asiento) (Saldo, error) {
	return f.apply(ctx, jugadorID, a, -int64(montoAsiento(a)))
}

func (f *fakeBackend) Acredita(ctx context.Context, jugadorID int64, a Asiento) (Saldo, error) {
	return f.apply(ctx, jugadorID, a, +int64(montoAsiento(a)))
}

func (f *fakeBackend) Saldo(ctx context.Context, jugadorID int64) (Saldo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	v, ok := f.sal[jugadorID]
	if !ok {
		return Saldo{}, pgx.ErrNoRows // misma semántica que el adapter pgx
	}
	return Saldo{Monto: Monto(v), Moneda: "VES"}, nil
}

// montoAsiento devuelve el monto del asiento (la primera entrada con valor
// distinto de cero): por contrato toda entrada toca un solo lado.
func montoAsiento(a Asiento) Monto {
	for _, e := range a.Entradas {
		if e.Debe > 0 {
			return e.Debe
		}
		if e.Haber > 0 {
			return e.Haber
		}
	}
	return 0
}

var _ Backend = &fakeBackend{}
