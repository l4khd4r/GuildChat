package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/l4khd4r/GuildChat/internal/model"
)

type FriendshipRepository struct {
	db *pgxpool.Pool
}

func NewFriendshipRepository(db *pgxpool.Pool) *FriendshipRepository {
	return &FriendshipRepository{
		db: db,
	}
}

// mapFriendshipError turns pgx errors into the package's sentinels so that
// callers can match them with errors.Is.
func mapFriendshipError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrFriendshipNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrFriendshipAlreadyExists
	}
	return err
}

func (r *FriendshipRepository) Create(ctx context.Context, requesterId int64, receiverId int64) (*model.Friendship, error) {

	now := time.Now()
	query := `
		INSERT INTO friendships (requester_id, receiver_id, status , created_at, updated_at)
		VALUES ($1, $2, $3 , $4 , $5 )
		RETURNING id, requester_id, receiver_id, status, created_at, updated_at
		`

	friendship := &model.Friendship{}
	err := r.db.QueryRow(ctx, query, requesterId, receiverId, model.FriendshipPending, now, now).Scan(
		&friendship.ID,
		&friendship.RequesterID,
		&friendship.ReceiverID,
		&friendship.Status,
		&friendship.CreatedAt,
		&friendship.UpdatedAt,
	)
	if err != nil {
		// A concurrent request for the same pair loses the race on
		// unique_friendship_pair; report it as an existing friendship.
		return nil, mapFriendshipError(err)
	}
	return friendship, nil
}

func (r *FriendshipRepository) GetBetweenUsers(ctx context.Context, userID1 int64, userID2 int64) (*model.Friendship, error) {
	friendship := &model.Friendship{}
	query := `
		SELECT id , requester_id , receiver_id , status , created_at  ,updated_at
		FROM friendships
		WHERE (requester_id = $1 AND receiver_id = $2) OR (requester_id = $2 AND receiver_id = $1)
		`
	err := r.db.QueryRow(ctx, query, userID1, userID2).Scan(
		&friendship.ID,
		&friendship.RequesterID,
		&friendship.ReceiverID,
		&friendship.Status,
		&friendship.CreatedAt,
		&friendship.UpdatedAt,
	)

	if err != nil {
		return nil, mapFriendshipError(err)
	}
	return friendship, nil
}
func (r *FriendshipRepository) Accept(ctx context.Context, FriendshipID int64, receiverId int64) (*model.Friendship, error) {
	friendship := &model.Friendship{}
	query := `
		UPDATE friendships
		SET status = $1, updated_at = NOW()
		WHERE id = $2
		AND receiver_id = $3
		AND status = $4
		RETURNING id, requester_id, receiver_id, status, created_at, updated_at
	`
	err := r.db.QueryRow(
		ctx,
		query,
		model.FriendshipAccepted,
		FriendshipID,
		receiverId,
		model.FriendshipPending,
	).Scan(
		&friendship.ID,
		&friendship.RequesterID,
		&friendship.ReceiverID,
		&friendship.Status,
		&friendship.CreatedAt,
		&friendship.UpdatedAt,
	)

	if err != nil {
		return nil, mapFriendshipError(err)
	}
	return friendship, nil
}

// Reject deletes the pending request instead of marking it rejected, so that
// the pair is free to send a new request later: unique_friendship_pair covers
// the pair whatever its status, and a leftover row would block them forever.
func (r *FriendshipRepository) Reject(ctx context.Context, FriendshipID int64, receiverId int64) (*model.Friendship, error) {
	friendship := &model.Friendship{}
	query := `
		DELETE FROM friendships
		WHERE id = $1
		AND receiver_id = $2
		AND status = $3
		RETURNING id, requester_id, receiver_id, status, created_at, updated_at
	`
	err := r.db.QueryRow(
		ctx,
		query,
		FriendshipID,
		receiverId,
		model.FriendshipPending,
	).Scan(
		&friendship.ID,
		&friendship.RequesterID,
		&friendship.ReceiverID,
		&friendship.Status,
		&friendship.CreatedAt,
		&friendship.UpdatedAt,
	)

	if err != nil {
		return nil, mapFriendshipError(err)
	}

	// RETURNING gives the row as it was before the delete, i.e. still pending.
	friendship.Status = model.FriendshipRejected
	return friendship, nil
}

func (r *FriendshipRepository) ListFriends(ctx context.Context, userID int64) ([]*model.User, error) {
	query := `
		SELECT u.id, u.username, u.email, u.created_at, u.updated_at
		FROM friendships f
		JOIN users u
		ON u.id = CASE
			WHEN f.requester_id = $1 THEN f.receiver_id
			ELSE f.requester_id
		END
		WHERE (f.requester_id = $1 OR f.receiver_id = $1)
		AND f.status = $2
		ORDER BY f.created_at DESC
	`
	rows, err := r.db.Query(ctx, query, userID, model.FriendshipAccepted)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	friends := make([]*model.User, 0)

	for rows.Next() {
		friend := &model.User{}
		err := rows.Scan(
			&friend.ID,
			&friend.Username,
			&friend.Email,
			&friend.CreatedAt,
			&friend.UpdatedAt,
		)

		if err != nil {
			return nil, err
		}
		friends = append(friends, friend)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return friends, nil
}

// scanFriendRequests reads rows shaped as (friendship, other user) into the
// list DTO. Both ListPendingRequests and ListSentRequests select that shape.
func scanFriendRequests(rows pgx.Rows) ([]*model.FriendRequest, error) {
	defer rows.Close()

	requests := make([]*model.FriendRequest, 0)

	for rows.Next() {
		request := &model.FriendRequest{User: &model.User{}}
		err := rows.Scan(
			&request.ID,
			&request.Status,
			&request.CreatedAt,
			&request.User.ID,
			&request.User.Username,
			&request.User.Email,
			&request.User.CreatedAt,
			&request.User.UpdatedAt,
		)

		if err != nil {
			return nil, err
		}
		requests = append(requests, request)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return requests, nil
}

// ListPendingRequests returns the requests userID has received and not yet
// answered, each with the user who sent it.
func (r *FriendshipRepository) ListPendingRequests(ctx context.Context, userID int64) ([]*model.FriendRequest, error) {
	query := `
		SELECT f.id, f.status, f.created_at,
		       u.id, u.username, u.email, u.created_at, u.updated_at
		FROM friendships f
		JOIN users u ON u.id = f.requester_id
		WHERE f.receiver_id = $1 AND f.status = $2
		ORDER BY f.created_at DESC
	`
	rows, err := r.db.Query(ctx, query, userID, model.FriendshipPending)
	if err != nil {
		return nil, err
	}

	return scanFriendRequests(rows)
}

// ListSentRequests returns the requests userID has sent that are still
// unanswered, each with the user they were sent to.
func (r *FriendshipRepository) ListSentRequests(ctx context.Context, userID int64) ([]*model.FriendRequest, error) {
	query := `
		SELECT f.id, f.status, f.created_at,
		       u.id, u.username, u.email, u.created_at, u.updated_at
		FROM friendships f
		JOIN users u ON u.id = f.receiver_id
		WHERE f.requester_id = $1 AND f.status = $2
		ORDER BY f.created_at DESC
	`
	rows, err := r.db.Query(ctx, query, userID, model.FriendshipPending)
	if err != nil {
		return nil, err
	}

	return scanFriendRequests(rows)
}
