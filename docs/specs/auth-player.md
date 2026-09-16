# Spec — Player auth, end-to-end (register / login / logout / me)

- **Status**: Implementable — hand to `/implement`.
- **Sources**: contract locked in #4 (grilling), Upstash client in #2 (research), session mechanism + hashing + no-verification in #1 (map). Visual direction: inherited from the Portada — #3 closed as not planned, the zinc-950 + amber-400 language stays, quality pass only.
- **Issue**: #5 · **Map**: #1.
- Anything this spec decides **beyond** what #4 locked is marked **[spec-added]**. Nothing here overrides #4.

## 1. Scope

End-to-end player auth: OpenAPI contract with codegen on both sides, Go `internal/auth` + `internal/ratelimit`, migration `000003` (`players` + `sessions`), `LoginForm` rewrite, SPA session gate + logout, and the `/terminos` page registration links to (the map flagged it as a blocker; a stub with structure ships here, final legal copy is David's).

**Non-goals** (locked): password recovery, email verification / OTP, admin login / RBAC, MFA, login-screen visual redesign, balance in auth payloads (wallets stays a separate fetch), single-session enforcement (concurrent sessions per player are allowed), wallets alignment — `balances`/`ledger_entries` keep `bigint jugador_id`; wiring `players.id` into wallets is #9's.

## 2. OpenAPI contract

Single source of truth: **`apps/api/openapi.yaml`** (new). Both sides generate from it. Zero hand-written auth types.

Generation:

- **Go**: oapi-codegen v2, std-http server mode, installed via `go get -tool` (Go 1.27). Config `apps/api/oapi-codegen.yaml`, output `internal/auth/oapi_gen.go` — the only generated Go file. Verify exact flags against the installed version.
- **TS**: `openapi-typescript` devDep in `apps/web`; script `"gen:api": "openapi-typescript ../../apps/api/openapi.yaml -o src/lib/api/types.gen.ts"`. Add `typescript` + `@astrojs/check` devDeps and `"check": "astro check"` — the type gate that makes "zero hand-written types" verifiable.

Base path `/api/v1`, JSON bodies. **Every error response is RFC 7807 `application/problem+json`**:

```
Problem { title: string, status: int, detail: string, code: string }
```

`detail` is Spanish (product language, parity with existing module errors). `code` enum: `email_taken` · `invalid_credentials` · `underage` · `terms_not_accepted` · `rate_limited` · `validation` · `unauthenticated` · `origin_rejected` **[spec-added]**.

| Endpoint | Request | Success | Errors |
|---|---|---|---|
| `POST /auth/register` | `{ email, password, birthdate, acceptsTerms }` | `201` + Player + Set-Cookie | `409 email_taken` · `422 validation / underage / terms_not_accepted` · `429 rate_limited` (shares the per-IP limiter) · `400 validation` (malformed JSON) |
| `POST /auth/login` | `{ email, password }` | `200` + Player + Set-Cookie | `401 invalid_credentials` · `429 rate_limited` · `400` |
| `POST /auth/logout` | — | `204` always, idempotent (clears cookie) | `403 origin_rejected` |
| `GET /auth/me` | — | `200` + Player (refreshes rolling TTL) | `401 unauthenticated` |

Component schemas:

- `RegisterRequest`: `email` (string, ≤254), `password` (string, 8–128), `birthdate` (string, `date`, `YYYY-MM-DD`), `acceptsTerms` (boolean, must be `true`).
- `LoginRequest`: `email`, `password`.
- `Player`: `{ id (uuid), email, birthdate (date), created_at (date-time) }` — no balance.
- `Problem`: as above.

Rules:

- **Register logs you in**: `201` sets the session cookie; the frontend makes no follow-up login call.
- Login never distinguishes bad email from bad password (`401 invalid_credentials`).
- `429` carries a `Retry-After` header (integer seconds) alongside the problem body.
- Malformed JSON → `400` problem+json `code: validation` **[spec-added]** (parity with the existing `decodeJSON` → 400 behavior).
- **Origin check** on all three POSTs: `Origin` header present and not in `SKYLAND_CORS_ORIGINS` → `403 origin_rejected` **[spec-added]**. Belt-and-braces CSRF; `SameSite=Lax` is the primary defense. Implies `SKYLAND_CORS_ORIGINS` must be set wherever the browser frontend runs (it already must be, for CORS).

Cookie (locked #4):

- Name `skyland_session`; value = base64url of 32 `crypto/rand` bytes (≥256-bit entropy).
- `HttpOnly; Secure; SameSite=Lax; Path=/`; Max-Age 7 days, absolute cap 30 days (`sessions.absolute_expires_at`).
- **Rolling is two-sided** **[spec-added]**: `/auth/me` refreshes the DB row **and re-sends `Set-Cookie`** with a fresh 7-day Max-Age — a browser-side cookie that only counts down from login would void the rolling TTL.
- Logout re-sets the same cookie with `Max-Age=0`.
- No dev-only flag branching: browsers accept `Secure` cookies on localhost.

## 3. Database — migration `000003`

Normative DDL (from #4):

```sql
BEGIN;
CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE players (
    id                uuid PRIMARY KEY,
    email             citext NOT NULL UNIQUE,
    birthdate         date NOT NULL,
    password_hash     text NOT NULL,              -- argon2id PHC string
    terms_accepted_at timestamptz NOT NULL,
    terms_version     text NOT NULL,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE sessions (
    id                  uuid PRIMARY KEY,
    player_id           uuid NOT NULL REFERENCES players(id),
    token_hash          bytea NOT NULL UNIQUE,    -- sha-256(token); never plaintext
    created_ip          inet NOT NULL,
    created_at          timestamptz NOT NULL DEFAULT now(),
    expires_at          timestamptz NOT NULL,     -- rolling
    absolute_expires_at timestamptz NOT NULL,
    revoked_at          timestamptz
);
COMMIT;
```

Down: `DROP TABLE sessions; DROP TABLE players; DROP EXTENSION IF EXISTS citext;`

Notes:

- `ponytail:` expired/revoked sessions accumulate until a cleanup job exists — manual `DELETE FROM sessions WHERE expires_at < now()`; upgrade path is a daily QStash job in `cmd/worker`.
- No index on `sessions.player_id`: every MVP query looks up by `token_hash` (unique index). Add one when the first player-scoped query exists.
- Email is trimmed + lowercased before insert; `citext` enforces the case-insensitive unique either way.

## 4. Go backend — `internal/auth` + `internal/ratelimit`

Module layout mirrors wallets: `Register(mux, …)` onto the shared httpapi mux, module-local HTTP helpers (do not refactor wallets' duplicated helpers in this effort), fake + pgx adapters behind one interface.

| File | Responsibility |
|---|---|
| `auth.go` | Service interface + types: `Register`, `Login`, `Verify`, `Revoke` (#9's language). `Player` struct. `TERMS_VERSION = "2026-09-01"` const — the future Terms page bumps it and re-acceptance becomes required. |
| `password.go` | argon2id: m=19456 KiB, t=2, p=1, salt 16 B, key 32 B, PHC string; constant-time verify. Embedded common-password list (~top 1000, SecLists, license header in-file), lowercase compare — exact cutoff not load-bearing. |
| `postgres.go` | pgx adapter: insert player (unique violation → `email_taken`), insert session, verify by `token_hash` (one indexed SELECT joining player), rolling-refresh UPDATE, revoke UPDATE. |
| `fake.go` | In-memory store for handler tests (and the dev fake server). |
| `http.go` | Handlers implementing the generated `ServerInterface`; problem+json writer; cookie set/clear; client-IP extraction; Origin check. |
| `oapi_gen.go` | Generated. Do not edit. |

Client IP **[spec-added]**: leftmost `X-Forwarded-For` (Railway sets it) → `RemoteAddr` host → `0.0.0.0` fallback. Same value feeds `created_ip` and the rate-limit key.

`internal/ratelimit/redis.go` — normative shape from #2:

```go
// Allow: fixed window, INCR + ExpireNX pipelined in 1 RTT.
func (l *Limiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error)
```

- go-redis/v9 over the TLS/RESP endpoint (`rediss://` from `SKYLAND_UPSTASH_REDIS_URL`); keys `rl:ip:<ip>` / `rl:email:<email>`.
- Budgets: login checks **both** (email 5/15 min, IP 20/15 min — the IP counter is shared with register); register checks IP only. Successful login `DEL rl:email:<email>`.
- Redis unreachable → **fail open** + `slog` warn **[spec-added]**: availability at draw-time beats brute-force margin during an Upstash blip.
- `Limiter` is an interface with an in-memory fake (handler tests stay deterministic; the Redis impl is a 4-line pipeline over a well-tested client — integration test optional, gated on `SKYLAND_TEST_REDIS_URL`).

Service semantics:

- **Register**: trim + lowercase email → validate (format ≤254; password 8–128 and not in the common list; birthdate parses `YYYY-MM-DD`, not future, ≥18 computed in `America/Caracas` **[spec-added: timezone]**; `acceptsTerms` true) → argon2 hash → insert player (`409` on taken) → create session → `201` + cookie.
- **Login**: rate-limit check **first** (it protects the argon2 cost) → fetch player by email; unknown email runs a dummy argon2 verify against a fixed hash **[spec-added: anti-enumeration timing]** → verify → create session → `200` + cookie + DEL email counter.
- **Verify** (`/me`): sha-256(token) → `SELECT … WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > now() AND absolute_expires_at > now()` → refresh `expires_at = least(now()+7d, absolute_expires_at)` → Player. Missing cookie, unknown/revoked/expired token → `401 unauthenticated`.
- **Revoke** (logout): `UPDATE … SET revoked_at = now() WHERE token_hash = $1 AND revoked_at IS NULL`; clear cookie; `204` regardless.
- New session row per login/register (no rotation — locked #4).

Wiring & config:

- `internal/httpapi/httpapi.go`: replace the `GET /api/v1/auth/status` stub with the real `auth.Register(...)` (parity with `wallets.Register`).
- `internal/platform/config`: **remove `JWTSecret`** and drop `SKYLAND_JWT_SECRET` from `.env.example` — obsolete under opaque tokens (ADR-0005). No new env vars: the Redis URL already exists; TTLs, budgets and the terms version are constants, not config (values that never change don't earn config).
- New Go deps, all mandated: promote `golang.org/x/crypto` (already indirect — argon2), `github.com/redis/go-redis/v9` (#2), oapi-codegen as a `go tool`. Nothing else.

## 5. Frontend — `apps/web`

**Same-origin API everywhere** (locked `SameSite=Lax` demands it):

- Dev: vite proxy in `astro.config.mjs` — `server.proxy` sends `/api` → `http://localhost:8080`. Dev stops depending on CORS entirely.
- Prod: `public/_redirects` gains `/api/* https://<railway-api-host>/api/:splat 200` (Cloudflare Pages proxy; the host is a deploy value).
- `PUBLIC_API_URL` stays unset → `API_BASE = '/api'`. The CORS middleware stays for direct-API consumers.

| File | Work |
|---|---|
| `src/lib/api/types.gen.ts` | Generated (§2). |
| `src/lib/api/auth.ts` | Typed client over the generated types: `register` / `login` / `logout` / `me`; `credentials: 'include'`; problem+json parsing → `{ code, detail }`; `/me` 401 surfaced to the caller. |
| `src/components/LoginForm.tsx` | Rewrite in place (single island, current zinc-950/amber-400 language — no redesign): login/register modes (keeps `?mode=register`), email, password with visibility toggle (`aria-label`), birthdate + Terms checkbox linking `/terminos` in a new tab (register only). States: idle / loading (submit disabled) / error (top box, code → copy below) / success → `/app`. Native input types carry the client hints; the server is the authority. |
| `src/components/app/session.tsx` | `SessionProvider` + `useSession`: `{ player, loading, logout() }`. Calls `/me` on mount; 401 → full-page redirect to `/login` (**not** a router navigate — `/login` is an Astro page outside the `/app` basename). |
| `src/components/app/App.tsx` | Gate runs first (ADR-0004: no lobby flash): loading → zinc-950 splash; no player → redirect. |
| `src/components/app/AppLayout.tsx` | Player email in the header + logout button (`logout()` → `/login`). Saldo stays as-is (wallets untouched). |
| `src/pages/terminos.astro` | Static page showing version `2026-09-01`. Structure: +18, juego responsable, sorteos propios (RNG), pago móvil, reglas de cuenta, privacidad. **Final legal copy is David's** — flagged with a TODO in the page. |

Error copy (code → Spanish, top-level box; `validation` shows the server `detail` verbatim):

| code | copy |
|---|---|
| `email_taken` | Ese correo ya tiene una cuenta. Entra con tu contraseña. |
| `invalid_credentials` | Correo o contraseña incorrectos. |
| `underage` | Debes tener 18 años o más para crear una cuenta. |
| `terms_not_accepted` | Debes aceptar los términos para continuar. |
| `rate_limited` | Demasiados intentos. Espera un momento y prueba de nuevo. |
| `unauthenticated` | Tu sesión expiró. Entra de nuevo. |
| network / other | No se pudo conectar con el servidor. Intenta más tarde. |

## 6. Test plan

Backend (`go test ./...`, `go vet`):

1. `http_test.go` — contract tests over the mux with fake store + fake limiter (`wallets_http_test.go` pattern): every status + code in the §2 table; problem+json content-type; cookie attributes (name, Max-Age, Path, HttpOnly, Secure, SameSite); `Retry-After` on 429; Origin reject → 403; logout idempotent (204 twice); `/me` 401 for missing / bogus / revoked / expired cookie.
2. `password_test.go` — hash/verify roundtrip; wrong password fails; params are exactly m/t/p/salt/key above; common-password rejected; age edges: 17y364d → reject, exactly-18 → accept, future date → reject.
3. `postgres_test.go` — gated on `SKYLAND_TEST_DATABASE_URL` (wallets pattern): taken email maps to 409; token stored hashed (plaintext appears nowhere); revoke → `/me` 401; rolling refresh respects the absolute cap.
4. `ratelimit` — fake-limiter budget logic (5th email attempt blocks, 21st IP attempt blocks, register counts against the IP budget); Redis impl integration optional via `SKYLAND_TEST_REDIS_URL`.
5. `BenchmarkHash` — documented target ≤ 500 ms/hash, a benchmark not a unit assert (hardware varies).

Frontend: `pnpm gen:api` regenerates cleanly; `pnpm check` (astro check) clean; `pnpm build:web` builds `/`, `/app`, `/login`, `/terminos`.

Manual smoke (docker Postgres + curl): register → `201` + `Set-Cookie` with the locked attributes → `/me` → `200` Player → logout → `204` → `/me` → `401`.

## 7. Work order (TDD at pre-agreed seams)

Seams pre-agreed here: the service interface (fake store) and the `Limiter` interface (fake limiter). RED against fakes, GREEN on real adapters.

1. `openapi.yaml` + codegen both sides — types compiling on both ends makes the contract real.
2. Migration `000003` up/down (golang-migrate).
3. `password.go` — `password_test.go` first.
4. `auth.go` service + `fake.go` — service tests first.
5. `http.go` handlers — contract tests first; cookies + Origin check.
6. `ratelimit` fake + `redis.go`, wired into login/register.
7. `postgres.go` — gated tests.
8. `httpapi` wiring; drop `JWTSecret`.
9. Frontend: types gen → `auth.ts` → `LoginForm` → session/gate/`AppLayout` → `/terminos` → `_redirects` + vite proxy.
10. Full gates once at the end: `go test ./...`, `go vet`, `pnpm check`, `pnpm build:web`, manual smoke.

## 8. Acceptance criteria

- [ ] `go test ./...` and `go vet` green (gated suites run when their env vars are set).
- [ ] `pnpm check` + `pnpm build:web` green; `/terminos` builds.
- [ ] Zero hand-written auth types in `apps/web`; `oapi_gen.go` is the only generated Go file.
- [ ] Manual smoke (§6) passes against local Postgres.
- [ ] ADR-0005 committed; `JWTSecret` gone from config and `.env.example`.
- [ ] Every **[spec-added]** decision survives David's review — they extend #4, never override it.
