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

func pickHeroBackground(backgrounds []models.HeroBackground) models.HeroBackground {
	valid := make([]models.HeroBackground, 0, len(backgrounds))
	for _, background := range backgrounds {
		// only keep ids whose media bytes still exist — a dangling pointer (e.g.
		// media deleted out from under the row) would otherwise render a broken
		// <img>. drop the dead side and fall back to the live one; drop the pair
		// if neither survives.
		desktopOK := mediaExists(background.DesktopMediaID)
		mobileOK := mediaExists(background.MobileMediaID)
		if !desktopOK && !mobileOK {
			continue
		}
		if !desktopOK {
			background.DesktopMediaID = background.MobileMediaID
		}
		if !mobileOK {
			background.MobileMediaID = background.DesktopMediaID
		}
		valid = append(valid, background)
	}
	if len(valid) == 0 {
		return models.HeroBackground{}
	}
	return valid[rand.IntN(len(valid))]
}

// loadHomeContent pulls the homepage's collections. First error wins — the page
// is all-or-nothing, so there is nothing the caller could do with the detail of
// which relation failed beyond what the driver already logs.
func loadHomeContent() (templates.HomeContent, error) {
	var content templates.HomeContent
	var err error
	if content.Places, err = database.GetPlaces(models.PlaceKindStory); err != nil {
		return content, err
	}
	if content.Honeymoon, err = database.GetPlaces(models.PlaceKindHoneymoon); err != nil {
		return content, err
	}
	if content.ParkingSpots, err = database.GetParkingSpots(); err != nil {
		return content, err
	}
	if content.Accommodations, err = database.GetAccommodations(); err != nil {
		return content, err
	}
	backgrounds, err := database.GetHeroBackgrounds()
	if err != nil {
		return content, err
	}
	content.HeroBackground = pickHeroBackground(backgrounds)
	return content, nil
}

func Home(c *fiber.Ctx) error {
	settings, err := database.GetSettings()
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
	content, err := loadHomeContent()
	if err != nil {
		return c.Status(500).SendString("failed to load homepage content")
	}
	lang := getLang(c)
	labelOverrides, err := database.GetHomepageLabels(lang)
	if err != nil {
		return c.Status(500).SendString("failed to load labels")
	}
	// the chat bubble only needs to know a persona exists; loading the profiles
	// here would drag kilobytes of prompt text into every homepage render.
	impersonationCount, err := database.CountImpersonations()
	if err != nil {
		return c.Status(500).SendString("failed to count impersonations")
	}
	bankConfigured := settings.BankAccountIBAN != "" && settings.BankAccountHolder != ""
	chatEnabled := ChatEnabled() && impersonationCount > 0
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
		settings.GroomName+" & "+settings.BrideName,
		strings.Join(ogDescParts, " · "),
		settings,
	)
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
	return Render(c, templates.Home(settings, content, registryItems, claimedAmounts, bankConfigured, chatEnabled, soundtrackEnabled, inviteUpdateURL, rsvpSubmitted, i18n.NewTWithOverrides(lang, labelOverrides), lang, ogMeta))
}
