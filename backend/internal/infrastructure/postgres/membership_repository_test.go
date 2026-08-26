package postgres_test

import (
	"slices"
	"testing"

	"github.com/diegobraga92/pudimproductivity/backend/internal/infrastructure/postgres"
	"github.com/diegobraga92/pudimproductivity/backend/internal/infrastructure/postgres/postgrestest"
)

func TestMembershipRepository_ListIDsForUser(t *testing.T) {
	postgrestest.SkipIfShort(t)
	ctx, pool := postgrestest.SetupPool(t)
	repo := postgres.NewMembershipRepository(pool)

	seed := `
		INSERT INTO task_lists (id, name, owner_id) VALUES
			('11111111-1111-1111-1111-111111111111', 'Owned',     'alice'),
			('22222222-2222-2222-2222-222222222222', 'Shared',    'bob'),
			('33333333-3333-3333-3333-333333333333', 'Deleted',   'alice'),
			('44444444-4444-4444-4444-444444444444', 'No access', 'carol');
		UPDATE task_lists SET deleted_at = NOW() WHERE id = '33333333-3333-3333-3333-333333333333';
		INSERT INTO task_list_shares (list_id, shared_with, role) VALUES
			('22222222-2222-2222-2222-222222222222', 'alice', 'editor');
	`
	if _, err := pool.Exec(ctx, seed); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Alice owns list 1 and is shared list 2, her soft-deleted list is excluded.
	ids, err := repo.ListIDsForUser(ctx, "alice", "user")
	if err != nil {
		t.Fatalf("ListIDsForUser(alice): %v", err)
	}
	if len(ids) != 2 ||
		!slices.Contains(ids, "11111111-1111-1111-1111-111111111111") ||
		!slices.Contains(ids, "22222222-2222-2222-2222-222222222222") {
		t.Fatalf("ListIDsForUser(alice) = %v, want her owned + shared lists", ids)
	}

	// A user with no lists gets an empty (non-nil) slice.
	empty, err := repo.ListIDsForUser(ctx, "nobody", "user")
	if err != nil {
		t.Fatalf("ListIDsForUser(nobody): %v", err)
	}
	if empty == nil || len(empty) != 0 {
		t.Fatalf("ListIDsForUser(nobody) = %v, want empty non-nil slice", empty)
	}

	// An empty user ID short-circuits to an empty result.
	if got, err := repo.ListIDsForUser(ctx, "", "user"); err != nil || got == nil || len(got) != 0 {
		t.Fatalf("ListIDsForUser(\"\") = %v (err %v), want empty", got, err)
	}

	// Admins see every non-deleted list regardless of ownership.
	admin, err := repo.ListIDsForUser(ctx, "admin-user", "admin")
	if err != nil {
		t.Fatalf("ListIDsForUser(admin): %v", err)
	}
	if len(admin) != 3 {
		t.Fatalf("ListIDsForUser(admin) = %v, want all 3 non-deleted lists", admin)
	}
	for _, want := range []string{
		"11111111-1111-1111-1111-111111111111",
		"22222222-2222-2222-2222-222222222222",
		"44444444-4444-4444-4444-444444444444",
	} {
		if !slices.Contains(admin, want) {
			t.Fatalf("admin list %v missing from %v", want, admin)
		}
	}
}
