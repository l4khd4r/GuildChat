package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/l4khd4r/GuildChat/internal/model"
)

// type User struct
// {
// 	ID int64
// 	Username string
// 	Email string
// 	PasswordHash string
// }
// users
// ├── id
// ├── username
// ├── email
// └── password_hash

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

func (r *UserRepository) CreateUser(ctx context.Context, username string, email string, passwordHash string) (*model.User, error) {
	user := &model.User{}

	query := `
		INSERT INTO users (username , email , password_hash)
		VALUES ($1 , $2 , $3)
		RETURNING id , username , email , password_hash , created_at , updated_at
		`
	err := r.db.QueryRow(
		ctx,
		query,
		username,
		email,
		passwordHash,
	).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, mapError(err)
	}
	return user, nil
}

// mapError turns pgx errors into the package's sentinels so that callers can
// match them with errors.Is.
func mapError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrUserNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		switch pgErr.ConstraintName {
		case "users_username_key":
			return ErrUsernameAlreadyExists
		case "users_email_key":
			return ErrEmailAlreadyExists
		}
	}
	return err
}

func (r *UserRepository) GetByID(ctx context.Context, id int64) (*model.User, error) {
	user := &model.User{}
	query := `
		SELECT id , username , email , password_hash , created_at , updated_at
		FROM users
		WHERE id = $1
		`
	err := r.db.QueryRow(
		ctx,
		query,
		id,
	).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	user := &model.User{}
	query := `
		SELECT id , username , email , password_hash , created_at , updated_at
		FROM users
		WHERE email = $1
		`
	err := r.db.QueryRow(
		ctx,
		query,
		email,
	).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}


func (r *UserRepository) Update(ctx context.Context , id int64 , username string, email string  ) ( *model.User, error ) {
	user := &model.User{}

	query := `
		UPDATE users
		SET username = $1,
			email = $2,
			updated_at = NOW()
		WHERE id = $3
		RETURNING id , username , email , password_hash , created_at , updated_at
		`
	err := r.db.QueryRow(
		ctx,
		query,
		username,
		email,
		id,
	).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, mapError(err)
	}
	return user, nil
}


func (r *UserRepository) Delete(ctx context.Context, id int64) error {
	query := `
		DELETE FROM users
		WHERE id = $1
		`
	cmdTag, err := r.db.Exec(
		ctx,
		query,
		id,
	)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}
