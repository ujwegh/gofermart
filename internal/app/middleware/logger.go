package middlware

import (
	"bytes"
	"github.com/ujwegh/gophermart/internal/app/logger"
	"go.uber.org/zap"
	"io"
	"net/http"
)

type responseRecorder struct {
	http.ResponseWriter
	status        int
	contentLength int
	body          bytes.Buffer
}

func (rr *responseRecorder) WriteHeader(statusCode int) {
	rr.status = statusCode
	rr.ResponseWriter.WriteHeader(statusCode)
}

func (rr *responseRecorder) Write(b []byte) (int, error) {
	n, err := rr.ResponseWriter.Write(b)
	if err == nil {
		rr.contentLength += n
		rr.body.Write(b)
	}
	return n, err
}

func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyMsg, err := getRequestBodyForLogging(r)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		logger.Log.Info("REQUEST:",
			zap.String("Method", r.Method),
			zap.String("Path", r.URL.Path),
			zap.String("Body", bodyMsg),
		)
		next.ServeHTTP(w, r)
	})
}

func ResponseLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rr := &responseRecorder{ResponseWriter: w}
		next.ServeHTTP(rr, r)
		body := rr.body.String()
		if len(body) == 0 {
			body = "empty body"
		}
		logger.Log.Info("RESPONSE:",
			zap.Int("Status", rr.status),
			zap.Int("Content-Length", rr.contentLength),
			zap.String("Body", body),
		)
	})
}

func getRequestBodyForLogging(r *http.Request) (string, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err.Error(), err
	}
	defer r.Body.Close()
	if len(body) == 0 {
		return "empty body", nil
	}
	r.Body = io.NopCloser(bytes.NewBuffer(body))
	return string(body), nil
}
