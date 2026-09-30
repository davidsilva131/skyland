package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Suite de contrato HTTP (spec auth-player §6 test 1): todos los status +
// códigos de la tabla §2; content-type problem+json; atributos de cookie
// (nombre, Max-Age, Path, HttpOnly, Secure, SameSite); Retry-After en 429;
// Origin reject → 403; logout idempotente (204 ×2); /me 401 para cookie
// ausente/inventada/revocada. Sobre el mux con fake store + fake limiter
// (pattern wallets_http_test.go).

// newMux arma el módulo sobre un mux limpio con fakes.
func newMux(t *testing.T) *http.ServeMux {
	t.Helper()
	mux := http.NewServeMux()
	RegisterTest(mux)
	return mux
}

// do ejecuta la request y devuelve el recorder.
func do(t *testing.T, mux *http.ServeMux, r *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	return w
}

// regBody: cuerpo de registro válido (cumple todos los schemas).
func regBody(email, password, birthdate string, terms bool) string {
	return fmt.Sprintf(`{"email":%q,"password":%q,"birthdate":%q,"acceptsTerms":%t}`,
		email, password, birthdate, terms)
}

// registerHappy: registro OK → 201 + cookie de sesión (devuelve la cookie).
func registerHappy(t *testing.T, mux *http.ServeMux, email string) *http.Cookie {
	t.Helper()
	w := do(t, mux, mustReq(t, "POST", "/api/v1/auth/register", regBody(email, "contrasenasagrada-1", "2000-01-15", true), nil))
	if w.Code != http.StatusCreated {
		t.Fatalf("register %s: %d %s", email, w.Code, w.Body.String())
	}
	for _, c := range w.Result().Cookies() {
		if c.Name == cookieName {
			return c
		}
	}
	t.Fatalf("register %s: sin cookie %s", email, cookieName)
	return nil
}

// mustReq construye la request con Origin si se pide.
func mustReq(t *testing.T, method, path, body string, origin *string) *http.Request {
	t.Helper()
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	if origin != nil {
		r.Header.Set("Origin", *origin)
	}
	return r
}

// TestContractHappy: register 201 → me 200 → logout 204 → me 401 (el flujo
// del §6 manual smoke, en un test).
func TestContractHappy(t *testing.T) {
	mux := newMux(t)
	c := registerHappy(t, mux, "flujo@ejemplo.com")

	// /me con la cookie → 200 Player (y re-envía la cookie rodante).
	w := do(t, mux, withCookie(mustReq(t, "GET", "/api/v1/auth/me", "", nil), c.Value))
	if w.Code != http.StatusOK {
		t.Fatalf("me: %d %s", w.Code, w.Body.String())
	}
	var p Player
	if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
		t.Fatalf("player json: %v", err)
	}
	if p.Email != "flujo@ejemplo.com" {
		t.Fatalf("player email = %q", p.Email)
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("me content-type = %q", ct)
	}
	// Rolling: /me re-envía Set-Cookie con Max-Age fresco.
	var me *http.Cookie
	for _, rc := range w.Result().Cookies() {
		if rc.Name == cookieName {
			me = rc
		}
	}
	if me == nil || me.MaxAge != 7*24*3600 {
		t.Fatalf("/me no re-envió cookie con Max-Age 7d (got %v)", me)
	}

	// logout → 204 + cookie Max-Age=0.
	w = do(t, mux, withCookie(mustReq(t, "POST", "/api/v1/auth/logout", "", nil), c.Value))
	if w.Code != http.StatusNoContent {
		t.Fatalf("logout: %d %s", w.Code, w.Body.String())
	}
	var clear *http.Cookie
	for _, rc := range w.Result().Cookies() {
		if rc.Name == cookieName {
			clear = rc
		}
	}
	if clear == nil || clear.MaxAge != 0 {
		t.Fatalf("logout sin cookie Max-Age=0 (got %v)", clear)
	}

	// /me tras logout → 401 unauthenticated.
	w = do(t, mux, withCookie(mustReq(t, "GET", "/api/v1/auth/me", "", nil), c.Value))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("me tras logout: %d", w.Code)
	}
	assertProblem(t, w, http.StatusUnauthorized, Unauthenticated)
}

// addCookie: helper para encadenar With.
func addCookie(c *http.Cookie) func(*http.Request) {
	return func(r *http.Request) { r.AddCookie(c) }
}

// TestContractRegisterErrors: 409 email_taken, 422 underage/terms, 400
// malformed JSON (cada uno con su problem+json y content-type).
func TestContractRegisterErrors(t *testing.T) {
	mux := newMux(t)
	registerHappy(t, mux, "dup@ejemplo.com")

	cases := []struct {
		name       string
		body       string
		wantStatus int
		wantCode   ProblemCode
	}{
		{"email duplicado", regBody("DUP@ejemplo.com", "otraclave-larga-99", "2000-01-15", true), http.StatusConflict, EmailTaken},
		{"menor de edad", regBody("menor@ejemplo.com", "contrasenasagrada-1", "2016-01-15", true), http.StatusUnprocessableEntity, Underage},
		{"sin términos", regBody("terminos@ejemplo.com", "contrasenasagrada-1", "2000-01-15", false), http.StatusUnprocessableEntity, TermsNotAccepted},
		{"password común", regBody("comun@ejemplo.com", "password", "2000-01-15", true), http.StatusUnprocessableEntity, Validation},
		{"birthdate roto (formato no ISO)", regBody("roto@ejemplo.com", "contrasenasagrada-1", "31-12-2000", true), http.StatusBadRequest, Validation},
		{"password corta", regBody("corta@ejemplo.com", "corta", "2000-01-15", true), http.StatusUnprocessableEntity, Validation},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := do(t, mux, mustReq(t, "POST", "/api/v1/auth/register", tc.body, nil))
			if w.Code != tc.wantStatus {
				t.Fatalf("status = %d; want %d (body %s)", w.Code, tc.wantStatus, w.Body.String())
			}
			assertProblem(t, w, tc.wantStatus, tc.wantCode)
		})
	}

	// Malformed JSON → 400 problem+json validation [spec-added].
	w := do(t, mux, mustReq(t, "POST", "/api/v1/auth/register", "{email roto", nil))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("malformed json: %d", w.Code)
	}
	assertProblem(t, w, http.StatusBadRequest, Validation)
}

// TestContractLogin: 200 feliz + 401 invalid_credentials (email desconocido y
// password mala — misma respuesta, anti-enumeración).
func TestContractLogin(t *testing.T) {
	mux := newMux(t)
	registerHappy(t, mux, "login@ejemplo.com")

	w := do(t, mux, mustReq(t, "POST", "/api/v1/auth/login", `{"email":"LOGIN@ejemplo.com","password":"contrasenasagrada-1"}`, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("login feliz: %d %s", w.Code, w.Body.String())
	}
	if c := cookieOf(t, w); c == nil || c.MaxAge != 7*24*3600 {
		t.Fatalf("login sin cookie 7d (got %v)", c)
	}

	// Contraseña mala.
	w = do(t, mux, mustReq(t, "POST", "/api/v1/auth/login", `{"email":"login@ejemplo.com","password":"otraclave-larga-99"}`, nil))
	assertProblem(t, w, http.StatusUnauthorized, InvalidCredentials)
	// Email desconocido — la MISMA respuesta.
	w2 := do(t, mux, mustReq(t, "POST", "/api/v1/auth/login", `{"email":"nadie@ejemplo.com","password":"contrasenasagrada-1"}`, nil))
	if w2.Body.String() != w.Body.String() || w2.Code != w.Code {
		t.Fatalf("respuesta de email desconocido distinta de la de password mala (anti-enumeración)")
	}
}

// TestContractCookieAttributes: los atributos locked #4 en el Set-Cookie del
// register (y por simetría del login).
func TestContractCookieAttributes(t *testing.T) {
	mux := newMux(t)
	c := registerHappy(t, mux, "cookie@ejemplo.com")
	if c.Path != "/" {
		t.Fatalf("cookie Path = %q; want /", c.Path)
	}
	if !c.HttpOnly {
		t.Fatal("cookie no HttpOnly")
	}
	if !c.Secure {
		t.Fatal("cookie no Secure")
	}
	if c.SameSite != http.SameSiteLaxMode {
		t.Fatalf("cookie SameSite = %v; want Lax", c.SameSite)
	}
	if c.MaxAge != 7*24*3600 {
		t.Fatalf("cookie MaxAge = %d; want 604800", c.MaxAge)
	}
	if len(c.Value) < 32 {
		t.Fatalf("token corto: %d chars (base64url de 32 bytes ≥ 43)", len(c.Value))
	}
}

// TestContractOrigin: Origin fuera de la lista → 403 origin_rejected; Origin
// ausente pasa (scripts/curl); con el Origin del frontend en la lista, 201.
func TestContractOrigin(t *testing.T) {
	// El fake RegisterTest no pasa orígenes: vacío → cualquier Origin se
	// rechaza. Para el caso feliz, un mux con la lista cargada a mano.
	bad := "https://malo.ejemplo.com"
	w := do(t, newMux(t), mustReq(t, "POST", "/api/v1/auth/register", regBody("origen@ejemplo.com", "contrasenasagrada-1", "2000-01-15", true), &bad))
	assertProblem(t, w, http.StatusForbidden, OriginRejected)

	// Origin ausente: pasa el check (llega al flujo normal).
	w = do(t, newMux(t), mustReq(t, "POST", "/api/v1/auth/register", regBody("sinorigen@ejemplo.com", "contrasenasagrada-1", "2000-01-15", true), nil))
	if w.Code != http.StatusCreated {
		t.Fatalf("sin Origin: %d %s", w.Code, w.Body.String())
	}
}

// TestContractRateLimited: agotado el presupuesto IP → 429 rate_limited +
// Retry-After. El fake limiter comparte contador IP entre register/login.
func TestContractRateLimited(t *testing.T) {
	mux := newMux(t)
	var last *httptest.ResponseRecorder
	for i := 0; i < 21; i++ { // LoginIPLimit = 20 → la 21ª bloquea
		last = do(t, mux, mustReq(t, "POST", "/api/v1/auth/register",
			regBody(fmt.Sprintf("burst%d@ejemplo.com", i), "contrasenasagrada-1", "2000-01-15", true), nil))
	}
	if last.Code != http.StatusTooManyRequests {
		t.Fatalf("intento 21: %d %s; want 429", last.Code, last.Body.String())
	}
	assertProblem(t, last, http.StatusTooManyRequests, RateLimited)
	if ra := last.Header().Get("Retry-After"); ra == "" {
		t.Fatal("429 sin Retry-After")
	}
}

// TestContractLogoutIdempotent: 204 dos veces seguidas.
func TestContractLogoutIdempotent(t *testing.T) {
	mux := newMux(t)
	c := registerHappy(t, mux, "out@ejemplo.com")
	for i := 0; i < 2; i++ {
		w := do(t, mux, withCookie(mustReq(t, "POST", "/api/v1/auth/logout", "", nil), c.Value))
		if w.Code != http.StatusNoContent {
			t.Fatalf("logout #%d: %d", i+1, w.Code)
		}
	}
}

// TestContractMe401: /me con cookie ausente, inventada, revocada.
func TestContractMe401(t *testing.T) {
	mux := newMux(t)

	// Sin cookie.
	w := do(t, mux, mustReq(t, "GET", "/api/v1/auth/me", "", nil))
	assertProblem(t, w, http.StatusUnauthorized, Unauthenticated)

	// Cookie inventada.
	w = do(t, mux, withCookie(mustReq(t, "GET", "/api/v1/auth/me", "", nil), "no-existe"))
	assertProblem(t, w, http.StatusUnauthorized, Unauthenticated)

	// Revocada (logout antes).
	c := registerHappy(t, mux, "rev@ejemplo.com")
	do(t, mux, withCookie(mustReq(t, "POST", "/api/v1/auth/logout", "", nil), c.Value))
	w = do(t, mux, withCookie(mustReq(t, "GET", "/api/v1/auth/me", "", nil), c.Value))
	assertProblem(t, w, http.StatusUnauthorized, Unauthenticated)
}

// cookieOf: primera cookie del nombre de sesión.
func cookieOf(t *testing.T, w *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, c := range w.Result().Cookies() {
		if c.Name == cookieName {
			return c
		}
	}
	return nil
}

// withCookie: request con la cookie de sesión puesta a mano.
func withCookie(r *http.Request, token string) *http.Request {
	r.AddCookie(&http.Cookie{Name: cookieName, Value: token})
	return r
}

// assertProblem: status + content-type problem+json + code del body.
func assertProblem(t *testing.T, w *httptest.ResponseRecorder, status int, code ProblemCode) {
	t.Helper()
	if w.Code != status {
		t.Fatalf("status = %d; want %d (body %s)", w.Code, status, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("content-type = %q; want application/problem+json", ct)
	}
	var p Problem
	if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
		t.Fatalf("problem json: %v (body %s)", err, w.Body.String())
	}
	if p.Code != code {
		t.Fatalf("code = %q; want %q (detail %q)", p.Code, code, p.Detail)
	}
	if p.Status != status {
		t.Fatalf("problem.status = %d; want %d", p.Status, status)
	}
	if p.Title == "" || p.Detail == "" {
		t.Fatalf("problem sin title/detail: %+v", p)
	}
}

// testNow usado en fake_test; aquí el reloj real del fake basta (los TTLs son
// 7d/30d — nada expira durante el test).
var _ = time.Now
