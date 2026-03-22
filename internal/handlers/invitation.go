package handlers

import (
	"database/sql"
	"strconv"

	"github.com/streambinder/foedus/internal/database"
	"github.com/streambinder/foedus/internal/i18n"
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

	settings, err := database.GetAllSettings()
	if err != nil {
		return c.Status(500).SendString("failed to load settings")
	}
	if !settings.IsConfigured() {
		return Render(c, templates.SetupGuard(getLang(c), getT(c)))
	}

	polls, err := database.GetAllPolls()
	if err != nil {
		return c.Status(500).SendString("failed to load polls")
	}

	lang := getLang(c)
	return Render(c, templates.Invitation(inv, settings, polls, inv.ViewedAt != nil, i18n.NewTWithOverrides(lang, settings.HomepageLabels[lang]), lang))
}

func MarkInvitationViewed(c *fiber.Ctx) error {
	inv, err := database.GetInvitationByCode(c.Params("code"))
	if err != nil {
		return c.Status(404).SendString("not found")
	}
	database.MarkInvitationViewed(inv.ID)
	return c.SendStatus(204)
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

	polls, err := database.GetAllPolls()
	if err != nil {
		return c.Status(500).SendString("failed to load polls")
	}

	// each guest has radio fields ceremony_<id> and reception_<id>: "1"=yes, "0"=no, absent=nil
	for _, g := range inv.Guests {
		ceremony := parseRSVPField(c.FormValue("ceremony_" + strconv.Itoa(g.ID)))
		reception := parseRSVPField(c.FormValue("reception_" + strconv.Itoa(g.ID)))
		if err := database.SetGuestRSVP(g.ID, ceremony, reception); err != nil {
			return c.Status(500).SendString("failed to update RSVP")
		}

		// parse poll checkbox fields: poll_{pollID}_{guestID} = "1" if checked
		answers := make(map[int]bool)
		for _, p := range polls {
			answers[p.ID] = c.FormValue("poll_"+strconv.Itoa(p.ID)+"_"+strconv.Itoa(g.ID)) == "1"
		}
		if err := database.SavePollAnswers(g.ID, answers); err != nil {
			return c.Status(500).SendString("failed to save poll answers")
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
