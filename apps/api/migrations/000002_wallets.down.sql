-- Reversa de wallets.
BEGIN;
DROP TABLE IF EXISTS ledger_entries CASCADE;
DROP TABLE IF EXISTS ledger_txns  CASCADE;
DROP TABLE IF EXISTS balances     CASCADE;
DROP FUNCTION IF EXISTS check_asiento_cuadrado();
COMMIT;
