package auth_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"

	"github.com/ethanwang/devpulse/api/internal/apperror"
	"github.com/ethanwang/devpulse/api/internal/auth"
)

// setupEcho creates an Echo instance with the apperror middleware,
// matching how the real server is configured.
func setupEcho() *echo.Echo {
	e := echo.New()
	e.HTTPErrorHandler = apperror.ErrorHandler(false)
	return e
}

func TestRegister_InvalidJSON(t *testing.T) {
	e := setupEcho()
	req := httptest.NewRequest(http.MethodPost, "/api/register", strings.NewReader(`{bad`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := auth.NewHandler(nil)
	_ = h.Register(c)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid request body")
}

func TestRegister_MissingEmail(t *testing.T) {
	e := setupEcho()
	body := `{"name":"Test","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := auth.NewHandler(nil)
	err := h.Register(c)

	// validate.Struct returns AppError; error handler would format it
	// In direct handler test, we check the error was returned
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Email")
}

func TestRegister_ShortPassword(t *testing.T) {
	e := setupEcho()
	body := `{"email":"a@b.com","name":"Test","password":"123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := auth.NewHandler(nil)
	err := h.Register(c)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least 8")
}

func TestRegister_InvalidEmail(t *testing.T) {
	e := setupEcho()
	body := `{"email":"not-an-email","name":"Test","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := auth.NewHandler(nil)
	err := h.Register(c)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "email")
}

func TestLogin_InvalidJSON(t *testing.T) {
	e := setupEcho()
	req := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(`{bad`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := auth.NewHandler(nil)
	_ = h.Login(c)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestLogin_MissingPassword(t *testing.T) {
	e := setupEcho()
	body := `{"email":"a@b.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := auth.NewHandler(nil)
	err := h.Login(c)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Password")
}
