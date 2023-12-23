package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	appErrors "github.com/ujwegh/gophermart/internal/app/errors"
	"github.com/ujwegh/gophermart/internal/app/repository"
	"net/http"
	"time"
)

type WithdrawalService interface {
	CreateWithdrawal(ctx context.Context, userUID *uuid.UUID, orderID string, amount float64) error
	GetWithdrawals(ctx context.Context, userUID *uuid.UUID) (*[]repository.Withdrawal, error)
}

type WithdrawalServiceImpl struct {
	withdrawalRepo repository.WithdrawalsRepository
	walletService  WalletService
}

func NewWithdrawalService(withdrawalRepo repository.WithdrawalsRepository, walletService WalletService) *WithdrawalServiceImpl {
	return &WithdrawalServiceImpl{
		withdrawalRepo: withdrawalRepo,
		walletService:  walletService,
	}
}

func (bs *WithdrawalServiceImpl) CreateWithdrawal(ctx context.Context, userUID *uuid.UUID, orderID string, amount float64) error {
	withdrawal := repository.Withdrawal{
		UserUUID:  *userUID,
		OrderID:   orderID,
		Amount:    amount,
		CreatedAt: time.Now(),
	}

	tx, err := bs.withdrawalRepo.GetDB().BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()
	wallet, err := bs.walletService.Debit(ctx, tx, userUID, amount)
	if err != nil {
		return err
	}
	if (wallet.Credits - wallet.Debits) < 0 {
		msg := "insufficient funds"
		return appErrors.NewWithCode(errors.New(msg), msg, http.StatusPaymentRequired)
	}
	err = bs.withdrawalRepo.CreateWithdrawal(ctx, tx, &withdrawal)
	if err != nil {
		return appErrors.NewWithCode(err, "create withdrawal", http.StatusInternalServerError)
	}

	return tx.Commit()
}

func (bs *WithdrawalServiceImpl) GetWithdrawals(ctx context.Context, userUID *uuid.UUID) (*[]repository.Withdrawal, error) {
	return bs.withdrawalRepo.GetWithdrawals(ctx, userUID)
}
