package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

func validateBindError(err error) error {
	error := strings.Split(err.Error(), " ")
	var expected, got, field string

	for _, w := range error {
		if strings.Contains(w, "expected=") {
			expected = strings.TrimSuffix(strings.Split(w, "=")[1], ",")
		}

		if strings.Contains(w, "got=") {
			got = strings.TrimSuffix(strings.Split(w, "=")[1], ",")

		}

		if strings.Contains(w, "field=") {
			field = strings.TrimSuffix(strings.Split(w, "=")[1], ",")
		}
	}
	return badRequest(fmt.Errorf("invalid type at %s. (got: %s | want: %s)", field, got, expected))
}

func bind(c echo.Context, i any) error {
	if err := c.Bind(&i); err != nil {
		if strings.Contains(err.Error(), "unmarshal") {
			return validateBindError(err)
		}
		return badRequest(err)
	}
	return nil
}

func badRequest(msg error) error {
	return echo.NewHTTPError(http.StatusBadRequest, msg.Error())
}

func internalError(msg error) error {
	return echo.NewHTTPError(http.StatusInternalServerError, msg.Error())
}
