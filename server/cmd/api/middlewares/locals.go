package middlewares

import (
	"reflect"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

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
