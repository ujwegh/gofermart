package handlers

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	appContext "github.com/ujwegh/gophermart/internal/app/context"
	"github.com/ujwegh/gophermart/internal/app/models"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type MockWalletService struct {
	mock.Mock
}

type MockWithdrawalService struct {
	mock.Mock
}

func (m *MockWalletService) CreateWallet(ctx context.Context, tx *sqlx.Tx, userUID *uuid.UUID) error {
	args := m.Called(ctx, tx, userUID)
	return args.Error(1)
}

func (m *MockWalletService) GetWallet(ctx context.Context, userUID *uuid.UUID) (*models.Wallet, error) {
	args := m.Called(ctx, userUID)
	return args.Get(0).(*models.Wallet), args.Error(1)
}

func (m *MockWalletService) Credit(ctx context.Context, tx *sqlx.Tx, userUID *uuid.UUID, amount float64) (*models.Wallet, error) {
	args := m.Called(ctx, tx, userUID, amount)
	return args.Get(0).(*models.Wallet), args.Error(1)
}

func (m *MockWalletService) Debit(ctx context.Context, tx *sqlx.Tx, userUID *uuid.UUID, amount float64) (*models.Wallet, error) {
	args := m.Called(ctx, tx, userUID, amount)
	return args.Get(0).(*models.Wallet), args.Error(1)
}

func (m *MockWalletService) GetBalance(ctx context.Context, userUID *uuid.UUID) (*models.UserBalance, error) {
	args := m.Called(ctx, userUID)
	return args.Get(0).(*models.UserBalance), args.Error(1)
}

func (m *MockWithdrawalService) CreateWithdrawal(ctx context.Context, userUID *uuid.UUID, order string, sum float64) error {
	args := m.Called(ctx, userUID, order, sum)
	return args.Error(0)
}

func (m *MockWithdrawalService) GetWithdrawals(ctx context.Context, userUID *uuid.UUID) (*[]models.Withdrawal, error) {
	args := m.Called(ctx, userUID)
	return args.Get(0).(*[]models.Withdrawal), args.Error(1)
}

func TestBalanceHandler_GetBalance(t *testing.T) {
	userUID := uuid.New()
	tests := []struct {
		name              string
		mockWalletService func() *MockWalletService
		contextTimeout    time.Duration
		userUID           *uuid.UUID
		wantErr           bool
		wantStatusCode    int
		wantResponseBody  string
	}{
		{
			name: "Successful Balance Retrieval",
			mockWalletService: func() *MockWalletService {
				m := &MockWalletService{}
				balance := &models.UserBalance{CurrentBalance: 100.0, WithdrawnBalance: 50.0}
				m.On("GetBalance", mock.Anything, mock.Anything).Return(balance, nil)
				return m
			},
			contextTimeout:   5 * time.Second,
			userUID:          &userUID,
			wantErr:          false,
			wantStatusCode:   http.StatusOK,
			wantResponseBody: "{\"current\":100.0,\"withdrawn\":50.0}", // Expected JSON response
		},
		{
			name: "Error in Fetching Balance",
			mockWalletService: func() *MockWalletService {
				m := &MockWalletService{}
				err := errors.New("internal server error")
				m.On("GetBalance", mock.Anything, mock.Anything).Return((*models.UserBalance)(nil), err)
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
			mockWalletService: func() *MockWalletService {
				m := &MockWalletService{}
				balance := &models.UserBalance{CurrentBalance: 100.0, WithdrawnBalance: 50.0}
				m.On("GetBalance", mock.Anything, mock.Anything).Return(balance, nil)
				return m
			},
			contextTimeout:   0,
			userUID:          &userUID,
			wantErr:          true,
			wantStatusCode:   http.StatusInternalServerError,
			wantResponseBody: "{\"code\":500,\"message\":\"Timeout exceeded\"}\n",
		},
		// Add more test cases as needed
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare the request and response recorder
			req, err := http.NewRequest("GET", "/api/user/balance", nil)
			assert.NoError(t, err)

			// Add user UID to the request context
			ctx := appContext.WithUserUID(req.Context(), tt.userUID)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			// Create BalanceHandler with mocked service
			bh := &BalanceHandler{
				walletService:  tt.mockWalletService(),
				contextTimeout: tt.contextTimeout,
			}

			// Call the method
			bh.GetBalance(w, req)

			// Validate the results
			assert.Equal(t, tt.wantStatusCode, w.Code)
			assert.JSONEq(t, tt.wantResponseBody, w.Body.String())
		})
	}
}

func TestBalanceHandler_GetWithdrawals(t *testing.T) {
	userUID := uuid.New()
	tests := []struct {
		name                  string
		mockWithdrawalService func() *MockWithdrawalService
		contextTimeout        time.Duration
		userUID               *uuid.UUID
		wantErr               bool
		wantStatusCode        int
		wantResponseBody      string
	}{
		{
			name: "Successful Withdrawal Retrieval",
			mockWithdrawalService: func() *MockWithdrawalService {
				m := &MockWithdrawalService{}
				withdrawals := &[]models.Withdrawal{
					{OrderID: "order1", Amount: 100.0, CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)},
					{OrderID: "order2", Amount: 200.0, CreatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC)},
				}
				m.On("GetWithdrawals", mock.Anything, mock.Anything).Return(withdrawals, nil)
				return m
			},
			contextTimeout: 5 * time.Second,
			userUID:        &userUID,
			wantErr:        false,
			wantStatusCode: http.StatusOK,
			wantResponseBody: `[
									{"order":"order1","sum":100,"processed_at":"2021-01-01T00:00:00Z"},
									{"order":"order2","sum":200,"processed_at":"2021-01-02T00:00:00Z"}
								]`,
		},
		{
			name: "No Withdrawals Found",
			mockWithdrawalService: func() *MockWithdrawalService {
				m := &MockWithdrawalService{}
				m.On("GetWithdrawals", mock.Anything, mock.Anything).Return(&[]models.Withdrawal{}, nil)
				return m
			},
			contextTimeout:   5 * time.Second,
			userUID:          &userUID,
			wantErr:          false,
			wantStatusCode:   http.StatusNoContent,
			wantResponseBody: "[]",
		},
		{
			name: "Error in Fetching Withdrawals",
			mockWithdrawalService: func() *MockWithdrawalService {
				m := &MockWithdrawalService{}
				err := errors.New("internal server error")
				m.On("GetWithdrawals", mock.Anything, mock.Anything).Return((*[]models.Withdrawal)(nil), err)
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
			mockWithdrawalService: func() *MockWithdrawalService {
				m := &MockWithdrawalService{}
				withdrawals := &[]models.Withdrawal{
					{OrderID: "order1", Amount: 100.0, CreatedAt: time.Now()},
				}
				m.On("GetWithdrawals", mock.Anything, mock.Anything).Return(withdrawals, nil)
				return m
			},
			contextTimeout:   0, // 0 seconds timeout to trigger the timeout error
			userUID:          &userUID,
			wantErr:          true,
			wantStatusCode:   http.StatusInternalServerError,
			wantResponseBody: "{\"code\":500,\"message\":\"Timeout exceeded\"}\n",
		},
		// Add more test cases as needed
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare the request and response recorder
			req, err := http.NewRequest("GET", "/api/withdrawals", nil)
			assert.NoError(t, err)

			// Add user UID to the request context
			ctx := appContext.WithUserUID(req.Context(), tt.userUID)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			// Create BalanceHandler with mocked service
			bh := &BalanceHandler{
				withdrawalService: tt.mockWithdrawalService(),
				contextTimeout:    tt.contextTimeout,
			}

			// Call the method
			bh.GetWithdrawals(w, req)

			// Validate the results
			assert.Equal(t, tt.wantStatusCode, w.Code)
			assert.JSONEq(t, tt.wantResponseBody, w.Body.String())
		})
	}
}

func TestBalanceHandler_Withdraw(t *testing.T) {
	userUID := uuid.New()
	tests := []struct {
		name                  string
		requestBody           string
		mockWithdrawalService func() *MockWithdrawalService
		contextTimeout        time.Duration
		userUID               *uuid.UUID
		wantErr               bool
		wantStatusCode        int
		wantResponseBody      string
	}{
		{
			name:        "Successful Withdrawal",
			requestBody: `{"order":"354188083613","sum":100.0}`,
			mockWithdrawalService: func() *MockWithdrawalService {
				m := &MockWithdrawalService{}
				m.On("CreateWithdrawal", mock.Anything, mock.Anything, "354188083613", 100.0).Return(nil)
				return m
			},
			contextTimeout: 5 * time.Second,
			userUID:        &userUID,
			wantErr:        false,
			wantStatusCode: http.StatusOK,
		},
		{
			name:        "Invalid Order ID",
			requestBody: `{"order":"123","sum":100.0}`,
			mockWithdrawalService: func() *MockWithdrawalService {
				m := &MockWithdrawalService{}
				return m
			},
			contextTimeout:   5 * time.Second,
			userUID:          &userUID,
			wantErr:          true,
			wantStatusCode:   http.StatusUnprocessableEntity,
			wantResponseBody: "{\"code\":422, \"message\":\"Invalid order ID\"}",
		},
		{
			name:        "Invalid Request Body",
			requestBody: `{"order":354188083613,"sum":"100.0"}`, // Malformed JSON
			mockWithdrawalService: func() *MockWithdrawalService {
				m := &MockWithdrawalService{}
				return m
			},
			contextTimeout:   5 * time.Second,
			userUID:          &userUID,
			wantErr:          true,
			wantStatusCode:   http.StatusBadRequest,
			wantResponseBody: "{\"code\":400, \"message\":\"Unable to parse body\"}",
		},
		{
			name:        "Error in Withdrawal Service",
			requestBody: `{"order":"354188083613","sum":100.0}`,
			mockWithdrawalService: func() *MockWithdrawalService {
				m := &MockWithdrawalService{}
				err := errors.New("internal server error")
				m.On("CreateWithdrawal", mock.Anything, mock.Anything, "354188083613", 100.0).Return(err)
				return m
			},
			contextTimeout: 5 * time.Second,
			userUID:        &userUID,
			wantErr:        true,
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name:        "Context Timeout",
			requestBody: `{"order":"354188083613","sum":100.0}`,
			mockWithdrawalService: func() *MockWithdrawalService {
				m := &MockWithdrawalService{}
				m.On("CreateWithdrawal", mock.Anything, mock.Anything, "354188083613", 100.0).Return(nil)
				return m
			},
			contextTimeout: 0, // 0 seconds timeout to trigger the timeout error
			userUID:        &userUID,
			wantErr:        true,
			wantStatusCode: http.StatusInternalServerError,
			// If your implementation returns a specific error message for timeout, include it here
			wantResponseBody: "{\"code\":500,\"message\":\"Timeout exceeded\"}\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare the request and response recorder
			body := strings.NewReader(tt.requestBody)
			req, err := http.NewRequest("POST", "/api/withdraw", body)
			assert.NoError(t, err)

			// Add user UID to the request context
			ctx := appContext.WithUserUID(req.Context(), tt.userUID)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			// Create BalanceHandler with mocked service
			bh := &BalanceHandler{
				withdrawalService: tt.mockWithdrawalService(),
				contextTimeout:    tt.contextTimeout,
			}

			// Call the method
			bh.Withdraw(w, req)

			// Validate the results
			assert.Equal(t, tt.wantStatusCode, w.Code)
			if tt.wantErr {
				if tt.wantResponseBody != "" {
					assert.JSONEq(t, tt.wantResponseBody, w.Body.String())
				} else {
					// Generic error response validation if no specific body is expected
					assert.JSONEq(t, "{\"code\":500,\"message\":\"Internal Server Error\"}\n", w.Body.String())
				}
			} else {
				if tt.wantResponseBody != "" {
					assert.Equal(t, tt.wantResponseBody, w.Body.String())
				} else {
					assert.Empty(t, w.Body.String())
				}
			}
		})
	}
}
