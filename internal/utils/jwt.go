package utils

import (
	"errors"
	"recharge-go/internal/config"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID   int64    `json:"user_id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
}

func GenerateJWT(userID int64, username string, roleNames []string) (string, string, error) {
	cfg := config.GetConfig()

	// Generate access token
	accessExpirationTime := time.Now().Add(time.Hour * time.Duration(cfg.JWT.Expire))
	accessClaims := &Claims{
		UserID:   userID,
		Username: username,
		Roles:    roleNames,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExpirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString([]byte(cfg.JWT.Secret))
	if err != nil {
		return "", "", err
	}

	// Generate refresh token
	refreshExpirationTime := time.Now().Add(time.Hour * time.Duration(cfg.JWT.RefreshExpire))
	refreshClaims := &Claims{
		UserID:   userID,
		Username: username,
		Roles:    roleNames,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshExpirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(cfg.JWT.RefreshSecret))
	if err != nil {
		return "", "", err
	}

	return accessTokenString, refreshTokenString, nil
}

func ValidateJWT(tokenString string, isRefresh bool) (*Claims, error) {
	cfg := config.GetConfig()
	claims := &Claims{}

	var secret string
	if isRefresh {
		secret = cfg.JWT.RefreshSecret
	} else {
		secret = cfg.JWT.Secret
	}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
