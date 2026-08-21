package repository
import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
)
type User struct
{
	ID int64
	Username string
	Email string
	PasswordHash string
}
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

func (r *UserRepository) CreateUser(ctx context.Context, username string , email string , passwordHash string  ) (*User , error){
	user := &User{}

	query := `
		INSERT INTO users (username , email , password_hash)
		VALUES ($1 , $2 , $3)
		RETURNING id , username , email , password_hash
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
	)


	if err != nil {
		return nil , err
	}
	return user , nil
}
