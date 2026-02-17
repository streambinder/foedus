package middleware

import (
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/basicauth"
)

func BasicAuth() fiber.Handler {
	user := os.Getenv("ADMIN_USER")
	pass := os.Getenv("ADMIN_PASSWORD")
	if user == "" || pass == "" {
		user = "admin"
		pass = "admin"
	}
	return basicauth.New(basicauth.Config{
		Users: map[string]string{user: pass},
	})
}
