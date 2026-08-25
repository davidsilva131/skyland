"""Módulo auth: registro, login, sesiones httpOnly y RBAC (jugador/admin/soporte)."""

from fastapi import APIRouter

router = APIRouter(tags=["auth"])


@router.get("/auth/status")
async def status() -> dict:
    """Esqueleto: aquí vivirán register/login/logout y el perfil del jugador."""
    return {"module": "auth", "ready": False}