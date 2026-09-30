package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/davidsilva131/skyland/apps/api/internal/ratelimit"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// handler implementa el ServerInterface generado (oapi_gen.go): solo HTTP —
// cookies, problem+json, Origin check, client IP, rate-limit y códigos del
// enum. El service (fake/pgx) no sabe nada de HTTP.
type handler struct {
	svc     Service
	limiter ratelimit.Limiter
	delim   ratelimit.Deleter // resetea el contador de email en login OK
	logger  *slog.Logger
	allowed map[string]bool // SKYLAND_CORS_ORIGINS para el Origin check
	now     func() time.Time
}

// RegisterRoutes registra las rutas del módulo (spec §4: espejo wallets —
// Register sobre el mux compartido; los helpers HTTP son module-local).
// BaseURL "/api/v1": el contrato (openapi.yaml servers: /) se sirve bajo el
// prefijo del monolito — parity con wallets (`/api/v1/wallets/...`).
func RegisterRoutes(mux *http.ServeMux, svc Service, limiter ratelimit.Limiter, logger *slog.Logger, origins []string) {
	allowed := map[string]bool{}
	for _, o := range origins {
		allowed[o] = true
	}
	// Deleter: el limiter de Redis lo trae (reset del contador de email en
	// login OK); el fake lo satisface vía el cast de abajo (no-op in-memory).
	h := &handler{svc: svc, limiter: limiter, delim: deleter(limiter), logger: logger, allowed: allowed, now: time.Now}
	HandlerFromMuxWithBaseURL(h, mux, "/api/v1")
}

// deleter: el Deleter del limiter si lo trae; no-op si no (fake).
func deleter(l ratelimit.Limiter) ratelimit.Deleter {
	if d, ok := l.(ratelimit.Deleter); ok {
		return d
	}
	return noopDeleter{}
}

type noopDeleter struct{}

func (noopDeleter) Del(context.Context, string) error { return nil }

// writeProblem escribe el problem+json RFC 7807: title/status/detail/code.
func writeProblem(w http.ResponseWriter, status int, code ProblemCode, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Problem{
		Code:   code,
		Title:  problemTitle(code),
		Detail: detail,
		Status: status,
	})
}

// problemTitle: título corto por código (el detail lleva el copy completo).
func problemTitle(code ProblemCode) string {
	switch code {
	case EmailTaken:
		return "Correo ya registrado"
	case InvalidCredentials:
		return "Credenciales inválidas"
	case Underage:
		return "Edad mínima"
	case TermsNotAccepted:
		return "Términos de servicio"
	case RateLimited:
		return "Límite de intentos"
	case Validation:
		return "Datos inválidos"
	case Unauthenticated:
		return "Sesión requerida"
	case OriginRejected:
		return "Origen no permitido"
	}
	return string(code)
}

// clientIP: leftmost X-Forwarded-For → RemoteAddr host → "0.0.0.0" (spec §4
// [spec-added]). El mismo valor feedea created_ip y la key del rate-limit.
//
//	ponytail: asume que el edge proxy SETEA XFF al cliente real; si lo
//	APPENDEA a un header spoofable, cambiar a rightmost — verificar con un
//	curl contra la API desplegada antes de fiarse del limiter por-IP.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i >= 0 {
			xff = xff[:i]
		}
		if ip := strings.TrimSpace(xff); ip != "" {
			return ip
		}
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	if r.RemoteAddr != "" && !strings.Contains(r.RemoteAddr, ":") {
		return r.RemoteAddr
	}
	return "0.0.0.0"
}

// sessionToken lee el token de la cookie skyland_session ("" si no está).
func sessionToken(r *http.Request) string {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return ""
	}
	return c.Value
}

// decodeJSON decodifica el cuerpo en v; malformed JSON → error → 400
// problem+json validation [spec-added], parity con decodeJSON de wallets.
func decodeJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("json inválido: %w", err)
	}
	return nil
}

// birthdateString convierte la fecha generada a "YYYY-MM-DD" y rechaza el
// valor cero: oapi-codegen no aplica `required` a campos del body JSON, así
// que una petición sin birthdate llega como Date{} → "0001-01-01", burlando
// el corte de +18 (code review #9/#11). "" → 422 validation.
func birthdateString(d openapi_types.Date) string {
	if d.Time.IsZero() {
		return ""
	}
	return d.Time.Format("2006-01-02")
}

// tooMany: 429 problem+json rate_limited + Retry-After (segundos, cap 60).
func (h *handler) tooMany(w http.ResponseWriter) {
	w.Header().Set("Retry-After", "60")
	writeProblem(w, http.StatusTooManyRequests, RateLimited,
		"Demasiados intentos. Espera un momento y prueba de nuevo.")
}

// guard: Origin check + presupuesto de IP (contador compartido con register
// — spec §4). false = respuesta ya escrita.
func (h *handler) guard(w http.ResponseWriter, r *http.Request) bool {
	if origin := r.Header.Get("Origin"); origin != "" && !h.allowed[origin] {
		writeProblem(w, http.StatusForbidden, OriginRejected,
			"Petición no permitida desde este origen.")
		return false
	}
	ok, err := h.limiter.Allow(r.Context(), ratelimit.IPKey(clientIP(r)), ratelimit.LoginIPLimit, ratelimit.AttemptWindow)
	if err != nil {
		h.logger.Warn("ratelimit: error consultando presupuesto IP", "err", err)
		// fail-open: error del limiter no bloquea al jugador (spec §4).
	}
	if !ok {
		h.tooMany(w)
		return false
	}
	return true
}

// Register POST /auth/register: rate-limit IP → validate → 201 + cookie.
func (h *handler) Register(w http.ResponseWriter, r *http.Request) {
	if !h.guard(w, r) {
		return
	}
	var req RegisterJSONRequestBody
	if err := decodeJSON(r, &req); err != nil {
		writeProblem(w, http.StatusBadRequest, Validation,
			"JSON inválido: revisa los datos del formulario.")
		return
	}
	res, err := h.svc.Register(r.Context(), string(req.Email), req.Password,
		birthdateString(req.Birthdate), bool(req.AcceptsTerms), h.now(), clientIP(r))
	if err != nil {
		if code, detail, ok := validationCode(err); ok {
			writeProblem(w, http.StatusUnprocessableEntity, code, detail)
			return
		}
		if errors.Is(err, ErrEmailTaken) {
			writeProblem(w, http.StatusConflict, EmailTaken,
				"Ese correo ya tiene una cuenta. Entra con tu contraseña.")
			return
		}
		h.logger.Error("auth: register", "err", err)
		writeProblem(w, http.StatusInternalServerError, Validation,
			"Error del servidor. Intenta más tarde.")
		return
	}
	setSession(w, res.Token)
	writeJSON(w, http.StatusCreated, res.Player)
}

// Login POST /auth/login: presupuesto IP (guard) + email, luego service.
// Login exitoso DEL el contador de email (spec §4).
func (h *handler) Login(w http.ResponseWriter, r *http.Request) {
	if !h.guard(w, r) {
		return
	}
	var req LoginJSONRequestBody
	if err := decodeJSON(r, &req); err != nil {
		writeProblem(w, http.StatusBadRequest, Validation,
			"JSON inválido: revisa los datos del formulario.")
		return
	}
	email := strings.ToLower(strings.TrimSpace(string(req.Email)))
	if ok, _ := h.limiter.Allow(r.Context(), ratelimit.EmailKey(email), ratelimit.LoginEmailLimit, ratelimit.AttemptWindow); !ok {
		h.tooMany(w)
		return
	}
	res, err := h.svc.Login(r.Context(), email, req.Password, h.now())
	if err != nil {
		// Solo las credenciales malas son 401; cualquier otro error (DB
		// caída, etc.) es 500 — nunca enmascarado como contraseña mala.
		if errors.Is(err, ErrInvalidCredentials) {
			// sin PII: el email no entra al log (code review #9/#11).
			h.logger.Info("auth: credenciales inválidas", "ip", clientIP(r))
			writeProblem(w, http.StatusUnauthorized, InvalidCredentials,
				"Correo o contraseña incorrectos.")
			return
		}
		h.logger.Error("auth: login", "err", err)
		writeProblem(w, http.StatusInternalServerError, Validation,
			"Error del servidor. Intenta más tarde.")
		return
	}
	if err := h.delim.Del(r.Context(), ratelimit.EmailKey(email)); err != nil {
		h.logger.Warn("ratelimit: reset del contador de email falló", "err", err)
	}
	setSession(w, res.Token)
	writeJSON(w, http.StatusOK, res.Player)
}

// Logout POST /auth/logout: revoca y limpia; 204 pase lo que pase
// (idempotente — spec §2/§4).
func (h *handler) Logout(w http.ResponseWriter, r *http.Request) {
	if origin := r.Header.Get("Origin"); origin != "" && !h.allowed[origin] {
		writeProblem(w, http.StatusForbidden, OriginRejected,
			"Petición no permitida desde este origen.")
		return
	}
	if token := sessionToken(r); token != "" {
		if err := h.svc.Revoke(r.Context(), token); err != nil {
			// El error real queda en el log: 204 pase lo que pase.
			h.logger.Warn("auth: revoke", "err", err)
		}
	}
	clearSession(w)
	w.WriteHeader(http.StatusNoContent)
}

// Me GET /auth/me: verify + rolling refresh (DB en el adapter) + re-send de
// la cookie con Max-Age fresco (TTL rodante dos lados, spec §2).
func (h *handler) Me(w http.ResponseWriter, r *http.Request) {
	token := sessionToken(r)
	if token == "" {
		writeProblem(w, http.StatusUnauthorized, Unauthenticated,
			"Tu sesión expiró. Entra de nuevo.")
		return
	}
	p, err := h.svc.Verify(r.Context(), token)
	if err != nil {
		writeProblem(w, http.StatusUnauthorized, Unauthenticated,
			"Tu sesión expiró. Entra de nuevo.")
		return
	}
	setSession(w, token)
	writeJSON(w, http.StatusOK, p)
}

// Cookie helpers (atributos locked #4: HttpOnly; Secure; SameSite=Lax;
// Path=/; Max-Age 7d; logout re-set con Max-Age=0).
const cookieName = "skyland_session"

func setSession(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   7 * 24 * 3600,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   0,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

// writeJSON módulo-local (ponytail: helper duplicado #3 — extraer el
// compartido en el próximo touch de cualquiera de los tres, spec §4).
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// Register arma el módulo auth en prod (pattern wallets.Register): Postgres
// store + limiter Redis sobre el mux compartido.
func Register(mux *http.ServeMux, svc Service, limiter ratelimit.Limiter, logger *slog.Logger, origins []string) {
	RegisterRoutes(mux, svc, limiter, logger, origins)
}

// RegisterTest arma el módulo con fake store + fake limiter (tests/dev sin
// DB — pattern wallets.RegisterTest).
func RegisterTest(mux *http.ServeMux) {
	RegisterRoutes(mux, NewFake(), ratelimit.NewFake(), slog.Default(), nil)
}
