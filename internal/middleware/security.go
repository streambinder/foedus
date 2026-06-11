package middleware

import "github.com/gofiber/fiber/v2"

// SecurityHeaders adds standard security headers to every response.
// CSP is intentionally broad because the app uses inline scripts and
// embeds third-party resources (Spotify iframe, Leaflet tiles, Google Fonts).
func SecurityHeaders() fiber.Handler {
	// built once, reused per-request
	const csp = "default-src 'self';" +
		" script-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net https://unpkg.com;" +
		" style-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net https://fonts.googleapis.com https://unpkg.com;" +
		" font-src 'self' https://fonts.gstatic.com;" +
		" img-src 'self' https://*.basemaps.cartocdn.com data:;" +
		" frame-src https://open.spotify.com;" +
		" connect-src 'self' https://*.basemaps.cartocdn.com;" +
		" base-uri 'none';" +
		" form-action 'self'"

	return func(c *fiber.Ctx) error {
		c.Set("Content-Security-Policy", csp)
		return c.Next()
	}
}
