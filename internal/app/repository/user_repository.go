package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
	appErrors "github.com/ujwegh/gophermart/internal/app/errors"
	"github.com/ujwegh/gophermart/internal/app/models"
)

type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	FindByEmail(ctx context.Context, email string) (*models.User, error)
}

type UserRepositoryImpl struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepositoryImpl {
	return &UserRepositoryImpl{db: db}
}

func (ur *UserRepositoryImpl) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `SELECT uuid, email, name, password_hash FROM users WHERE email = $1;`
	user := models.User{}
	err := ur.db.GetContext(ctx, &user, query, email)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if user.Email == "" {
		return nil, appErrors.New(err, "User not found")
	}
	return &user, nil
}

func (ur *UserRepositoryImpl) Create(ctx context.Context, user *models.User) error {
	tx, err := ur.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	query := `INSERT INTO users (uuid, email, name, password_hash) VALUES ($1, $2, $3, $4);`
	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, query, user.UUID, user.Email, user.PasswordHash)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return appErrors.New(err, "User already exists")
		}
		if err := tx.Rollback(); err != nil {
			return fmt.Errorf("rollback transaction: %w", err)
		}
		return fmt.Errorf("exec statement: %w", err)
	}
	return tx.Commit()
}
