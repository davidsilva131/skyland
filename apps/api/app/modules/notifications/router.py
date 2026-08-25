"""Módulo notifications: SSE (contador de sorteo, resultados) + email + push web."""

from fastapi import APIRouter

router = APIRouter(tags=["notifications"])


@router.get("/notifications/status")
async def status() -> dict:
    """Esqueleto: aquí vivirán los streams SSE y el envío de notificaciones."""
    return {"module": "notifications", "ready": False}