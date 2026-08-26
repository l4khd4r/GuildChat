package repository

import (
	"context"
	"errors"
	"time"

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
		return nil, err
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
		return nil, errors.New("friendship not found")
	}
	return friendship, nil
}
func (r *FriendshipRepository) Accept(ctx context.Context , FriendshipID int64 , receiverId int64) (*model.Friendship, error) {
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
		return nil , errors.New("friendship not found or not pending")
	}
	return friendship , nil
}
