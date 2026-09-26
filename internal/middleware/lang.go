package middleware

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/streambinder/foedus/internal/i18n"
)

func LangDetect() fiber.Handler {
	return func(c *fiber.Ctx) error {
		lang := i18n.DetectLang(c.Get("Accept-Language"))
		c.Locals("lang", lang)
		c.Locals("t", i18n.NewT(lang))
		slog.Debug("language detected", "request_id", c.Locals("request_id"), "lang", lang)
		return c.Next()
	}
}
