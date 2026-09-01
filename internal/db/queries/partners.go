package queries

import (
	"context"
	"fmt"
)

// PartnerCredential is a partner's id plus its hashed API key. It is consumed
// by internal/auth during bearer-key authentication.
type PartnerCredential struct {
	PartnerID  string
	APIKeyHash string
}

// ListPartnerCredentials returns every partner's id and hashed API key.
//
// MVP note: authentication iterates these rows and bcrypt-compares each hash
// because the partners table stores only a hash (no key id / prefix). This is
// fine while partner count is small; if it grows, add an indexed key-id column
// and look up by that instead of scanning all partners.
func (s *Store) ListPartnerCredentials(ctx context.Context) ([]PartnerCredential, error) {
	rows, err := s.pool.Query(ctx, `SELECT id::text, api_key_hash FROM partners`)
	if err != nil {
		return nil, fmt.Errorf("queries: list partner credentials: %w", err)
	}
	defer rows.Close()

	var out []PartnerCredential
	for rows.Next() {
		var c PartnerCredential
		if err := rows.Scan(&c.PartnerID, &c.APIKeyHash); err != nil {
			return nil, fmt.Errorf("queries: scan partner credential: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("queries: iterate partner credentials: %w", err)
	}
	return out, nil
}
