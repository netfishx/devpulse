package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethanwang/devpulse/api/internal/jwtutil"
	mw "github.com/ethanwang/devpulse/api/internal/middleware"
)

const testSecret = "test-secret-key"

func TestJWTAuth_ValidToken(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)

	token, err := jwtutil.Generate(42, testSecret)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := mw.JWTAuth(testSecret)(func(c *echo.Context) error {
		userID, err := mw.GetUserID(c)
		require.NoError(t, err)
		assert.Equal(t, int64(42), userID)
		return c.String(http.StatusOK, "ok")
	})

	err = handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestJWTAuth_MissingHeader(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := mw.JWTAuth(testSecret)(func(c *echo.Context) error {
		return c.String(http.StatusOK, "should not reach")
	})

	err := handler(c)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing token")
}

func TestJWTAuth_InvalidToken(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := mw.JWTAuth(testSecret)(func(c *echo.Context) error {
		return c.String(http.StatusOK, "should not reach")
	})

	err := handler(c)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token")
}
