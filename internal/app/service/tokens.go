package service

import (
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"github.com/ujwegh/gophermart/internal/app/config"
	appErrors "github.com/ujwegh/gophermart/internal/app/errors"
	"regexp"
	"time"
)

type TokenService interface {
	GetUserEmail(tokenString string) (string, error)
	GenerateToken(userEmail string) (string, error)
}

type Claims struct {
	jwt.RegisteredClaims
	UserEmail string
}

type TokenServiceImpl struct {
	secretKey     string
	tokenLifetime time.Duration
}

var emailRegex = regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`)

func NewTokenService(cfg config.AppConfig) *TokenServiceImpl {
	return &TokenServiceImpl{
		secretKey:     cfg.TokenSecretKey,
		tokenLifetime: time.Duration(cfg.TokenLifetimeSec) * time.Second,
	}
}

func (ts TokenServiceImpl) GetUserEmail(tokenString string) (string, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(ts.secretKey), nil
		})
	if err != nil {
		return "", appErrors.New(err, "failed to parse token")
	}

	if !token.Valid {
		return "", appErrors.New(errors.New("token error"), "token is not valid")
	}

	if !emailRegex.MatchString(claims.UserEmail) {
		return "", appErrors.New(errors.New("token error"), "invalid email in token")
	}

	return claims.UserEmail, nil
}

func (ts TokenServiceImpl) GenerateToken(userEmail string) (string, error) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "gophermart",
			Subject:   "auth token",
			ExpiresAt: jwt.NewNumericDate(now.Add(ts.tokenLifetime)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
		UserEmail: userEmail,
	})

	tokenString, err := token.SignedString([]byte(ts.secretKey))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}
