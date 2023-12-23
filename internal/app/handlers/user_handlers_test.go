package handlers

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	appErrors "github.com/ujwegh/gophermart/internal/app/errors"
	"github.com/ujwegh/gophermart/internal/app/repository"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type MockUserService struct {
	mock.Mock
}
type MockTokenService struct {
	mock.Mock
}

func (m *MockUserService) Create(ctx context.Context, login, password string) (*repository.User, error) {
	args := m.Called(ctx, login, password)
	return args.Get(0).(*repository.User), args.Error(1)
}

func (m *MockUserService) GetByUserLogin(ctx context.Context, login string) (*repository.User, error) {
	args := m.Called(ctx, login)
	return args.Get(0).(*repository.User), args.Error(1)
}

func (m *MockUserService) Authenticate(ctx context.Context, login, password string) (*repository.User, error) {
	args := m.Called(ctx, login, password)
	return args.Get(0).(*repository.User), args.Error(1)
}

func (m *MockTokenService) GetUserLogin(tokenString string) (string, error) {
	args := m.Called(tokenString)
	return args.String(0), args.Error(1)
}

func (m *MockTokenService) GenerateToken(login string) (string, error) {
	args := m.Called(login)
	return args.String(0), args.Error(1)
}

func TestUserHandler_Login(t *testing.T) {
	tests := []struct {
		name             string
		request          string
		mockUserService  func() *MockUserService
		mockTokenService func() *MockTokenService
		contextTimeout   time.Duration
		wantErr          bool
		wantResponse     string
		wantStatusCode   int
	}{
		{
			name:    "Successful Login",
			request: `{"login":"testuser","password":"password"}`,
			mockUserService: func() *MockUserService {
				m := &MockUserService{}
				user := &repository.User{
					UUID:         uuid.New(),
					Login:        "testuser",
					PasswordHash: "passwordhash",
					CreatedAt:    time.Now(),
				}
				m.On("Authenticate", mock.Anything, "testuser", "password").Return(user, nil)
				return m
			},
			mockTokenService: func() *MockTokenService {
				m := &MockTokenService{}
				m.On("GenerateToken", "testuser").Return("secret-token", nil)
				return m
			},
			contextTimeout: 5 * time.Second,
			wantErr:        false,
			wantResponse:   "Bearer secret-token",
			wantStatusCode: http.StatusOK,
		},
		{
			name:    "Invalid Password",
			request: `{"login":"testuser","password":"password"}`,
			mockUserService: func() *MockUserService {
				m := &MockUserService{}
				err := appErrors.NewWithCode(errors.New(""), "Invalid password", http.StatusUnauthorized)
				m.On("Authenticate", mock.Anything, "testuser", "password").Return((*repository.User)(nil), err)
				return m
			},
			mockTokenService: func() *MockTokenService {
				m := &MockTokenService{}
				return m
			},
			contextTimeout: 5 * time.Second,
			wantErr:        true,
			wantResponse:   "{\"code\":401,\"message\":\"Invalid password\"}\n",
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name:    "Invalid Login Format",
			request: `{"login":"","password":"password"}`,
			mockUserService: func() *MockUserService {
				m := &MockUserService{}
				return m
			},
			mockTokenService: func() *MockTokenService {
				return &MockTokenService{}
			},
			contextTimeout: 5 * time.Second,
			wantErr:        true,
			wantResponse:   "{\"code\":400,\"message\":\"Login and password are required\"}\n",
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:    "Error in Token Generation",
			request: `{"login":"testuser","password":"password"}`,
			mockUserService: func() *MockUserService {
				m := &MockUserService{}
				user := &repository.User{
					UUID:         uuid.New(),
					Login:        "testuser",
					PasswordHash: "passwordhash",
					CreatedAt:    time.Now(),
				}
				m.On("Authenticate", mock.Anything, "testuser", "password").Return(user, nil)
				return m
			},
			mockTokenService: func() *MockTokenService {
				m := &MockTokenService{}
				m.On("GenerateToken", "testuser").Return("", errors.New("token generation error"))
				return m
			},
			contextTimeout: 5 * time.Second,
			wantErr:        true,
			wantResponse:   "{\"code\":500,\"message\":\"Unable to generate token\"}\n",
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name:    "Context Timeout",
			request: `{"login":"testuser","password":"password"}`,
			mockUserService: func() *MockUserService {
				m := &MockUserService{}
				user := &repository.User{
					UUID:         uuid.New(),
					Login:        "testuser",
					PasswordHash: "passwordhash",
					CreatedAt:    time.Now(),
				}
				m.On("Authenticate", mock.Anything, "testuser", "password").Return(user, nil)
				return m
			},
			mockTokenService: func() *MockTokenService {
				m := &MockTokenService{}
				m.On("GenerateToken", "testuser").Return("secret-token", nil)
				return m
			},
			contextTimeout: 0 * time.Second,
			wantErr:        true,
			wantResponse:   "{\"code\":500,\"message\":\"Timeout exceeded\"}\n", // Adjust the message as needed
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name:    "Invalid JSON Request",
			request: `{"login":testuser,"password":"password"}`, // Malformed JSON
			mockUserService: func() *MockUserService {
				return &MockUserService{}
			},
			mockTokenService: func() *MockTokenService {
				return &MockTokenService{}
			},
			contextTimeout: 5 * time.Second,
			wantErr:        true,
			wantResponse:   "{\"code\":400,\"message\":\"Unable to parse body\"}\n",
			wantStatusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare the request and response recorder
			body := strings.NewReader(tt.request)
			req, err := http.NewRequest("POST", "/api/user/login", body)
			assert.NoError(t, err)
			w := httptest.NewRecorder()

			// Create UserHandler with mocked services
			uh := &UserHandler{
				userService:    tt.mockUserService(),
				tokenService:   tt.mockTokenService(),
				contextTimeout: tt.contextTimeout,
			}

			// Call the method
			uh.Login(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			if tt.wantErr {
				assert.JSONEq(t, tt.wantResponse, w.Body.String())
				return
			} else {
				assert.Equal(t, tt.wantResponse, w.Body.String())
			}
		})
	}
}

func TestUserHandler_Register(t *testing.T) {
	tests := []struct {
		name             string
		request          string
		mockUserService  func() *MockUserService
		mockTokenService func() *MockTokenService
		contextTimeout   time.Duration
		wantErr          bool
		wantResponse     string
		wantStatusCode   int
	}{
		{
			name:    "Successful Registration",
			request: `{"login":"newuser","password":"newpassword"}`,
			mockUserService: func() *MockUserService {
				m := &MockUserService{}
				user := &repository.User{
					UUID:         uuid.New(),
					Login:        "newuser",
					PasswordHash: "passwordhash",
					CreatedAt:    time.Now()}
				m.On("Create", mock.Anything, "newuser", "newpassword").Return(user, nil)
				return m
			},
			mockTokenService: func() *MockTokenService {
				m := &MockTokenService{}
				m.On("GenerateToken", "newuser").Return("secret-token", nil)
				return m
			},
			contextTimeout: 5 * time.Second,
			wantErr:        false,
			wantResponse:   "Bearer secret-token",
			wantStatusCode: http.StatusOK,
		},
		{
			name:    "Invalid Input",
			request: `{"login":"","password":"newpassword"}`,
			mockUserService: func() *MockUserService {
				m := &MockUserService{}
				user := &repository.User{UUID: uuid.New(), Login: "newuser", PasswordHash: "passwordhash", CreatedAt: time.Now()}
				m.On("Create", mock.Anything, "newuser", "newpassword").Return(user, nil)
				return m
			},
			mockTokenService: func() *MockTokenService {
				m := &MockTokenService{}
				m.On("GenerateToken", "newuser").Return("secret-token", nil)
				return m
			},
			contextTimeout: 5 * time.Second,
			wantErr:        true,
			wantResponse:   "{\"code\":400,\"message\":\"Login and password are required\"}\n",
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:    "Error in User Creation",
			request: `{"login":"newuser","password":"newpassword"}`,
			mockUserService: func() *MockUserService {
				m := &MockUserService{}
				err := appErrors.New(errors.New(""), "User already exists")
				m.On("Create", mock.Anything, "newuser", "newpassword").Return((*repository.User)(nil), err)
				return m
			},
			mockTokenService: func() *MockTokenService {
				return &MockTokenService{}
			},
			contextTimeout: 5 * time.Second,
			wantErr:        true,
			wantResponse:   "{\"code\":500,\"message\":\"User already exists\"}\n",
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name:    "Error in Token Generation",
			request: `{"login":"newuser","password":"newpassword"}`,
			mockUserService: func() *MockUserService {
				m := &MockUserService{}
				user := &repository.User{UUID: uuid.New(), Login: "newuser", PasswordHash: "passwordhash", CreatedAt: time.Now()}
				m.On("Create", mock.Anything, "newuser", "newpassword").Return(user, nil)
				return m
			},
			mockTokenService: func() *MockTokenService {
				m := &MockTokenService{}
				m.On("GenerateToken", "newuser").Return("", errors.New("token generation error"))
				return m
			},
			contextTimeout: 5 * time.Second,
			wantErr:        true,
			wantResponse:   "{\"code\":500,\"message\":\"Unable to generate token\"}\n",
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name:    "Context Timeout",
			request: `{"login":"newuser","password":"newpassword"}`,
			mockUserService: func() *MockUserService {
				m := &MockUserService{}
				user := &repository.User{UUID: uuid.New(), Login: "newuser", PasswordHash: "passwordhash", CreatedAt: time.Now()}
				m.On("Create", mock.Anything, "newuser", "newpassword").Return(user, nil)
				return m
			},
			mockTokenService: func() *MockTokenService {
				m := &MockTokenService{}
				m.On("GenerateToken", "newuser").Return("secret-token", nil)
				return m
			},
			contextTimeout: 0 * time.Second,
			wantErr:        true,
			wantResponse:   "{\"code\":500,\"message\":\"Timeout exceeded\"}\n",
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name:    "Invalid JSON Request",
			request: `{"login":newuser,"password":"newpassword"}`, // Malformed JSON
			mockUserService: func() *MockUserService {
				return &MockUserService{}
			},
			mockTokenService: func() *MockTokenService {
				return &MockTokenService{}
			},
			contextTimeout: 5 * time.Second,
			wantErr:        true,
			wantResponse:   "{\"code\":400,\"message\":\"Unable to parse body\"}\n",
			wantStatusCode: http.StatusBadRequest,
		},
		// Add more test cases as needed
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare the request and response recorder
			body := strings.NewReader(tt.request)
			req, err := http.NewRequest("POST", "/api/user/register", body)
			assert.NoError(t, err)
			w := httptest.NewRecorder()

			// Create UserHandler with mocked services
			uh := &UserHandler{
				userService:    tt.mockUserService(),
				tokenService:   tt.mockTokenService(),
				contextTimeout: tt.contextTimeout,
			}

			// Call the method
			uh.Register(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			if tt.wantErr {
				assert.JSONEq(t, tt.wantResponse, w.Body.String())
			} else {
				assert.Equal(t, tt.wantResponse, w.Body.String())
			}
		})
	}
}
