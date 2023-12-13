package clients

import (
	"bytes"
	"fmt"
	"github.com/sethgrid/pester"
	"github.com/ujwegh/gophermart/internal/app/config"
	"github.com/ujwegh/gophermart/internal/app/logger"
	"go.uber.org/ratelimit"
	"go.uber.org/zap"
	"io"
	"net/http"
	"time"
)

type (
	AccrualClient interface {
		GetOrderInfo(orderID string) (*AccrualResponseDto, error)
	}
	AccrualClientImpl struct {
		ServiceURL   string
		pesterClient *pester.Client
		rateLimiter  ratelimit.Limiter
	}
	//easyjson:json
	AccrualResponseDto struct {
		OrderID       string        `json:"order"`
		AccrualStatus AccrualStatus `json:"status"`
		Accrual       float64       `json:"accrual"`
	}
	LoggingRoundTripper struct {
		Proxied http.RoundTripper
	}
	responseRecorder struct {
		http.ResponseWriter
		status        int
		contentLength int
		body          bytes.Buffer
	}

	AccrualStatus string
)

const (
	REGISTERED AccrualStatus = "REGISTERED"
	PROCESSING AccrualStatus = "PROCESSING"
	INVALID    AccrualStatus = "INVALID"
	PROCESSED  AccrualStatus = "PROCESSED"
)

func NewAccrualClient(c config.AppConfig) *AccrualClientImpl {
	ratePerSecond := c.AccrualMaxRequestsPerMinute / 1

	rateLimiter := ratelimit.New(ratePerSecond)
	pesterClient := pester.New()

	pesterClient.Concurrency = 1 // Since we are rate-limiting, concurrency should be 1
	pesterClient.MaxRetries = 0
	pesterClient.KeepLog = true
	pesterClient.Timeout = time.Duration(c.AccrualSystemRequestTimeoutSec) * time.Second
	pesterClient.RetryOnHTTP429 = false
	pesterClient.Transport = &LoggingRoundTripper{Proxied: http.DefaultTransport}

	return &AccrualClientImpl{
		ServiceURL:   c.AccrualSystemAddress,
		pesterClient: pesterClient,
		rateLimiter:  rateLimiter,
	}
}

func (ac *AccrualClientImpl) GetOrderInfo(orderID string) (*AccrualResponseDto, error) {
	// Wait for the next available opportunity to send a request
	ac.rateLimiter.Take()

	resp, err := ac.pesterClient.Get(ac.ServiceURL + "/api/orders/" + orderID)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	body, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()

	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		return nil, fmt.Errorf("error making request to get order info by orderID: %s", orderID)
	} else if resp.StatusCode == 204 {
		return nil, fmt.Errorf("order with orderID: " + orderID + " not registered yet")
	}

	dto := &AccrualResponseDto{}
	err = dto.UnmarshalJSON(body)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling response to DTO: %w", err)
	}

	return dto, nil
}

func (ac *LoggingRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	logRequest(r)
	response, err := ac.Proxied.RoundTrip(r)
	if err != nil {
		logger.Log.Error("accrual request error", zap.Error(err))
		return nil, err
	}
	logResponse(response)
	return response, nil
}

func logResponse(response *http.Response) {
	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		logger.Log.Error("accrual response error", zap.Error(err))
		return
	}
	response.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	body := string(bodyBytes)
	if len(body) == 0 {
		body = "empty body"
	}

	logger.Log.Info("ACCRUAL RESPONSE:",
		zap.Int("Status", response.StatusCode),
		zap.Int64("Content-Length", response.ContentLength),
		zap.String("Body", body),
	)
}

func logRequest(r *http.Request) {
	bodyMsg, err := getRequestBodyForLogging(r)
	if err != nil {
		logger.Log.Error("accrual log request error", zap.Error(err))
		return
	}
	logger.Log.Info("ACCRUAL REQUEST:",
		zap.String("Method", r.Method),
		zap.String("Path", r.URL.String()),
		zap.String("Body", bodyMsg),
	)
}

func getRequestBodyForLogging(r *http.Request) (string, error) {
	if r.Body == nil || r.ContentLength == 0 {
		return "empty body", nil
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return "", fmt.Errorf("error reading request body: %w", err)
	}
	defer r.Body.Close()
	r.Body = io.NopCloser(bytes.NewBuffer(body))
	return string(body), nil
}
