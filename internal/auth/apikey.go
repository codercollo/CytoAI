// Package auth implements minimal partner API-key authentication
// (docs/spec.md §6): "partner API key in Authorization: Bearer <key> header,
// hashed at rest."
//
// Choice documented: keys are generated with crypto/rand (never math/rand) and
// stored as bcrypt hashes. bcrypt is chosen over a SHA-256+pepper scheme
// because it is simpler to justify for an MVP, has a built-in cost factor, and
// CompareHashAndPassword is constant-time — no naive string equality on the
// secret or its hash.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/codercollo/cytoai/internal/db/queries"
	"golang.org/x/crypto/bcrypt"
)

// ErrInvalidAPIKey is returned when a provided key does not match any partner.
var ErrInvalidAPIKey = errors.New("auth: invalid API key")

// bcryptCost is the bcrypt work factor. 12 is a reasonable MVP default
// (interactive-login-like latency); raise for production.
const bcryptCost = 12

// credentialLookup is the minimal store surface Authenticator needs.
// *queries.Store satisfies it via ListPartnerCredentials.
type credentialLookup interface {
	ListPartnerCredentials(ctx context.Context) ([]queries.PartnerCredential, error)
}

// Authenticator authenticates bearer API keys against the partner store.
type Authenticator struct {
	creds credentialLookup
}

// New constructs an Authenticator over a credential lookup (typically a
// *queries.Store).
func New(creds credentialLookup) *Authenticator {
	return &Authenticator{creds: creds}
}

// GenerateAPIKey returns a new 256-bit random API key, base64url-encoded. It
// uses crypto/rand — this is a credential, never math/rand.
func GenerateAPIKey() (string, error) {
	buf := make([]byte, 32) // 256 bits
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("auth: generate key: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// HashAPIKey returns the bcrypt hash of a key for storage at rest.
func HashAPIKey(key string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(key), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("auth: hash key: %w", err)
	}
	return string(h), nil
}

// Authenticate verifies a provided bearer key and returns the owning partner
// id. Comparison uses bcrypt.CompareHashAndPassword (constant-time), never
// == on the secret or its hash.
//
// SCOPING NOTE: this function only returns the authenticated partnerID. It
// does NOT, by itself, scope data access. The HTTP handlers (Phase 7 in
// internal/api) MUST filter every riders/batteries/scores/portfolio query by
// this partnerID so a partner can only ever see/act on their own rows.
func (a *Authenticator) Authenticate(ctx context.Context, providedKey string) (string, error) {
	if providedKey == "" {
		return "", ErrInvalidAPIKey
	}

	creds, err := a.creds.ListPartnerCredentials(ctx)
	if err != nil {
		return "", fmt.Errorf("auth: lookup credentials: %w", err)
	}

	for _, c := range creds {
		if bcrypt.CompareHashAndPassword([]byte(c.APIKeyHash), []byte(providedKey)) == nil {
			return c.PartnerID, nil
		}
	}
	return "", ErrInvalidAPIKey
}
