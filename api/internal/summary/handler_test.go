package summary

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
)

func TestListSummaries_MissingAuth(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/summaries", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := NewHandler(nil)
	err := h.List(c)
	assert.Error(t, err)
}

func TestListWeekly_MissingAuth(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/summaries/weekly", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := NewHandler(nil)
	err := h.ListWeekly(c)
	assert.Error(t, err)
}

func TestListMonthly_MissingAuth(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/summaries/monthly", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := NewHandler(nil)
	err := h.ListMonthly(c)
	assert.Error(t, err)
}

func TestHeatmap_MissingAuth(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/summaries/heatmap", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := NewHandler(nil)
	err := h.Heatmap(c)
	assert.Error(t, err)
}

func TestCommitCountToLevel(t *testing.T) {
	tests := []struct {
		count int
		want  int
	}{
		{0, 0},
		{1, 1},
		{3, 1},
		{4, 2},
		{9, 2},
		{10, 3},
		{19, 3},
		{20, 4},
		{100, 4},
	}

	for _, tt := range tests {
		got := commitCountToLevel(tt.count)
		assert.Equal(t, tt.want, got, "commitCountToLevel(%d)", tt.count)
	}
}
