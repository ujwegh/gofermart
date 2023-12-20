package handlers

import (
	"context"
	"fmt"
	"github.com/ShiraazMoollatjie/goluhn"
	appContext "github.com/ujwegh/gophermart/internal/app/context"
	appErrors "github.com/ujwegh/gophermart/internal/app/errors"
	"github.com/ujwegh/gophermart/internal/app/models"
	"github.com/ujwegh/gophermart/internal/app/service"
	"io"
	"net/http"
	"time"
)

type (
	BalanceHandler struct {
		walletService     service.WalletService
		withdrawalService service.WithdrawalService
		contextTimeout    time.Duration
	}

	//easyjson:json
	BalanceDto struct {
		CurrentBalance   float64 `json:"current"`
		WithdrawnBalance float64 `json:"withdrawn"`
	}
	//easyjson:json
	WithdrawRequestDTO struct {
		Order string  `json:"order"`
		Sum   float64 `json:"sum"`
	}
	//easyjson:json
	WithdrawalDTO struct {
		OrderID     string    `json:"order"`
		Sum         float64   `json:"sum"`
		ProcessedAt time.Time `json:"processed_at"`
	}
	//easyjson:json
	WithdrawalDtoSlice []WithdrawalDTO
)

func NewBalanceHandler(contextTimeoutSec int, walletService service.WalletService, withdrawalService service.WithdrawalService) *BalanceHandler {
	return &BalanceHandler{
		walletService:     walletService,
		withdrawalService: withdrawalService,
		contextTimeout:    time.Duration(contextTimeoutSec) * time.Second,
	}
}

// GetBalance godoc
// @Summary Getting the user's current balance
// @Description The handler returns the current amount of loyalty points and the total amount of points
// withdrawn during the entire registration period for an authorized user.
// @Tags balance
// @Produce json
// @Success 200 {object} BalanceDto "Current and withdrawn loyalty points"
// @Failure 401 {object} ErrorResponse "Unauthorized - The user is not authorized"
// @Failure 500 {object} ErrorResponse "Internal Server Error"
// @Security ApiKeyAuth
// @Router /api/user/balance [get]
func (bh *BalanceHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), bh.contextTimeout)
	defer cancel()
	userUID := appContext.UserUID(r.Context())

	balance, err := bh.walletService.GetBalance(ctx, userUID)
	if err != nil {
		PrepareError(w, err)
		return
	}
	balanceDto := BalanceDto{
		CurrentBalance:   balance.CurrentBalance,
		WithdrawnBalance: balance.WithdrawnBalance,
	}
	json, err := balanceDto.MarshalJSON()
	if err != nil {
		PrepareError(w, fmt.Errorf("unable to marshal json: %w", err))
		return
	}

	err = appContext.GetContextError(ctx)
	if err != nil {
		PrepareError(w, err)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(json)
}

// Withdraw godoc
// @Summary Request for debiting funds
// @Description The handler allows an authorized user to debit points from their account for a hypothetical new order.
// @Tags balance
// @Accept json
// @Produce json
// @Param withdrawal body WithdrawRequestDTO true "Withdrawal Request"
// @Success 200 "Successful processing of the request"
// @Failure 400 {object} ErrorResponse "Bad Request - Unable to read body or parse body"
// @Failure 401 {object} ErrorResponse "Unauthorized - The user is not authorized"
// @Failure 402 {object} ErrorResponse "Payment Required - Insufficient funds in the account"
// @Failure 422 {object} ErrorResponse "Unprocessable Entity - Incorrect order number format"
// @Failure 500 {object} ErrorResponse "Internal Server Error"
// @Security ApiKeyAuth
// @Router /api/user/balance/withdraw [post]
func (bh *BalanceHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), bh.contextTimeout)
	defer cancel()
	userUID := appContext.UserUID(r.Context())

	body, err := io.ReadAll(r.Body)
	if err != nil {
		err = appErrors.NewWithCode(err, errMsgEnableReadBody, http.StatusBadRequest)
		PrepareError(w, err)
		return
	}

	request := WithdrawRequestDTO{}
	err = request.UnmarshalJSON(body)
	if err != nil {
		err = appErrors.NewWithCode(err, "Unable to parse body", http.StatusBadRequest)
		PrepareError(w, err)
		return
	}

	err = goluhn.Validate(request.Order)
	if err != nil {
		err = appErrors.NewWithCode(err, "Invalid order ID", http.StatusUnprocessableEntity)
		PrepareError(w, err)
		return
	}
	err = bh.withdrawalService.CreateWithdrawal(ctx, userUID, request.Order, request.Sum)
	if err != nil {
		PrepareError(w, err)
		return
	}

	err = appContext.GetContextError(ctx)
	if err != nil {
		PrepareError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// GetWithdrawals godoc
// @Summary Receiving information about the withdrawal of funds
// @Description The handler returns information about the withdrawal of funds,
// sorted by the time of withdrawal from oldest to newest for an authorized user.
// @Tags withdrawals
// @Produce json
// @Success 200 {array} WithdrawalDTO "List of withdrawals with details"
// @Success 204 "No withdrawals to display"
// @Failure 401 {object} ErrorResponse "Unauthorized - The user is not authorized"
// @Failure 500 {object} ErrorResponse "Internal Server Error"
// @Security ApiKeyAuth
// @Router /api/user/withdrawals [get]
func (bh *BalanceHandler) GetWithdrawals(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), bh.contextTimeout)
	defer cancel()
	userUID := appContext.UserUID(r.Context())

	withdrawals, err := bh.withdrawalService.GetWithdrawals(ctx, userUID)
	if err != nil {
		PrepareError(w, err)
		return
	}
	if len(*withdrawals) == 0 {
		w.WriteHeader(http.StatusNoContent)
		fmt.Fprintf(w, "%s", "[]")
		return
	}
	response := bh.mapWithdrawalsToWithdrawalDtoSlice(withdrawals)
	rawBytes, err := response.MarshalJSON()
	if err != nil {
		PrepareError(w, fmt.Errorf("unable to marshal response: %w", err))
		return
	}

	err = appContext.GetContextError(ctx)
	if err != nil {
		PrepareError(w, err)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "%s", rawBytes)

}

func (bh *BalanceHandler) mapWithdrawalsToWithdrawalDtoSlice(slice *[]models.Withdrawal) WithdrawalDtoSlice {
	var responseSlice []WithdrawalDTO
	for _, item := range *slice {
		responseItem := WithdrawalDTO{
			OrderID:     item.OrderID,
			Sum:         item.Amount,
			ProcessedAt: item.CreatedAt,
		}
		responseSlice = append(responseSlice, responseItem)
	}
	return responseSlice
}
