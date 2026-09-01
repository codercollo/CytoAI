CREATE TABLE telemetry_readings (
    id BIGSERIAL PRIMARY KEY,
    battery_id UUID NOT NULL REFERENCES batteries(id),
    reading_at TIMESTAMPTZ NOT NULL,
    state_of_charge NUMERIC,
    voltage NUMERIC,
    temperature_c NUMERIC,
    cycle_count INT,
    depth_of_discharge NUMERIC,
    distance_km_since_last NUMERIC
);
