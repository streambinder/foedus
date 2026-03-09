package handlers

import (
	"database/sql"
	"strconv"

	"github.com/streambinder/foedus/internal/database"
	"github.com/streambinder/foedus/templates"
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

	// each guest has radio fields ceremony_<id> and reception_<id>: "1"=yes, "0"=no, absent=nil
	for _, g := range inv.Guests {
		ceremony := parseRSVPField(c.FormValue("ceremony_" + strconv.Itoa(g.ID)))
		reception := parseRSVPField(c.FormValue("reception_" + strconv.Itoa(g.ID)))
		if err := database.SetGuestRSVP(g.ID, ceremony, reception); err != nil {
			return c.Status(500).SendString("failed to update RSVP")
		}
	}

	return c.Redirect("/" + code)
}

func parseRSVPField(val string) *bool {
	switch val {
	case "1":
		t := true
		return &t
	case "0":
		f := false
		return &f
	default:
		return nil
	}
}
