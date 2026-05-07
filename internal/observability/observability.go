package observability

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"log/slog"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
)

const (
	RequestIDHeader = "X-Request-ID"
	requestIDKey    = "request_id"
)

func Init() *slog.Logger {
	level := parseLevel(os.Getenv("LOG_LEVEL"))
	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if strings.EqualFold(strings.TrimSpace(os.Getenv("LOG_FORMAT")), "json") {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)

	stdLogger := slog.NewLogLogger(handler, level)
	log.SetFlags(0)
	log.SetOutput(stdLogger.Writer())

	logger.Info(
		"logger initialized",
		"format", logFormat(),
		"level", level.String(),
	)

	return logger
}

func RequestIDFromFiber(c *fiber.Ctx) string {
	if requestID, ok := c.Locals(requestIDKey).(string); ok && requestID != "" {
		return requestID
	}
	return ""
}

func SetRequestID(c *fiber.Ctx, requestID string) {
	c.Locals(requestIDKey, requestID)
	c.Set(RequestIDHeader, requestID)
}

func LoggerFromFiber(c *fiber.Ctx) *slog.Logger {
	logger := slog.With(
		"request_id", RequestIDFromFiber(c),
		"method", c.Method(),
		"route", RoutePattern(c),
		"ip", c.IP(),
	)

	if lang, ok := c.Locals("lang").(string); ok && lang != "" {
		logger = logger.With("lang", lang)
	}
	if user := AdminUsername(c); user != "" {
		logger = logger.With("admin_user", user)
	}

	return logger
}

func RoutePattern(c *fiber.Ctx) string {
	if route := c.Route(); route != nil && route.Path != "" {
		return route.Path
	}
	return c.Path()
}

func AdminUsername(c *fiber.Ctx) string {
	if username, ok := c.Locals("username").(string); ok {
		return username
	}
	return ""
}

func GenerateRequestID() string {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return "request-id-unavailable"
	}
	return hex.EncodeToString(buf)
}

func Redact(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) <= 4 {
		return "****"
	}
	return value[:2] + "..." + value[len(value)-2:]
}

func Truncate(value string, max int) string {
	value = strings.TrimSpace(value)
	if max <= 0 || len(value) <= max {
		return value
	}
	if max <= 3 {
		return value[:max]
	}
	return value[:max-3] + "..."
}

func parseLevel(raw string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func logFormat() string {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("LOG_FORMAT")), "json") {
		return "json"
	}
	return "text"
}
