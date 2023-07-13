package main

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/wotlk888/gesellschaft-hale/server/models"
)

func ErrorHandler() func(c *fiber.Ctx, err error) error {
	return func(c *fiber.Ctx, err error) error {
		code := fiber.StatusInternalServerError

		// Retrieve the custom status code if it's a *fiber.Error
		var e *fiber.Error
		if errors.As(err, &e) {
			code = e.Code
		}

		// Set Content-Type: text/plain; charset=utf-8
		c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
		// Return status code with error message
		return c.Status(code).JSON(struct {
			Error string `json:"error"`
		}{
			Error: err.Error(),
		})

	}
}

func adminOnlyMiddleware(c *fiber.Ctx) error {
	u, err := models.GetUserFromJWT(c)
	if err != nil {
		return err
	}

	if u.Role == models.ROLE_ADMIN {
		return c.Next()
	}

	return fiber.ErrUnauthorized
}

func localsIDMiddleware(c *fiber.Ctx) error {
	requestUrl := c.OriginalURL()
	parts := strings.Split(requestUrl, "/")
	idParam := parts[len(parts)-1]
	storeIDInLocals(c, idParam)

	return c.Next()
}
func campaignMiddleware(c *fiber.Ctx) error {
	u, err := models.GetUserFromJWT(c)
	if err != nil {
		return err
	}

	id, has := getIDInLocals(c)
	if !has {
		fmt.Println("we are in a no id needed zone")
		return c.Next()
	}

	fmt.Println("we are in an id needed zone, and the id is ->", id)

	// campaign exists at all?
	if _, err := models.GetCampaign(id); err != nil {
		return err
	}

	// skip validation of ownership, admin can edit all
	if u.Role == models.ROLE_ADMIN {
		return c.Next()
	}

	if err := models.VerifyCampaignOwnership(u, id); err != nil {
		return err
	}

	return c.Next()
}

func storeIDInLocals(c *fiber.Ctx, x string) {
	// due to how we pull the id, it might not be an integer
	if x == "" {
		c.Locals("id", nil)
		return
	}

	id64, err := strconv.ParseUint(x, 10, 32)
	if err != nil {
		c.Locals("id", nil)
		return
	}

	fmt.Println("Reached storage")
	id := uint(id64)

	c.Locals("id", &id)
}

func getIDInLocals(c *fiber.Ctx) (uint, bool) {
	// use * for nil, to not trigger with 0, but return uint for convenience of usage
	param := c.Locals("id")

	if rv := reflect.ValueOf(param); !rv.IsValid() || rv.IsNil() {
		return 0, false
	} // its not nil, so we can say it's *uint for sure, as we store that.

	id := param.(*uint)

	return *id, true
}
