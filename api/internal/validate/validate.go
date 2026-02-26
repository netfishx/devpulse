package validate

import (
	"fmt"

	"github.com/go-playground/validator/v10"

	"github.com/ethanwang/devpulse/api/internal/apperror"
)

var v = validator.New()

// Struct validates a struct using go-playground/validator tags.
// Returns an *apperror.AppError on failure (handler can just `return err`).
func Struct(req any) error {
	if err := v.Struct(req); err != nil {
		ve, ok := err.(validator.ValidationErrors)
		if !ok {
			return apperror.BadRequest("invalid request")
		}
		return apperror.BadRequest(formatFieldError(ve[0]))
	}
	return nil
}

func formatFieldError(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", fe.Field())
	case "email":
		return "invalid email format"
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", fe.Field(), fe.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters", fe.Field(), fe.Param())
	default:
		return fmt.Sprintf("%s is invalid", fe.Field())
	}
}
