package service

import (
	"context"
	"fmt"
	"github.com/ujwegh/gophermart/internal/app/logger"
	"github.com/ujwegh/gophermart/internal/app/models"
	"github.com/ujwegh/gophermart/internal/app/repository"
	"github.com/ujwegh/gophermart/internal/app/service/clients"
	"go.uber.org/zap"
	"time"
)

type OrderProcessor interface {
	ProcessOrder(order *models.Order) error
}

type OrderProcessorImpl struct {
	orderRepo        repository.OrderRepository
	orderCache       OrderCache
	walletService    WalletService
	accrualClient    clients.AccrualClient
	processOrderChan chan models.Order
}

func NewOrderProcessor(orderRepo repository.OrderRepository,
	orderCache OrderCache,
	walletService WalletService,
	accrualClient clients.AccrualClient,
	processOrderChan chan models.Order) *OrderProcessorImpl {
	o := &OrderProcessorImpl{
		orderRepo:        orderRepo,
		orderCache:       orderCache,
		walletService:    walletService,
		accrualClient:    accrualClient,
		processOrderChan: processOrderChan,
	}
	o.ProcessUnfinishedOrders()
	return o
}

func (op *OrderProcessorImpl) ProcessUnfinishedOrders() {
	logger.Log.Info("start processing unfinished orders")
	totalOrders, err := op.orderRepo.CountUnprocessedOrders()
	if err != nil {
		logger.Log.Error("failed to count unprocessed orders", zap.Error(err))
		return
	}
	if totalOrders != 0 {
		cnt := 0
		for cnt < totalOrders {
			limit := 20
			offset := cnt
			orders, err := op.orderRepo.GetUnprocessedOrders(limit, offset)
			if err != nil {
				logger.Log.Error("failed to get unprocessed orders", zap.Error(err))
				return
			}
			for _, order := range *orders {
				op.processOrderChan <- order
			}
			cnt += 20
		}
	}
	logger.Log.Info("published unprocessed orders", zap.Int("total_orders", totalOrders))
}

func (op *OrderProcessorImpl) ProcessOrders(ctx context.Context) {
	for {
		select {
		case order := <-op.processOrderChan:
			logger.Log.Debug("processing order", zap.String("order_id", order.ID))
			orderInfo, err := op.accrualClient.GetOrderInfo(order.ID)
			if err != nil {
				logger.Log.Debug("error getting order info", zap.Error(err))
				op.orderCache.AddOrder(&order)
				continue
			}
			order.Accrual = &orderInfo.Accrual
			order.Status = mapAccrualResponseStatus(orderInfo)
			order.UpdatedAt = time.Now()

			err = op.updateOrder(&order)
			if err != nil {
				logger.Log.Error("failed to update order", zap.Error(err))
			}
		case <-ctx.Done():
			return
		}
	}
}

func (op *OrderProcessorImpl) updateOrder(order *models.Order) error {
	ctx := context.Background()

	db := op.orderRepo.GetDB()
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		op.orderCache.AddOrder(order)
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	if err := op.orderRepo.UpdateOrder(ctx, tx, order); err != nil {
		op.orderCache.AddOrder(order)
		if err := tx.Rollback(); err != nil {
			return fmt.Errorf("failed to rollback transaction: %w", err)
		}
		return fmt.Errorf("failed to update order: %w", err)
	}
	_, err = op.walletService.Credit(ctx, tx, &order.UserUUID, *order.Accrual)
	if err != nil {
		if err := tx.Rollback(); err != nil {
			return fmt.Errorf("failed to rollback transaction: %w", err)
		}
		op.orderCache.AddOrder(order)
		return fmt.Errorf("failed to credit: %w", err)
	}

	if err := tx.Commit(); err != nil {
		op.orderCache.AddOrder(order)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func mapAccrualResponseStatus(accrualResponse *clients.AccrualResponseDto) models.Status {
	switch accrualResponse.AccrualStatus {
	case clients.PROCESSING:
		return models.PROCESSING
	case clients.REGISTERED:
		return models.NEW
	case clients.INVALID:
		return models.INVALID
	case clients.PROCESSED:
		return models.PROCESSED
	}
	return models.INVALID
}
