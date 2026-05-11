package middleware

import (
	"log/slog"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/basicauth"
	"github.com/streambinder/foedus/internal/observability"
)

func BasicAuth() fiber.Handler {
	users := make(map[string]string)
	// check ADMIN_USER/ADMIN_PASSWORD, then ADMIN_USER1..9/ADMIN_PASSWORD1..9
	for _, suffix := range []string{"", "1", "2", "3", "4", "5", "6", "7", "8", "9"} {
		user := os.Getenv("ADMIN_USER" + suffix)
		pass := os.Getenv("ADMIN_PASSWORD" + suffix)
		if user != "" && pass != "" {
			users[user] = pass
		}
	}
	if len(users) == 0 {
		// refuse to boot rather than expose dashboard with admin/admin
		slog.Error("no admin credentials configured: set ADMIN_USER and ADMIN_PASSWORD (or ADMIN_USER1..9 / ADMIN_PASSWORD1..9)")
		panic("no admin credentials configured")
	}
	slog.Info("basic auth configured", "admin_users", len(users))
	return basicauth.New(basicauth.Config{
		Users: users,
		Unauthorized: func(c *fiber.Ctx) error {
			observability.LoggerFromFiber(c).Warn("dashboard authentication failed")
			c.Set(fiber.HeaderWWWAuthenticate, "basic realm=Restricted")
			return c.SendStatus(fiber.StatusUnauthorized)
		},
	})
}
