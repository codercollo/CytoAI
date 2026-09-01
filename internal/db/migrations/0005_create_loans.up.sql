CREATE TABLE loans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rider_id UUID NOT NULL REFERENCES riders(id),
    battery_id UUID REFERENCES batteries(id),
    principal_kes NUMERIC,
    battery_value_kes NUMERIC,
    term_months INT,
    daily_installment_kes NUMERIC,
    started_at DATE
);
