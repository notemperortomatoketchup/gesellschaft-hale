package models

import (
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func (u *User) GenerateJWT() (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":       u.ID,
		"username": u.Username,
		"role":     u.Role,
	})

	tokenStr, err := token.SignedString([]byte("NEwBigSecretNo303003"))
	if err != nil {
		return "", err
	}

	return tokenStr, nil
}

func GetUserFromJWT(c *fiber.Ctx) (*User, error) {
	user := c.Locals("user").(*jwt.Token)
	claims := user.Claims.(jwt.MapClaims)
	id := uint(claims["id"].(float64))

	u, err := GetUserByID(id)
	if err != nil {
		return nil, err
	}

	return u, nil
}
