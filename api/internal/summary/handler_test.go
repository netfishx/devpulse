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
