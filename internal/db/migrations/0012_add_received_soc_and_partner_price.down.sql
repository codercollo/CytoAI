ALTER TABLE partners
    DROP COLUMN IF EXISTS price_per_kwh_kes;

ALTER TABLE swap_events
    DROP COLUMN IF EXISTS received_state_of_charge;
