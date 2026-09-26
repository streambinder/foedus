package handlers

import (
	"strconv"
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/streambinder/foedus/internal/database"
)

type cachedMedia struct {
	mime  string
	bytes []byte
}

const mediaCacheMaxBytes = 64 * 1024 * 1024 // 64MB total cap

var (
	mediaCacheMu    sync.RWMutex
	mediaCache      = make(map[int]cachedMedia)
	mediaCacheBytes int
)

func getCachedMedia(id int) (cachedMedia, bool, error) {
	mediaCacheMu.RLock()
	if entry, ok := mediaCache[id]; ok {
		mediaCacheMu.RUnlock()
		return entry, true, nil
	}
	mediaCacheMu.RUnlock()

	media, err := database.GetMedia(id)
	if err != nil {
		return cachedMedia{}, false, err
	}
	entry := cachedMedia{mime: media.Mime, bytes: media.Bytes}

	mediaCacheMu.Lock()
	if _, ok := mediaCache[id]; !ok {
		// crude eviction: if over cap, drop everything before adding
		if mediaCacheBytes+len(entry.bytes) > mediaCacheMaxBytes {
			mediaCache = make(map[int]cachedMedia)
			mediaCacheBytes = 0
		}
		mediaCache[id] = entry
		mediaCacheBytes += len(entry.bytes)
	}
	mediaCacheMu.Unlock()

	return entry, true, nil
}

func MediaImage(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return c.SendStatus(fiber.StatusNotFound)
	}

	etag := `"m` + strconv.Itoa(id) + `"`
	if c.Get(fiber.HeaderIfNoneMatch) == etag {
		return c.SendStatus(fiber.StatusNotModified)
	}

	entry, ok, err := getCachedMedia(id)
	if err != nil || !ok {
		return c.SendStatus(fiber.StatusNotFound)
	}

	c.Set(fiber.HeaderCacheControl, "public, max-age=31536000, immutable")
	c.Set(fiber.HeaderETag, etag)
	c.Set(fiber.HeaderContentType, entry.mime)
	c.Set(fiber.HeaderContentLength, strconv.Itoa(len(entry.bytes)))
	return c.Send(entry.bytes)
}
