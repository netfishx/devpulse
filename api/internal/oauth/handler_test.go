package oauth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"

	"github.com/ethanwang/devpulse/api/internal/oauth"
)

func TestGitHubRedirect_ReturnsURL(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/oauth/github", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	svc := oauth.NewService(nil, oauth.GitHubConfig{
		ClientID:    "test-client-id",
		CallbackURL: "http://localhost/callback",
	})
	h := oauth.NewHandler(svc)
	err := h.GitHubRedirect(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "github.com/login/oauth/authorize")
	assert.Contains(t, rec.Body.String(), "test-client-id")
}

func TestGitHubCallback_MissingCode(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/oauth/github/callback", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := oauth.NewHandler(nil)
	err := h.GitHubCallback(c)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing code")
}
