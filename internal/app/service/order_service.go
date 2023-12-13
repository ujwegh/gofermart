package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	appErrors "github.com/ujwegh/gophermart/internal/app/errors"
	"github.com/ujwegh/gophermart/internal/app/models"
	"github.com/ujwegh/gophermart/internal/app/repository"
	"net/http"
	"time"
)

type OrderService interface {
	CreateOrder(ctx context.Context, orderID string, userUID *uuid.UUID) (*models.Order, error)
	GetOrderByID(ctx context.Context, orderID string) (*models.Order, error)
	GetOrders(ctx context.Context, uid *uuid.UUID) (*[]models.Order, error)
}

type OrderServiceImpl struct {
	orderRepo     repository.OrderRepository
	walletService WalletService
	orderChan     chan models.Order
}

func NewOrderService(orderRepo repository.OrderRepository, walletService WalletService, processOrderChan chan models.Order) *OrderServiceImpl {
	return &OrderServiceImpl{
		orderRepo:     orderRepo,
		walletService: walletService,
		orderChan:     processOrderChan,
	}
}

func (os *OrderServiceImpl) CreateOrder(ctx context.Context, orderID string, userUID *uuid.UUID) (*models.Order, error) {
	order, err := os.GetOrderByID(ctx, orderID)
	appErr := &appErrors.ResponseCodeError{}
	if err != nil && !errors.As(err, appErr) {
		return nil, err
	}

	if order != nil && userUID.String() != order.UserUUID.String() {
		msg := "order already created by another user"
		return nil, appErrors.NewWithCode(errors.New(msg), msg, http.StatusConflict)
	} else if order != nil && userUID.String() == order.UserUUID.String() {
		msg := "repeated order"
		return nil, appErrors.New(errors.New(msg), msg)
	}

	now := time.Now()
	newOrder := &models.Order{
		ID:        orderID,
		UserUUID:  *userUID,
		Status:    models.NEW,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err = os.orderRepo.CreateOrder(ctx, newOrder); err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}
	os.orderChan <- *newOrder // send order to process channel
	return newOrder, nil
}

func (os *OrderServiceImpl) GetOrderByID(ctx context.Context, orderID string) (*models.Order, error) {
	return os.orderRepo.GetOrderByID(ctx, orderID)
}

func (os *OrderServiceImpl) GetOrders(ctx context.Context, uid *uuid.UUID) (*[]models.Order, error) {
	orders, err := os.orderRepo.GetOrdersByUserUID(ctx, uid)
	if err != nil {
		return nil, err
	}
	return orders, nil
}
