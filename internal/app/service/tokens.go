package service

import (
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"github.com/ujwegh/gophermart/internal/app/config"
	"time"
)

type TokenService interface {
	GetUserLogin(tokenString string) (string, error)
	GenerateToken(userEmail string) (string, error)
}

type Claims struct {
	jwt.RegisteredClaims
	UserLogin string
}

type TokenServiceImpl struct {
	secretKey     string
	tokenLifetime time.Duration
}

func NewTokenService(cfg config.AppConfig) *TokenServiceImpl {
	return &TokenServiceImpl{
		secretKey:     cfg.TokenSecretKey,
		tokenLifetime: time.Duration(cfg.TokenLifetimeSec) * time.Second,
	}
}

func (ts TokenServiceImpl) GetUserLogin(tokenString string) (string, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(ts.secretKey), nil
		})
	if err != nil {
		return "", fmt.Errorf("token error: failed to parse token: %w", err)
	}

	if !token.Valid {
		return "", fmt.Errorf("token error: %w", errors.New("token is not valid"))
	}

	if claims.UserLogin == "" {
		return "", fmt.Errorf("token error: %w", errors.New("empty login in token"))
	}

	return claims.UserLogin, nil
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
		UserLogin: userEmail,
	})

	tokenString, err := token.SignedString([]byte(ts.secretKey))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}
