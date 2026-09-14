# ADR-0003 — Arquitectura: Monolito modular + Worker (sin microservicios en el MVP)

- **Fecha**: 2026-09-14 (Rev. 2)
- **Estado**: Aceptado — Rev. 2 (módulos Go)
- **Contexto**: Skyland necesita organizar frontend (Astro + React SPA en Cloudflare Pages) y backend (Go + Postgres + Upstash en Railway) para soportar un catálogo de juegos extensible, picos de 100–500 apuestas/min, dinero real (ledger, sorteos, recargas) y un equipo de un solo desarrollador con presupuesto limitado (~$15–25/mes). El backend se decidió primero en Python/FastAPI (Rev. 1) y se migró a Go 1.27 en septiembre 2026 (ADR-0001 Rev. 4) por corrección del dinero vía compilador, goroutines para SSE y menor huella de RAM en Railway.
- **Decisión**:
  - **Monolito modular** (no microservicios): un solo proceso Go con módulos por dominio — `auth` (sesiones httpOnly revocables + RBAC), `wallets` (saldo + ledger de doble entrada), `games` (motor de apuestas genérico con contratos `BetMarket`/`ResultSource` + juegos concretos), `payments` (recargas + `PagoVerifier` intercambiable), `notifications` (SSE, email, push), `admin` (RBAC soporte/admin). Los módulos se comunican por funciones internas, nunca por HTTP interno. Paquete `internal/` de Go impone el límite (nada externo importa módulos internos).
  - **Worker hermano**: segundo proceso con el mismo binario (`cmd/worker`) para todo lo diferido: sorteos programados, cierre de apuestas, liquidación de pagos, confirmación de recargas. Orquestado con **QStash** (colas + cron). No es un servicio distribuido; comparte la misma base de datos y el mismo paquete de código.
  - **Regla de oro**: el camino del dinero (apuesta → débito → ledger) es una **transacción ACID local en Postgres**. Nada de transacciones distribuidas, nada de Redis en el camino crítico.
  - **Contrato front↔back**: **OpenAPI spec-first** — el YAML es la fuente de verdad; `oapi-codegen` genera los tipos e interfaces del servidor Go, y `openapi-typescript` genera el cliente TypeScript del frontend. Un solo contrato, cero tipos a mano en TS.
  - **Estructura del repo** (monorepo):
    ```
    skyland/
    ├── apps/
    │   ├── web/        # Astro (público) + React SPA en /app
    │   └── api/        # Go: cmd/api + cmd/worker + internal/{auth,wallets,games,payments,notifications,admin} + migrations/
    ├── infra/          # docker-compose.dev.yml (Postgres local), Dockerfiles
    ├── docs/           # ADRs + agent docs
    ├── AGENTS.md · CONTEXT.md
    ```
  - **Despliegue**: un Dockerfile multi-stage → Railway: servicio `api` (+ binario api) y servicio `worker` (mismo build, distinto comando). Migraciones golang-migrate como job del pipeline. Frontend → Cloudflare Pages.
- **Por qué**: Con un dev solo y dinero real, el monolito minimiza la superficie operativa y garantiza integridad transaccional; el worker separa lo síncrono de lo diferido sin partir el dominio. Microservicios se evaluarán solo si aparece un dominio con requisitos de escala/deploy independientes (plan de salida: el módulo `games` es el primer candidato a extracción, sus contratos ya están aislados).
- **Consecuencias**: Un solo punto de despliegue (simple); la separación módulo→servicio futuro es mecánica; el worker debe vigilarse para no dormirse (Railway: sin sleep en worker). QStash como única cola del sistema. Sin ORM: SQL explícito con pgx — el costo de scaffold por módulo es constante pero el SQL queda auditable (crítico en el ledger).
