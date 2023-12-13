package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
	appErrors "github.com/ujwegh/gophermart/internal/app/errors"
	"github.com/ujwegh/gophermart/internal/app/models"
)

type UserRepository interface {
	Create(ctx context.Context, tx *sqlx.Tx, user *models.User) error
	FindByLogin(ctx context.Context, login string) (*models.User, error)
	GetDB() *sqlx.DB
}

type UserRepositoryImpl struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepositoryImpl {
	return &UserRepositoryImpl{db: db}
}

func (ur *UserRepositoryImpl) FindByLogin(ctx context.Context, login string) (*models.User, error) {
	query := `SELECT * FROM users WHERE login = $1;`
	user := models.User{}
	err := ur.db.GetContext(ctx, &user, query, login)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, appErrors.New(err, "User not found")
		}
		return nil, fmt.Errorf("get user: %w", err)
	}
	return &user, nil
}

func (ur *UserRepositoryImpl) Create(ctx context.Context, tx *sqlx.Tx, user *models.User) error {
	query := `INSERT INTO users (uuid, login, password_hash, created_at) VALUES ($1, $2, $3, $4);`
	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, user.UUID, user.Login, user.PasswordHash, user.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return appErrors.New(err, "User already exists")
		}
		return fmt.Errorf("exec statement: %w", err)
	}
	return nil
}

func (ur *UserRepositoryImpl) GetDB() *sqlx.DB {
	return ur.db
}
