package handlers

import (
	"regexp"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/streambinder/foedus/internal/database"
	"github.com/streambinder/foedus/internal/observability"
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
	logger := handlerLogger(c)

	if !spotify.Enabled() {
		logger.Warn("soundtrack search rejected", "reason", "spotify disabled")
		return c.SendStatus(fiber.StatusNotFound)
	}

	if !checkSoundtrackRateLimit(c.IP()) {
		logger.Warn("soundtrack search rate limited")
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "rate limited"})
	}

	query := c.Query("q")
	if query == "" || len(query) > soundtrackMaxQueryLen {
		logger.Warn("soundtrack search invalid query", "query_len", len(query))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid query"})
	}

	tracks, err := spotify.Search(query, soundtrackSearchLimit)
	if err != nil {
		logger.Error("soundtrack search failed", "query_len", len(query), "error", err.Error())
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "search failed"})
	}

	logger.Info("soundtrack search completed", "query_len", len(query), "results", len(tracks))
	return c.JSON(tracks)
}

func SoundtrackAdd(c *fiber.Ctx) error {
	logger := handlerLogger(c)

	if !spotify.Enabled() {
		logger.Warn("soundtrack add rejected", "reason", "spotify disabled")
		return c.SendStatus(fiber.StatusNotFound)
	}

	if !checkSoundtrackRateLimit(c.IP()) {
		logger.Warn("soundtrack add rate limited")
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "rate limited"})
	}

	var req struct {
		URI string `json:"uri"`
	}
	if err := c.BodyParser(&req); err != nil || req.URI == "" {
		logger.Warn("soundtrack add invalid request", "error", errString(err))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	settings, err := database.GetAllSettings()
	if err != nil {
		logger.Error("soundtrack add failed to load settings", "error", err.Error())
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	playlistID := spotifyPlaylistID(settings.SpotifyPlaylist)
	if playlistID == "" {
		logger.Warn("soundtrack add rejected", "reason", "playlist not configured")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "no playlist configured"})
	}

	if err := spotify.AddToPlaylist(playlistID, req.URI); err != nil {
		logger.Error("soundtrack add failed",
			"playlist_id", observability.Redact(playlistID),
			"track_uri", observability.Redact(req.URI),
			"error", err.Error(),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to add track"})
	}

	logger.Info("soundtrack track added",
		"playlist_id", observability.Redact(playlistID),
		"track_uri", observability.Redact(req.URI),
	)
	return c.JSON(fiber.Map{"ok": true})
}

func spotifyPlaylistID(rawURL string) string {
	m := playlistIDRe.FindStringSubmatch(rawURL)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
