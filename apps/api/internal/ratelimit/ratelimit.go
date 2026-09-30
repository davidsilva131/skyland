// Package ratelimit: presupuestos por IP/email (spec auth-player §4, shape
// normativo de #2). Fixed window en Redis; interface con fake para los tests.
package ratelimit

import (
	"context"
	"time"
)

// Limiter es el contrato (spec §4): unAllow por clave, presupuesto y ventana.
// El Redis impl es el prod; el fake in-memory cubre los handler tests.
type Limiter interface {
	// Allow: fixed window — true si el intento cabe en el presupuesto.
	Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error)
}

// Presupuestos (spec §4, normativo): login chequea ambos, registro solo IP.
const (
	// LoginEmailLimit: 5 intentos por email cada 15 min.
	LoginEmailLimit = 5
	// LoginIPLimit: 20 intentos por IP cada 15 min (contador compartido con
	// register — spec §4).
	LoginIPLimit = 20
	// AttemptWindow: ventana de los presupuestos.
	AttemptWindow = 15 * time.Minute
)

// Claves de Redis (spec §4).
func IPKey(ip string) string   { return "rl:ip:" + ip }
func EmailKey(e string) string { return "rl:email:" + e }

// Del borra la clave del email tras un login exitoso (reset del contador).
type Deleter interface {
	Del(ctx context.Context, key string) error
}
