package model

import "time"

// we are talking now about the dto ( domain object) of the user, which is the model of the user, and this is the object that we will use to represent the user in our application. This is the object that we will use to interact with the database, and this is the object that we will use to return to the client.

type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
