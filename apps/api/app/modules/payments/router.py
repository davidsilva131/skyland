"""Módulo payments: recargas por pago móvil.

`PagoVerifier` es intercambiable: manual (backoffice acredita) hoy,
API de conciliación BDV/Mercantil cuando exista cuenta empresarial.
"""

from fastapi import APIRouter

router = APIRouter(tags=["payments"])


@router.get("/payments/status")
async def status() -> dict:
    """Esqueleto: aquí viven las recargas, PagoVerifier y los límites diarios."""
    return {"module": "payments", "ready": False}