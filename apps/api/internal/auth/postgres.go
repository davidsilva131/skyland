package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

// Postgres es el adapter pgx de Service (spec §4 postgres.go): insert player
// (unique violation → ErrEmailTaken), insert session, verify por token_hash
// (un SELECT indexado con join al player), rolling-refresh UPDATE, revoke.
//
// Esquema: migrations/000003_players_sessions.up.sql.
type postgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgres arma el Service sobre pgx (pattern wallets.NewPostgres).
func NewPostgres(pool *pgxpool.Pool) Service { return postgresStore{pool} }

var _ Service = postgresStore{}

// Register: valida → hash → insert player (único) → insert session.
func (s postgresStore) Register(ctx context.Context, email, password, birthdate string, acceptsTerms bool, now time.Time, createdIP string) (SessionResult, error) {
	e, verr := validateRegistration(email, password, birthdate, acceptsTerms, now)
	if verr != nil {
		return SessionResult{}, verr
	}
	hash, err := hashPassword(password)
	if err != nil {
		return SessionResult{}, err
	}
	var id openapi_types.UUID
	if _, err = randRead(id[:]); err != nil {
		return SessionResult{}, err
	}
	id[6], id[8] = (id[6]&0x0f)|0x40, (id[8]&0x3f)|0x80

	// terms_accepted_at = now (no solo la versión: el timestamp es la prueba).
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return SessionResult{}, err
	}
	defer tx.Rollback(ctx)

	if _, err = tx.Exec(ctx,
		`INSERT INTO players (id, email, birthdate, password_hash, terms_accepted_at, terms_version)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		id, e, birthdate, hash, now, TERMS_VERSION); err != nil {
		if isUniqueViolation(err) {
			return SessionResult{}, ErrEmailTaken
		}
		return SessionResult{}, err
	}

	token, err := newToken()
	if err != nil {
		return SessionResult{}, err
	}
	if err = insertSession(ctx, tx, id, token, now, createdIP); err != nil {
		return SessionResult{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return SessionResult{}, err
	}
	return SessionResult{Token: token, Player: newPlayer(id, e, birthdate, now)}, nil
}

// Login: fetch player por email; desconocido quema argon2 (anti-enumeración);
// verify contra el hash; nueva sesión (una fila por login, sin rotación).
func (s postgresStore) Login(ctx context.Context, email, password string, now time.Time) (SessionResult, error) {
	e := strings.ToLower(strings.TrimSpace(email))
	var (
		id        openapi_types.UUID
		hash      string
		birthdate string
	)
	err := s.pool.QueryRow(ctx,
		`SELECT id, password_hash, birthdate FROM players WHERE email = $1`, e).
		Scan(&id, &hash, &birthdate)
	if errors.Is(err, pgx.ErrNoRows) {
		dummyVerify() // mismo coste que un login real
		return SessionResult{}, ErrInvalidCredentials
	}
	if err != nil {
		return SessionResult{}, err
	}
	if !verifyPassword(password, hash) {
		return SessionResult{}, ErrInvalidCredentials
	}
	token, err := newToken()
	if err != nil {
		return SessionResult{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return SessionResult{}, err
	}
	defer tx.Rollback(ctx)
	if err = insertSession(ctx, tx, id, token, now, ""); err != nil {
		return SessionResult{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return SessionResult{}, err
	}
	return SessionResult{Token: token, Player: newPlayer(id, e, birthdate, now)}, nil
}

// Verify: sha-256(token) → un SELECT (sesión por token_hash con join al
// player) → rolling refresh. Condiciones del spec: revoked IS NULL,
// expires_at > now(), absolute_expires_at > now().
func (s postgresStore) Verify(ctx context.Context, token string) (Player, error) {
	sum := sha256Sum(token)
	row := s.pool.QueryRow(ctx, `
		SELECT p.id, p.email::text, p.birthdate, p.created_at, se.expires_at, se.absolute_expires_at
		FROM sessions se
		JOIN players p ON p.id = se.player_id
		WHERE se.token_hash = $1
		  AND se.revoked_at IS NULL
		  AND se.expires_at > now()
		  AND se.absolute_expires_at > now()`, sum[:])
	var (
		id          openapi_types.UUID
		email       string
		birthdate   string
		createdAt   time.Time
		expires     time.Time
		absoluteExp time.Time
	)
	if err := row.Scan(&id, &email, &birthdate, &createdAt, &expires, &absoluteExp); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Player{}, ErrUnauthenticated
		}
		return Player{}, err
	}
	// Rolling refresh: expires_at = least(now()+7d, absolute_expires_at).
	nuevo := minTime(time.Now().Add(7*24*time.Hour), absoluteExp)
	if _, err := s.pool.Exec(ctx,
		`UPDATE sessions SET expires_at = $2 WHERE token_hash = $1`, sum[:], nuevo); err != nil {
		return Player{}, err
	}
	return newPlayer(id, email, birthdate, createdAt), nil
}

// Revoke: UPDATE … SET revoked_at = now() WHERE token_hash = $1 AND revoked_at
// IS NULL. Idempotente: no existe o ya revocada → UPDATE 0 filas, nil.
func (s postgresStore) Revoke(ctx context.Context, token string) error {
	sum := sha256Sum(token)
	_, err := s.pool.Exec(ctx,
		`UPDATE sessions SET revoked_at = now()
		 WHERE token_hash = $1 AND revoked_at IS NULL`, sum[:])
	return err
}

// insertSession crea la fila de sesión (hash, nunca el token). createdIP es
// el IP del cliente (spec §4: mismo valor que el rate-limit); fallback
// "0.0.0.0" cuando no hay IP parseable (tests).
func insertSession(ctx context.Context, tx pgx.Tx, playerID openapi_types.UUID, token string, now time.Time, createdIP string) error {
	if createdIP == "" {
		createdIP = "0.0.0.0"
	}
	sum := sha256Sum(token)
	_, err := tx.Exec(ctx, `
		INSERT INTO sessions (id, player_id, token_hash, created_ip, expires_at, absolute_expires_at)
		VALUES (gen_random_uuid(), $1, $2, $5, $3, $4)`,
		playerID, sum[:], now.Add(7*24*time.Hour), now.Add(30*24*time.Hour), createdIP)
	return err
}

// newPlayer arma el schema Player generado desde la fila.
func newPlayer(id openapi_types.UUID, email, birthdate string, createdAt time.Time) Player {
	b, _ := time.ParseInLocation("2006-01-02", birthdate, caracasTZ)
	return Player{
		Id:        id,
		Email:     email,
		Birthdate: openapi_types.Date{Time: b},
		CreatedAt: createdAt,
	}
}

// isUniqueViolation: pgconn.PgError 23505 → ErrEmailTaken (citext lowercase
// unique del DDL).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
