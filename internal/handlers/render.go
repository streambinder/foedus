package handlers

import (
	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
)

func Render(c *fiber.Ctx, component templ.Component) error {
	c.Set("Content-Type", "text/html")
	c.Set(fiber.HeaderCacheControl, "no-cache")
	return adaptor.HTTPHandler(templ.Handler(component))(c)
}
