# ADR-0003 — Arquitectura: Monolito modular + Worker (sin microservicios en el MVP)

- **Fecha**: 2026-08-25
- **Estado**: Aceptado
- **Contexto**: Skyland necesita organizar frontend (Astro + React SPA en Cloudflare Pages) y backend (FastAPI + Postgres + Upstash en Railway) para soportar un catálogo de juegos extensible, picos de 100–500 apuestas/min, dinero real (ledger, sorteos, recargas) y un equipo de un solo desarrollador con presupuesto limitado (~$15–25/mes).
- **Decisión**:
  - **Monolito modular** (no microservicios): un solo proceso FastAPI con módulos por dominio — `auth`, `wallets` (saldo + ledger de doble entrada), `games` (motor de apuestas genérico con contratos `BetMarket`/`ResultSource` + juegos concretos), `payments` (recargas + `PagoVerifier` intercambiable), `notifications` (SSE, email, push), `admin` (RBAC soporte/admin). Los módulos se comunican por funciones internas, nunca por HTTP interno.
  - **Worker hermano**: segundo proceso con el mismo código (`worker.py`) para todo lo diferido: sorteos programados, cierre de apuestas, liquidación de pagos, confirmación de recargas. Orquestado con **QStash** (colas + cron). No es un servicio distribuido; comparte la misma base de datos y el mismo paquete de código.
  - **Regla de oro**: el camino del dinero (apuesta → débito → ledger) es una **transacción ACID local en Postgres**. Nada de transacciones distribuidas, nada de Redis en el camino crítico.
  - **Contrato front↔back**: FastAPI genera **OpenAPI** → el frontend genera su cliente TypeScript con `openapi-typescript`. Un solo contrato, cero drift entre Python y TS.
  - **Estructura del repo** (monorepo):
    ```
    skyland/
    ├── apps/
    │   ├── web/        # Astro (público) + React SPA en /app
    │   └── api/        # FastAPI (app/) + worker.py + migraciones Alembic
    ├── infra/          # docker-compose.dev.yml (Postgres local), Dockerfiles
    ├── docs/           # ADRs + agent docs
    ├── AGENTS.md · CONTEXT.md
    ```
  - **Despliegue**: una imagen Docker → Railway: servicio `api` (uvicorn) + servicio `worker` (mismo build). Migraciones Alembic como job del pipeline. Frontend → Cloudflare Pages.
- **Por qué**: Con un dev solo y dinero real, el monolito minimiza la superficie operativa y garantiza integridad transaccional; el worker separa lo síncrono de lo diferido sin partir el dominio. Microservicios se evaluarán solo si aparece un dominio con requisitos de escala/deploy independientes (plan de salida: el módulo `games` es el primer candidato a extracción, sus contratos ya están aislados).
- **Consecuencias**: Un solo punto de despliegue (simple); la separación módulo→servicio futuro es mecánica; el worker debe vigilarse para no dormirse (Railway: sin sleep en worker). QStash como única cola del sistema.