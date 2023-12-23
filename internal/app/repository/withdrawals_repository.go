package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"time"
)

type (
	Withdrawal struct {
		ID        int64     `db:"id"`
		UserUUID  uuid.UUID `db:"user_uuid"`
		OrderID   string    `db:"order_id"`
		Amount    float64   `db:"amount"`
		CreatedAt time.Time `db:"created_at"`
	}
	WithdrawalsRepository interface {
		CreateWithdrawal(ctx context.Context, tx *sqlx.Tx, withdrawal *Withdrawal) error
		GetWithdrawals(ctx context.Context, userUID *uuid.UUID) (*[]Withdrawal, error)
		GetDB() *sqlx.DB
	}
	WithdrawalsRepositoryImpl struct {
		db *sqlx.DB
	}
)

func NewWithdrawalsRepository(db *sqlx.DB) *WithdrawalsRepositoryImpl {
	return &WithdrawalsRepositoryImpl{db: db}
}

func (wr *WithdrawalsRepositoryImpl) CreateWithdrawal(ctx context.Context, tx *sqlx.Tx, withdrawal *Withdrawal) error {
	query := `INSERT INTO withdrawals (user_uuid, order_id, amount, created_at) VALUES ($1, $2, $3, $4);`
	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, withdrawal.UserUUID, withdrawal.OrderID, withdrawal.Amount, withdrawal.CreatedAt)
	if err != nil {
		return fmt.Errorf("exec statement: %w", err)
	}
	return nil
}

func (wr *WithdrawalsRepositoryImpl) GetWithdrawals(ctx context.Context, userUID *uuid.UUID) (*[]Withdrawal, error) {
	query := `SELECT * FROM withdrawals WHERE user_uuid = $1 order by created_at;`
	withdrawals := make([]Withdrawal, 0)
	err := wr.db.SelectContext(ctx, &withdrawals, query, userUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &withdrawals, nil
		}
		return nil, fmt.Errorf("read withdrawals: %w", err)
	}
	return &withdrawals, nil
}

func (wr *WithdrawalsRepositoryImpl) GetDB() *sqlx.DB {
	return wr.db
}
