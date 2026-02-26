package oauth

import (
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/ethanwang/devpulse/api/internal/apperror"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts OAuth routes. All routes require authentication.
func (h *Handler) RegisterRoutes(api *echo.Group) {
	api.GET("/oauth/github", h.GitHubRedirect)
	api.GET("/oauth/github/callback", h.GitHubCallback)
}

// GitHubRedirect returns the GitHub OAuth authorization URL.
func (h *Handler) GitHubRedirect(c *echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"url": h.svc.GitHubAuthURL(),
	})
}

// GitHubCallback exchanges the authorization code for a token.
func (h *Handler) GitHubCallback(c *echo.Context) error {
	code := c.QueryParam("code")
	if code == "" {
		return apperror.BadRequest("missing code parameter")
	}

	userID, ok := c.Get("userID").(int64)
	if !ok {
		return apperror.Unauthorized("not authenticated")
	}

	if err := h.svc.ExchangeGitHubCode(c.Request().Context(), userID, code); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "connected"})
}
