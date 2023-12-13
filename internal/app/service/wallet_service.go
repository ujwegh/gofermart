package service

import (
	"context"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	appErrors "github.com/ujwegh/gophermart/internal/app/errors"
	"github.com/ujwegh/gophermart/internal/app/models"
	"github.com/ujwegh/gophermart/internal/app/repository"
	"time"
)

type WalletService interface {
	CreateWallet(ctx context.Context, tx *sqlx.Tx, userUID *uuid.UUID) error
	GetWallet(ctx context.Context, userUID *uuid.UUID) (*models.Wallet, error)
	Credit(ctx context.Context, tx *sqlx.Tx, userUID *uuid.UUID, amount float64) (*models.Wallet, error)
	Debit(ctx context.Context, tx *sqlx.Tx, userUID *uuid.UUID, amount float64) (*models.Wallet, error)
	GetBalance(ctx context.Context, uid *uuid.UUID) (*models.UserBalance, error)
}

type WalletServiceImpl struct {
	walletRepo repository.WalletRepository
}

func NewWalletService(walletRepo repository.WalletRepository) *WalletServiceImpl {
	return &WalletServiceImpl{walletRepo: walletRepo}
}

func (ws *WalletServiceImpl) CreateWallet(ctx context.Context, tx *sqlx.Tx, userUID *uuid.UUID) error {
	now := time.Now()
	newWallet := models.Wallet{
		UserUUID:  *userUID,
		Credits:   0,
		Debits:    0,
		CreatedAt: now,
		UpdatedAt: now,
	}
	err := ws.walletRepo.CreateWallet(ctx, tx, &newWallet)
	if err != nil {
		return appErrors.New(err, "create wallet")
	}
	return nil
}

func (ws *WalletServiceImpl) GetWallet(ctx context.Context, userUID *uuid.UUID) (*models.Wallet, error) {
	wallet, err := ws.walletRepo.GetWallet(ctx, userUID)
	if err != nil {
		return nil, appErrors.New(err, "get wallet")
	}
	return wallet, nil
}

func (ws *WalletServiceImpl) Credit(ctx context.Context, tx *sqlx.Tx, userUID *uuid.UUID, amount float64) (*models.Wallet, error) {
	return ws.walletRepo.Credit(ctx, tx, userUID, amount)
}

func (ws *WalletServiceImpl) Debit(ctx context.Context, tx *sqlx.Tx, userUID *uuid.UUID, amount float64) (*models.Wallet, error) {
	return ws.walletRepo.Debit(ctx, tx, userUID, amount)
}

func (ws *WalletServiceImpl) GetBalance(ctx context.Context, uid *uuid.UUID) (*models.UserBalance, error) {
	wallet, err := ws.GetWallet(ctx, uid)
	if err != nil {
		return nil, err
	}
	return &models.UserBalance{
		CurrentBalance:   wallet.Credits - wallet.Debits,
		WithdrawnBalance: wallet.Debits,
	}, nil
}
