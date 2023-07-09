package main

import (
	"github.com/golang-jwt/jwt"
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

func getUserFromJWT(c echo.Context) (*User, error) {

	tokenStr := c.Request().Header.Get("Token")
	if tokenStr == "" {
		return nil, internalError(protocol.ErrNotAuthenticated)
	}

	claims := jwt.MapClaims{}

	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return jwtsecret, nil
	})

	if _, ok := token.Claims.(jwt.MapClaims); !ok || !token.Valid {
		return nil, internalError(err)
	}

	user, err := getUser(claims["username"].(string))
	if err != nil {
		return nil, internalError(err)
	}

	return user, nil
}
