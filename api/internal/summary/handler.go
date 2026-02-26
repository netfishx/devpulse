package summary

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v5"

	mw "github.com/ethanwang/devpulse/api/internal/middleware"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(g *echo.Group) {
	g.GET("/summaries", h.List)
}

func (h *Handler) List(c *echo.Context) error {
	userID, err := mw.GetUserID(c)
	if err != nil {
		return err
	}

	days, _ := strconv.Atoi(c.QueryParam("days"))

	resp, err := h.svc.List(c.Request().Context(), userID, days)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, resp)
}
