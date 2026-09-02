-- swap_events: one row per physical battery swap (Spiro/Ampersand-style).
-- battery_id here is "which physical unit this rider happened to be carrying",
-- NOT collateral — it can differ swap to swap. Do not read loans.battery_id
-- semantics into this column.
CREATE TABLE swap_events (
    id BIGSERIAL PRIMARY KEY,
    rider_id UUID NOT NULL REFERENCES riders(id),
    battery_id UUID NOT NULL REFERENCES batteries(id),
    station_id TEXT,
    swapped_at TIMESTAMPTZ NOT NULL,
    returned_state_of_charge NUMERIC,
    returned_temperature_c NUMERIC,
    returned_cycle_count INT,
    returned_depth_of_discharge NUMERIC,
    distance_km_since_last_swap NUMERIC
);

-- financing_model makes the lease-vs-swap distinction explicit so scoring
-- branches on this field rather than inferring it from battery_id IS NULL.
-- Existing lease-model rows keep their one-battery collateral semantics via
-- the default.
ALTER TABLE loans
    ADD COLUMN financing_model TEXT NOT NULL DEFAULT 'leased_fixed'
    CHECK (financing_model IN ('leased_fixed', 'swap_network'));
