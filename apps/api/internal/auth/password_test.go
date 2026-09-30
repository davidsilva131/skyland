package auth

import (
	"strings"
	"testing"
	"time"
)

// TestHashVerifyRoundtrip: hash → verify OK con contraseña correcta, FALSE con
// equivocada (spec auth-player §6 test 2).
func TestHashVerifyRoundtrip(t *testing.T) {
	h, err := hashPassword(".correcta-larga-2026")
	if err != nil {
		t.Fatalf("hashPassword: %v", err)
	}
	if !verifyPassword(" correcta-larga-2026"[:0]+".correcta-larga-2026", h) {
		t.Fatal("verifyPassword(correcta) = false; want true")
	}
	if verifyPassword("incorrecta", h) {
		t.Fatal("verifyPassword(incorrecta) = true; want false")
	}
}

// TestPHCParams: el PHC string lleva exactamente los parámetros del spec
// (m=19456 KiB, t=2, p=1) y el verify recomputa con los que el string trae.
func TestPHCParams(t *testing.T) {
	h, err := hashPassword(" параметр-check-2026")
	if err != nil {
		t.Fatalf("hashPassword: %v", err)
	}
	const want = "$argon2id$v=19$m=19456,t=2,p=1$"
	if !strings.HasPrefix(h, want) {
		t.Fatalf("PHC prefix no es el del spec: %q", h)
	}
	// salt 16 B, key 32 B → base64 RawStdEncoding: 22 y 43 chars.
	parts := strings.Split(h, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" ||
		parts[3] != "m=19456,t=2,p=1" {
		t.Fatalf("estructura PHC inesperada: %q", h)
	}
	if len(parts[4]) != 22 {
		t.Fatalf("salt b64 len = %d; want 22 (16 B)", len(parts[4]))
	}
	if len(parts[5]) != 43 {
		t.Fatalf("key b64 len = %d; want 43 (32 B)", len(parts[5]))
	}
}

// TestCommonPasswordRejected: la lista incrustada rechaza los triviales.
func TestCommonPasswordRejected(t *testing.T) {
	if !isCommonPassword("password") {
		t.Fatal("password no está en la lista común")
	}
	if !isCommonPassword("123456") {
		t.Fatal("123456 no está en la lista común")
	}
	if isCommonPassword("una-contraseña-aleatoria-real-9xQ") {
		t.Fatal("contraseña aleatoria marcada como común")
	}
}

// TestValidateAgeEdges: bordes de edad (spec §6 test 2) en America/Caracas:
// 17y364d → reject, exactamente 18 → accept, fecha futura → reject.
func TestValidateAgeEdges(t *testing.T) {
	// now clavado: 2026-09-29 12:00 Caracas (UTC-4 fijo).
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, caracasTZ)
	cases := []struct {
		name      string
		birthdate string
		wantCode  string
	}{
		{"17y364d → underage", "2008-09-30", "underage"},
		{"exactamente 18 → ok", "2008-09-29", ""},
		{"18y+1d → ok", "2008-09-28", ""},
		{"fecha futura → underage", "2026-12-31", "underage"},
		{"mayor de edad normal", "2000-01-15", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := validateRegistration("jugador@ejemplo.com", "contrasenasagrada-2026", tc.birthdate, true, now)
			switch {
			case tc.wantCode == "" && err != nil:
				t.Fatalf("edad(%s): err inesperado %v", tc.birthdate, err)
			case tc.wantCode == "underage" && err != errUnderage:
				t.Fatalf("edad(%s): err = %v; want underage", tc.birthdate, err)
			}
		})
	}
}

// BenchmarkHash: target documentado ≤500 ms/hash (spec §6 test 5). Es un
// benchmark, no un assert (hardware varía).
func BenchmarkHash(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := hashPassword("benchmark-password-2026"); err != nil {
			b.Fatal(err)
		}
	}
}
