package handlers

import (
	"database/sql"
	"strconv"

	"github.com/dpucci/foedus/internal/database"
	"github.com/dpucci/foedus/templates"
	"github.com/gofiber/fiber/v2"
)

func ViewInvitation(c *fiber.Ctx) error {
	code := c.Params("code")
	inv, err := database.GetInvitationByCode(code)
	if err == sql.ErrNoRows {
		return c.Status(404).SendString("invitation not found")
	}
	if err != nil {
		return c.Status(500).SendString("failed to load invitation")
	}

	// mark as viewed on first open
	database.MarkInvitationViewed(inv.ID)

	settings, err := database.GetAllSettings()
	if err != nil {
		return c.Status(500).SendString("failed to load settings")
	}

	return Render(c, templates.Invitation(inv, settings, getT(c), getLang(c)))
}

func UpdateInvitationRSVP(c *fiber.Ctx) error {
	code := c.Params("code")
	inv, err := database.GetInvitationByCode(code)
	if err == sql.ErrNoRows {
		return c.Status(404).SendString("invitation not found")
	}
	if err != nil {
		return c.Status(500).SendString("failed to load invitation")
	}

	// each guest has a form field "confirmed_<id>" that's "1" if checked, absent if not
	for _, g := range inv.Guests {
		confirmed := c.FormValue("confirmed_"+strconv.Itoa(g.ID)) == "1"
		if err := database.SetGuestConfirmed(g.ID, confirmed); err != nil {
			return c.Status(500).SendString("failed to update RSVP")
		}
	}

	return c.Redirect("/" + code)
}
