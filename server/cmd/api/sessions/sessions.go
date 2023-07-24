package sessions

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wotlk888/gesellschaft-hale/protocol"
	"github.com/wotlk888/gesellschaft-hale/server/models"
	"gorm.io/gorm"
)

type Session struct {
	ID        uint      `json:"session_id" gorm:"column:session_id"`
	UserID    uint      `json:"user_id" gorm:"column:user_id"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at"`
	ExpiresAt time.Time `json:"expires_at" gorm:"column:expires_at"`
}

func New(userID uint, exp time.Duration) (*Session, error) {
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

	s, err := getSessionByID(uint(sID))
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

func getSessionByID(id uint) (*Session, error) {
	var s Session
	if err := models.DB.Model(&Session{}).Where("session_id = ?", fmt.Sprint(id)).First(&s).Error; err != nil {
		return nil, err
	}

	return &s, nil
}

func GetUserFromSession(c *fiber.Ctx) (*models.User, error) {
	user := c.Locals("user").(models.User)
	id := *user.ID

	u, err := models.GetUserByID(id)
	if err != nil {
		return nil, err
	}

	return u, nil
}

func (s *Session) Store() error {
	if err := models.DB.Create(&s).Error; err != nil {
		return err
	}

	return nil
}

func (s *Session) Delete() error {
	if err := models.DB.Delete(&s).Error; err != nil {
		return err
	}

	return nil
}

func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}
