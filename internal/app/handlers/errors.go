package handlers

import (
	"errors"
	appErrors "github.com/ujwegh/gophermart/internal/app/errors"
	"github.com/ujwegh/gophermart/internal/app/logger"
	"go.uber.org/zap"
	"net/http"
)

//easyjson:json
type ErrorResponse struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func PrepareError(w http.ResponseWriter, err error) {
	var codeErr appErrors.ResponseCodeError
	logger.Log.Error("internal error: ", zap.Error(err))
	if errors.As(err, &codeErr) {
		WriteJSONErrorResponse(w, codeErr.Msg(), codeErr.Code())
		return
	}
	// Default error handling
	WriteJSONErrorResponse(w, "Internal Server Error", http.StatusInternalServerError)
}

func WriteJSONErrorResponse(w http.ResponseWriter, message string, code int) {
	er := ErrorResponse{
		Message: message,
		Code:    code,
	}
	w.Header().Set("Content-Type", "application/json")
	json, err := ErrorResponse.MarshalJSON(er)
	if err != nil {
		logger.Log.Error("failed to marshal error response", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(code)
	w.Write(json)
}
