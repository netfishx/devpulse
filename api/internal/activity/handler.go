package activity

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
	g.GET("/activities", h.List)
	g.GET("/activities/top-repos", h.TopRepos)
}

func (h *Handler) TopRepos(c *echo.Context) error {
	userID, err := mw.GetUserID(c)
	if err != nil {
		return err
	}

	days, _ := strconv.Atoi(c.QueryParam("days"))
	source := c.QueryParam("source")

	resp, err := h.svc.TopRepos(c.Request().Context(), userID, days, source)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) List(c *echo.Context) error {
	userID, err := mw.GetUserID(c)
	if err != nil {
		return err
	}

	page, _ := strconv.Atoi(c.QueryParam("page"))
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))
	source := c.QueryParam("source")

	resp, err := h.svc.List(c.Request().Context(), userID, page, perPage, source)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, resp)
}
