package models

import (
	"github.com/google/uuid"
	"time"
)

type (
	User struct {
		UUID         uuid.UUID `db:"uuid"`
		Login        string    `db:"login"`
		PasswordHash string    `db:"password_hash"`
		CreatedAt    time.Time `db:"created_at"`
	}
	Order struct {
		ID        string    `db:"id"`
		UserUUID  uuid.UUID `db:"user_uuid"`
		Status    Status    `db:"status"`
		Accrual   *float64  `db:"accrual"`
		CreatedAt time.Time `db:"created_at"`
		UpdatedAt time.Time `db:"updated_at"`
	}
	UserBalance struct {
		CurrentBalance   float64
		WithdrawnBalance float64
	}
	Withdrawal struct {
		ID        int64     `db:"id"`
		UserUUID  uuid.UUID `db:"user_uuid"`
		OrderID   string    `db:"order_id"`
		Amount    float64   `db:"amount"`
		CreatedAt time.Time `db:"created_at"`
	}
	Wallet struct {
		ID        int64     `db:"id"`
		UserUUID  uuid.UUID `db:"user_uuid"`
		Credits   float64   `db:"credits"`
		Debits    float64   `db:"debits"`
		CreatedAt time.Time `db:"created_at"`
		UpdatedAt time.Time `db:"updated_at"`
	}
)

type Status string

func (s Status) String() string {
	return string(s)
}

const (
	NEW        Status = "NEW"
	PROCESSING Status = "PROCESSING"
	INVALID    Status = "INVALID"
	PROCESSED  Status = "PROCESSED"
)
