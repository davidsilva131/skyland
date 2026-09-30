BEGIN;
CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE players (
    id                uuid PRIMARY KEY,
    email             citext NOT NULL UNIQUE,
    birthdate         date NOT NULL,
    password_hash     text NOT NULL,              -- argon2id PHC string
    terms_accepted_at timestamptz NOT NULL,
    terms_version     text NOT NULL,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE sessions (
    id                  uuid PRIMARY KEY,
    player_id           uuid NOT NULL REFERENCES players(id),
    token_hash          bytea NOT NULL UNIQUE,    -- sha-256(token); never plaintext
    created_ip          inet NOT NULL,
    created_at          timestamptz NOT NULL DEFAULT now(),
    expires_at          timestamptz NOT NULL,     -- rolling
    absolute_expires_at timestamptz NOT NULL,
    revoked_at          timestamptz
);
COMMIT;
