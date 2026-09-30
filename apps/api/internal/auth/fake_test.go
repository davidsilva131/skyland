package auth

import (
	"context"
	"testing"
	"time"
)

// fixedNow: reloj clavado para los tests de servicio (spec §7 paso 4:
// service tests antes que los handlers).
var testNow = time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)

// TestFakeServiceRegisterLogin: contrato del Service contra el fake (spec §7
// paso 4: service tests primero). Register loguea; email duplicado
// case-insensitive → ErrEmailTaken; login de contraseña mala →
// ErrInvalidCredentials; login de email desconocido → ErrInvalidCredentials.
func TestFakeServiceRegisterLogin(t *testing.T) {
	s := NewFake()
	ctx := context.Background()

	res, err := s.Register(ctx, "Jugador@Ejemplo.COM", "contrasenasagrada-1", "2000-01-15", true, testNow, "203.0.113.7")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if res.Token == "" {
		t.Fatal("register sin token de sesión")
	}
	if want := "jugador@ejemplo.com"; res.Player.Email != want {
		t.Fatalf("email normalizado = %q; want %q", res.Player.Email, want)
	}

	// Email duplicado (case-insensitive) → ErrEmailTaken.
	_, err = s.Register(ctx, "jugador@ejemplo.com", "otraclavelarga99", "2001-02-20", true, testNow, "203.0.113.7")
	if !errorIs(err, ErrEmailTaken) {
		t.Fatalf("email duplicado: err = %v; want ErrEmailTaken", err)
	}

	// Login con la contraseña real (case-insensitive en el email).
	ok, err := s.Login(ctx, "JUGADOR@ejemplo.com", "contrasenasagrada-1", testNow)
	if err != nil {
		t.Fatalf("login válido: %v", err)
	}
	if ok.Token == "" {
		t.Fatal("login sin token")
	}

	// Login con contraseña equivocada → ErrInvalidCredentials.
	if _, err := s.Login(ctx, "jugador@ejemplo.com", "otraclavelarga99", testNow); !errorIs(err, ErrInvalidCredentials) {
		t.Fatalf("login contraseña mala: err = %v; want ErrInvalidCredentials", err)
	}

	// Login de email desconocido → ErrInvalidCredentials (nunca otra cosa).
	if _, err := s.Login(ctx, "nadie@ejemplo.com", "contrasenasagrada-1", testNow); !errorIs(err, ErrInvalidCredentials) {
		t.Fatalf("login desconocido: err = %v; want ErrInvalidCredentials", err)
	}
}

// TestFakeServiceVerifyRevoke: /me + logout contra el fake.
func TestFakeServiceVerifyRevoke(t *testing.T) {
	s := NewFake()
	ctx := context.Background()

	res, err := s.Register(ctx, "sesion@ejemplo.com", "contrasenasagrada-1", "2000-01-15", true, testNow, "203.0.113.7")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	// Verify con el token fresco → player.
	p, err := s.Verify(ctx, res.Token)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if p.Email != "sesion@ejemplo.com" {
		t.Fatalf("verify email = %q", p.Email)
	}

	// Token inventado → ErrUnauthenticated.
	if _, err := s.Verify(ctx, "no-existe"); !errorIs(err, ErrUnauthenticated) {
		t.Fatalf("token inventado: err = %v; want ErrUnauthenticated", err)
	}

	// Logout idempotente: revoke ×2 ok.
	if err := s.Revoke(ctx, res.Token); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if err := s.Revoke(ctx, res.Token); err != nil {
		t.Fatalf("revoke idempotente: %v", err)
	}

	// Revocado → 401.
	if _, err := s.Verify(ctx, res.Token); !errorIs(err, ErrUnauthenticated) {
		t.Fatalf("token revocado: err = %v; want ErrUnauthenticated", err)
	}
}

// errorIs: errors.Is a mano (los errores del fake se devuelven sin envolver;
// el wrapper del validate sí usa %w — recorrido manual que cubre ambos).
func errorIs(err, target error) bool {
	for {
		if err == target {
			return true
		}
		u, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
}
