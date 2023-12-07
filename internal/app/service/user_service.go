package service

import (
	"context"
	"errors"
	"fmt"
	appErrors "github.com/ujwegh/gophermart/internal/app/errors"
	"github.com/ujwegh/gophermart/internal/app/models"
	"github.com/ujwegh/gophermart/internal/app/repository"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"strings"
)

type UserService interface {
	Create(ctx context.Context, email, password string) (*models.User, error)
	Authenticate(ctx context.Context, email, password string) (*models.User, error)
	GetByUserEmail(ctx context.Context, email string) (*models.User, error)
}

type UserServiceImpl struct {
	userRepo repository.UserRepository
}

func NewUserService(userRepo repository.UserRepository) *UserServiceImpl {
	return &UserServiceImpl{userRepo: userRepo}
}

func (us *UserServiceImpl) Authenticate(ctx context.Context, email, password string) (*models.User, error) {
	email = strings.ToLower(email)
	user, err := us.GetByUserEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, appErrors.NewWithCode(err, "Invalid password", http.StatusUnauthorized)
	}
	return user, nil
}

func (us *UserServiceImpl) GetByUserEmail(ctx context.Context, email string) (*models.User, error) {
	email = strings.ToLower(email)
	user, err := us.userRepo.FindByEmail(ctx, email)
	if err != nil {
		appErr := &appErrors.ResponseCodeError{}
		if errors.As(err, appErr) {
			return nil, appErr
		}
		return nil, fmt.Errorf("find user: %w", err)
	}
	return user, nil
}

func (us *UserServiceImpl) Create(ctx context.Context, email, password string) (*models.User, error) {
	email = strings.ToLower(email)
	passwordHash := generatePasswordHash(password)
	user := &models.User{
		Email:        email,
		PasswordHash: passwordHash,
	}
	err := us.userRepo.Create(ctx, user)
	if err != nil {
		appErr := &appErrors.ResponseCodeError{}
		if errors.As(err, appErr) {
			return nil, appErrors.NewWithCode(err, appErr.Msg(), http.StatusConflict)
		}
		return nil, fmt.Errorf("create user: %w", err)
	}
	return user, nil
}

func generatePasswordHash(password string) string {
	hashedBytes, err := bcrypt.GenerateFromPassword(
		[]byte(password), bcrypt.DefaultCost)
	if err != nil {
		panic(fmt.Errorf("generate hash error: %w", err))
	}
	return string(hashedBytes)
}
