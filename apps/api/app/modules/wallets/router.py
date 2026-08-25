"""Módulo wallets: saldo + ledger de doble entrada (camino del dinero, SOLO Postgres)."""

from fastapi import APIRouter

router = APIRouter(tags=["wallets"])


@router.get("/wallets/status")
async def status() -> dict:
    """Esqueleto: aquí vivirá el ledger (movimientos, saldos, transacciones atómicas)."""
    return {"module": "wallets", "ready": False}