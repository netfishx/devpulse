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
	g.GET("/summaries/weekly", h.ListWeekly)
	g.GET("/summaries/monthly", h.ListMonthly)
	g.GET("/summaries/heatmap", h.Heatmap)
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

func (h *Handler) ListWeekly(c *echo.Context) error {
	userID, err := mw.GetUserID(c)
	if err != nil {
		return err
	}

	weeks, _ := strconv.Atoi(c.QueryParam("weeks"))

	resp, err := h.svc.ListWeekly(c.Request().Context(), userID, weeks)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) ListMonthly(c *echo.Context) error {
	userID, err := mw.GetUserID(c)
	if err != nil {
		return err
	}

	months, _ := strconv.Atoi(c.QueryParam("months"))

	resp, err := h.svc.ListMonthly(c.Request().Context(), userID, months)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) Heatmap(c *echo.Context) error {
	userID, err := mw.GetUserID(c)
	if err != nil {
		return err
	}

	days, _ := strconv.Atoi(c.QueryParam("days"))

	resp, err := h.svc.Heatmap(c.Request().Context(), userID, days)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, resp)
}
