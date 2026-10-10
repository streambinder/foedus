package observability

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func newTestApp(t *testing.T, handler fiber.Handler) *fiber.App {
	t.Helper()
	app := fiber.New()
	app.Get("/test/:id", handler)
	t.Cleanup(func() { _ = app.Shutdown() })
	return app
}

func doRequest(t *testing.T, app *fiber.App, path string) string {
	t.Helper()
	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, path, nil))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func TestInitFormatsAndLevels(t *testing.T) {
	for _, tc := range []struct {
		name   string
		format string
		level  string
	}{
		{"text default level", "", ""},
		{"json format", "json", "debug"},
		{"json uppercase with spaces", " JSON ", "warn"},
		{"text warning level", "text", "warning"},
		{"text error level", "text", "error"},
		{"unknown level falls back to info", "text", "verbose"},
		{"info level", "text", "info"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("LOG_FORMAT", tc.format)
			t.Setenv("LOG_LEVEL", tc.level)
			logger := Init()
			if logger == nil {
				t.Fatal("Init returned a nil logger")
			}
		})
	}
}

func TestRequestIDHelpers(t *testing.T) {
	app := newTestApp(t, func(c *fiber.Ctx) error {
		if got := RequestIDFromFiber(c); got != "" {
			t.Errorf("fresh context request id = %q, want empty", got)
		}
		SetRequestID(c, "req-123")
		if got := RequestIDFromFiber(c); got != "req-123" {
			t.Errorf("request id = %q, want req-123", got)
		}
		if got := c.GetRespHeader(RequestIDHeader); got != "req-123" {
			t.Errorf("response header = %q, want req-123", got)
		}
		// a non-string local must not be mistaken for a request id
		c.Locals(requestIDKey, 42)
		if got := RequestIDFromFiber(c); got != "" {
			t.Errorf("non-string request id local = %q, want empty", got)
		}
		return c.SendString("ok")
	})
	doRequest(t, app, "/test/1")
}

func TestLoggerFromFiber(t *testing.T) {
	app := newTestApp(t, func(c *fiber.Ctx) error {
		if logger := LoggerFromFiber(c); logger == nil {
			t.Error("LoggerFromFiber returned nil")
		}
		c.Locals("lang", "it")
		c.Locals("username", "admin")
		if logger := LoggerFromFiber(c); logger == nil {
			t.Error("LoggerFromFiber with lang and user returned nil")
		}
		// wrongly typed locals are ignored
		c.Locals("lang", 7)
		c.Locals("username", 7)
		if logger := LoggerFromFiber(c); logger == nil {
			t.Error("LoggerFromFiber with mistyped locals returned nil")
		}
		return c.SendString("ok")
	})
	doRequest(t, app, "/test/9")
}

func TestRoutePattern(t *testing.T) {
	app := newTestApp(t, func(c *fiber.Ctx) error {
		if got := RoutePattern(c); got != "/test/:id" {
			t.Errorf("RoutePattern = %q, want /test/:id", got)
		}
		return c.SendString("ok")
	})
	doRequest(t, app, "/test/42")
}

func TestAdminUsername(t *testing.T) {
	app := newTestApp(t, func(c *fiber.Ctx) error {
		if got := AdminUsername(c); got != "" {
			t.Errorf("AdminUsername without local = %q, want empty", got)
		}
		c.Locals("username", "davide")
		if got := AdminUsername(c); got != "davide" {
			t.Errorf("AdminUsername = %q, want davide", got)
		}
		c.Locals("username", 42)
		if got := AdminUsername(c); got != "" {
			t.Errorf("AdminUsername with mistyped local = %q, want empty", got)
		}
		return c.SendString("ok")
	})
	doRequest(t, app, "/test/1")
}

func TestGenerateRequestID(t *testing.T) {
	first := GenerateRequestID()
	second := GenerateRequestID()
	if len(first) != 24 {
		t.Errorf("request id %q has length %d, want 24 hex chars", first, len(first))
	}
	if first == second {
		t.Error("two generated request ids are identical")
	}
}

func TestRedact(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want string
	}{
		{"", ""},
		{"   ", ""},
		{"ab", "****"},
		{"abcd", "****"},
		{"abcde", "ab...de"},
		{"  secret-token  ", "se...en"},
	} {
		if got := Redact(tc.in); got != tc.want {
			t.Errorf("Redact(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   string
		max  int
		want string
	}{
		{"empty", "", 5, ""},
		{"shorter than max", "abc", 5, "abc"},
		{"exactly max", "abcde", 5, "abcde"},
		{"zero max returns input", "abcdef", 0, "abcdef"},
		{"negative max returns input", "abcdef", -1, "abcdef"},
		{"max three hard cut", "abcdef", 3, "abc"},
		{"max two hard cut", "abcdef", 2, "ab"},
		{"truncated with ellipsis", "abcdefghij", 7, "abcd..."},
		{"trims before measuring", "  abcdefgh  ", 5, "ab..."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Truncate(tc.in, tc.max); got != tc.want {
				t.Errorf("Truncate(%q, %d) = %q, want %q", tc.in, tc.max, got, tc.want)
			}
		})
	}
}

func TestParseLevelAndLogFormat(t *testing.T) {
	if got := parseLevel("debug"); got.String() != "DEBUG" {
		t.Errorf("parseLevel(debug) = %v", got)
	}
	if got := parseLevel(" WaRn "); got.String() != "WARN" {
		t.Errorf("parseLevel(warn) = %v", got)
	}
	if got := parseLevel("warning"); got.String() != "WARN" {
		t.Errorf("parseLevel(warning) = %v", got)
	}
	if got := parseLevel("error"); got.String() != "ERROR" {
		t.Errorf("parseLevel(error) = %v", got)
	}
	if got := parseLevel(""); got.String() != "INFO" {
		t.Errorf("parseLevel(empty) = %v", got)
	}
	if got := parseLevel("trace"); got.String() != "INFO" {
		t.Errorf("parseLevel(trace) = %v", got)
	}

	t.Setenv("LOG_FORMAT", "json")
	if got := logFormat(); got != "json" {
		t.Errorf("logFormat = %q, want json", got)
	}
	t.Setenv("LOG_FORMAT", "text")
	if got := logFormat(); got != "text" {
		t.Errorf("logFormat = %q, want text", got)
	}
	if !strings.EqualFold(logFormat(), "text") {
		t.Error("logFormat sanity check failed")
	}
}
