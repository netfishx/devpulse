package middleware

import (
	"strings"

	"github.com/labstack/echo/v5"

	"github.com/ethanwang/devpulse/api/internal/apperror"
	"github.com/ethanwang/devpulse/api/internal/jwtutil"
)

// ContextKeyUserID is the key used to store the authenticated user ID in echo.Context.
const ContextKeyUserID = "userID"

// JWTAuth returns middleware that validates Bearer tokens and sets userID in context.
func JWTAuth(secret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			header := c.Request().Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				return apperror.Unauthorized("missing token")
			}

			tokenStr := strings.TrimPrefix(header, "Bearer ")
			userID, err := jwtutil.Parse(tokenStr, secret)
			if err != nil {
				return apperror.Unauthorized("invalid token")
			}

			c.Set(ContextKeyUserID, userID)
			return next(c)
		}
	}
}

// GetUserID extracts the authenticated user ID from context.
func GetUserID(c *echo.Context) (int64, error) {
	id, ok := c.Get(ContextKeyUserID).(int64)
	if !ok {
		return 0, apperror.Unauthorized("not authenticated")
	}
	return id, nil
}
