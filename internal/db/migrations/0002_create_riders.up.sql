CREATE TABLE riders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    external_ref TEXT,
    partner_id UUID NOT NULL REFERENCES partners(id),
    onboarded_at TIMESTAMPTZ DEFAULT now()
);
