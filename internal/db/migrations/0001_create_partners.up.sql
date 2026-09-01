-- partners table (docs/spec.md §4). First migration; also enables
-- gen_random_uuid() for PostgreSQL versions where it lives in pgcrypto.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE partners (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    api_key_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now()
);
