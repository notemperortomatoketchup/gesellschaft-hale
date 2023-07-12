package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// type CustomContext struct {
// 	echo.Context
// 	id   *uint
// 	user *User
// }

// func makeCustomContext(c echo.Context) *CustomContext {
// 	return &CustomContext{
// 		c,
// 		nil,
// 		nil,
// 	}
// }
// func (c *CustomContext) GetID() uint {
// 	return *c.id
// }

// func (c *CustomContext) SetID(id uint) {
// 	c.id = &id
// }

// func (c *CustomContext) setUser(u *User) {
// 	c.user = u
// }

// func (c *CustomContext) getUser() *User {
// 	return c.user
// }

// func adminMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
// 	return func(c echo.Context) error {
// 		cc := makeCustomContext(c)
// 		u, err := getUserFromJWT(c)
// 		if err != nil {
// 			return err
// 		}

// 		if u.Role != ROLE_ADMIN {
// 			return echo.ErrUnauthorized
// 		}

// 		return next(cc)
// 	}
// }

// func verifyIDMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
// 	return func(c echo.Context) error {
// 		cc := c.(*CustomContext)
// 		_, err := getIDFromCtx(cc)
// 		if err != nil {
// 			return err
// 		}

// 		return next(c)
// 	}
// }
// func verifyOwnershipMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
// 	return func(c echo.Context) error {
// 		cc := makeCustomContext(c)
// 		if strings.Contains(c.Request().URL.String(), "create") {
// 			next(cc)
// 		}

// 		id, err := getIDFromCtx(cc)
// 		if err != nil {
// 			return err
// 		}

// 		if _, err := getCampaign(id); err != nil {
// 			return badRequest(err)
// 		}

// 		u, err := getUserFromJWT(cc)
// 		if err != nil {
// 			return badRequest(protocol.ErrUserNotFound)
// 		}

// 		if u.Role == ROLE_ADMIN {
// 			return next(cc)
// 		}

// 		if err := verifyCampaignOwnership(u, id); err != nil {
// 			return badRequest(protocol.ErrCampaignUnowned)
// 		}

// 		return next(cc)
// 	}
// }

// func verifyUserMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
// 	return func(c echo.Context) error {
// 		cc := makeCustomContext(c)
// 		u, err := getUserFromJWT(c)
// 		if err != nil {
// 			return err
// 		}

// 		cc.setUser(u)

// 		return next(cc)
// 	}
// }

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
	u, err := getUserFromJWT(c)
	if err != nil {
		return err
	}

	if u.Role == ROLE_ADMIN {
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
	u, err := getUserFromJWT(c)
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
	if _, err := getCampaign(id); err != nil {
		return err
	}

	// skip validation of ownership, admin can edit all
	if u.Role == ROLE_ADMIN {
		return c.Next()
	}

	if err := verifyCampaignOwnership(u, id); err != nil {
		return err
	}

	return c.Next()
}
