// Package worker: proceso hermano de la API para tareas diferidas del dominio.
//
// Mismo código que la API, proceso hermano. Orquestado con QStash (colas + cron):
//   - sorteos: cierre de apuestas → RNG (crypto/rand) → resultado → liquidación de ganadoras
//   - recargas: verificación (PagoVerifier) → acreditación de saldo
//   - notificaciones: email / push tras eventos
//
// El camino del dinero siempre termina en una transacción ACID local en Postgres.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var startedAt = time.Now()

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	logger.Info("Worker skyland iniciado (esqueleto). Conectándose a QStash en el siguiente paso…")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// TODO: suscribirse a colas QStash y registrar cron de sorteos.
	<-ctx.Done()
	logger.Info("Worker skyland detenido", "uptime", time.Since(startedAt).String())
}
