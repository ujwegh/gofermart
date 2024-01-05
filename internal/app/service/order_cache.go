package service

import (
	"github.com/patrickmn/go-cache"
	"github.com/ujwegh/gophermart/internal/app/logger"
	"github.com/ujwegh/gophermart/internal/app/repository"
	"go.uber.org/zap"
	"time"
)

type OrderCache interface {
	AddOrder(order *repository.Order)
}

type OrderCacheImpl struct {
	*cache.Cache
	orderChan chan repository.Order
}

func NewOrderCache(defaultExpiration, cleanupInterval time.Duration, orderChan chan repository.Order) *OrderCacheImpl {
	c := cache.New(defaultExpiration, cleanupInterval)
	c.OnEvicted(func(key string, value interface{}) {
		order, ok := value.(repository.Order)
		if !ok {
			return
		}
		orderChan <- order
	})
	return &OrderCacheImpl{
		Cache:     c,
		orderChan: orderChan,
	}
}

func (c *OrderCacheImpl) AddOrder(order *repository.Order) {
	err := c.Add(order.ID, *order, cache.DefaultExpiration)
	if err != nil {
		logger.Log.Debug("Order already exists in cache", zap.String("order_id", order.ID))
	}
}
