package models

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wotlk888/gesellschaft-hale/protocol"
	"gorm.io/gorm"
)

type Session struct {
	ID          uint      `json:"session_id" gorm:"column:session_id"`
	UserID      uint      `json:"user_id" gorm:"column:user_id"`
	CreatedAt   time.Time `json:"created_at" gorm:"column:created_at"`
	ExpiresAt   time.Time `json:"expires_at" gorm:"column:expires_at"`
	LastUpdated time.Time `json:"last_updated" gorm:"column:last_updated"`
}

func NewSession(userID uint, exp time.Duration) (*Session, error) {
	// erase existing sessions

	s := &Session{
		ID:        uint(uuid.New().ID()),
		UserID:    userID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(exp),
	}

	if err := s.Store(); err != nil {
		return nil, err
	}

	return s, nil
}

func GetSession(c *fiber.Ctx) (*Session, error) {
	idStr := c.Cookies("session_id")
	if idStr == "" {
		return nil, protocol.ErrNotAuthenticated
	}

	sID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return nil, protocol.ErrSessionFormat
	}

	s, err := GetSessionByID(uint(sID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, protocol.ErrSessionInvalid
		}
		return nil, err
	}

	if expired := s.IsExpired(); expired {
		return nil, protocol.ErrSessionExpired
	}

	return s, nil
}

func GetSessionByID(id uint) (*Session, error) {
	var s Session
	if err := DB.Model(&Session{}).Where("session_id = ?", fmt.Sprint(id)).First(&s).Error; err != nil {
		return nil, err
	}

	return &s, nil
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

func (s *Session) Store() error {
	if err := DB.Create(&s).Error; err != nil {
		return err
	}

	return nil
}

func (s *Session) Delete() error {
	if err := DB.Delete(&s).Error; err != nil {
		return err
	}

	return nil
}

func (s *Session) Update() error {
	if err := DB.Save(&s).Error; err != nil {
		return err
	}
	return nil
}

func (s *Session) UpdateExpiration(exp time.Duration) error {
	if time.Since(s.LastUpdated) < 30*time.Minute {
		return nil
	}

	s.LastUpdated = time.Now()
	s.ExpiresAt = time.Now().Add(exp)

	if err := s.Update(); err != nil {
		return err
	}

	return nil
}

func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}
