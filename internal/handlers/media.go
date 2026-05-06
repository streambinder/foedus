package handlers

import (
	"encoding/hex"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/streambinder/foedus/internal/database"
)

type cachedMediaImage struct {
	mimeType string
	data     []byte
}

var mediaCache = struct {
	sync.RWMutex
	images map[string]cachedMediaImage
}{}

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
	images, err := getMediaCache()
	if err != nil {
		return "", nil, false
	}
	image, ok := images[token]
	if !ok {
		return "", nil, false
	}
	return image.mimeType, image.data, true
}

func getMediaCache() (map[string]cachedMediaImage, error) {
	mediaCache.RLock()
	if mediaCache.images != nil {
		images := mediaCache.images
		mediaCache.RUnlock()
		return images, nil
	}
	mediaCache.RUnlock()

	mediaCache.Lock()
	defer mediaCache.Unlock()
	if mediaCache.images != nil {
		return mediaCache.images, nil
	}

	images, err := buildMediaCache()
	if err != nil {
		return nil, err
	}
	mediaCache.images = images
	return mediaCache.images, nil
}

func buildMediaCache() (map[string]cachedMediaImage, error) {
	images := make(map[string]cachedMediaImage)
	add := func(image string) {
		if image == "" || !strings.HasPrefix(image, "data:image/") {
			return
		}
		token := imageToken(image)
		if _, exists := images[token]; exists {
			return
		}
		mimeType, data, err := decodeDataURI(image)
		if err != nil {
			return
		}
		images[token] = cachedMediaImage{mimeType: mimeType, data: data}
	}

	settings, err := database.GetAllSettings()
	if err != nil {
		return nil, err
	}
	add(settings.CeremonyImage)
	add(settings.ReceptionImage)
	add(settings.SharePreviewImage)
	for _, place := range settings.Places {
		add(place.Image)
	}
	for _, place := range settings.HoneymoonLocations {
		add(place.Image)
	}
	for _, background := range settings.HomepageHeroBackgrounds {
		add(background.DesktopImage)
		add(background.MobileImage)
	}

	items, err := database.GetAllRegistryItems()
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		add(item.Image)
	}

	return images, nil
}

func invalidateMediaCache() {
	mediaCache.Lock()
	mediaCache.images = nil
	mediaCache.Unlock()
}
