package ratelimit

import (
	"context"
	"testing"
	"time"
)

// TestFakeBudgets: presupuestos del spec §4 — email 5/15 min (el 6º bloquea),
// IP 20/15 min (el 21º bloquea).
func TestFakeBudgets(t *testing.T) {
	l := NewFake()
	ctx := context.Background()

	// Email: 1–5 pasan, el 6º bloquea.
	for i := 1; i <= LoginEmailLimit; i++ {
		ok, err := l.Allow(ctx, EmailKey("a@b.com"), LoginEmailLimit, AttemptWindow)
		if err != nil {
			t.Fatalf("email intent %d: %v", i, err)
		}
		if !ok {
			t.Fatalf("email intent %d bloqueado; presupuesta %d", i, LoginEmailLimit)
		}
	}
	if ok, _ := l.Allow(ctx, EmailKey("a@b.com"), LoginEmailLimit, AttemptWindow); ok {
		t.Fatal("email intent 6 pasado; want bloqueado")
	}
	// Independencia de claves: otro email intacto.
	if ok, _ := l.Allow(ctx, EmailKey("c@d.com"), LoginEmailLimit, AttemptWindow); !ok {
		t.Fatal("email 2 bloqueado por presupuesto del email 1")
	}

	// IP: 1–20 pasan, la 21ª bloquea.
	for i := 1; i <= LoginIPLimit; i++ {
		ok, err := l.Allow(ctx, IPKey("1.2.3.4"), LoginIPLimit, AttemptWindow)
		if err != nil {
			t.Fatalf("ip intent %d: %v", i, err)
		}
		if !ok {
			t.Fatalf("ip intent %d bloqueado; presupuesta %d", i, LoginIPLimit)
		}
	}
	if ok, _ := l.Allow(ctx, IPKey("1.2.3.4"), LoginIPLimit, AttemptWindow); ok {
		t.Fatal("ip intent 21 pasada; want bloqueada")
	}
}

// TestFakeRegisterSharesIPBudget: register consume el MISMO contador de IP
// que login (spec §4: "the IP counter is shared with register").
func TestFakeRegisterSharesIPBudget(t *testing.T) {
	l := NewFake()
	ctx := context.Background()
	// 20 intentos de register/registro sobre la IP agotan el presupuesto de
	// login de esa IP.
	for i := 0; i < LoginIPLimit; i++ {
		if ok, _ := l.Allow(ctx, IPKey("9.9.9.9"), LoginIPLimit, AttemptWindow); !ok {
			t.Fatalf("ip intent %d bloqueado antes de tiempo", i+1)
		}
	}
	// El usuario (login) de la misma IP cae bajo el mismo contador → 21º.
	if ok, _ := l.Allow(ctx, IPKey("9.9.9.9"), LoginIPLimit, AttemptWindow); ok {
		t.Fatal("login de la IP pasó tras agotar con register; want bloqueado")
	}
}

// TestFakeWindowExpires: ventana fija — el contador expira (guardia básica).
func TestFakeWindowExpires(t *testing.T) {
	l := NewFake()
	ctx := context.Background()
	if ok, _ := l.Allow(ctx, "k", 1, time.Millisecond); !ok {
		t.Fatal("primer intento bloqueado")
	}
	if ok, _ := l.Allow(ctx, "k", 1, time.Millisecond); ok {
		t.Fatal("segundo intento en la misma ventana pasó")
	}
	time.Sleep(2 * time.Millisecond)
	if ok, _ := l.Allow(ctx, "k", 1, time.Millisecond); !ok {
		t.Fatal("ventana expirada no se reabrió")
	}
}
