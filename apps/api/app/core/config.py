from functools import lru_cache

from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    """Configuración central de Skyland API. Se sobreescribe con variables de entorno (.env)."""

    model_config = SettingsConfigDict(env_file=".env", env_prefix="SKYLAND_", extra="ignore")

    app_name: str = "Skyland API"
    environment: str = "development"  # development | production

    # --- Base de datos (única fuente de verdad) ---
    database_url: str = "postgresql+asyncpg://skyland:skyland_dev@localhost:5432/skyland"

    # --- Upstash (Redis serverless + QStash) ---
    upstash_redis_url: str | None = None  # https://<region>.upstash.io
    upstash_redis_token: str | None = None
    qstash_token: str | None = None

    # --- Auth / sesiones ---
    jwt_secret: str = "dev-secret-change-me"
    jwt_expire_minutes: int = 60 * 24 * 7  # 7 días

    # --- CORS (frontend) ---
    cors_origins: str = "http://localhost:4321,http://localhost:4322,https://skyland.pages.dev"

    @property
    def cors_origin_list(self) -> list[str]:
        return [o.strip() for o in self.cors_origins.split(",") if o.strip()]


@lru_cache
def get_settings() -> Settings:
    return Settings()