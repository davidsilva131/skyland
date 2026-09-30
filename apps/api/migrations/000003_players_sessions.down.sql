-- Reversa de players + sessions.
BEGIN;
DROP TABLE sessions;
DROP TABLE players;
DROP EXTENSION IF EXISTS citext;
COMMIT;
