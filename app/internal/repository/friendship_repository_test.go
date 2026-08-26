package repository

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestFriendshipRepository_ListFriends(t *testing.T) {
	if os.Getenv("RUN_DB_TESTS") != "1" {
		t.Skip("set RUN_DB_TESTS=1 to run database tests")
	}

	ctx := context.Background()

	db, err := pgxpool.New(ctx,
		"postgres://postgres:postgres@localhost:5432/guildchat",
	)
	if err != nil {
		t.Fatalf("failed to create db pool: %v", err)
	}
	defer db.Close()

	repo := NewFriendshipRepository(db)

	friends, err := repo.ListFriends(ctx, 3)
	if err != nil {
		t.Fatalf("ListFriends failed: %v", err)
	}

	for _, friend := range friends {
		t.Logf(
			"friend: id=%d username=%s email=%s",
			friend.ID,
			friend.Username,
			friend.Email,
		)
	}

	if len(friends) == 0 {
		t.Fatal("expected user 3 to have at least one friend")
	}

	foundUser1 := false

	for _, friend := range friends {
		if friend.ID == 1 {
			foundUser1 = true
			break
		}
	}

	if !foundUser1 {
		t.Errorf("expected user 1 to be a friend of user 3")
	}
}
