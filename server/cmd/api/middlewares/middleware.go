package middlewares

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/wotlk888/gesellschaft-hale/protocol"
	"github.com/wotlk888/gesellschaft-hale/server/cmd/api/sessions"
	"github.com/wotlk888/gesellschaft-hale/server/models"
)

func SessionChecker(c *fiber.Ctx) error {
	s, err := sessions.GetSession(c)
	if err != nil {
		return err
	}

	if expired := s.IsExpired(); expired {
		return protocol.ErrSessionExpired
	}

	u, err := models.GetUserByID(s.UserID)
	if err != nil {
		return err
	}

	c.Locals("user", *u)

	return c.Next()
}

func AdminOnly(c *fiber.Ctx) error {
	u, err := sessions.GetUserFromSession(c)
	if err != nil {
		return err
	}

	if u.Role == models.ROLE_ADMIN {
		return c.Next()
	}

	return fiber.ErrUnauthorized
}

func LocalsStorer(c *fiber.Ctx) error {
	requestUrl := c.OriginalURL()
	parts := strings.Split(requestUrl, "/")
	idParam := parts[len(parts)-1]
	StoreIDInLocals(c, idParam)

	return c.Next()
}

func CampaignChecker(c *fiber.Ctx) error {
	u, err := sessions.GetUserFromSession(c)
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

func StoreIDInLocals(c *fiber.Ctx, x string) {
	// due to how we pull the id, it might not be an integer
	if x == "" {
		c.Locals("id", nil)
		return
	}

	id64, err := strconv.ParseUint(x, 10, 32)
	if err != nil {
		c.Locals("id", nil)
		return
	}

	id := uint(id64)

	c.Locals("id", &id)
}

func GetIDInLocals(c *fiber.Ctx) (uint, bool) {
	// use * for nil, to not trigger with 0, but return uint for convenience of usage
	param := c.Locals("id")

	if rv := reflect.ValueOf(param); !rv.IsValid() || rv.IsNil() {
		return 0, false
	} // its not nil, so we can say it's *uint for sure, as we store that.

	id := param.(*uint)

	return *id, true
}
