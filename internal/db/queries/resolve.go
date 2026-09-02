package queries

import (
	"context"
	"fmt"
)

// ResolveBatteryID maps a partner-supplied battery ref (either an
// external_ref or a UUID id) to the batteries.id UUID.
func (s *Store) ResolveBatteryID(ctx context.Context, partnerID, ref string) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `
		SELECT DISTINCT b.id::text
		FROM batteries b
		JOIN loans l ON l.battery_id = b.id
		JOIN riders r ON r.id = l.rider_id
		WHERE (b.external_ref = $1 OR b.id::text = $1)
		  AND r.partner_id = $2
		LIMIT 1`, ref, partnerID,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("queries: resolve battery id %q for partner %q: %w", ref, partnerID, err)
	}
	return id, nil
}

// ResolveLoanID maps a partner-supplied loan ref (either an external_ref or a
// UUID id) to the loans.id UUID.

func (s *Store) ResolveLoanID(ctx context.Context, partnerID, ref string) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `
		SELECT l.id::text
		FROM loans l
		JOIN riders r ON r.id = l.rider_id
		WHERE (l.external_ref = $1 OR l.id::text = $1)
		  AND r.partner_id = $2
		LIMIT 1`, ref, partnerID,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("queries: resolve loan id %q for partner %q: %w", ref, partnerID, err)
	}
	return id, nil
}

// ResolveRiderID maps a partner-supplied rider ref (either an external_ref or a
// UUID id) to the riders.id UUID, scoped to the authenticated partner.
func (s *Store) ResolveRiderID(ctx context.Context, partnerID, ref string) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `
		SELECT id::text
		FROM riders
		WHERE (external_ref = $1 OR id::text = $1)
		  AND partner_id = $2
		LIMIT 1`, ref, partnerID,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("queries: resolve rider id %q for partner %q: %w", ref, partnerID, err)
	}
	return id, nil
}

// ResolveBatteryIDByExternalRef maps a battery ref to the batteries.id UUID by
// external_ref (or id) alone. Swap-network batteries are not linked to a
// partner through loans (loans.battery_id is NULL), so partner scoping through
// the loan join used by ResolveBatteryID does not apply. MVP tradeoff: battery
// external_refs are treated as operator-assigned and unique across the fleet.
func (s *Store) ResolveBatteryIDByExternalRef(ctx context.Context, ref string) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `
		SELECT id::text
		FROM batteries
		WHERE external_ref = $1 OR id::text = $1
		LIMIT 1`, ref,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("queries: resolve battery id %q: %w", ref, err)
	}
	return id, nil
}

// RiderBelongsToPartner reports whether a rider belongs to a partner. The
// API layer uses this to enforce per-partner data scoping.
func (s *Store) RiderBelongsToPartner(ctx context.Context, partnerID, riderID string) (bool, error) {
	var ok bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM riders WHERE id = $1 AND partner_id = $2)`, riderID, partnerID,
	).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("queries: rider belongs to partner: %w", err)
	}
	return ok, nil
}
