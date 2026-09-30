package auth

import (
	"context"
	"errors"
	"time"
)

// Errores de dominio (spec auth-player §4): el handler los traduce a los
// códigos del enum ProblemCode (uno a uno; el copy español del detail vive en
// http.go).
var (
	// ErrEmailTaken: el correo ya tiene una cuenta → 409.
	ErrEmailTaken = errors.New("ese correo ya tiene una cuenta")
	// ErrInvalidCredentials: login malo → 401 (nunca distingue email de password).
	ErrInvalidCredentials = errors.New("correo o contraseña incorrectos")
	// ErrUnauthenticated: sesión ausente/desconocida/expirada/revocada → 401.
	ErrUnauthenticated = errors.New("sesión expirada o inválida")
	// ErrValidation cubre 422 genérico (schema: validation/underage/terms).
	ErrValidation = errors.New("datos inválidos")
)

// SessionResult: token opaco (va en la cookie skyland_session) + el Player
// del schema generado (oapi_gen.go) que la respuesta serializa tal cual —
// cero tipos hand-written en el borde del API.
type SessionResult struct {
	Token  string
	Player Player
}

// Service es el servicio de auth (spec §4 auth.go): register / login / verify
// / revoke — el lenguaje de #9. Adapters: fake (tests/dev) + postgres (prod).
//
//	ponytail: una interface, dos adapters planos; un tercer proveedor
//	obligaría a revisar el split, no antes.
type Service interface {
	// Registro: valida → hash argon2id → insert player → crea sesión (loguea
	// al registro, spec §2). Devuelve códigos del enum o ErrEmailTaken.
	// createdIP es el IP del cliente (feeds sessions.created_ip, spec §4).
	Register(ctx context.Context, email, password, birthdate string, acceptsTerms bool, now time.Time, createdIP string) (SessionResult, error)
	// Login: rate-limit fuera de aquí (http.go); unknown email quema argon2
	// contra un hash fijo (anti-enumeración) y devuelve ErrInvalidCredentials.
	Login(ctx context.Context, email, password string, now time.Time) (SessionResult, error)
	// Verify por token opaco: TTL rodante dos lados (spec §2: DB refresh +
	// re-send cookie). ErrUnauthenticated si no hay sesión válida.
	Verify(ctx context.Context, token string) (Player, error)
	// Revoke logout: idempotente (204 pase lo que pase).
	Revoke(ctx context.Context, token string) error
}
