# Skyland 🎲

Plataforma de apuestas multi-juego (latino tradicional: animalitos, caballos y más). Los jugadores recargan saldo vía pago móvil (Venezuela), apuestan en los juegos del catálogo y sus ganancias se acreditan al saldo.

## Stack

| Capa | Tecnología | Deploy |
|---|---|---|
| Frontend | Astro (público) + React SPA en `/app`, TailwindCSS, PWA | Cloudflare Pages |
| Backend | Python + FastAPI (monolito modular) + worker | Railway |
| Datos | PostgreSQL 16 (SQLAlchemy 2 + Alembic) · Upstash Redis/QStash | Railway / Upstash |
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

### Backend (`apps/api`)

```bash
cd apps/api
uv sync                 # instala dependencias
uv run alembic upgrade head   # migraciones
uv run uvicorn app.main:app --reload
# worker en otra terminal:
uv run python worker.py
```

Requiere Postgres local: `docker compose -f infra/docker-compose.dev.yml up -d` y `.env` (ver `.env.example`).

### Frontend (`apps/web`)

```bash
cd apps/web
npm install
npm run dev
npm run build   # genera el estático para Cloudflare Pages
```

## Reglas de oro

1. El **camino del dinero** (apuesta → débito → ledger) es una transacción ACID local en Postgres. Sin transacciones distribuidas.
2. Nada de lógica de dinero en Edge Functions / serverless.
3. Montos en mínima unidad + `currency` (multi-moneda lista desde el día 1).
4. Contrato front↔back generado desde OpenAPI (FastAPI → cliente TS). Cero tipos a mano.
5. RNG CSPRNG con `secrets` + `pg_advisory_lock` en sorteos.