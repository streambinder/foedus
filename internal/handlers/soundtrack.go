package handlers

import (
	"log"
	"regexp"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/streambinder/foedus/internal/database"
	"github.com/streambinder/foedus/internal/spotify"
)

var playlistIDRe = regexp.MustCompile(`playlist/([a-zA-Z0-9]+)`)

const (
	soundtrackRateLimit   = 10 // requests per minute per IP
	soundtrackSearchLimit = 5  // max results per search
	soundtrackMaxQueryLen = 100
)

var soundtrackRateLimiter sync.Map // ip -> []time.Time

func init() {
	go func() {
		for range time.Tick(5 * time.Minute) {
			cutoff := time.Now().Add(-1 * time.Minute)
			soundtrackRateLimiter.Range(func(k, v any) bool {
				times := v.([]time.Time)
				var recent []time.Time
				for _, t := range times {
					if t.After(cutoff) {
						recent = append(recent, t)
					}
				}
				if len(recent) == 0 {
					soundtrackRateLimiter.Delete(k)
				} else {
					soundtrackRateLimiter.Store(k, recent)
				}
				return true
			})
		}
	}()
}

func checkSoundtrackRateLimit(ip string) bool {
	now := time.Now()
	cutoff := now.Add(-1 * time.Minute)
	raw, _ := soundtrackRateLimiter.LoadOrStore(ip, []time.Time{})
	times := raw.([]time.Time)
	var recent []time.Time
	for _, t := range times {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}
	if len(recent) >= soundtrackRateLimit {
		soundtrackRateLimiter.Store(ip, recent)
		return false
	}
	soundtrackRateLimiter.Store(ip, append(recent, now))
	return true
}

func SoundtrackEnabled() bool {
	return spotify.Enabled()
}

func SoundtrackSearch(c *fiber.Ctx) error {
	if !spotify.Enabled() {
		return c.SendStatus(fiber.StatusNotFound)
	}

	if !checkSoundtrackRateLimit(c.IP()) {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "rate limited"})
	}

	query := c.Query("q")
	if query == "" || len(query) > soundtrackMaxQueryLen {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid query"})
	}

	tracks, err := spotify.Search(query, soundtrackSearchLimit)
	if err != nil {
		log.Printf("soundtrack: search error q=%q: %v", query, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "search failed"})
	}

	return c.JSON(tracks)
}

func SoundtrackAdd(c *fiber.Ctx) error {
	if !spotify.Enabled() {
		return c.SendStatus(fiber.StatusNotFound)
	}

	if !checkSoundtrackRateLimit(c.IP()) {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "rate limited"})
	}

	var req struct {
		URI string `json:"uri"`
	}
	if err := c.BodyParser(&req); err != nil || req.URI == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	settings, err := database.GetAllSettings()
	if err != nil {
		log.Printf("soundtrack: failed to load settings: %v", err)
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	playlistID := spotifyPlaylistID(settings.SpotifyPlaylist)
	if playlistID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "no playlist configured"})
	}

	if err := spotify.AddToPlaylist(playlistID, req.URI); err != nil {
		log.Printf("soundtrack: add track error uri=%s: %v", req.URI, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to add track"})
	}

	log.Printf("soundtrack: track added uri=%s ip=%s", req.URI, c.IP())
	return c.JSON(fiber.Map{"ok": true})
}

func spotifyPlaylistID(rawURL string) string {
	m := playlistIDRe.FindStringSubmatch(rawURL)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}
