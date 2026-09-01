CREATE TABLE batteries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    external_ref TEXT,
    manufacturer TEXT,
    rated_capacity_wh NUMERIC,
    commissioned_at TIMESTAMPTZ
);
