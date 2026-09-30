package ratelimit

import (
	"context"
	"sync"
	"time"
)

// fakeLimiter: in-memory fixed-window (handler tests determinísticos — spec
// §4 "Limiter is an interface with an in-memory fake").
//
//	ponytail: ventana fija por clave sin limpieza — los tests viven segundos;
//	upgrade path: purga perezosa si entra al prod dev-run largo.
type fakeLimiter struct {
	mu     sync.Mutex
	gotten map[string]*fakeWindow
}

type fakeWindow struct {
	count   int
	expires time.Time
}

// NewFake arma el Limiter in-memory.
func NewFake() Limiter { return &fakeLimiter{gotten: map[string]*fakeWindow{}} }

func (f *fakeLimiter) Allow(_ context.Context, key string, limit int, window time.Duration) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	now := time.Now()
	w, ok := f.gotten[key]
	if !ok || now.After(w.expires) {
		// Ventana nueva: arranca en el primer intento.
		if limit <= 0 {
			return false, nil // presupuesto 0 = siempre bloqueado
		}
		f.gotten[key] = &fakeWindow{count: 1, expires: now.Add(window)}
		return true, nil
	}
	if w.count+1 > limit {
		return false, nil
	}
	w.count++
	return true, nil
}
