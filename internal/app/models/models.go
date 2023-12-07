package models

import "github.com/google/uuid"

type (
	//easyjson:json
	User struct {
		UUID         uuid.UUID `json:"uuid" db:"uuid"`
		Email        string    `json:"email" db:"email"`
		Name         string    `json:"name" db:"name"`
		PasswordHash string    `json:"password_hash" db:"password_hash"`
	}
)
