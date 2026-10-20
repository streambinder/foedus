package middleware

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestSecurityHeadersServesBaseCSP(t *testing.T) {
	app := fiber.New()
	app.Use(SecurityHeaders())
	app.Get("/", func(c *fiber.Ctx) error { return c.SendString("ok") })

	_, header, _ := doRequest(t, app, httptest.NewRequest("GET", "/", nil))
	csp := header.Get("Content-Security-Policy")
	if csp != baseCSP {
		t.Fatalf("CSP = %q, want the base policy", csp)
	}
	if strings.Contains(csp, "nonce-") {
		t.Fatalf("base CSP must not carry a nonce: %q", csp)
	}
}

func TestCSPNonceAddsNonceToScriptSrcAndLocals(t *testing.T) {
	app := fiber.New()
	app.Use(SecurityHeaders())
	app.Use(CSPNonce())
	var localNonce string
	app.Get("/", func(c *fiber.Ctx) error {
		localNonce, _ = c.Locals(CSPNonceKey).(string)
		return c.SendString("ok")
	})

	_, header, _ := doRequest(t, app, httptest.NewRequest("GET", "/", nil))
	csp := header.Get("Content-Security-Policy")
	if localNonce == "" {
		t.Fatal("nonce not exposed via Locals")
	}
	if !strings.Contains(csp, "script-src 'nonce-"+localNonce+"' 'self'") {
		t.Fatalf("CSP does not carry the Locals nonce in script-src: %q", csp)
	}
	if strings.Contains(csp, "unsafe-eval") {
		t.Fatalf("CSP must never gain unsafe-eval: %q", csp)
	}
	if !strings.Contains(csp, "script-src-attr 'unsafe-inline'") {
		t.Fatalf("CSP does not keep unsafe-inline for handler attributes: %q", csp)
	}
	// apart from the nonce and script-src-attr, the policy equals the base one
	want := strings.Replace(baseCSP, "script-src ", "script-src 'nonce-"+localNonce+"' ", 1) +
		"; script-src-attr 'unsafe-inline'"
	if csp != want {
		t.Fatalf("CSP = %q, want %q", csp, want)
	}
}

func TestCSPNonceIsFreshPerRequest(t *testing.T) {
	app := fiber.New()
	app.Use(CSPNonce())
	nonces := map[string]bool{}
	app.Get("/", func(c *fiber.Ctx) error {
		nonce, _ := c.Locals(CSPNonceKey).(string)
		nonces[nonce] = true
		return c.SendString("ok")
	})

	for i := 0; i < 8; i++ {
		_, _, _ = doRequest(t, app, httptest.NewRequest("GET", "/", nil))
	}
	if len(nonces) != 8 {
		t.Fatalf("got %d distinct nonces over 8 requests, want 8", len(nonces))
	}
}
