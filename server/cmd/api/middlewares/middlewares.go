package middlewares

import (
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/wotlk888/gesellschaft-hale/protocol"
	"github.com/wotlk888/gesellschaft-hale/server/models"
)

func SessionChecker(c *fiber.Ctx) error {
	s, err := models.GetSession(c)
	if err != nil {
		return err
	}

	if expired := s.IsExpired(); expired {
		return protocol.ErrSessionExpired
	}

	// update session, set it to 48 hours again to expires.
	// not mandatory to return on error.
	if err := s.UpdateExpiration(48 * time.Hour); err != nil {
		log.Println("UpdateExpiration(): ", err)
	}

	u, err := models.GetUserByID(s.UserID)
	if err != nil {
		return err
	}

	c.Locals("user", *u)

	return c.Next()
}

func AdminOnly(c *fiber.Ctx) error {
	u, err := models.GetUserFromSession(c)
	if err != nil {
		return err
	}

	if u.Role == models.ROLE_ADMIN {
		return c.Next()
	}

	return fiber.ErrUnauthorized
}

func IDStorer(c *fiber.Ctx) error {
	requestUrl := c.OriginalURL()
	parts := strings.Split(requestUrl, "/")
	idParam := parts[len(parts)-1]
	StoreIDInLocals(c, idParam)

	return c.Next()
}

func CampaignChecker(c *fiber.Ctx) error {
	u, err := models.GetUserFromSession(c)
	if err != nil {
		return err
	}

	id, has := GetIDInLocals(c)
	if !has {
		return c.Next()
	}

	// campaign exists at all?
	if _, err := models.GetCampaign(id); err != nil {
		return err
	}

	// skip validation of ownership, admin can edit all
	if u.Role == models.ROLE_ADMIN {
		return c.Next()
	}

	if err := models.VerifyCampaignOwnership(u, id); err != nil {
		return err
	}

	return c.Next()
}
