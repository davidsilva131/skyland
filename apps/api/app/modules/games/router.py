"""Módulo games: catálogo extensible + motor de apuestas genérico.

Contratos de dominio:
- `BetMarket`: mercado de apuestas de un juego (tipos de jugada, payouts, límites).
- `ResultSource`: fuente de resultados (RNG propio hoy; lotería/hipódromo mañana).
"""

from fastapi import APIRouter

router = APIRouter(tags=["games"])


@router.get("/games")
async def list_games() -> dict:
    """Catálogo de juegos disponibles. La configuración vive en JSONB por juego."""
    return {"games": []}