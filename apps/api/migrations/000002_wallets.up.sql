-- wallets: Saldo + ledger de doble entrada (ADR-0001 Rev. 4).
-- Montos enteros en la mínima unidad; currency ISO 4217 (VES hoy, multi-moneda desde el día 1).
BEGIN;

CREATE TABLE balances (
    jugador_id bigint      NOT NULL,
    currency   text        NOT NULL,
    amount     bigint      NOT NULL DEFAULT 0 CHECK (amount >= 0),
    PRIMARY KEY (jugador_id, currency)
);

CREATE TABLE ledger_txns (
    id       bigserial PRIMARY KEY,
    currency text    NOT NULL,
    detalle  text    NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE ledger_entries (
    txn_id     bigint NOT NULL REFERENCES ledger_txns(id),
    jugador_id bigint NOT NULL,
    debe       bigint NOT NULL DEFAULT 0 CHECK (debe   >= 0),
    haber      bigint NOT NULL DEFAULT 0 CHECK (haber  >= 0),
    CHECK ((debe > 0) <> (haber > 0))
);

-- Suma de la partida siempre cierra a 0: constraints de doble entrada por txn.
CREATE FUNCTION check_asiento_cuadrado() RETURNS trigger AS $$
BEGIN
    IF (SELECT sum(debe) FROM ledger_entries WHERE txn_id = NEW.txn_id) <>
       (SELECT sum(haber) FROM ledger_entries WHERE txn_id = NEW.txn_id) THEN
        RAISE EXCEPTION 'asiento % descuadrado', NEW.txn_id;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE CONSTRAINT TRIGGER trg_asiento_cuadrado
    AFTER INSERT ON ledger_entries
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION check_asiento_cuadrado();

COMMIT;
