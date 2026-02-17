package handlers

import (
	"strconv"
	"time"

	"github.com/dpucci/foedus/internal/database"
	"github.com/dpucci/foedus/templates"
	"github.com/gofiber/fiber/v2"
)

func setFlash(c *fiber.Ctx, msg string) {
	c.Cookie(&fiber.Cookie{
		Name:    "flash",
		Value:   msg,
		Expires: time.Now().Add(5 * time.Second),
		Path:    "/dashboard",
	})
}

func getFlash(c *fiber.Ctx) string {
	msg := c.Cookies("flash")
	if msg != "" {
		// expire cookie immediately
		c.Cookie(&fiber.Cookie{
			Name:    "flash",
			Value:   "",
			Expires: time.Now().Add(-1 * time.Hour),
			Path:    "/dashboard",
		})
	}
	return msg
}

func DashboardIndex(c *fiber.Ctx) error {
	settings, err := database.GetAllSettings()
	if err != nil {
		return c.Status(500).SendString("failed to load settings")
	}
	guests, err := database.GetAllGuests()
	if err != nil {
		return c.Status(500).SendString("failed to load guests")
	}
	csrfToken, _ := c.Locals("csrf").(string)
	return Render(c, templates.Dashboard(settings, guests, csrfToken, getFlash(c)))
}

func SaveSettings(c *fiber.Ctx) error {
	keys := []string{
		"spouse1_name", "spouse2_name", "ceremony_date",
		"church_name", "church_address",
		"party_venue", "party_address", "details",
	}
	for _, key := range keys {
		if err := database.UpdateSetting(key, c.FormValue(key)); err != nil {
			return c.Status(500).SendString("failed to save settings")
		}
	}
	setFlash(c, "Settings saved.")
	return c.Redirect("/dashboard")
}

func AddGuest(c *fiber.Ctx) error {
	if err := database.CreateGuest(
		c.FormValue("name"),
		c.FormValue("email"),
		c.FormValue("plus_one") == "1",
		c.FormValue("dietary_notes"),
		c.FormValue("notes"),
	); err != nil {
		return c.Status(500).SendString("failed to add guest")
	}
	setFlash(c, "Guest added.")
	return c.Redirect("/dashboard")
}

func EditGuestPage(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).SendString("invalid id")
	}
	guest, err := database.GetGuest(id)
	if err != nil {
		return c.Status(404).SendString("guest not found")
	}
	csrfToken, _ := c.Locals("csrf").(string)
	return Render(c, templates.EditGuest(guest, csrfToken))
}

func UpdateGuest(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).SendString("invalid id")
	}
	if err := database.UpdateGuest(
		id,
		c.FormValue("name"),
		c.FormValue("email"),
		c.FormValue("plus_one") == "1",
		c.FormValue("dietary_notes"),
		c.FormValue("notes"),
	); err != nil {
		return c.Status(500).SendString("failed to update guest")
	}
	setFlash(c, "Guest updated.")
	return c.Redirect("/dashboard")
}

func DeleteGuest(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).SendString("invalid id")
	}
	if err := database.DeleteGuest(id); err != nil {
		return c.Status(500).SendString("failed to delete guest")
	}
	setFlash(c, "Guest deleted.")
	return c.Redirect("/dashboard")
}
