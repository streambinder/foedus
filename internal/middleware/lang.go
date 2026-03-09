package middleware

import (
	"github.com/streambinder/foedus/internal/i18n"
	"github.com/gofiber/fiber/v2"
)

func LangDetect() fiber.Handler {
	return func(c *fiber.Ctx) error {
		lang := i18n.DetectLang(c.Get("Accept-Language"))
		c.Locals("lang", lang)
		c.Locals("t", i18n.NewT(lang))
		return c.Next()
	}
}
