-- anomaly_flags stores non-gating fraud/anomaly findings (Feature B).
-- One generic table flags either a telemetry_reading or a repayment_event via
-- (entity_type, entity_id). entity_id is TEXT because telemetry/repayment use
-- BIGSERIAL ids today while other future entities (riders/batteries) use UUID.
CREATE TABLE anomaly_flags (
    id BIGSERIAL PRIMARY KEY,
    entity_type TEXT NOT NULL,
    entity_id TEXT NOT NULL,
    flagged_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    rule_or_model TEXT NOT NULL,
    severity TEXT NOT NULL,
    reason TEXT NOT NULL,
    resolved BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT anomaly_flags_entity_type_check
        CHECK (entity_type IN ('telemetry_reading', 'repayment_event'))
);

CREATE INDEX anomaly_flags_entity_idx
    ON anomaly_flags (entity_type, entity_id);
