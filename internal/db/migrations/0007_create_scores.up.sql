CREATE TABLE scores (
    id BIGSERIAL PRIMARY KEY,
    rider_id UUID REFERENCES riders(id),
    battery_id UUID REFERENCES batteries(id),
    battery_health_index NUMERIC,
    repayment_risk_index NUMERIC,
    cyto_score NUMERIC,
    scored_at TIMESTAMPTZ DEFAULT now(),
    model_version TEXT
);
