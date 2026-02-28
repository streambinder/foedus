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
	registryItems, err := database.GetAllRegistryItems()
	if err != nil {
		return c.Status(500).SendString("failed to load registry items")
	}
	soldItems, err := database.GetSoldRegistryItemIDs()
	if err != nil {
		return c.Status(500).SendString("failed to load sold items")
	}
	return Render(c, templates.Home(settings, registryItems, soldItems, StripeEnabled(), getT(c), getLang(c)))
}
