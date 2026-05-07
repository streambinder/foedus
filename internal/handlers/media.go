package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/streambinder/foedus/internal/database"
)

func MediaImage(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return c.SendStatus(fiber.StatusNotFound)
	}

	etag := `"m` + strconv.Itoa(id) + `"`
	if c.Get(fiber.HeaderIfNoneMatch) == etag {
		return c.SendStatus(fiber.StatusNotModified)
	}

	media, err := database.GetMedia(id)
	if err != nil {
		return c.SendStatus(fiber.StatusNotFound)
	}

	c.Set(fiber.HeaderCacheControl, "public, max-age=31536000, immutable")
	c.Set(fiber.HeaderETag, etag)
	c.Set(fiber.HeaderContentType, media.Mime)
	return c.Send(media.Bytes)
}
