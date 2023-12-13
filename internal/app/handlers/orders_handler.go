package handlers

import (
	"context"
	"errors"
	"fmt"
	"github.com/ShiraazMoollatjie/goluhn"
	appContext "github.com/ujwegh/gophermart/internal/app/context"
	appErrors "github.com/ujwegh/gophermart/internal/app/errors"
	"github.com/ujwegh/gophermart/internal/app/models"
	"github.com/ujwegh/gophermart/internal/app/service"
	"io"
	"net/http"
	"strings"
	"time"
)

type (
	OrdersHandler struct {
		orderService   service.OrderService
		contextTimeout time.Duration
	}

	//easyjson:json
	OrderDTO struct {
		OrderID    string    `json:"number"`
		Status     string    `json:"status"`
		Accrual    *float64  `json:"accrual,omitempty"`
		UploadedAt time.Time `json:"uploaded_at"`
	}
	//easyjson:json
	OrderDTOSlice []OrderDTO
)

func NewOrdersHandler(contextTimeoutSec int, orderService service.OrderService) *OrdersHandler {
	return &OrdersHandler{
		orderService:   orderService,
		contextTimeout: time.Duration(contextTimeoutSec) * time.Second,
	}
}

func (oh *OrdersHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), oh.contextTimeout)
	defer cancel()

	orderID, err := io.ReadAll(r.Body)
	if err != nil {
		err = appErrors.NewWithCode(err, errMsgEnableReadBody, http.StatusBadRequest)
		PrepareError(w, err)
		return
	}
	userUID := appContext.UserUID(r.Context())

	stringOrderID := string(orderID)
	err = goluhn.Validate(stringOrderID)
	if err != nil {
		err = appErrors.NewWithCode(err, "Invalid order ID", http.StatusUnprocessableEntity)
		PrepareError(w, err)
		return
	}
	_, err = oh.orderService.CreateOrder(ctx, stringOrderID, userUID)
	appErr := &appErrors.ResponseCodeError{}
	if err != nil && errors.As(err, appErr) && strings.Contains(appErr.Msg(), "repeated order") {
		w.WriteHeader(http.StatusOK)
		return
	} else if err != nil {
		PrepareError(w, err)
		return
	}

	err = appContext.GetContextError(ctx)
	if err != nil {
		PrepareError(w, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (oh *OrdersHandler) GetOrders(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), oh.contextTimeout)
	defer cancel()

	userUID := appContext.UserUID(r.Context())

	orders, err := oh.orderService.GetOrders(ctx, userUID)
	if err != nil {
		PrepareError(w, err)
		return
	}
	if len(*orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	response := oh.mapOrdersToOrderDtoSlice(orders)
	rawBytes, err := response.MarshalJSON()
	if err != nil {
		PrepareError(w, fmt.Errorf("marshal response: %w", err))
		return
	}
	err = appContext.GetContextError(ctx)
	if err != nil {
		PrepareError(w, err)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(rawBytes)
}

func (oh *OrdersHandler) mapOrdersToOrderDtoSlice(slice *[]models.Order) OrderDTOSlice {
	var responseSlice []OrderDTO
	for _, item := range *slice {
		responseItem := OrderDTO{
			OrderID:    item.ID,
			Status:     item.Status.String(),
			Accrual:    item.Accrual,
			UploadedAt: item.CreatedAt,
		}
		responseSlice = append(responseSlice, responseItem)
	}
	return responseSlice
}
