package main

import (
	"fmt"

	"github.com/wotlk888/gesellschaft-hale/protocol"
	"golang.org/x/crypto/bcrypt"
)

const (
	ROLE_USER int = iota
	ROLE_ADMIN
)

type User struct {
	ID             *uint  `json:"id,omitempty"`
	Username       string `json:"username"`
	HashedPassword string `json:"hashed_password"`
	Role           int    `json:"role"`
}

func (u *User) SetUsername(username string) *User {
	u.Username = username
	return u
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
	if err := db.DB.From("users").Update(&u).Eq("id", fmt.Sprint(*u.ID)).Execute(&results); err != nil {
		return err
	}
	return nil
}

func (u *User) Delete() error {
	var results []any
	if err := db.DB.From("users").Delete().Eq("id", fmt.Sprint(*u.ID)).Execute(&results); err != nil {
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

func (u *User) HasCampaign(id uint) (bool, error) {
	var campaigns []Campaign

	if err := db.DB.From("campaigns").Select().Eq("owner_id", fmt.Sprint(u.ID)).Eq("campaign_id", fmt.Sprint(id)).Execute(&campaigns); err != nil {
		return false, err
	}
	if len(campaigns) == 0 {
		return false, protocol.ErrCampaignUnowned
	}

	return true, nil
}
