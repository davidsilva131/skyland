# Skyland 🎲

Plataforma de apuestas multi-juego (latino tradicional: animalitos, caballos y más). Los jugadores recargan saldo vía pago móvil (Venezuela), apuestan en los juegos del catálogo y sus ganancias se acreditan al saldo.

## Stack

| Capa | Tecnología | Deploy |
|---|---|---|
| Frontend | Astro (público) + React SPA en `/app`, TailwindCSS, PWA | Cloudflare Pages |
| Backend | Go 1.27 (monolito modular + worker, pgx/v5) | Railway |
| Datos | PostgreSQL 16 (pgx + golang-migrate) · Upstash Redis/QStash | Railway / Upstash |
| Tiempo real | SSE (contador de sorteo, resultados) | — |
| CI/CD | GitHub Actions → CF Pages + Railway | — |

Decisiones de arquitectura en [`docs/adr/`](docs/adr/). Glosario del dominio en [`CONTEXT.md`](CONTEXT.md).

## Estructura

```
apps/
├── web/     # Astro + React (landing, login, SPA /app: lobby, juegos, backoffice)
└── api/     # FastAPI: app/ (módulos auth, wallets, games, payments, notifications, admin) + worker.py
infra/       # docker-compose.dev.yml (Postgres local), Dockerfiles
docs/        # ADRs + docs de agentes
```

## Desarrollo

### Backend (`apps/api` — Go)

```bash
# requiere Postgres local: docker compose -f infra/docker-compose.dev.yml up -d
# y apps/api/.env (ver .env.example)

cd apps/api
go test ./...                       # tests
go run ./cmd/migrate up  (o: migrate -path migrations -database "$SKYLAND_DATABASE_URL" up)
go run ./cmd/api                    # API en :8080 (usa PORT para cambiar)
go run ./cmd/worker                 # worker en otra terminal
```

Código en `cmd/{api,worker}` + `internal/{auth,wallets,games,payments,notifications,admin}` (monolito modular — ADR-0003). Migraciones: golang-migrate (`migrations/`).

### Frontend (`apps/web`)

```bash
pnpm install          # instala dependencias (workspaces desde la raíz)
pnpm dev:web          # dev server desde la raíz (o cd apps/web && pnpm dev)
pnpm build:web        # genera el estático para Cloudflare Pages
```

## Reglas de oro

1. El **camino del dinero** (apuesta → débito → ledger) es una transacción ACID local en Postgres. Sin transacciones distribuidas.
2. Nada de lógica de dinero en Edge Functions / serverless.
3. Montos en mínima unidad + `currency` (multi-moneda lista desde el día 1).
4. Contrato front↔back: OpenAPI spec-first (`oapi-codegen` → Go, `openapi-typescript` → TS). Cero tipos a mano.
5. RNG CSPRNG con `crypto/rand` + `pg_advisory_lock` en sorteos.