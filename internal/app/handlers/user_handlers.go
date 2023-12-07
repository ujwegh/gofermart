package handlers

import (
	"context"
	appContext "github.com/ujwegh/gophermart/internal/app/context"
	appErrors "github.com/ujwegh/gophermart/internal/app/errors"
	"github.com/ujwegh/gophermart/internal/app/models"
	"github.com/ujwegh/gophermart/internal/app/service"
	"io"
	"net/http"
	"time"
)

const errMsgEnableReadBody = "Unable to read body"

type (
	UserHandler struct {
		userService    service.UserService
		tokenService   service.TokenService
		contextTimeout time.Duration
	}
	//easyjson:json
	UserLoginDto struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}
	//easyjson:json
	UserRegisterDto struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}
	//easyjson:json
	AccessTokenDto struct {
		AccessToken string `json:"access_token"`
	}
)

func NewUserHandler(userService service.UserService, tokenService service.TokenService, contextTimeoutSec int) *UserHandler {
	return &UserHandler{
		userService:    userService,
		tokenService:   tokenService,
		contextTimeout: time.Duration(contextTimeoutSec) * time.Second,
	}
}

func (uh *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), uh.contextTimeout)
	defer cancel()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		err = appErrors.NewWithCode(err, errMsgEnableReadBody, http.StatusBadRequest)
		PrepareError(w, err)
		return
	}
	registerDto := UserRegisterDto{}
	err = registerDto.UnmarshalJSON(body)
	if err != nil {
		err = appErrors.NewWithCode(err, "Unable to parse body", http.StatusBadRequest)
		PrepareError(w, err)
		return
	}

	user, err := uh.userService.Create(ctx, registerDto.Login, registerDto.Password)
	if err != nil {
		PrepareError(w, err)
		return
	}

	accessTokenDtoBytes, err := uh.generateToken(user, err)
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
	w.Write(accessTokenDtoBytes)
}

func (uh *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), uh.contextTimeout)
	defer cancel()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		err = appErrors.NewWithCode(err, errMsgEnableReadBody, http.StatusBadRequest)
		PrepareError(w, err)
		return
	}

	loginDto := UserLoginDto{}
	err = loginDto.UnmarshalJSON(body)
	if err != nil {
		err = appErrors.NewWithCode(err, "Unable to parse body", http.StatusBadRequest)
		PrepareError(w, err)
		return
	}

	user, err := uh.userService.Authenticate(ctx, loginDto.Login, loginDto.Password)
	if err != nil {
		PrepareError(w, err)
		return
	}

	accessTokenDtoBytes, err := uh.generateToken(user, err)
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
	w.Write(accessTokenDtoBytes)
}

func (uh *UserHandler) generateToken(user *models.User, err error) ([]byte, error) {
	token, err := uh.tokenService.GenerateToken(user.Email)
	if err != nil {
		return nil, appErrors.NewWithCode(err, "Unable to generate token", http.StatusInternalServerError)
	}
	accessTokenDto := AccessTokenDto{AccessToken: token}
	accessTokenDtoBytes, err := accessTokenDto.MarshalJSON()
	if err != nil {
		return nil, appErrors.NewWithCode(err, "Unable to marshal token", http.StatusInternalServerError)
	}
	return accessTokenDtoBytes, nil
}
