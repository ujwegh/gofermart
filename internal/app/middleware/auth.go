package middlware

import (
	"context"
	appContext "github.com/ujwegh/gophermart/internal/app/context"
	"github.com/ujwegh/gophermart/internal/app/handlers"
	"github.com/ujwegh/gophermart/internal/app/logger"
	"github.com/ujwegh/gophermart/internal/app/service"
	"go.uber.org/zap"
	"net/http"
	"strings"
	"time"
)

type AuthMiddleware struct {
	tokenService   service.TokenService
	userService    service.UserService
	contextTimeout time.Duration
}

func NewAuthMiddleware(tokenService service.TokenService, userService service.UserService, contextTimeoutSec int) AuthMiddleware {
	return AuthMiddleware{
		tokenService:   tokenService,
		userService:    userService,
		contextTimeout: time.Duration(contextTimeoutSec) * time.Second,
	}
}

func (am *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(context.Background(), am.contextTimeout)
		defer cancel()

		authHeader := r.Header.Get("Authorization")
		token := strings.Split(authHeader, "Bearer ")[1]

		userEmail, err := am.tokenService.GetUserEmail(token)
		if err != nil {
			logger.Log.Error("failed to get user email", zap.Error(err))
			handlers.WriteJSONErrorResponse(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
			return
		}

		user, err := am.userService.GetByUserEmail(ctx, userEmail)
		if err != nil {
			logger.Log.Error("failed to get user", zap.Error(err))
			handlers.WriteJSONErrorResponse(w, "Unauthorized: User not found", http.StatusUnauthorized)
			return
		}

		err = appContext.GetContextError(ctx)
		if err != nil {
			handlers.PrepareError(w, err)
			return
		}

		r = r.WithContext(appContext.WithUserUID(r.Context(), &user.UUID))
		next.ServeHTTP(w, r)
	})
}
