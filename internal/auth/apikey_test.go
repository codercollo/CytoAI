package auth

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/codercollo/cytoai/internal/db/queries"
	"golang.org/x/crypto/bcrypt"
)

type fakeStore struct {
	creds []queries.PartnerCredential
	err   error
}

func (f fakeStore) ListPartnerCredentials(ctx context.Context) ([]queries.PartnerCredential, error) {
	return f.creds, f.err
}

func TestGenerateAPIKey(t *testing.T) {
	k1, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey() error: %v", err)
	}
	k2, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey() error: %v", err)
	}

	if k1 == "" || k2 == "" {
		t.Fatal("generated keys must be non-empty")
	}
	if k1 == k2 {
		t.Error("two generated keys must differ")
	}
	if _, err := base64.RawURLEncoding.DecodeString(k1); err != nil {
		t.Errorf("key is not base64url-encoded: %v", err)
	}
	if len(k1) < 32 {
		t.Errorf("key too short: %d chars, want >= 32", len(k1))
	}
}

func TestHashAPIKey(t *testing.T) {
	key := "correct horse battery staple"

	hash, err := HashAPIKey(key)
	if err != nil {
		t.Fatalf("HashAPIKey() error: %v", err)
	}
	if hash == key {
		t.Error("hash must not equal the plaintext key")
	}
	// A bcrypt hash carries a $2a$/$2b$/$2y$ prefix.
	if !strings.HasPrefix(hash, "$2") {
		t.Errorf("expected a bcrypt hash, got %q", hash)
	}

	// Correct key verifies via bcrypt (constant-time compare).
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(key)); err != nil {
		t.Errorf("correct key should verify: %v", err)
	}
	// Wrong key must be rejected — this is the timing-safe comparison path.
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("wrong key")); err == nil {
		t.Error("wrong key must not verify")
	}
}

func TestAuthenticate(t *testing.T) {
	key, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey() error: %v", err)
	}
	hash, err := HashAPIKey(key)
	if err != nil {
		t.Fatalf("HashAPIKey() error: %v", err)
	}

	store := fakeStore{creds: []queries.PartnerCredential{
		{PartnerID: "partner-1", APIKeyHash: hash},
	}}
	a := New(store)

	// Correct key authenticates and returns the partner id.
	got, err := a.Authenticate(context.Background(), key)
	if err != nil {
		t.Fatalf("Authenticate(correct) error: %v", err)
	}
	if got != "partner-1" {
		t.Errorf("Authenticate() = %q, want partner-1", got)
	}

	// Wrong key is rejected.
	if _, err := a.Authenticate(context.Background(), "wrong-key"); !errors.Is(err, ErrInvalidAPIKey) {
		t.Errorf("Authenticate(wrong) error = %v, want ErrInvalidAPIKey", err)
	}

	// Empty key is rejected without touching the store.
	if _, err := a.Authenticate(context.Background(), ""); !errors.Is(err, ErrInvalidAPIKey) {
		t.Errorf("Authenticate(empty) error = %v, want ErrInvalidAPIKey", err)
	}
}
