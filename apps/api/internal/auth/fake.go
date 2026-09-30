package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"strings"
	"sync"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

// fakeStore es el adapter in-memory de Service: tests de handler + dev fake
// (httpapi.RegisterFake). Misma interface que Postgres.
//
//	ponytail: algo por store; el upgrade path es postgres.go, no aquí.
type fakeStore struct {
	mu       sync.Mutex
	players  map[string]*fakePlayerRow // email normalizado → fila
	sessions map[[32]byte]*fakeSession // token sha-256 → sesión
}

type fakePlayerRow struct {
	id           openapi_types.UUID
	email        string
	birthdate    string
	passwordHash string
	termsVersion string
	createdAt    time.Time
}

type fakeSession struct {
	playerEmail       string
	expiresAt         time.Time
	absoluteExpiresAt time.Time
	revoked           bool
}

// NewFake arma el Service in-memory (pattern wallets.NewFake).
func NewFake() Service { return newFakeStore() }

func newFakeStore() *fakeStore {
	return &fakeStore{
		players:  map[string]*fakePlayerRow{},
		sessions: map[[32]byte]*fakeSession{},
	}
}

// uuidFromEmailOrRand: el fake genera id determinista? No — crypto/rand v4.
func newUUID() (openapi_types.UUID, error) {
	var u openapi_types.UUID
	// uuid.UUID es [16]byte (alias de github.com/google/uuid.UUID): v4 a mano
	// — 6 bits de versión/variante sobre crypto/rand, sin depender del
	// paquete google/uuid que entra como indirect de oapi-codegen/runtime.
	if _, err := randRead(u[:]); err != nil {
		return u, err
	}
	u[6] = (u[6] & 0x0f) | 0x40 // versión 4
	u[8] = (u[8] & 0x3f) | 0x80 // variante RFC 4122
	return u, nil
}

// createSession records: insert sesión con TTL rodante/absoluto.
func (f *fakeStore) createSession(email string, now time.Time, token string) {
	sum := sha256Sum(token)
	f.sessions[sum] = &fakeSession{
		playerEmail:       email,
		expiresAt:         now.Add(7 * 24 * time.Hour),
		absoluteExpiresAt: now.Add(30 * 24 * time.Hour),
	}
}

// playerRowToSchema: fila fake → Player (schema generado).
func playerRowToSchema(row *fakePlayerRow) Player {
	b, _ := time.Parse("2006-01-02", row.birthdate)
	return Player{
		Id:        row.id,
		Email:     row.email,
		Birthdate: openapi_types.Date{Time: b},
		CreatedAt: row.createdAt,
	}
}

var _ Service = (*fakeStore)(nil)

func (f *fakeStore) Register(ctx context.Context, email, password, birthdate string, acceptsTerms bool, now time.Time, createdIP string) (SessionResult, error) {
	e, verr := validateRegistration(email, password, birthdate, acceptsTerms, now)
	if verr != nil {
		return SessionResult{}, verr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, taken := f.players[e]; taken {
		return SessionResult{}, ErrEmailTaken
	}
	hash, err := hashPassword(password)
	if err != nil {
		return SessionResult{}, err
	}
	id, err := newUUID()
	if err != nil {
		return SessionResult{}, err
	}
	row := &fakePlayerRow{
		id:           id,
		email:        e,
		birthdate:    birthdate,
		passwordHash: hash,
		termsVersion: TERMS_VERSION,
		createdAt:    now,
	}
	f.players[e] = row
	token, err := newToken()
	if err != nil {
		return SessionResult{}, err
	}
	f.createSession(e, now, token)
	return SessionResult{Token: token, Player: playerRowToSchema(row)}, nil
}

func (f *fakeStore) Login(ctx context.Context, email, password string, now time.Time) (SessionResult, error) {
	e := strings.ToLower(strings.TrimSpace(email))
	f.mu.Lock()
	defer f.mu.Unlock()
	row, ok := f.players[e]
	if !ok {
		dummyVerify() // anti-enumeración: mismo coste que un login real
		return SessionResult{}, ErrInvalidCredentials
	}
	if !verifyPassword(password, row.passwordHash) {
		return SessionResult{}, ErrInvalidCredentials
	}
	token, err := newToken()
	if err != nil {
		return SessionResult{}, err
	}
	f.createSession(e, now, token)
	return SessionResult{Token: token, Player: playerRowToSchema(row)}, nil
}

func (f *fakeStore) Verify(ctx context.Context, token string) (Player, error) {
	sum := sha256Sum(token)
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.sessions[sum]
	if !ok || s.revoked || !time.Now().Before(s.expiresAt) || !time.Now().Before(s.absoluteExpiresAt) {
		return Player{}, ErrUnauthenticated
	}
	// TTL rodante: refresh expires_at = min(now+7d, absolute).
	s.expiresAt = minTime(time.Now().Add(7*24*time.Hour), s.absoluteExpiresAt)
	return playerRowToSchema(f.players[s.playerEmail]), nil
}

func (f *fakeStore) Revoke(ctx context.Context, token string) error {
	sum := sha256Sum(token)
	f.mu.Lock()
	defer f.mu.Unlock()
	if s, ok := f.sessions[sum]; ok {
		s.revoked = true
	}
	return nil // idempotente: revocar lo que no existe no falla
}

// sha256Sum: sha-256 del token (el único identificador que toca el store).
func sha256Sum(token string) [32]byte {
	return sha256.Sum256([]byte(token))
}

// randRead: crypto/rand.Read con firma corta.
func randRead(b []byte) (int, error) { return rand.Read(b) }

// minTime: el menor de dos tiempos (TTL rodante contra el cap absoluto).
func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
