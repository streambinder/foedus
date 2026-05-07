package handlers

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/streambinder/foedus/internal/database"
	"github.com/streambinder/foedus/internal/models"
	"github.com/streambinder/foedus/templates"
)

const (
	defaultOGImagePath   = "./static/og-preview.png"
	defaultOGImageType   = "image/png"
	defaultOGImageWidth  = "1200"
	defaultOGImageHeight = "630"
)

func BuildOGMeta(baseURL, pageURL, title, description string, settings models.WeddingSettings) templates.OGMeta {
	imageType := defaultOGImageType
	imageWidth := defaultOGImageWidth
	imageHeight := defaultOGImageHeight
	if settings.SharePreviewMediaID > 0 {
		if mime, _, err := database.GetMediaMeta(settings.SharePreviewMediaID); err == nil && mime != "" {
			imageType = mime
			// dimensions unknown without decoding bytes — drop them
			imageWidth = ""
			imageHeight = ""
		}
	}
	return templates.OGMeta{
		Title:       title,
		Description: description,
		URL:         pageURL,
		ImageURL:    baseURL + "/og-image",
		ImageType:   imageType,
		ImageWidth:  imageWidth,
		ImageHeight: imageHeight,
	}
}

func ogCeremonyLocation(settings models.WeddingSettings) string {
	var parts []string
	if settings.CeremonyAddress != "" {
		parts = append(parts, settings.CeremonyAddress)
	}
	if settings.CeremonyCity != "" {
		parts = append(parts, settings.CeremonyCity)
	}
	if len(parts) > 0 {
		return strings.Join(parts, ", ")
	}
	return settings.CeremonyLocation
}

func OGImage(c *fiber.Ctx) error {
	c.Set("Cache-Control", "public, max-age=3600")

	settings, err := database.GetAllSettings()
	if err == nil && settings.SharePreviewMediaID > 0 {
		if media, mediaErr := database.GetMedia(settings.SharePreviewMediaID); mediaErr == nil {
			c.Set(fiber.HeaderContentType, media.Mime)
			return c.Send(media.Bytes)
		}
	}

	return c.SendFile(defaultOGImagePath)
}
