package ratelimit

import (
	"context"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"
)

// testLogger: logger silencioso para los tests.
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// TestRedisAllow: integración optativa — solo corre con
// SKYLAND_TEST_REDIS_URL (spec §4: "integration test optional, gated on
// SKYLAND_TEST_REDIS_URL").
func TestRedisAllow(t *testing.T) {
	url := os.Getenv("SKYLAND_TEST_REDIS_URL")
	if url == "" {
		t.Skip("SKYLAND_TEST_REDIS_URL no definida: integración Redis omitida")
	}
	l := NewRedis(url, testLogger())
	ctx := context.Background()
	key := "rl:test:" + time.Now().Format(time.RFC3339Nano)

	for i := 1; i <= 3; i++ {
		ok, err := l.Allow(ctx, key, 3, time.Minute)
		if err != nil {
			t.Fatalf("Allow %d: %v", i, err)
		}
		if !ok {
			t.Fatalf("Allow %d bloqueado antes del límite", i)
		}
	}
	if ok, err := l.Allow(ctx, key, 3, time.Minute); err != nil || ok {
		t.Fatalf("4º intento: ok=%v err=%v; want bloqueado", ok, err)
	}
	if err := l.Del(ctx, key); err != nil {
		t.Fatalf("Del: %v", err)
	}
	if ok, err := l.Allow(ctx, key, 3, time.Minute); err != nil || !ok {
		t.Fatalf("tras Del: ok=%v err=%v; want permitido", ok, err)
	}
}
