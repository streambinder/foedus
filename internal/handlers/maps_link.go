package handlers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// mapsPlace is the place a Google Maps link carries: the name from the
// /place/ path segment and the coordinates from the pin data, the @
// viewport or a numeric query, in that order of precision.
type mapsPlace struct {
	Name string  `json:"name"`
	Lat  float64 `json:"lat"`
	Lng  float64 `json:"lng"`
}

var (
	mapsPinPattern   = regexp.MustCompile(`!3d(-?\d+(?:\.\d+)?)!4d(-?\d+(?:\.\d+)?)`)
	mapsAtPattern    = regexp.MustCompile(`@(-?\d+(?:\.\d+)?),(-?\d+(?:\.\d+)?)`)
	mapsCoordPattern = regexp.MustCompile(`^(-?\d+(?:\.\d+)?)[, ](-?\d+(?:\.\d+)?)$`)
)

// mapsHostPattern matches a google country domain at the end of the
// host, so lookalikes like google.com.evil.example never pass.
var mapsHostPattern = regexp.MustCompile(`(^|\.)google\.[a-z]{2,}(\.[a-z]{2})?$`)

// isMapsHost reports whether host belongs to Google Maps. Short links
// live on the goo.gl hosts; full links on any google country domain.
func isMapsHost(host string) bool {
	host = strings.ToLower(host)
	switch host {
	case "maps.app.goo.gl", "goo.gl", "maps.google.com":
		return true
	}
	return mapsHostPattern.MatchString(host)
}

func isShortMapsHost(host string) bool {
	host = strings.ToLower(host)
	return host == "maps.app.goo.gl" || host == "goo.gl"
}

func validCoords(lat, lng float64) bool {
	return lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180 && (lat != 0 || lng != 0)
}

// parseMapsURL extracts the place from an expanded Google Maps URL.
func parseMapsURL(raw string) (mapsPlace, bool) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return mapsPlace{}, false
	}
	place := mapsPlace{}
	if idx := strings.Index(u.Path, "/place/"); idx >= 0 {
		rest := u.Path[idx+len("/place/"):]
		if end := strings.Index(rest, "/"); end >= 0 {
			rest = rest[:end]
		}
		if name, err := url.PathUnescape(rest); err == nil {
			// google writes spaces in place names as plus signs
			place.Name = strings.TrimSpace(strings.ReplaceAll(name, "+", " "))
		}
	}
	coords := func(match []string) bool {
		if match == nil {
			return false
		}
		lat, latErr := strconv.ParseFloat(match[1], 64)
		lng, lngErr := strconv.ParseFloat(match[2], 64)
		if latErr != nil || lngErr != nil || !validCoords(lat, lng) {
			return false
		}
		place.Lat, place.Lng = lat, lng
		return true
	}
	if coords(mapsPinPattern.FindStringSubmatch(raw)) ||
		coords(mapsAtPattern.FindStringSubmatch(raw)) {
		return place, true
	}
	for _, key := range []string{"query", "q"} {
		if coords(mapsCoordPattern.FindStringSubmatch(strings.TrimSpace(u.Query().Get(key)))) {
			return place, true
		}
	}
	return mapsPlace{}, false
}

// mapsRedirectPolicy keeps link expansion on Google Maps hosts, so the
// endpoint can never be pointed at an arbitrary target.
func mapsRedirectPolicy(req *http.Request, via []*http.Request) error {
	if len(via) >= 5 {
		return errors.New("too many redirects")
	}
	if !isMapsHost(req.URL.Host) {
		return fmt.Errorf("redirect to non-maps host %q", req.URL.Host)
	}
	return nil
}

var mapsHTTPClient = &http.Client{Timeout: 6 * time.Second, CheckRedirect: mapsRedirectPolicy}

// expandMapsLink follows a short maps link and returns the final URL.
func expandMapsLink(raw string) (string, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, raw, nil)
	if err != nil {
		return "", err
	}
	// a desktop agent receives the standard redirect to the maps page
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36")
	resp, err := mapsHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	return resp.Request.URL.String(), nil
}

// ResolveMapsLink answers the place carried by a Google Maps link the
// admin pasted into a venue field. Full links parse in place; short
// links are expanded server-side because the browser cannot follow
// their redirect under CORS.
func ResolveMapsLink(c *fiber.Ctx) error {
	logger := handlerLogger(c)
	raw := strings.TrimSpace(c.Query("url"))
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || !isMapsHost(u.Host) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "not a maps link"})
	}
	if place, ok := parseMapsURL(raw); ok {
		return c.JSON(place)
	}
	if !isShortMapsHost(u.Host) {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "no place found in link"})
	}
	final, err := expandMapsLink(raw)
	if err != nil {
		logger.Warn("maps link expansion failed", "error", err.Error())
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "could not resolve the link"})
	}
	place, ok := parseMapsURL(final)
	if !ok {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "no place found in link"})
	}
	return c.JSON(place)
}
