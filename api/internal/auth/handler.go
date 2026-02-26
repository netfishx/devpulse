package auth

import (
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/ethanwang/devpulse/api/internal/validate"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterPublicRoutes mounts unauthenticated auth routes.
func (h *Handler) RegisterPublicRoutes(api *echo.Group) {
	api.POST("/register", h.Register)
	api.POST("/login", h.Login)
}

// RegisterProtectedRoutes mounts authenticated auth routes.
// The caller is responsible for applying JWT middleware to the group.
func (h *Handler) RegisterProtectedRoutes(api *echo.Group) {
	api.GET("/me", h.Me)
}

func (h *Handler) Register(c *echo.Context) error {
	var req RegisterRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}
	if err := validate.Struct(req); err != nil {
		return err
	}

	user, err := h.svc.Register(c.Request().Context(), req)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, user)
}

func (h *Handler) Login(c *echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}
	if err := validate.Struct(req); err != nil {
		return err
	}

	resp, err := h.svc.Login(c.Request().Context(), req)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) Me(c *echo.Context) error {
	userID, ok := c.Get("userID").(int64)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
	}

	user, err := h.svc.GetMe(c.Request().Context(), userID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, user)
}
