# ADR-0001 — Stack y despliegue de Skyland (decisión final)

- **Fecha**: 2026-08-25
- **Estado**: Aceptado — Rev. 3 (final)
- **Contexto**: Plataforma de apuestas multi-juego con saldo real (VES hoy, multi-moneda a futuro), picos de 100–500 apuestas/min antes de cada sorteo, recargas vía pago móvil venezolano (verificación manual al inicio, API bancaria después), presupuesto ~$10/mes, escala objetivo 100–1.000 concurrentes. El usuario prefiere Cloudflare, rechaza Vercel, y quiere un backend sólido desde el día 1 (descartó Supabase por riesgo de Acceptable Use Policy sobre gambling y por evitar una migración futura innecesaria).
- **Decisión**:
  - **Frontend**: **Astro** (HTML estático puro) para el sitio público — landing, login/registro, FAQ, términos — + **React + TypeScript + Vite** como SPA en `/app` (lobby, juegos, backoffice), todo desplegado en **Cloudflare Pages** ($0, `skyland.pages.dev`; dominio propio ~$1–3/año antes del lanzamiento). Patrón: una sola isla React grande bajo `/app`, no islas dispersas.
  - **Backend**: **Python + FastAPI** (async, Pydantic para validación estricta de montos/apuestas), desplegado en **Railway** (PaaS que el equipo ya usa): API + PostgreSQL 16 managed + backups incluidos. Sin VPS en el MVP; migrar a VPS propio es trivial (Dockerfile + env vars) si el costo o el control lo exigen.
  - **Datos**: **PostgreSQL** (única fuente de verdad). `JSONB` para la configuración flexible de cada juego (la flexibilidad documental que se buscaba con MongoDB, sin perder ACID). SQLAlchemy 2.0 + Alembic.
  - **Cache/colas**: **Upstash Redis + QStash** (serverless, pay-as-you-go, SDK compatible con Redis): rate limiting por jugador, topes diarios, contador de apuestas del sorteo, cache, cron de sorteos y colas.
  - **Regla de oro**: el **camino crítico del dinero** (apuesta atómica → débito → ledger de doble entrada) corre **solo en Postgres, en una transacción**, con constraints y locks. Upstash nunca bloquea el camino del dinero (llamadas en paralelo o asíncronas).
  - **Auth**: sesiones propias httpOnly revocables + RBAC (Admin / Soporte).
  - **Tiempo real**: **SSE** (Server-Sent Events) para contador de sorteo y resultados en vivo. WebSockets solo si un juego futuro lo exige.
  - **RNG**: CSPRNG (`secrets` de Python) en el backend, sorteos con locks en Postgres (`pg_advisory_lock`).
  - **Pagos**: contrato `PagoVerifier` desacoplado — verificación **manual** (backoffice acredita) hoy, **API de conciliación BDV/Mercantil** cuando exista cuenta empresarial.
  - **CI/CD**: GitHub Actions → `wrangler pages deploy` (frontend) + deploy a Railway (backend) + migraciones Alembic en cada PR.
  - **Excluido**: Vercel (preferencia explícita), Supabase (AUP gambling + migración), MongoDB para el dinero (integridad/constraints), Edge Functions serverless para lógica de dinero (control y auditabilidad), app nativa (Play Store prohíbe apuestas reales).
- **Por qué**: El manejo de dinero exige integridad garantizada por la base (Postgres), validación estricta en el borde (Pydantic) y control total del RNG (servidor propio). Railway + Upstash eliminan la administración (DBA/Redis) manteniendo portabilidad total (Docker + SDK compatible). Astro da la landing instantánea con SEO; React concentra la interacción donde vive.
- **Consecuencias**: Costo ~$8–15/mes (Railway usage-based + Upstash, en centavos). Responsabilidad operativa del esquema de datos y backups en Railway (aceptable: managed). Latencia ~70–120ms desde VE (aceptable). El stack queda abierto a migrar a VPS/Hetzner con cambios de configuración, no de código.

# ADR-0002 — Resultados de los juegos por RNG propio de la Casa (sin provably fair en el MVP)

- **Fecha**: 2026-08-25
- **Estado**: Aceptado
- **Contexto**: La fuente de resultados se decidió como RNG generado por la plataforma (a diferencia de jugar contra loterías oficiales). El usuario descartó provably fair (hash público por sorteo) en el MVP.
- **Decisión**: Los resultados de cada juego provienen de un **CSPRNG del backend** (`secrets` de Python), generados dentro de transacciones con locks (`pg_advisory_lock`) para garantizar consistencia entre cierre de apuestas, resultado y pagos. El diseño expone un contrato `ResultSource` por juego (hoy: RNG; mañana: cualquier fuente externa) sin tocar el motor de apuestas.
- **Por qué**: RNG propio da control total de frecuencia y payouts (RTP 90% objetivo), coste cero y simplicidad. El contrato por juego mantiene la puerta abierta a resultados externos (lotería/hipódromo) sin rediseño.
- **Consecuencias**: La confianza del jugador depende de la reputación de la Casa (riesgo asumido y revisitado). Sin provably fair, el RNG debe ser auditado internamente y el ledger debe permitir reconstruir cualquier sorteo (semilla + parámetros + apuestas registradas). Es la decisión con mayor exposición legal/regulatoria del proyecto — revisar cuando el marco regulatorio venezolano de apuestas en línea avance.