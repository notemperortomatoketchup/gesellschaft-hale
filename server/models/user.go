package models

import (
	"github.com/wotlk888/gesellschaft-hale/protocol"
	"golang.org/x/crypto/bcrypt"
)

const (
	ROLE_USER int = iota
	ROLE_ADMIN
)

type User struct {
	ID       *uint  `json:"id,omitempty" gorm:"id,primarykey"`
	Username string `json:"username" gorm:"username"`
	Hashed   string `json:"hashed_password" gorm:"hashed"`
	Role     int    `json:"role" gorm:"role"`
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
	u.Hashed = string(hashed)
	return nil
}

func (u *User) Insert() error {
	if err := db.Create(&u).Error; err != nil {
		return err
	}
	return nil
}

func (u *User) Update() error {
	if err := db.Save(&u).Error; err != nil {
		return err
	}
	return nil
}

func (u *User) Delete() error {
	if err := db.Delete(&u).Error; err != nil {
		return err
	}
	return nil

}

func (u *User) IsPassword(raw string) error {
	err := bcrypt.CompareHashAndPassword([]byte(u.Hashed), []byte(raw))
	if err != nil {
		return protocol.ErrIncorrectPassword
	}
	return nil
}

func (u *User) HasCampaign(id uint) (bool, error) {
	var campaign Campaign
	if err := db.Table("campaigns").Where("owner_id = ? AND id = ?", u.ID, id).First(&campaign).Error; err != nil {
		return false, err
	}

	return true, nil
}
