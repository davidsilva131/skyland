from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from app.core.config import get_settings
from app.modules.auth.router import router as auth_router
from app.modules.games.router import router as games_router

settings = get_settings()

app = FastAPI(
    title=settings.app_name,
    version="0.1.0",
    description="API de Skyland — apuestas multi-juego (Venezuela). Contrato OpenAPI para el frontend.",
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=settings.cors_origin_list,
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Módulos del monolito modular (cada uno con sus routers y servicios)
app.include_router(auth_router, prefix="/api/v1")
app.include_router(games_router, prefix="/api/v1")


@app.get("/api/v1/health")
async def health() -> dict:
    return {"status": "ok", "app": settings.app_name, "env": settings.environment}