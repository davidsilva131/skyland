package auth

import (
	"errors"
	"net/mail"
	"strings"
	"time"
)

// TERMS_VERSION: fuente de verdad de la versión de términos. terminos.astro
// muestra el mismo valor; bump → cambio de copy + re-aceptación obligatoria
// (spec auth-player §4 auth.go).
const TERMS_VERSION = "2026-09-01"

// Errores de validación de registro: códigos del enum ProblemCode. Los
// sentinels viajan del validate al handler; el copy español del detail vive
// en http.go (problemDetail).
var (
	errBadEmail     = errors.New("correo inválido: revisa el formato")
	errBadPassword  = errors.New("la contraseña debe tener 8 a 128 caracteres y no ser una común")
	errBadBirthdate = errors.New("fecha de nacimiento inválida: usa el formato AAAA-MM-DD")
	errUnderage     = errors.New("underage: Debes tener 18 años o más para crear una cuenta")
	errTerms        = errors.New("terms_not_accepted: Debes aceptar los términos para continuar")
)

// validationCode¿?: el error es un 422 del validate con código del enum —
// devuelve (code, detail, true); los sentinels con prefijo "code: detail"
// usan el prefijo como código.
func validationCode(err error) (ProblemCode, string, bool) {
	switch err {
	case errBadEmail:
		return Validation, errBadEmail.Error(), true
	case errBadPassword:
		return Validation, errBadPassword.Error(), true
	case errBadBirthdate:
		return Validation, errBadBirthdate.Error(), true
	}
	// underage / terms_not_accepted.
	if s := err.Error(); len(s) > 0 {
		if i := indexByteN(s, ':'); i > 0 {
			return ProblemCode(s[:i]), s[i+2:], true
		}
	}
	return "", "", false
}

func indexByteN(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

// Zona del corte de edad: America/Caracas es UTC-4 fijo (Venezuela eliminó el
// DST en 2016) — hardcode -04:00 para no depender del TZ del proceso.
var caracasTZ = time.FixedZone("America/Caracas", -4*3600)

// validateRegistration valida email/password/birthdate/acceptsTerms del borde
// (sin tocar la base). Devuelve el email normalizado (trim + lowercase) o uno
// de los sentinels de arriba. now inyectable: los tests clavan el reloj. Un
// birthday futuro sale > corte → underage (cubre también la fecha futura).
func validateRegistration(email, password, birthdate string, acceptsTerms bool, now time.Time) (normalizedEmail string, err error) {
	// Email: trim + lowercase (spec §3), formato + ≤254 (schema).
	e := strings.ToLower(strings.TrimSpace(email))
	if _, err := mail.ParseAddress(e); err != nil || len(e) > 254 {
		return "", errBadEmail
	}

	// Password: 8–128 y no trivial (spec §4 password.go, lista SecLists).
	if l := len(password); l < 8 || l > 128 {
		return "", errBadPassword
	}
	if isCommonPassword(strings.ToLower(password)) {
		return "", errBadPassword
	}

	// Birthdate: YYYY-MM-DD estricto (schema format:date normativo).
	b, err := time.ParseInLocation("2006-01-02", birthdate, caracasTZ)
	if err != nil {
		return "", errBadBirthdate
	}

	// Edad ≥18 computada en America/Caracas [spec-added: timezone].
	// Corte = 23:59:59 del 18º cumpleaños: nacido ese día o antes → ≥18;
	// nacido después → underage (cubre también fecha futura).
	today := now.In(caracasTZ)
	cutoff := time.Date(today.Year()-18, today.Month(), today.Day(), 23, 59, 59, 0, caracasTZ)
	if b.After(cutoff) {
		return "", errUnderage
	}

	if !acceptsTerms {
		return "", errTerms
	}
	return e, nil
}
