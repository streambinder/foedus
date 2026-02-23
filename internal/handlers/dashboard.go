package handlers

import (
	"net/url"
	"strconv"
	"time"

	"github.com/dpucci/foedus/internal/database"
	"github.com/dpucci/foedus/internal/i18n"
	"github.com/dpucci/foedus/templates"
	"github.com/gofiber/fiber/v2"
)

func setFlash(c *fiber.Ctx, msg string) {
	c.Cookie(&fiber.Cookie{
		Name:    "flash",
		Value:   url.QueryEscape(msg),
		Expires: time.Now().Add(5 * time.Second),
		Path:    "/dashboard",
	})
}

func getFlash(c *fiber.Ctx) string {
	raw := c.Cookies("flash")
	if raw == "" {
		return ""
	}
	// expire cookie immediately
	c.Cookie(&fiber.Cookie{
		Name:    "flash",
		Value:   "",
		Expires: time.Now().Add(-1 * time.Hour),
		Path:    "/dashboard",
	})
	msg, _ := url.QueryUnescape(raw)
	return msg
}

func getT(c *fiber.Ctx) i18n.T {
	if t, ok := c.Locals("t").(i18n.T); ok {
		return t
	}
	return i18n.NewT("en")
}

func getLang(c *fiber.Ctx) string {
	if lang, ok := c.Locals("lang").(string); ok {
		return lang
	}
	return "en"
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
	return Render(c, templates.Dashboard(settings, guests, csrfToken, getFlash(c), getT(c), getLang(c)))
}

func SaveSettings(c *fiber.Ctx) error {
	keys := []string{
		"spouse1_name", "spouse2_name", "ceremony_date",
		"church_name", "church_address",
		"party_venue", "party_address",
	}
	for _, key := range keys {
		if err := database.UpdateSetting(key, c.FormValue(key)); err != nil {
			return c.Status(500).SendString("failed to save settings")
		}
	}
	setFlash(c, getT(c)("flash.settings_saved"))
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
	setFlash(c, getT(c)("flash.guest_added"))
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
	return Render(c, templates.EditGuest(guest, csrfToken, getT(c), getLang(c)))
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
	setFlash(c, getT(c)("flash.guest_updated"))
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
	setFlash(c, getT(c)("flash.guest_deleted"))
	return c.Redirect("/dashboard")
}
