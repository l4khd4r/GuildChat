package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/l4khd4r/GuildChat/internal/model"
)

type ConversationRepository struct {
	db *pgxpool.Pool
}

func NewConversationRepository(db *pgxpool.Pool) *ConversationRepository {
	return &ConversationRepository{
		db: db,
	}
}

func (r *ConversationRepository) GetDM(ctx context.Context, userID1 int64, userID2 int64) (*model.Conversation, error) {

	query := `
		SELECT c.id , c.type ,c.name , c.created_by , c.created_at , c.updated_at
		FROM conversations c
		JOIN conversation_members cm1 ON c.id = cm1.conversation_id
		JOIN conversation_members cm2 ON c.id = cm2.conversation_id
		WHERE c.type = $3
			AND cm1.user_id = $1
			AND cm2.user_id = $2
	`
	conversation := &model.Conversation{}

	err := r.db.QueryRow(ctx, query, userID1, userID2, model.ConversationDM).Scan(
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

func (r *ConversationRepository) CreateConversation(ctx context.Context, conversationType string, name *string, createdBy int64) (*model.Conversation, error) {
	query := `
		INSERT INTO conversations (type, name, created_by)
		VALUES ($1, $2, $3)
		RETURNING id, type, name, created_by, created_at, updated_at
	`

	conversation := &model.Conversation{}

	err := r.db.QueryRow(
		ctx,
		query,
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

func (r *ConversationRepository) AddMember(ctx context.Context, conversationID int64, userID int64, role string) error {
	query := `
		INSERT INTO conversation_members (conversation_id, user_id , role)
		VALUES ($1, $2, $3)
	`

	_, err := r.db.Exec(ctx, query, conversationID, userID, role)
	return err
}

// CreateRoom inserts a room and the creator's owner row as one unit.
//
// The two statements run in a transaction because a room without its owner is
// not a lesser room, it is a broken one: nobody can administer it, and the
// creator cannot even see it, since every read joins the caller through
// conversation_members. Either both rows land or neither does.
//
// (GetOrCreateDM still does its inserts unwrapped and has exactly that failure
// mode; it wants the same treatment.)
func (r *ConversationRepository) CreateRoom(ctx context.Context, name string, createdBy int64) (*model.Conversation, error) {

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}

	// No-op once the transaction has been committed; this is the rollback for
	// every path that returns early below.
	defer tx.Rollback(ctx)

	conversation := &model.Conversation{}

	err = tx.QueryRow(
		ctx,
		`
			INSERT INTO conversations (type, name, created_by)
			VALUES ($1, $2, $3)
			RETURNING id, type, name, created_by, created_at, updated_at
		`,
		model.ConversationRoom,
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

	_, err = tx.Exec(
		ctx,
		`
			INSERT INTO conversation_members (conversation_id, user_id, role)
			VALUES ($1, $2, $3)
		`,
		conversation.ID,
		createdBy,
		model.MemberOwner,
	)

	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return conversation, nil
}

// ListUserConversations returns every conversation the user belongs to,
// newest activity first, with the per-row extras described on
// model.ConversationListEntry.
//
// It is deliberately one round trip. The obvious version -- list the
// conversations, then loop and query the members of each -- is an N+1: twenty
// conversations become twenty-one queries. Both extras are folded into the
// same statement instead:
//
//   - member_count comes from a scalar sub-select counting the membership rows
//     of that conversation.
//   - the other user comes from a LATERAL sub-query, which is just a sub-query
//     allowed to reference the row being joined (c.id here). Its ON clause is
//     `c.type = $2`, so it only runs for DMs; for a room the join finds no
//     match and every o.* column arrives NULL.
//
// That NULL is why the other-user fields scan into pointers below: a room has
// no other user, and a nil *int64 is how the driver reports it.
// $1 is the caller and $2 is model.ConversationDM. Both callers below append
// their own tail (an ORDER BY, or a WHERE narrowing to one id), so the shape of
// a conversation entry is defined once. Keeping it in one place matters because
// the caller is joined *into* the query: a conversation the user is not a
// member of cannot come out of it, so this string is the authorisation rule as
// much as it is the projection.
const conversationEntryQuery = `
	SELECT
		c.id, c.type, c.name, c.created_by, c.created_at, c.updated_at,
		(
			SELECT COUNT(*)
			FROM conversation_members mc
			WHERE mc.conversation_id = c.id
		) AS member_count,
		o.id, o.username, o.email, o.created_at, o.updated_at
	FROM conversations c
	JOIN conversation_members me
		ON me.conversation_id = c.id
		AND me.user_id = $1
	LEFT JOIN LATERAL (
		SELECT u.id, u.username, u.email, u.created_at, u.updated_at
		FROM conversation_members om
		JOIN users u ON u.id = om.user_id
		WHERE om.conversation_id = c.id
			AND om.user_id <> $1
		LIMIT 1
	) o ON c.type = $2
`

// rowScanner is satisfied by both pgx.Row and pgx.Rows, so the single-row and
// the many-row query can share one scan.
type rowScanner interface {
	Scan(dest ...any) error
}

// scanConversationEntry reads one row of conversationEntryQuery.
func scanConversationEntry(row rowScanner) (*model.ConversationListEntry, error) {
	conversation := &model.Conversation{}
	var memberCount int

	// nil for a room, set for a DM
	var otherID *int64
	var otherUsername, otherEmail *string
	var otherCreatedAt, otherUpdatedAt *time.Time

	err := row.Scan(
		&conversation.ID,
		&conversation.Type,
		&conversation.Name,
		&conversation.CreatedBy,
		&conversation.CreatedAt,
		&conversation.UpdatedAt,
		&memberCount,
		&otherID,
		&otherUsername,
		&otherEmail,
		&otherCreatedAt,
		&otherUpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	entry := &model.ConversationListEntry{
		Conversation: conversation,
		MemberCount:  memberCount,
	}

	if otherID != nil {
		entry.OtherUser = &model.User{
			ID:        *otherID,
			Username:  *otherUsername,
			Email:     *otherEmail,
			CreatedAt: *otherCreatedAt,
			UpdatedAt: *otherUpdatedAt,
		}
	}

	return entry, nil
}

func (r *ConversationRepository) ListUserConversations(ctx context.Context, userID int64) ([]*model.ConversationListEntry, error) {

	query := conversationEntryQuery + `
		ORDER BY c.updated_at DESC
	`

	rows, err := r.db.Query(ctx, query, userID, model.ConversationDM)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	entries := make([]*model.ConversationListEntry, 0)

	for rows.Next() {
		entry, err := scanConversationEntry(rows)
		if err != nil {
			return nil, err
		}

		entries = append(entries, entry)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return entries, nil
}

// GetUserConversation returns one conversation, in the same shape as an entry
// of the list, provided the user is a member of it.
//
// Membership is not checked separately: conversationEntryQuery joins the caller
// in, so a conversation the user does not belong to produces no row and comes
// back as ErrNotFound -- the same answer as an id that does not exist. That is
// on purpose. Distinguishing "not yours" from "no such conversation" would let
// anyone probe ids to learn which conversations exist.
func (r *ConversationRepository) GetUserConversation(ctx context.Context, userID int64, conversationID int64) (*model.ConversationListEntry, error) {

	query := conversationEntryQuery + `
		WHERE c.id = $3
	`

	row := r.db.QueryRow(ctx, query, userID, model.ConversationDM, conversationID)

	entry, err := scanConversationEntry(row)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrConversationNotFound
	}

	if err != nil {
		return nil, err
	}

	return entry, nil
}
