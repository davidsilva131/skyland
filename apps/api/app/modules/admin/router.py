"""Módulo admin: backoffice (roles Admin/Soporte) — recargas, jugadores, juegos, ledger."""

from fastapi import APIRouter

router = APIRouter(tags=["admin"])


@router.get("/admin/status")
async def status() -> dict:
    """Esqueleto: aquí vivirá el backoffice protegido por RBAC."""
    return {"module": "admin", "ready": False}