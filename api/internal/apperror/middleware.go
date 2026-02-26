package apperror

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v5"
)

// ErrorHandler returns an Echo error handler that maps AppError to JSON responses.
func ErrorHandler(exposeInternal bool) func(c *echo.Context, err error) {
	return func(c *echo.Context, err error) {
		var appErr *AppError
		if errors.As(err, &appErr) {
			if appErr.Err != nil {
				slog.Error("request error", "title", appErr.Title, "detail", appErr.Detail, "cause", appErr.Err)
			}
			c.JSON(appErr.Code, map[string]string{
				"error": appErr.Detail,
			})
			return
		}

		// Echo's own HTTPError (e.g. 404 from router)
		var he *echo.HTTPError
		if errors.As(err, &he) {
			c.JSON(he.Code, map[string]string{
				"error": he.Message,
			})
			return
		}

		// Unknown error → 500
		slog.Error("unexpected error", "error", err)
		c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "an unexpected error occurred",
		})
	}
}
