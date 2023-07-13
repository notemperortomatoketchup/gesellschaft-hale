package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func validateBindError(err error) error {
	error := strings.Split(err.Error(), " ")

	got := error[3]
	expected := error[11]
	field := error[8]

	// it outputs structName.field -> I remove structName, only return the json field name.
	structNameIndex := strings.IndexFunc(field, func(r rune) bool {
		if r == '.' {
			return true
		}
		return false
	})
	field = field[structNameIndex+1:]

	return badRequest(fmt.Errorf("invalid type at %s (got: %s | want: %s)", field, got, expected))
}

func bind(c *fiber.Ctx, i any) []*ErrorResponse {
	var errors []*ErrorResponse

	if err := c.BodyParser(i); err != nil {
		if strings.Contains(err.Error(), "unmarshal") {
			bindErr := validateBindError(err)
			errors = append(errors, &ErrorResponse{
				FailedField: "binding",
				Hint:        bindErr.Error(),
			})
		}
	}

	structErrs := ValidateStruct(i)
	if structErrs != nil {
		errors = append(errors, structErrs...)
	}

	if errors != nil {
		return errors
	}

	return nil
}

func badRequest(msg error) error {
	return fiber.NewError(http.StatusBadRequest, msg.Error())
}

func internalError(msg error) error {
	return fiber.NewError(http.StatusInternalServerError, msg.Error())
}

func validationError(c *fiber.Ctx, v any) error {
	return c.Status(fiber.StatusBadRequest).JSON(v)
}
