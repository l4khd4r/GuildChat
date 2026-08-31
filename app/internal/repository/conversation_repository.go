package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/l4khd4r/GuildChat/internal/model"
)


type ConversationRepository struct {
	db *pgxpool.Pool
}

func NewConversationRepository(db *pgxpool.Pool) *ConversationRepository {
	return &ConversationRepository{
		db : db,
	}
}


func (r *ConversationRepository) GetDM( ctx context.Context,userID1 int64, userID2 int64) (*model.Conversation, error) {

	query :=`
		SELECT c.id , c.type ,c.name , c.created_by , c.created_at , c.updated_at
		FROM conversations c
		JOIN conversation_members cm1 ON c.id = cm1.conversation_id
		JOIN conversation_members cm2 ON c.id = cm2.conversation_id
		WHERE c.type = $3
			AND cm1.user_id = $1
			AND cm2.user_id = $2
	`
	conversation := &model.Conversation{}

	err := r.db.QueryRow(ctx, query , userID1 , userID2 , model.ConversationDM).Scan(
		&conversation.ID,
		&conversation.Type,
		&conversation.Name,
		&conversation.CreatedBy,
		&conversation.CreatedAt,
		&conversation.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return conversation, nil
}

func (r *ConversationRepository) CreateConversation(ctx context.Context  , conversationType string , name *string  ,createdBy  int64 ) (*model.Conversation , error){
	query := `
		INSERT INTO conversations (type, name, created_by)
		VALUES ($1, $2, $3)
		RETURNING id, type, name, created_by, created_at, updated_at
	`

	conversation := &model.Conversation{}


	err := r.db.QueryRow(
		ctx,
		query ,
		conversationType,
		name,
		createdBy,
	).Scan(
		&conversation.ID,
		&conversation.Type,
		&conversation.Name,
		&conversation.CreatedBy,
		&conversation.CreatedAt,
		&conversation.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return conversation, nil
}


func (r *ConversationRepository) AddMember(ctx context.Context, conversationID int64, userID int64 , role string ) error {
	query := `
		INSERT INTO conversation_members (conversation_id, user_id , role)
		VALUES ($1, $2, $3)
	`

	_ , err := r.db.Exec(ctx,  query , conversationID,  userID, role);
	return err
}
