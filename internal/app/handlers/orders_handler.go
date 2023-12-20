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

// CreateOrder godoc
// @Summary Loading order number
// @Description The handler is only available to authenticated users and is used to upload a new order number.
//
//	The order number is a sequence of digits of arbitrary length and can be validated using the Luhn algorithm.
//
// @Tags order
// @Accept plain
// @Produce json
// @Param order body string true "Order Number"
// @Success 200 "The order number has already been uploaded by this user"
// @Success 202 "The new order number has been accepted for processing"
// @Failure 400 {object} ErrorResponse "Bad Request - Unable to read body or incorrect request format"
// @Failure 401 {object} ErrorResponse "Unauthorized - The user is not authenticated"
// @Failure 409 {object} ErrorResponse "Conflict - The order number has already been uploaded by another user"
// @Failure 422 {object} ErrorResponse "Unprocessable Entity - Incorrect order number format"
// @Failure 500 {object} ErrorResponse "Internal Server Error"
// @Security ApiKeyAuth
// @Router /api/user/orders [post]
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

// GetOrders godoc
// @Summary Getting a list of downloaded order numbers
// @Description The handler returns a list of order numbers sorted by loading time from oldest to newest for an authorized user.
// @Description The response includes the order number, status, accrual (if available), and the upload timestamp.
// @Tags orders
// @Produce json
// @Success 200 {array} OrderDTO "List of orders with details"
// @Success 204 "No orders to display"
// @Failure 401 {object} ErrorResponse "Unauthorized - The user is not authorized"
// @Failure 500 {object} ErrorResponse "Internal Server Error"
// @Security ApiKeyAuth
// @Router /api/user/orders [get]
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
