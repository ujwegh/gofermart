package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	appErrors "github.com/ujwegh/gophermart/internal/app/errors"
	"net/http"
	"time"
)

type (
	Order struct {
		ID        string    `db:"id"`
		UserUUID  uuid.UUID `db:"user_uuid"`
		Status    Status    `db:"status"`
		Accrual   *float64  `db:"accrual"`
		CreatedAt time.Time `db:"created_at"`
		UpdatedAt time.Time `db:"updated_at"`
	}
	Status          string
	OrderRepository interface {
		CreateOrder(ctx context.Context, order *Order) error
		GetOrderByID(ctx context.Context, orderID string) (*Order, error)
		GetOrdersByUserUID(ctx context.Context, userUID *uuid.UUID) (*[]Order, error)
		UpdateOrder(ctx context.Context, tx *sqlx.Tx, order *Order) error
		CountUnprocessedOrders() (int, error)
		GetUnprocessedOrders(limit int, offset int) (*[]Order, error)
		GetDB() *sqlx.DB
	}
	OrderRepositoryImpl struct {
		db *sqlx.DB
	}
)

func (s Status) String() string {
	return string(s)
}

const (
	NEW        Status = "NEW"
	PROCESSING Status = "PROCESSING"
	INVALID    Status = "INVALID"
	PROCESSED  Status = "PROCESSED"
)

func NewOrderRepository(db *sqlx.DB) *OrderRepositoryImpl {
	return &OrderRepositoryImpl{db: db}
}

func (or *OrderRepositoryImpl) CreateOrder(ctx context.Context, order *Order) error {
	tx, err := or.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	query := `INSERT INTO orders (id, user_uuid, status, created_at, updated_at) VALUES ($1, $2, $3, $4, $5);`
	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, order.ID, order.UserUUID, order.Status.String(), order.CreatedAt, order.UpdatedAt)
	if err != nil {
		if err := tx.Rollback(); err != nil {
			return fmt.Errorf("rollback transaction: %w", err)
		}
		return err
	}
	return tx.Commit()
}

func (or *OrderRepositoryImpl) GetOrderByID(ctx context.Context, orderID string) (*Order, error) {
	query := `SELECT * FROM orders WHERE id = $1;`
	order := &Order{}
	err := or.db.GetContext(ctx, order, query, orderID)
	if err != nil {
		return nil, appErrors.NewWithCode(err, "Order not found", http.StatusNotFound)
	}
	return order, nil
}

func (or *OrderRepositoryImpl) GetOrdersByUserUID(ctx context.Context, userUID *uuid.UUID) (*[]Order, error) {
	query := `SELECT * FROM orders WHERE user_uuid = $1 order by created_at desc;`
	orders := make([]Order, 0)
	err := or.db.SelectContext(ctx, &orders, query, userUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &orders, nil
		}
		return nil, fmt.Errorf("read user orders: %w", err)
	}
	return &orders, nil
}

func (or *OrderRepositoryImpl) UpdateOrder(ctx context.Context, tx *sqlx.Tx, order *Order) error {
	query := `UPDATE orders SET status = $1, accrual = $2, updated_at = $3 WHERE id = $4`
	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, order.Status.String(), order.Accrual, order.UpdatedAt, order.ID)
	if err != nil {
		return fmt.Errorf("execute statement: %w", err)
	}
	return nil
}

func (or *OrderRepositoryImpl) CountUnprocessedOrders() (int, error) {
	query := `SELECT count(*) FROM orders WHERE status = 'NEW' or status = 'PROCESSING'`
	var count int
	err := or.db.Get(&count, query)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (or *OrderRepositoryImpl) GetUnprocessedOrders(limit int, offset int) (*[]Order, error) {
	query := `SELECT * FROM orders WHERE status = 'NEW' or status = 'PROCESSING' limit $1 offset $2`
	orders := make([]Order, 0)
	err := or.db.Select(&orders, query, limit, offset)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &orders, nil
		}
		return nil, fmt.Errorf("read not finished orders: %w", err)
	}
	return &orders, nil
}

func (or *OrderRepositoryImpl) GetDB() *sqlx.DB {
	return or.db
}
