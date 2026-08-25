"""Proceso worker de Skyland: tareas diferidas del dominio.

Mismo código que la API, proceso hermano. Orquestado con QStash (colas + cron):
- sorteos: cierre de apuestas → RNG → resultado → liquidación de ganadoras
- recargas: verificación (PagoVerifier) → acreditación de saldo
- notificaciones: email / push tras eventos

El camino del dinero siempre termina en una transacción ACID local en Postgres.
"""
import logging
import time

log = logging.getLogger("skyland.worker")


def main() -> None:
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s %(message)s")
    log.info("Worker skyland iniciado (esqueleto). Conectándose a QStash en el siguiente paso…")
    # TODO: suscribirse a colas QStash y registrar cron de sorteos.
    time.sleep(3600)


if __name__ == "__main__":
    main()