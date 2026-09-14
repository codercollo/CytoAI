ALTER TABLE swap_events
    ADD COLUMN received_state_of_charge NUMERIC;

ALTER TABLE partners
    ADD COLUMN price_per_kwh_kes NUMERIC DEFAULT 2.90;
