package handlers

import (
	"database/sql"
	"math/rand/v2"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/streambinder/foedus/internal/database"
	"github.com/streambinder/foedus/internal/i18n"
	"github.com/streambinder/foedus/internal/models"
	"github.com/streambinder/foedus/templates"
)

// mediaExists reports whether a media row is actually present. id<=0 is "absent"
// so it doubles as the zero-id check.
func mediaExists(id int) bool {
	if id <= 0 {
		return false
	}
	_, _, err := database.GetMediaMeta(id)
	return err == nil
}

func pickHomepageHeroBackground(backgrounds []models.HomepageHeroBackground) models.HomepageHeroBackground {
	if len(backgrounds) == 0 {
		return models.HomepageHeroBackground{}
	}

	valid := make([]models.HomepageHeroBackground, 0, len(backgrounds))
	for _, bg := range backgrounds {
		// only keep ids whose media bytes still exist — a dangling pointer (e.g.
		// media deleted out from under the setting) would otherwise render a broken
		// <img>. drop the dead side and fall back to the live one; drop the pair if
		// neither survives.
		desktopOK := mediaExists(bg.DesktopMediaID)
		mobileOK := mediaExists(bg.MobileMediaID)
		if !desktopOK && !mobileOK {
			continue
		}
		if !desktopOK {
			bg.DesktopMediaID = bg.MobileMediaID
		}
		if !mobileOK {
			bg.MobileMediaID = bg.DesktopMediaID
		}
		valid = append(valid, bg)
	}
	if len(valid) == 0 {
		return models.HomepageHeroBackground{}
	}
	return valid[rand.IntN(len(valid))]
}

func Home(c *fiber.Ctx) error {
	settings, err := database.GetAllSettings()
	if err != nil {
		return c.Status(500).SendString("failed to load settings")
	}
	if !settings.IsConfigured() {
		return Render(c, templates.SetupGuard(getLang(c), getT(c)))
	}
	registryItems, err := database.GetAllRegistryItems()
	if err != nil {
		return c.Status(500).SendString("failed to load registry items")
	}
	claimedAmounts, err := database.GetClaimedAmountsByItem()
	if err != nil {
		return c.Status(500).SendString("failed to load claimed amounts")
	}
	lang := getLang(c)
	bankConfigured := settings.BankAccountIBAN != "" && settings.BankAccountHolder != ""
	chatEnabled := ChatEnabled() && len(settings.Impersonations) > 0
	soundtrackEnabled := SoundtrackEnabled()
	baseURL := c.Protocol() + "://" + c.Hostname()
	var ogDescParts []string
	if settings.CeremonyDatetime != "" {
		ogDescParts = append(ogDescParts, i18n.FormatDatetimeUniversal(settings.CeremonyDatetime))
	}
	if ogLocation := ogCeremonyLocation(settings); ogLocation != "" {
		ogDescParts = append(ogDescParts, ogLocation)
	}
	ogMeta := BuildOGMeta(
		baseURL,
		baseURL+"/",
		settings.Spouse1Name+" & "+settings.Spouse2Name,
		strings.Join(ogDescParts, " · "),
		settings,
	)
	heroBackground := pickHomepageHeroBackground(settings.HomepageHeroBackgrounds)
	inviteUpdateURL := ""
	rsvpSubmitted := false
	if inviteCode := strings.TrimSpace(c.Query("invite")); inviteCode != "" {
		if inv, err := database.GetInvitationByCode(inviteCode); err == nil {
			inviteUpdateURL = "/" + inv.Code + "?no_redirect=1"
			rsvpSubmitted = c.Query("submitted") == "1"
		} else if err != sql.ErrNoRows {
			return c.Status(500).SendString("failed to load invitation")
		}
	}
	return Render(c, templates.Home(settings, heroBackground, registryItems, claimedAmounts, bankConfigured, chatEnabled, soundtrackEnabled, inviteUpdateURL, rsvpSubmitted, i18n.NewTWithOverrides(lang, settings.HomepageLabels[lang]), lang, ogMeta))
}
