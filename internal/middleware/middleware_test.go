package middleware

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/streambinder/foedus/internal/i18n"
)

func doRequest(t *testing.T, app *fiber.App, req *http.Request) (int, http.Header, string) {
	t.Helper()
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, resp.Header, string(body)
}

func TestBasicAuth(t *testing.T) {
	t.Setenv("ADMIN_USER", "admin")
	t.Setenv("ADMIN_PASSWORD", "s3cret")
	t.Setenv("ADMIN_USER1", "second")
	t.Setenv("ADMIN_PASSWORD1", "pass1")
	// a user without a password is skipped
	t.Setenv("ADMIN_USER2", "nopass")
	t.Setenv("ADMIN_PASSWORD2", "")

	app := fiber.New()
	app.Use(BasicAuth())
	app.Get("/", func(c *fiber.Ctx) error { return c.SendString("welcome") })
	t.Cleanup(func() { _ = app.Shutdown() })

	status, _, _ := doRequest(t, app, httptest.NewRequest(fiber.MethodGet, "/", nil))
	if status != fiber.StatusUnauthorized {
		t.Errorf("no credentials: status = %d, want 401", status)
	}

	req := httptest.NewRequest(fiber.MethodGet, "/", nil)
	req.SetBasicAuth("admin", "wrong")
	status, headers, _ := doRequest(t, app, req)
	if status != fiber.StatusUnauthorized {
		t.Errorf("wrong password: status = %d, want 401", status)
	}
	if got := headers.Get(fiber.HeaderWWWAuthenticate); got != "basic realm=Restricted" {
		t.Errorf("WWW-Authenticate = %q", got)
	}

	req = httptest.NewRequest(fiber.MethodGet, "/", nil)
	req.SetBasicAuth("admin", "s3cret")
	status, _, body := doRequest(t, app, req)
	if status != fiber.StatusOK || body != "welcome" {
		t.Errorf("valid credentials: status = %d body = %q", status, body)
	}

	req = httptest.NewRequest(fiber.MethodGet, "/", nil)
	req.SetBasicAuth("second", "pass1")
	status, _, _ = doRequest(t, app, req)
	if status != fiber.StatusOK {
		t.Errorf("second admin: status = %d, want 200", status)
	}

	req = httptest.NewRequest(fiber.MethodGet, "/", nil)
	req.SetBasicAuth("nopass", "anything")
	status, _, _ = doRequest(t, app, req)
	if status != fiber.StatusUnauthorized {
		t.Errorf("passwordless user: status = %d, want 401", status)
	}
}

func TestBasicAuthPanicsWithoutCredentials(t *testing.T) {
	for _, suffix := range []string{"", "1", "2", "3", "4", "5", "6", "7", "8", "9"} {
		t.Setenv("ADMIN_USER"+suffix, "")
		t.Setenv("ADMIN_PASSWORD"+suffix, "")
	}
	defer func() {
		if recover() == nil {
			t.Fatal("BasicAuth did not panic without any admin credentials")
		}
	}()
	BasicAuth()
}

func TestLangDetect(t *testing.T) {
	app := fiber.New()
	app.Use(LangDetect())
	app.Get("/", func(c *fiber.Ctx) error {
		lang, _ := c.Locals("lang").(string)
		translator, _ := c.Locals("t").(i18n.T)
		if translator == nil {
			return c.SendString(lang + ":no-translator")
		}
		return c.SendString(lang + ":" + translator("title.setup"))
	})
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest(fiber.MethodGet, "/", nil)
	req.Header.Set("Accept-Language", "it-IT,it;q=0.9")
	_, _, body := doRequest(t, app, req)
	if body != "it:Configurazione necessaria" {
		t.Errorf("italian request body = %q", body)
	}

	_, _, body = doRequest(t, app, httptest.NewRequest(fiber.MethodGet, "/", nil))
	if body != "en:Setup Required" {
		t.Errorf("default request body = %q", body)
	}
}

func TestSecurityHeaders(t *testing.T) {
	app := fiber.New()
	app.Use(SecurityHeaders())
	app.Get("/", func(c *fiber.Ctx) error { return c.SendString("ok") })
	t.Cleanup(func() { _ = app.Shutdown() })

	_, headers, _ := doRequest(t, app, httptest.NewRequest(fiber.MethodGet, "/", nil))
	csp := headers.Get("Content-Security-Policy")
	if !strings.HasPrefix(csp, "default-src 'self';") {
		t.Errorf("CSP header = %q", csp)
	}
	if !strings.Contains(csp, "form-action 'self'") {
		t.Errorf("CSP header missing form-action: %q", csp)
	}
}

func TestRequestContext(t *testing.T) {
	app := fiber.New()
	app.Use(RequestContext())
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString(c.GetRespHeader("X-Request-ID"))
	})
	t.Cleanup(func() { _ = app.Shutdown() })

	// a client-supplied id is kept
	req := httptest.NewRequest(fiber.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "client-id-1")
	_, _, body := doRequest(t, app, req)
	if body != "client-id-1" {
		t.Errorf("request id = %q, want client-id-1", body)
	}

	// a missing id is generated
	_, _, body = doRequest(t, app, httptest.NewRequest(fiber.MethodGet, "/", nil))
	if len(body) != 24 {
		t.Errorf("generated request id = %q, want 24 hex chars", body)
	}

	// an over-long id is replaced
	req = httptest.NewRequest(fiber.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", strings.Repeat("a", 200))
	_, _, body = doRequest(t, app, req)
	if len(body) != 24 {
		t.Errorf("over-long request id was kept: %q", body)
	}

	// a whitespace-only id is replaced
	req = httptest.NewRequest(fiber.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "   ")
	_, _, body = doRequest(t, app, req)
	if len(body) != 24 {
		t.Errorf("whitespace request id was kept: %q", body)
	}
}

func TestAccessLog(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: ErrorHandler})
	app.Use(RequestContext())
	app.Use(AccessLog())
	app.Get("/ok", func(c *fiber.Ctx) error { return c.SendString("ok") })
	app.Get("/healthz", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })
	app.Get("/bad", func(c *fiber.Ctx) error { return fiber.NewError(fiber.StatusBadRequest, "bad") })
	app.Get("/boom", func(c *fiber.Ctx) error { return errors.New("kaboom") })
	t.Cleanup(func() { _ = app.Shutdown() })

	for path, want := range map[string]int{
		"/ok":      fiber.StatusOK,
		"/healthz": fiber.StatusOK,
		"/bad":     fiber.StatusBadRequest,
		"/boom":    fiber.StatusInternalServerError,
	} {
		status, _, _ := doRequest(t, app, httptest.NewRequest(fiber.MethodGet, path, nil))
		if status != want {
			t.Errorf("GET %s: status = %d, want %d", path, status, want)
		}
	}
}

func TestErrorHandler(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: ErrorHandler})
	app.Use(RequestContext())
	app.Get("/fiber", func(c *fiber.Ctx) error { return fiber.NewError(fiber.StatusTeapot, "teapot") })
	app.Get("/plain", func(c *fiber.Ctx) error { return errors.New("mystery") })
	t.Cleanup(func() { _ = app.Shutdown() })

	status, _, body := doRequest(t, app, httptest.NewRequest(fiber.MethodGet, "/fiber", nil))
	if status != fiber.StatusTeapot || body != "teapot" {
		t.Errorf("fiber error: status = %d body = %q", status, body)
	}
	status, _, body = doRequest(t, app, httptest.NewRequest(fiber.MethodGet, "/plain", nil))
	if status != fiber.StatusInternalServerError || body != "internal server error" {
		t.Errorf("plain error: status = %d body = %q", status, body)
	}
}

func TestStatusCodeFromResult(t *testing.T) {
	for _, tc := range []struct {
		name     string
		err      error
		fallback int
		want     int
	}{
		{"no error keeps fallback", nil, 201, 201},
		{"no error zero fallback is ok", nil, 0, 200},
		{"fiber error wins", fiber.NewError(404, "missing"), 200, 404},
		{"wrapped fiber error wins", fmtError{inner: fiber.NewError(403, "no")}, 200, 403},
		{"plain error with error fallback", errors.New("x"), 500, 500},
		{"plain error with ok fallback is 500", errors.New("x"), 200, 500},
		{"plain error with zero fallback is 500", errors.New("x"), 0, 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := statusCodeFromResult(tc.err, tc.fallback); got != tc.want {
				t.Errorf("statusCodeFromResult = %d, want %d", got, tc.want)
			}
		})
	}
}

type fmtError struct{ inner error }

func (e fmtError) Error() string { return "wrapped: " + e.inner.Error() }
func (e fmtError) Unwrap() error { return e.inner }

func TestRequestLogLevel(t *testing.T) {
	if got := RequestLogLevel(200); got.String() != "INFO" {
		t.Errorf("RequestLogLevel(200) = %v", got)
	}
	if got := RequestLogLevel(404); got.String() != "WARN" {
		t.Errorf("RequestLogLevel(404) = %v", got)
	}
	if got := RequestLogLevel(503); got.String() != "ERROR" {
		t.Errorf("RequestLogLevel(503) = %v", got)
	}
}
