package ratelimit

import (
	"context"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// redisLimiter es el limiter de Redis (Upstash TLS/RESP): fixed window con
// INCR + ExpireNX pipelined en 1 RTT (shape normativo de #2). go-redis
// ParseURL cubre rediss:// que la URL de Upstash trae por defecto.
type redisLimiter struct {
	redis *redis.Client
	log   *slog.Logger
}

// NewRedis arma el limiter sobre go-redis/v9. fail-open ante caída del
// Redis (spec §4: disponibilidad > margen anti-fuerza durante un blip).
func NewRedis(redisURL string, log *slog.Logger) *redisLimiter {
	opts, err := redis.ParseURL(redisURL)
	if err != nil || redisURL == "" {
		log.Error("ratelimit: redis URL inválida, fail-open permanente", "err", err)
		return &redisLimiter{redis: nil, log: log}
	}
	return &redisLimiter{redis: redis.NewClient(opts), log: log}
}

var _ Limiter = (*redisLimiter)(nil)

// Allow: fixed window, INCR + ExpireNX pipelined en 1 RTT. Redis caído →
// fail-open + slog warn [spec-added]: disponibilidad a la hora del sorteo.
func (l *redisLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	if l.redis == nil {
		l.log.Warn("ratelimit: redis no disponible, fail-open", "key", key)
		return true, nil
	}
	pipe := l.redis.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.ExpireNX(ctx, key, window)
	if _, err := pipe.Exec(ctx); err != nil {
		l.log.Warn("ratelimit: redis fallando, fail-open", "key", key, "err", err)
		return true, nil
	}
	n := int(incr.Val())
	return n <= limit, nil
}

// Del resetea el contador del email tras un login exitoso.
func (l *redisLimiter) Del(ctx context.Context, key string) error {
	if l.redis == nil {
		return nil
	}
	return l.redis.Del(ctx, key).Err()
}
