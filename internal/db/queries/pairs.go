package queries

import (
	"context"
	"fmt"

	"github.com/codercollo/cytoai/internal/domain"
)

// RiderBatteryPairsForPartner returns every distinct rider/battery pair owned
// by a partner that has at least one loan — i.e. the same universe
// PortfolioScores' JOIN can reach.
func (s *Store) RiderBatteryPairsForPartner(ctx context.Context, partnerID string) ([]domain.RiderBatteryPair, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT l.rider_id::text, l.battery_id::text
		FROM loans l
		JOIN riders r ON r.id = l.rider_id
		WHERE r.partner_id = $1
		  AND l.battery_id IS NOT NULL
		ORDER BY l.rider_id::text, l.battery_id::text`, partnerID)
	if err != nil {
		return nil, fmt.Errorf("queries: rider battery pairs: %w", err)
	}
	defer rows.Close()

	var out []domain.RiderBatteryPair
	for rows.Next() {
		var p domain.RiderBatteryPair
		if err := rows.Scan(&p.RiderID, &p.BatteryID); err != nil {
			return nil, fmt.Errorf("queries: scan rider battery pair: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("queries: iterate rider battery pairs: %w", err)
	}
	return out, nil
}
