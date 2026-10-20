package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// baseCSP is the Content-Security-Policy served on every response.
// It is intentionally broad because the app uses inline scripts and
// embeds third-party resources (Spotify iframe, Leaflet tiles, Google Fonts).
const baseCSP = "default-src 'self';" +
	" script-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net https://unpkg.com;" +
	" style-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net https://fonts.googleapis.com https://unpkg.com;" +
	" font-src 'self' https://fonts.gstatic.com;" +
	" img-src 'self' blob: https://*.basemaps.cartocdn.com https://i.scdn.co data:;" +
	" frame-src https://open.spotify.com;" +
	" connect-src 'self' https://*.basemaps.cartocdn.com https://nominatim.openstreetmap.org https://photon.komoot.io;" +
	" base-uri 'none';" +
	" form-action 'self'"

// CSPNonceKey is the Locals key carrying the per-request CSP nonce.
const CSPNonceKey = "csp_nonce"

// SecurityHeaders adds standard security headers to every response.
func SecurityHeaders() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("Content-Security-Policy", baseCSP)
		return c.Next()
	}
}

// CSPNonce generates a fresh nonce for the request, exposes it to templates
// via Locals(CSPNonceKey) and adds it to the CSP script-src directive.
// Apply it only to route groups that load Datastar (dashboard, invitation):
// Datastar's CSP mode authorises its expression compiler with this nonce
// instead of requiring 'unsafe-eval'. Templates echo the nonce in a
// data-nonce attribute on the <html> element; a nonce is single-use and
// must never be reused across responses.
//
// Browsers ignore 'unsafe-inline' in script-src as soon as a nonce is
// present. Inline script elements on those pages therefore carry the nonce
// attribute themselves. Inline event handler attributes cannot carry a
// nonce, so script-src-attr keeps 'unsafe-inline' for them and preserves
// the base policy behaviour for attributes only.
func CSPNonce() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var raw [16]byte
		if _, err := rand.Read(raw[:]); err != nil {
			return err
		}
		nonce := base64.StdEncoding.EncodeToString(raw[:])
		c.Locals(CSPNonceKey, nonce)
		c.Set("Content-Security-Policy",
			strings.Replace(baseCSP, "script-src ", "script-src 'nonce-"+nonce+"' ", 1)+
				"; script-src-attr 'unsafe-inline'")
		return c.Next()
	}
}
