package middleware

import (
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/basicauth"
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
		users["admin"] = "admin"
	}
	return basicauth.New(basicauth.Config{
		Users: users,
	})
}
