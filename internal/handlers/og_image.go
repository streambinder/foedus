package handlers

import (
	"encoding/base64"
	"strings"

	"github.com/streambinder/foedus/internal/database"
	"github.com/gofiber/fiber/v2"
)

func OGImage(c *fiber.Ctx) error {
	c.Set("Cache-Control", "public, max-age=3600")

	settings, err := database.GetAllSettings()
	if err == nil && settings.SharePreviewImage != "" {
		// parse data URI: "data:<mime>;base64,<data>"
		if idx := strings.Index(settings.SharePreviewImage, ","); idx != -1 {
			header := settings.SharePreviewImage[:idx]
			encoded := settings.SharePreviewImage[idx+1:]
			mimeType := "image/png"
			if start := strings.Index(header, ":"); start != -1 {
				end := strings.Index(header, ";")
				if end != -1 {
					mimeType = header[start+1 : end]
				}
			}
			data, decodeErr := base64.StdEncoding.DecodeString(encoded)
			if decodeErr == nil {
				c.Set("Content-Type", mimeType)
				return c.Send(data)
			}
		}
	}

	// fallback: serve static PNG favicon
	return c.SendFile("./static/favicon.png")
}
