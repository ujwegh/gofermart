package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	appErrors "github.com/ujwegh/gophermart/internal/app/errors"
	"github.com/ujwegh/gophermart/internal/app/models"
	"github.com/ujwegh/gophermart/internal/app/repository"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"time"
)

type UserService interface {
	Create(ctx context.Context, login, password string) (*models.User, error)
	Authenticate(ctx context.Context, login, password string) (*models.User, error)
	GetByUserLogin(ctx context.Context, login string) (*models.User, error)
}

type UserServiceImpl struct {
	userRepo      repository.UserRepository
	walletService WalletService
}

func NewUserService(userRepo repository.UserRepository, walletService WalletService) *UserServiceImpl {
	return &UserServiceImpl{
		userRepo:      userRepo,
		walletService: walletService,
	}
}

func (us *UserServiceImpl) Authenticate(ctx context.Context, login, password string) (*models.User, error) {
	user, err := us.GetByUserLogin(ctx, login)
	if err != nil {
		return nil, err
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, appErrors.NewWithCode(err, "Invalid password", http.StatusUnauthorized)
	}
	return user, nil
}

func (us *UserServiceImpl) GetByUserLogin(ctx context.Context, login string) (*models.User, error) {
	user, err := us.userRepo.FindByLogin(ctx, login)
	if err != nil {
		appErr := &appErrors.ResponseCodeError{}
		if errors.As(err, appErr) {
			return nil, appErr
		}
		return nil, fmt.Errorf("find user: %w", err)
	}
	return user, nil
}

func (us *UserServiceImpl) Create(ctx context.Context, login, password string) (*models.User, error) {
	passwordHash := generatePasswordHash(password)
	user := &models.User{
		UUID:         uuid.New(),
		Login:        login,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now(),
	}
	tx, err := us.userRepo.GetDB().BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := us.userRepo.Create(ctx, tx, user); err != nil {
		appErr := &appErrors.ResponseCodeError{}
		if errors.As(err, appErr) {
			return nil, appErrors.NewWithCode(err, appErr.Msg(), http.StatusConflict)
		}
		return nil, fmt.Errorf("create user: %w", err)
	}

	err = us.walletService.CreateWallet(ctx, tx, &user.UUID)
	if err != nil {
		return nil, err
	}

	return user, tx.Commit()
}

func generatePasswordHash(password string) string {
	hashedBytes, err := bcrypt.GenerateFromPassword(
		[]byte(password), bcrypt.DefaultCost)
	if err != nil {
		panic(fmt.Errorf("generate hash error: %w", err))
	}
	return string(hashedBytes)
}
