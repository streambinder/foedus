package handlers

import (
	"bytes"
	"encoding/base64"
	"image"
	"strconv"
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
	imageType, imageWidth, imageHeight := sharePreviewMetadata(settings.SharePreviewImage)
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

func sharePreviewMetadata(dataURI string) (string, string, string) {
	if dataURI == "" {
		return defaultOGImageType, defaultOGImageWidth, defaultOGImageHeight
	}

	mimeType, data, err := decodeDataURI(dataURI)
	if err != nil {
		return defaultOGImageType, defaultOGImageWidth, defaultOGImageHeight
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return mimeType, "", ""
	}

	return mimeType, strconv.Itoa(cfg.Width), strconv.Itoa(cfg.Height)
}

func decodeDataURI(dataURI string) (string, []byte, error) {
	idx := strings.Index(dataURI, ",")
	if idx == -1 {
		return "", nil, fiber.ErrBadRequest
	}

	header := dataURI[:idx]
	encoded := dataURI[idx+1:]
	mimeType := defaultOGImageType
	if start := strings.Index(header, ":"); start != -1 {
		end := strings.Index(header, ";")
		if end != -1 {
			mimeType = header[start+1 : end]
		}
	}

	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", nil, err
	}

	return mimeType, data, nil
}

func OGImage(c *fiber.Ctx) error {
	c.Set("Cache-Control", "public, max-age=3600")

	settings, err := database.GetAllSettings()
	if err == nil && settings.SharePreviewImage != "" {
		if mimeType, data, decodeErr := decodeDataURI(settings.SharePreviewImage); decodeErr == nil {
			c.Set("Content-Type", mimeType)
			return c.Send(data)
		}
	}

	return c.SendFile(defaultOGImagePath)
}
