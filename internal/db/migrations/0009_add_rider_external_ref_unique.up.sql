-- Enforce one external_ref per partner for non-null refs. NULL refs stay
-- unlimited (partial index), matching riders.external_ref's nullable schema.
-- This backs POST /v1/riders duplicate detection via the 23505 unique
-- violation (constraint name: riders_partner_external_ref_unique).
CREATE UNIQUE INDEX riders_partner_external_ref_unique
    ON riders (partner_id, external_ref)
    WHERE external_ref IS NOT NULL;
