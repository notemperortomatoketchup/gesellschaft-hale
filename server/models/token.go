package models

import (
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func GenerateJWT(secret []byte, u *User) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":       u.ID,
		"username": u.Username,
		"role":     u.Role,
	})

	tokenStr, err := token.SignedString(secret)
	if err != nil {
		return "", err
	}

	return tokenStr, nil
}

func GetUserFromSession(c *fiber.Ctx) (*User, error) {
	user := c.Locals("user").(User)
	id := *user.ID

	u, err := GetUserByID(id)
	if err != nil {
		return nil, err
	}

	return u, nil
}
