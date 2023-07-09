package main

import (
	"github.com/wotlk888/gesellschaft-hale/protocol"
	"golang.org/x/crypto/bcrypt"
)

const (
	ROLE_USER Role = iota
	ROLE_ADMIN
)

type Role int

type User struct {
	ID             int    `json:"id,omitempty"`
	Username       string `json:"username"`
	HashedPassword string `json:"hashed_password"`
	Role           int    `json:"role"`
}

func (r Role) String() string {
	switch r {
	case ROLE_USER:
		return "User"
	case ROLE_ADMIN:
		return "Admin"
	default:
		return "Unknown"
	}
}

func (u *User) SetUsername(username string) {
	u.Username = username
}

func (u *User) SetPassword(password string) error {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// store as text in postgres.
	u.HashedPassword = string(hashed)
	return nil
}

func (u *User) Insert() error {
	var results []User
	if err := db.DB.From("users").Insert(&u).Execute(&results); err != nil {
		return err
	}
	return nil
}

func (u *User) Update() error {
	var results []User
	if err := db.DB.From("users").Update(&u).Eq("username", u.Username).Execute(&results); err != nil {
		return err
	}
	return nil
}

func (u *User) IsPassword(raw string) error {
	err := bcrypt.CompareHashAndPassword([]byte(u.HashedPassword), []byte(raw))
	if err != nil {
		return protocol.ErrIncorrectPassword
	}
	return nil
}
