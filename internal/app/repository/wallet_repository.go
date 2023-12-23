package repository

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"time"
)

type (
	Wallet struct {
		ID        int64     `db:"id"`
		UserUUID  uuid.UUID `db:"user_uuid"`
		Credits   float64   `db:"credits"`
		Debits    float64   `db:"debits"`
		CreatedAt time.Time `db:"created_at"`
		UpdatedAt time.Time `db:"updated_at"`
	}
	WalletRepository interface {
		CreateWallet(ctx context.Context, tx *sqlx.Tx, wallet *Wallet) error
		GetWallet(ctx context.Context, userUID *uuid.UUID) (*Wallet, error)
		Credit(ctx context.Context, tx *sqlx.Tx, userUID *uuid.UUID, amount float64) (*Wallet, error)
		Debit(ctx context.Context, tx *sqlx.Tx, userUID *uuid.UUID, amount float64) (*Wallet, error)
	}
	WalletRepositoryImpl struct {
		db *sqlx.DB
	}
)

func NewWalletRepository(db *sqlx.DB) *WalletRepositoryImpl {
	return &WalletRepositoryImpl{db: db}
}

func (wr *WalletRepositoryImpl) CreateWallet(ctx context.Context, tx *sqlx.Tx, wallet *Wallet) error {
	query := `INSERT INTO wallets (user_uuid, credits, debits, created_at, updated_at)
			  VALUES ($1, $2, $3, $4, $5) returning id;`
	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	err = stmt.QueryRowContext(ctx, wallet.UserUUID, wallet.Credits, wallet.Debits, wallet.CreatedAt, wallet.UpdatedAt).Scan(&wallet.ID)
	if err != nil {
		return fmt.Errorf("exec statement: %w", err)
	}
	return nil
}

func (wr *WalletRepositoryImpl) GetWallet(ctx context.Context, userUID *uuid.UUID) (*Wallet, error) {
	query := `SELECT * FROM wallets WHERE user_uuid = $1;`
	wallet := Wallet{}
	err := wr.db.GetContext(ctx, &wallet, query, userUID)
	if err != nil {
		return nil, fmt.Errorf("get wallet: %w", err)
	}
	return &wallet, nil
}

func (wr *WalletRepositoryImpl) Credit(ctx context.Context, tx *sqlx.Tx, userUID *uuid.UUID, amount float64) (*Wallet, error) {
	query := `UPDATE wallets SET credits = credits + $1 WHERE user_uuid = $2 returning *;`
	wallet := Wallet{}
	err := tx.GetContext(ctx, &wallet, query, amount, userUID)
	if err != nil {
		return nil, fmt.Errorf("credit: %w", err)
	}
	return &wallet, nil
}

func (wr *WalletRepositoryImpl) Debit(ctx context.Context, tx *sqlx.Tx, userUID *uuid.UUID, amount float64) (*Wallet, error) {
	query := `UPDATE wallets SET debits = debits + $1 WHERE user_uuid = $2 returning *;`
	wallet := Wallet{}
	err := tx.GetContext(ctx, &wallet, query, amount, userUID)
	if err != nil {
		return nil, fmt.Errorf("debit: %w", err)
	}
	return &wallet, nil
}
