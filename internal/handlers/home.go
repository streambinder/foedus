package handlers

import (
	"github.com/dpucci/foedus/internal/database"
	"github.com/dpucci/foedus/templates"
	"github.com/gofiber/fiber/v2"
)

func Home(c *fiber.Ctx) error {
	settings, err := database.GetAllSettings()
	if err != nil {
		return c.Status(500).SendString("failed to load settings")
	}
	guests, err := database.GetAllGuests()
	if err != nil {
		return c.Status(500).SendString("failed to load guests")
	}
	return Render(c, templates.Home(settings, guests))
}
