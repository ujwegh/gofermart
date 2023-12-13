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

/** GetBalance
 * @api {get} Returns the current balance of a user's loyalty points account
 */
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

/**
 * @api {post} Requests to write off points from the savings account towards payment of a new order
 */
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

/**
 * @api {get} Returns information about the withdrawal of funds from the savings account by the user
 */
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
