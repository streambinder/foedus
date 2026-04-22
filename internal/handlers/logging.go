package handlers

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/streambinder/foedus/internal/observability"
)

func handlerLogger(c *fiber.Ctx) *slog.Logger {
	return observability.LoggerFromFiber(c)
}
