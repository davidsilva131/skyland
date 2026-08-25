# Skyland API — imagen de producción (uv)
FROM ghcr.io/astral-sh/uv:python3.11-bookworm-slim AS base

WORKDIR /app
ENV UV_COMPILE_BYTECODE=1 \
    UV_LINK_MODE=copy \
    PYTHONUNBUFFERED=1

# Dependencias (capa cacheable)
COPY pyproject.toml uv.lock ./
RUN uv sync --frozen --no-install-project --no-dev

# Código
COPY . .
RUN uv sync --frozen --no-dev

EXPOSE 8000

# API (el worker se lanza con el mismo build: `uv run python worker.py`)
CMD ["uv", "run", "uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8000"]