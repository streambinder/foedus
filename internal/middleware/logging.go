package middleware

import (
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/streambinder/foedus/internal/observability"
)

func RequestContext() fiber.Handler {
	return func(c *fiber.Ctx) error {
		requestID := strings.TrimSpace(c.Get(observability.RequestIDHeader))
		if requestID == "" || len(requestID) > 128 {
			requestID = observability.GenerateRequestID()
		}

		observability.SetRequestID(c, requestID)
		return c.Next()
	}
}

func AccessLog() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// skip healthcheck pings: high-frequency, low-signal, would swamp the journal
		if c.Path() == "/healthz" {
			return c.Next()
		}

		start := time.Now()
		err := c.Next()

		status := statusCodeFromResult(err, c.Response().StatusCode())
		logger := observability.LoggerFromFiber(c).With(
			"status", status,
			"duration_ms", time.Since(start).Milliseconds(),
			"bytes_in", c.Request().Header.ContentLength(),
			"bytes_out", c.Response().Header.ContentLength(),
			"query_args", c.Request().URI().QueryArgs().Len(),
			"user_agent", observability.Truncate(c.Get(fiber.HeaderUserAgent), 120),
		)

		if err != nil {
			logger = logger.With("error", err.Error())
		}

		switch {
		case status >= fiber.StatusInternalServerError:
			logger.Error("request completed")
		case status >= fiber.StatusBadRequest:
			logger.Warn("request completed")
		default:
			logger.Info("request completed")
		}

		return err
	}
}

func ErrorHandler(c *fiber.Ctx, err error) error {
	observability.SetRequestID(c, observability.RequestIDFromFiber(c))

	fe := new(fiber.Error)
	if errors.As(err, &fe) {
		return c.Status(fe.Code).SendString(fe.Message)
	}

	observability.LoggerFromFiber(c).Error("unhandled server error", "error", err.Error())
	return c.Status(fiber.StatusInternalServerError).SendString("internal server error")
}

func statusCodeFromResult(err error, fallback int) int {
	if err == nil {
		if fallback == 0 {
			return fiber.StatusOK
		}
		return fallback
	}

	fe := new(fiber.Error)
	if errors.As(err, &fe) {
		return fe.Code
	}

	if fallback >= fiber.StatusBadRequest {
		return fallback
	}
	return fiber.StatusInternalServerError
}

func RequestLogLevel(status int) slog.Level {
	switch {
	case status >= fiber.StatusInternalServerError:
		return slog.LevelError
	case status >= fiber.StatusBadRequest:
		return slog.LevelWarn
	default:
		return slog.LevelInfo
	}
}
