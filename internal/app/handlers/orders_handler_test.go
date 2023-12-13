package handlers

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	appContext "github.com/ujwegh/gophermart/internal/app/context"
	appErrors "github.com/ujwegh/gophermart/internal/app/errors"
	"github.com/ujwegh/gophermart/internal/app/models"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type MockOrderService struct {
	mock.Mock
}

func (m *MockOrderService) CreateOrder(ctx context.Context, orderID string, userUID *uuid.UUID) (*models.Order, error) {
	args := m.Called(ctx, orderID, userUID)
	return args.Get(0).(*models.Order), args.Error(1)
}

func (m *MockOrderService) GetOrderByID(ctx context.Context, orderID string) (*models.Order, error) {
	args := m.Called(ctx, orderID)
	return args.Get(0).(*models.Order), args.Error(1)
}

func (m *MockOrderService) GetOrders(ctx context.Context, uid *uuid.UUID) (*[]models.Order, error) {
	args := m.Called(ctx, uid)
	return args.Get(0).(*[]models.Order), args.Error(1)
}

func TestOrdersHandler_CreateOrder(t *testing.T) {
	tests := []struct {
		name             string
		requestBody      string
		mockOrderService func() *MockOrderService
		contextTimeout   time.Duration
		wantErr          bool
		wantStatusCode   int
		wantResponseBody string
	}{
		{
			name:        "Successful Order Creation",
			requestBody: "354188083613",
			mockOrderService: func() *MockOrderService {
				m := &MockOrderService{}
				m.On("CreateOrder", mock.Anything, "354188083613", mock.Anything).Return(&models.Order{}, nil)
				return m
			},
			contextTimeout:   5 * time.Second,
			wantErr:          false,
			wantStatusCode:   http.StatusAccepted,
			wantResponseBody: "",
		},
		{
			name:        "Invalid Order ID",
			requestBody: `"123"`,
			mockOrderService: func() *MockOrderService {
				m := &MockOrderService{}
				return m
			},
			contextTimeout:   5 * time.Second,
			wantErr:          true,
			wantStatusCode:   http.StatusUnprocessableEntity,
			wantResponseBody: "{\"code\":422,\"message\":\"Invalid order ID\"}\n",
		},
		{
			name:        "Repeated Order",
			requestBody: "354188083613",
			mockOrderService: func() *MockOrderService {
				m := &MockOrderService{}
				err := appErrors.New(errors.New(""), "repeated order")
				m.On("CreateOrder", mock.Anything, "354188083613", mock.Anything).Return((*models.Order)(nil), err)
				return m
			},
			contextTimeout:   5 * time.Second,
			wantErr:          false,
			wantStatusCode:   http.StatusOK,
			wantResponseBody: "",
		},
		{
			name:        "Error in Order Creation",
			requestBody: "354188083613",
			mockOrderService: func() *MockOrderService {
				m := &MockOrderService{}
				err := errors.New("internal server error")
				m.On("CreateOrder", mock.Anything, "354188083613", mock.Anything).Return((*models.Order)(nil), err)
				return m
			},
			contextTimeout:   5 * time.Second,
			wantErr:          true,
			wantStatusCode:   http.StatusInternalServerError,
			wantResponseBody: "{\"code\":500,\"message\":\"Internal Server Error\"}\n",
		},
		{
			name:        "Context Timeout",
			requestBody: "354188083613",
			mockOrderService: func() *MockOrderService {
				m := &MockOrderService{}
				m.On("CreateOrder", mock.Anything, "354188083613", mock.Anything).Return(&models.Order{}, nil)
				return m
			},
			contextTimeout:   0,
			wantErr:          true,
			wantStatusCode:   http.StatusInternalServerError,
			wantResponseBody: "{\"code\":500,\"message\":\"Timeout exceeded\"}\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare the request and response recorder
			body := strings.NewReader(tt.requestBody)
			req, err := http.NewRequest("POST", "/api/user/orders", body)
			assert.NoError(t, err)
			w := httptest.NewRecorder()

			// Create OrdersHandler with mocked service
			oh := &OrdersHandler{
				orderService:   tt.mockOrderService(),
				contextTimeout: tt.contextTimeout,
			}

			// Call the method
			oh.CreateOrder(w, req)

			// Validate the results
			assert.Equal(t, tt.wantStatusCode, w.Code)
			if tt.wantErr {
				assert.JSONEq(t, tt.wantResponseBody, w.Body.String())
			} else {
				assert.Empty(t, w.Body.String())
			}
		})
	}
}

// Define the mock methods for OrderService as needed

func TestOrdersHandler_GetOrders(t *testing.T) {
	userUID := uuid.New()
	tests := []struct {
		name             string
		mockOrderService func() *MockOrderService
		contextTimeout   time.Duration
		userUID          *uuid.UUID
		wantErr          bool
		wantStatusCode   int
		wantResponseBody string
	}{
		{
			name: "Successful Retrieval of Orders",
			mockOrderService: func() *MockOrderService {
				m := &MockOrderService{}
				var accrual = 55.6
				orders := &[]models.Order{
					{ID: "order1", Status: models.NEW, Accrual: nil, CreatedAt: time.Now()},
					{ID: "order2", Status: models.PROCESSED, Accrual: &accrual, CreatedAt: time.Now()},
				}
				m.On("GetOrders", mock.Anything, mock.Anything).Return(orders, nil)
				return m
			},
			contextTimeout:   5 * time.Second,
			userUID:          &userUID,
			wantErr:          false,
			wantStatusCode:   http.StatusOK,
			wantResponseBody: "",
		},
		{
			name: "No Orders Found",
			mockOrderService: func() *MockOrderService {
				m := &MockOrderService{}
				m.On("GetOrders", mock.Anything, mock.Anything).Return(&[]models.Order{}, nil)
				return m
			},
			contextTimeout:   5 * time.Second,
			userUID:          &userUID,
			wantErr:          false,
			wantStatusCode:   http.StatusNoContent,
			wantResponseBody: "",
		},
		{
			name: "Error in Fetching Orders",
			mockOrderService: func() *MockOrderService {
				m := &MockOrderService{}
				err := errors.New("internal server error")
				m.On("GetOrders", mock.Anything, mock.Anything).Return((*[]models.Order)(nil), err)
				return m
			},
			contextTimeout:   5 * time.Second,
			userUID:          &userUID,
			wantErr:          true,
			wantStatusCode:   http.StatusInternalServerError,
			wantResponseBody: "{\"code\":500,\"message\":\"Internal Server Error\"}\n",
		},
		{
			name: "Context Timeout",
			mockOrderService: func() *MockOrderService {
				m := &MockOrderService{}
				orders := &[]models.Order{
					{ID: "order1", Status: models.NEW, Accrual: nil, CreatedAt: time.Now()},
				}
				m.On("GetOrders", mock.Anything, mock.Anything).Return(orders, nil)
				return m
			},
			contextTimeout:   0,
			userUID:          &userUID,
			wantErr:          true,
			wantStatusCode:   http.StatusInternalServerError,
			wantResponseBody: "{\"code\":500,\"message\":\"Timeout exceeded\"}\n",
		},
		{
			name: "Empty Orders",
			mockOrderService: func() *MockOrderService {
				m := &MockOrderService{}
				m.On("GetOrders", mock.Anything, mock.Anything).Return(&[]models.Order{}, nil)
				return m
			},
			contextTimeout:   5,
			userUID:          &userUID,
			wantErr:          false,
			wantStatusCode:   http.StatusNoContent,
			wantResponseBody: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare the request and response recorder
			req, err := http.NewRequest("GET", "/api/user/orders", nil)
			assert.NoError(t, err)

			// Add user UID to the request context
			ctx := appContext.WithUserUID(req.Context(), tt.userUID)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			// Create OrdersHandler with mocked service
			oh := &OrdersHandler{
				orderService:   tt.mockOrderService(),
				contextTimeout: tt.contextTimeout,
			}

			// Call the method
			oh.GetOrders(w, req)

			// Validate the results
			assert.Equal(t, tt.wantStatusCode, w.Code)
			if tt.wantErr {
				assert.JSONEq(t, tt.wantResponseBody, w.Body.String())
			}
		})
	}
}
