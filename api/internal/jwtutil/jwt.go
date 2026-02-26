package jwtutil

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const TokenExpiry = 7 * 24 * time.Hour

// Generate creates a signed JWT with the given user ID.
func Generate(userID int64, secret string) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(TokenExpiry).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// Parse validates a token string and returns the user ID.
func Parse(tokenStr, secret string) (int64, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return 0, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, jwt.ErrTokenInvalidClaims
	}
	// sub is stored as float64 in MapClaims (JSON number)
	subFloat, ok := claims["sub"].(float64)
	if !ok {
		return 0, jwt.ErrTokenInvalidClaims
	}
	return int64(subFloat), nil
}
