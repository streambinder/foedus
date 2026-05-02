package handlers

import (
	"database/sql"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/streambinder/foedus/internal/database"
	"github.com/streambinder/foedus/internal/i18n"
	"github.com/streambinder/foedus/internal/models"
	"github.com/streambinder/foedus/internal/observability"
	"github.com/streambinder/foedus/templates"
)

func ViewInvitation(c *fiber.Ctx) error {
	logger := handlerLogger(c)
	code := c.Params("code")
	inv, err := database.GetInvitationByCode(code)
	if err == sql.ErrNoRows {
		logger.Warn("invitation view not found", "invitation_code", observability.Redact(code))
		return c.Status(404).SendString("invitation not found")
	}
	if err != nil {
		logger.Error("invitation view failed to load invitation", "invitation_code", observability.Redact(code), "error", err.Error())
		return c.Status(500).SendString("failed to load invitation")
	}

	settings, err := database.GetAllSettings()
	if err != nil {
		logger.Error("invitation view failed to load settings", "invitation_code", observability.Redact(code), "error", err.Error())
		return c.Status(500).SendString("failed to load settings")
	}
	if !settings.IsConfigured() {
		logger.Info("invitation view redirected to setup guard", "invitation_code", observability.Redact(code))
		return Render(c, templates.SetupGuard(getLang(c), getT(c)))
	}

	polls, err := database.GetAllPolls()
	if err != nil {
		logger.Error("invitation view failed to load polls", "invitation_code", observability.Redact(code), "error", err.Error())
		return c.Status(500).SendString("failed to load polls")
	}

	noRedirect := c.Query("no_redirect") == "1"
	if templates.InvitationAnswered(inv) && !noRedirect {
		logger.Info("invitation view redirected to homepage", "invitation_code", observability.Redact(code), "guest_count", len(inv.Guests))
		return c.Redirect("/?invite=" + inv.Code)
	}

	lang := getLang(c)
	t := i18n.NewTWithOverrides(lang, settings.HomepageLabels[lang])
	baseURL := c.Protocol() + "://" + c.Hostname()
	var ogDescParts []string
	if settings.CeremonyDatetime != "" {
		ogDescParts = append(ogDescParts, i18n.FormatDatetimeUniversal(settings.CeremonyDatetime))
	}
	if ogLocation := ogCeremonyLocation(settings); ogLocation != "" {
		ogDescParts = append(ogDescParts, ogLocation)
	}
	title := invitationTitle(t, inv, settings.Spouse1Name, settings.Spouse2Name)
	ogMeta := BuildOGMeta(
		baseURL,
		baseURL+"/"+code,
		title,
		strings.Join(ogDescParts, " · "),
		settings,
	)
	logger.Info("invitation rendered",
		"invitation_code", observability.Redact(code),
		"guest_count", len(inv.Guests),
		"poll_count", len(polls),
		"viewed", inv.ViewedAt != nil,
	)
	return Render(c, templates.Invitation(inv, settings, polls, inv.ViewedAt != nil, noRedirect, t, lang, ogMeta, title))
}

func MarkInvitationViewed(c *fiber.Ctx) error {
	logger := handlerLogger(c)
	code := c.Params("code")
	inv, err := database.GetInvitationByCode(code)
	if err != nil {
		logger.Warn("invitation mark viewed failed", "invitation_code", observability.Redact(code), "error", errString(err))
		return c.Status(404).SendString("not found")
	}
	if err := database.MarkInvitationViewed(inv.ID); err != nil {
		logger.Error("invitation mark viewed failed", "invitation_code", observability.Redact(code), "invitation_id", inv.ID, "error", err.Error())
		return c.Status(500).SendString("failed to update invitation")
	}
	logger.Info("invitation marked viewed", "invitation_code", observability.Redact(code), "invitation_id", inv.ID)
	return c.SendStatus(204)
}

func UpdateInvitationRSVP(c *fiber.Ctx) error {
	logger := handlerLogger(c)
	code := c.Params("code")
	inv, err := database.GetInvitationByCode(code)
	if err == sql.ErrNoRows {
		logger.Warn("invitation rsvp not found", "invitation_code", observability.Redact(code))
		return c.Status(404).SendString("invitation not found")
	}
	if err != nil {
		logger.Error("invitation rsvp failed to load invitation", "invitation_code", observability.Redact(code), "error", err.Error())
		return c.Status(500).SendString("failed to load invitation")
	}

	polls, err := database.GetAllPolls()
	if err != nil {
		logger.Error("invitation rsvp failed to load polls", "invitation_code", observability.Redact(code), "error", err.Error())
		return c.Status(500).SendString("failed to load polls")
	}

	// each guest has radio fields ceremony_<id> and reception_<id>: "1"=yes, "0"=no, absent=nil
	answeredGuests := 0
	for _, g := range inv.Guests {
		ceremony := parseRSVPField(c.FormValue("ceremony_" + strconv.Itoa(g.ID)))
		reception := parseRSVPField(c.FormValue("reception_" + strconv.Itoa(g.ID)))
		if err := database.SetGuestRSVP(g.ID, ceremony, reception); err != nil {
			logger.Error("invitation rsvp guest update failed", "invitation_code", observability.Redact(code), "guest_id", g.ID, "error", err.Error())
			return c.Status(500).SendString("failed to update RSVP")
		}
		answeredGuests++

		// parse poll switch fields: poll_{pollID}_{guestID} = "1" if checked
		answers := make(map[int]models.PollAnswer)
		for _, p := range polls {
			fieldSuffix := strconv.Itoa(p.ID) + "_" + strconv.Itoa(g.ID)
			answer := c.FormValue("poll_"+fieldSuffix) == "1"
			notes := ""
			if answer {
				notes = strings.TrimSpace(c.FormValue("poll_notes_" + fieldSuffix))
			}
			answers[p.ID] = models.PollAnswer{PollID: p.ID, Answer: answer, Notes: notes}
		}
		if err := database.SavePollAnswers(g.ID, answers); err != nil {
			logger.Error("invitation poll answers save failed", "invitation_code", observability.Redact(code), "guest_id", g.ID, "answers", len(answers), "error", err.Error())
			return c.Status(500).SendString("failed to save poll answers")
		}
	}

	redirectURL := "/?invite=" + code + "&submitted=1"
	logger.Info("invitation rsvp updated",
		"invitation_code", observability.Redact(code),
		"guest_count", len(inv.Guests),
		"answered_guests", answeredGuests,
		"poll_count", len(polls),
	)
	return c.Redirect(redirectURL)
}

func invitationTitle(t i18n.T, inv models.Invitation, spouse1, spouse2 string) string {
	label := strings.TrimSpace(inv.Label)
	if label == "" {
		label = database.DefaultInvitationLabel(inv.Guests)
	}
	title := t("invitation.title")
	if label != "" {
		title += " " + label
	}
	if spouse1 != "" && spouse2 != "" {
		title += " · " + spouse1 + " & " + spouse2
	}
	return title
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
