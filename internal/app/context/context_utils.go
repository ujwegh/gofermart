package context

import (
	"context"
	"github.com/google/uuid"
	appErrors "github.com/ujwegh/gophermart/internal/app/errors"
	"net/http"
)

type key string

const userUIDKey key = "userUID"
const errorKey key = "error"

func WithUserUID(ctx context.Context, userUID *uuid.UUID) context.Context {
	return context.WithValue(ctx, userUIDKey, userUID)
}

func UserUID(ctx context.Context) *uuid.UUID {
	val := ctx.Value(userUIDKey)
	userUID, ok := val.(*uuid.UUID)
	if !ok {
		return nil
	}
	return userUID
}

func GetContextError(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		var errMsg string
		var errCode int

		switch err {
		case context.Canceled:
			errMsg, errCode = "Request canceled", http.StatusInternalServerError
		case context.DeadlineExceeded:
			errMsg, errCode = "Timeout exceeded", http.StatusInternalServerError
		default:
			errMsg, errCode = "Context error", http.StatusInternalServerError
		}
		return appErrors.NewWithCode(err, errMsg, errCode)
	}
	return nil
}
