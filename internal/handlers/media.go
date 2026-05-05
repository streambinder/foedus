package handlers

import (
	"database/sql"
	"encoding/hex"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/streambinder/foedus/internal/database"
)

func MediaImage(c *fiber.Ctx) error {
	token := strings.TrimSpace(c.Params("token"))
	if len(token) != 64 {
		return c.SendStatus(fiber.StatusNotFound)
	}
	if _, err := hex.DecodeString(token); err != nil {
		return c.SendStatus(fiber.StatusNotFound)
	}

	etag := `"` + token + `"`
	if c.Get(fiber.HeaderIfNoneMatch) == etag {
		return c.SendStatus(fiber.StatusNotModified)
	}

	if mimeType, data, ok := findImageByToken(token); ok {
		c.Set(fiber.HeaderCacheControl, "public, max-age=31536000, immutable")
		c.Set(fiber.HeaderETag, etag)
		c.Set(fiber.HeaderContentType, mimeType)
		return c.Send(data)
	}

	return c.SendStatus(fiber.StatusNotFound)
}

func findImageByToken(token string) (string, []byte, bool) {
	settings, err := database.GetAllSettings()
	if err == nil {
		if mimeType, data, ok := matchDataImage(token, settings.CeremonyImage); ok {
			return mimeType, data, true
		}
		if mimeType, data, ok := matchDataImage(token, settings.ReceptionImage); ok {
			return mimeType, data, true
		}
		if mimeType, data, ok := matchDataImage(token, settings.SharePreviewImage); ok {
			return mimeType, data, true
		}
		for _, place := range settings.Places {
			if mimeType, data, ok := matchDataImage(token, place.Image); ok {
				return mimeType, data, true
			}
		}
		for _, place := range settings.HoneymoonLocations {
			if mimeType, data, ok := matchDataImage(token, place.Image); ok {
				return mimeType, data, true
			}
		}
		for _, background := range settings.HomepageHeroBackgrounds {
			if mimeType, data, ok := matchDataImage(token, background.DesktopImage); ok {
				return mimeType, data, true
			}
			if mimeType, data, ok := matchDataImage(token, background.MobileImage); ok {
				return mimeType, data, true
			}
		}
	} else if err != sql.ErrNoRows {
		return "", nil, false
	}

	items, err := database.GetAllRegistryItems()
	if err != nil {
		return "", nil, false
	}
	for _, item := range items {
		if mimeType, data, ok := matchDataImage(token, item.Image); ok {
			return mimeType, data, true
		}
	}

	return "", nil, false
}

func matchDataImage(token, image string) (string, []byte, bool) {
	if image == "" || imageToken(image) != token || !strings.HasPrefix(image, "data:image/") {
		return "", nil, false
	}
	mimeType, data, err := decodeDataURI(image)
	if err != nil {
		return "", nil, false
	}
	return mimeType, data, true
}
