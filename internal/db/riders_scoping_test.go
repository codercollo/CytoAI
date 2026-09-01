package db

import (
	"context"
	"errors"
	"testing"

	"github.com/codercollo/cytoai/internal/db/queries"
	"github.com/codercollo/cytoai/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestListRiders_PartnerScoped(t *testing.T) {
	pool := integrationPool(t)
	ctx := context.Background()
	q := queries.New(pool)
	a := newPartner(t, pool, "rider-list-a")
	b := newPartner(t, pool, "rider-list-b")
	defer deleteRiderFixture(t, pool, "", "", "", a)
	defer deleteRiderFixture(t, pool, "", "", "", b)

	ra, err := q.CreateRider(ctx, a, domain.RiderRegistration{ExternalRef: sptr("list_a")})
	if err != nil {
		t.Fatalf("create rider A: %v", err)
	}
	defer deleteRiderFixture(t, pool, ra.Rider.ID, "", "", "")

	rb, err := q.CreateRider(ctx, b, domain.RiderRegistration{ExternalRef: sptr("list_b")})
	if err != nil {
		t.Fatalf("create rider B: %v", err)
	}
	defer deleteRiderFixture(t, pool, rb.Rider.ID, "", "", "")

	listA, err := q.ListRiders(ctx, a, 100, 0)
	if err != nil {
		t.Fatalf("ListRiders(A): %v", err)
	}
	if len(listA) != 1 || listA[0].Rider.ID != ra.Rider.ID {
		t.Errorf("ListRiders(A) = %+v, want only rider %s", listA, ra.Rider.ID)
	}

	listB, err := q.ListRiders(ctx, b, 100, 0)
	if err != nil {
		t.Fatalf("ListRiders(B): %v", err)
	}
	if len(listB) != 1 || listB[0].Rider.ID != rb.Rider.ID {
		t.Errorf("ListRiders(B) = %+v, want only rider %s", listB, rb.Rider.ID)
	}
}

func TestRiderByID_CrossPartnerReturnsNotFound(t *testing.T) {
	pool := integrationPool(t)
	ctx := context.Background()
	q := queries.New(pool)
	a := newPartner(t, pool, "rider-byid-a")
	b := newPartner(t, pool, "rider-byid-b")
	defer deleteRiderFixture(t, pool, "", "", "", a)
	defer deleteRiderFixture(t, pool, "", "", "", b)

	ra, err := q.CreateRider(ctx, a, domain.RiderRegistration{ExternalRef: sptr("byid_a")})
	if err != nil {
		t.Fatalf("create rider: %v", err)
	}
	defer deleteRiderFixture(t, pool, ra.Rider.ID, "", "", "")

	if _, err := q.RiderByID(ctx, b, ra.Rider.ID); !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("RiderByID(cross-partner) err = %v, want pgx.ErrNoRows", err)
	}
}

func TestCreateRider_DuplicateRefPerPartner(t *testing.T) {
	pool := integrationPool(t)
	ctx := context.Background()
	q := queries.New(pool)
	p := newPartner(t, pool, "rider-dup-p")
	defer deleteRiderFixture(t, pool, "", "", "", p)

	first, err := q.CreateRider(ctx, p, domain.RiderRegistration{ExternalRef: sptr("dup")})
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	defer deleteRiderFixture(t, pool, first.Rider.ID, "", "", "")

	_, err = q.CreateRider(ctx, p, domain.RiderRegistration{ExternalRef: sptr("dup")})
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		t.Fatalf("second create err = %v, want unique violation 23505", err)
	}

	// The same external_ref is allowed under a different partner.
	p2 := newPartner(t, pool, "rider-dup-p2")
	defer deleteRiderFixture(t, pool, "", "", "", p2)
	second, err := q.CreateRider(ctx, p2, domain.RiderRegistration{ExternalRef: sptr("dup")})
	if err != nil {
		t.Fatalf("create same ref under another partner: %v", err)
	}
	defer deleteRiderFixture(t, pool, second.Rider.ID, "", "", "")
}
