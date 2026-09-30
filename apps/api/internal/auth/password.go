package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Parámetros argon2id (spec auth-player §4): m=19456 KiB, t=2, p=1,
// salt 16 B, key 32 B. Constantes: valores que nunca cambian no ganan config.
const (
	argonMemory  uint32 = 19456 // KiB (19 MiB)
	argonTime    uint32 = 2
	argonThreads uint8  = 1
	argonSaltLen        = 16
	argonKeyLen         = 32
)

// ErrCommonPassword: la contraseña está en la lista de triviales incrustada.
var ErrCommonPassword = errors.New("la contraseña es demasiado común")

// hashPassword genera el PHC string argon2id con los parámetros fijos del módulo.
func hashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("salt: %w", err)
	}
	key := argon2.IDKey([]byte(password), salt, argonMemory, argonTime, argonThreads, argonKeyLen)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argonMemory, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key)), nil
}

// verifyPassword compara en tiempo constante contra el PHC string.
// Un PHC válido de otro tipo de argon (i / d) se trata como contraseña
// incorrecta: nunca paniquea, nunca compara de más.
func verifyPassword(password, phc string) bool {
	parts := strings.Split(phc, "$") // ["", "argon2id", "v=19", "m=..,t=..,p=..", salt, key]
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false
	}
	var m, t uint32
	var p uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &m, &t, &p); err != nil {
		return false
	}
	salt, err := base64.RawStdEncoding.Strict().DecodeString(parts[4])
	if err != nil || len(salt) == 0 {
		return false
	}
	want, err := base64.RawStdEncoding.Strict().DecodeString(parts[5])
	if err != nil || len(want) == 0 {
		return false
	}
	got := argon2.IDKey([]byte(password), salt, m, t, p, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1
}

// dummyVerify quema el mismo coste de argon2 con una contraseña de nombre
// fijo: anti-enumeración — login de email desconocido tarda lo mismo que uno
// real (spec auth-player §4).
var (
	dummyHash, _ = hashPassword("dummy-password-anti-enumeration")
)

// dummyVerify corre una verificación argon2 completa contra dummyHash.
func dummyVerify() {
	_ = verifyPassword("wrong-password-anti-enumeration", dummyHash)
}

// tokenBytes son los 32 crypto/rand bytes de la cookie (≥256-bit entropy).
const tokenLen = 32

// newToken devuelve un token opaco fresco (base64url de 32 random bytes).
func newToken() (string, error) {
	b := make([]byte, tokenLen)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
