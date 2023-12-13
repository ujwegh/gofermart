package service

import (
	"github.com/patrickmn/go-cache"
	"github.com/ujwegh/gophermart/internal/app/logger"
	"github.com/ujwegh/gophermart/internal/app/models"
	"go.uber.org/zap"
	"time"
)

type OrderCache interface {
	AddOrder(order *models.Order)
}

type OrderCacheImpl struct {
	*cache.Cache
	orderChan chan models.Order
}

func NewOrderCache(defaultExpiration, cleanupInterval time.Duration, orderChan chan models.Order) *OrderCacheImpl {
	c := cache.New(defaultExpiration, cleanupInterval)
	c.OnEvicted(func(key string, value interface{}) {
		order, ok := value.(models.Order)
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

func (c *OrderCacheImpl) AddOrder(order *models.Order) {
	err := c.Add(order.ID, *order, cache.DefaultExpiration)
	if err != nil {
		logger.Log.Debug("Order already exists in cache", zap.String("order_id", order.ID))
	}
}
