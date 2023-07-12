package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/wotlk888/gesellschaft-hale/protocol"
)

func (u *User) generateJWT() (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":       u.ID,
		"username": u.Username,
		"role":     u.Role,
	})

	tokenStr, err := token.SignedString(jwtsecret)
	if err != nil {
		return "", err
	}

	return tokenStr, nil
}

func verifyJWT(c echo.Context) error {
	token, ok := c.Get("user").(*jwt.Token)
	if !ok {
		return internalError(protocol.ErrNotAuthenticated)
	}

	_, ok = token.Claims.(jwt.MapClaims)
	if !ok {
		return internalError(protocol.ErrNotAuthenticated)
	}
	return nil
}

func getUserFromJWT(c *fiber.Ctx) (*User, error) {
	user := c.Locals("user").(*jwt.Token)
	claims := user.Claims.(jwt.MapClaims)
	id := uint(claims["id"].(float64))

	u, err := getUserByID(id)
	if err != nil {
		return nil, internalError(err)
	}

	return u, nil
}
