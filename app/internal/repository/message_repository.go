package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/l4khd4r/GuildChat/internal/model"
)

type MessageRepository struct {
	db *pgxpool.Pool
}

func NewMessageRepository(db *pgxpool.Pool) *MessageRepository {
	return &MessageRepository{db: db}
}

// CreateMessage stores one message and marks its conversation as freshly
// active, as one unit.
//
// The two writes share a transaction because the timestamp is what orders the
// caller's sidebar: a message that landed without touching its conversation
// would sit in a chat the user has no reason to open, and the list would claim
// the conversation had been quiet since it was created.
//
// The insert is idempotent on client_msg_id. The odd-looking DO UPDATE that
// assigns a column to itself is deliberate: DO NOTHING returns no rows on a
// conflict, which would cost a second query to find the message that already
// exists. Treating the conflict as an update makes RETURNING hand back the
// existing row in the same statement, so a retry gets the original message.
//
// The sender is joined in through a CTE rather than fetched separately. The
// response embeds the whole user, and a second round trip for a row the
// database already has in hand is a waste on the hottest write in the app.

func (r *MessageRepository) CreateMessage(ctx context.Context, conversationID int64, senderID int64, body string, clientMsgID *string) (*model.MessageEntry, error) {
	tx, err := r.db.Begin(ctx)

	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	message := &model.Message{}
	user := &model.User{}
	query := `
                      WITH inserted AS (
                              INSERT INTO messages (conversation_id, sender_id, body, client_msg_id)
                              VALUES ($1, $2, $3, $4)
                              ON CONFLICT (conversation_id, sender_id, client_msg_id)
                              DO UPDATE SET body = messages.body
                              RETURNING id, conversation_id, sender_id, body, client_msg_id, created_at, edited_at
                      )
                      SELECT
                              m.id, m.conversation_id, m.sender_id, m.body, m.client_msg_id,
                              m.created_at, m.edited_at,
                              u.id, u.username, u.email, u.created_at, u.updated_at
                      FROM inserted m
                      JOIN users u ON u.id = m.sender_id
              `

	err1 := tx.QueryRow(ctx,
		query,
		conversationID,
		senderID,
		body,
		clientMsgID).Scan(
		&message.ID,
		&message.ConversationID,
		&message.SenderID,
		&message.Body,
		&message.ClientMsgID,
		&message.CreatedAt,
		&message.EditedAt,
		&user.ID,
		&user.Username,
		&user.Email,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err1 != nil {
		return nil, err1
	}

	_, err2 := tx.Exec(
		ctx,
		`UPDATE conversations SET updated_at = NOW() WHERE id = $1`,
		conversationID,
	)
	if err2 != nil {
		return nil, err2
	}

	if err3 := tx.Commit(ctx); err3 != nil {
		return nil, err3
	}

	return &model.MessageEntry{
		Message: message,
		Sender:  user,
	}, nil
}

/*
ListMessages returns one page of a conversation's messages, newest first,
each with its sender

the page is keyed on id rather than an offset. `before` is the id the client already has
and wants to read backwards from;
nil asks for newset page.
Both forms are a seek into idx_messages_conversation_id folowed by a scan of exactly `limit` rows,
so scrolling into a long conversation cost the same as openinnig it

the Null check on the cursor keeps this one statement intead of two nearly
identical ones. The cast on $2 is required, not cosmetic: without it postgres cannot infer a type for parameter that only ever appears next to NULL.

Like ListMembers, this does not check who is asking.
The service establishes
membership first, so the permission lisves in one place.
*/
func (r *MessageRepository) ListMessages(ctx context.Context, conversationID int64, before *int64, limit int) ([]*model.MessageEntry, error) {
	query := `
		SELECT m.id, m.conversation_id , m.sender_id, m.body,
			m.client_msg_id, m.created_at, m.edited_at,
			u.id , u.username, u.email , u.created_at , u.updated_at
		FROM messages m
		JOIN users u ON u.id = m.sender_id
		WHERE m.conversation_id = $1 AND ($2::BIGINT IS NULL OR m.id < $2)
		ORDER BY m.id DESC
		LIMIT $3
	`

	rows, err := r.db.Query(ctx, query, conversationID, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]*model.MessageEntry, 0, limit)

	for rows.Next() {
		message := &model.Message{}
		user := &model.User{}

		if err := rows.Scan(
			&message.ID,
			&message.ConversationID,
			&message.SenderID,
			&message.Body,
			&message.ClientMsgID,
			&message.CreatedAt,
			&message.EditedAt,
			&user.ID,
			&user.Username,
			&user.Email,
			&user.CreatedAt,
			&user.UpdatedAt,
		); err != nil {
			return nil, err
		}

		entry := &model.MessageEntry{
			Message: message,
			Sender:  user,
		}

		if err := rows.Err(); err != nil {
			return nil, err
		}
		entries = append(entries, entry)

	}

	return entries, nil
}
