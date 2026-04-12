package handlers

import (
	"math/rand/v2"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/streambinder/foedus/internal/database"
	"github.com/streambinder/foedus/internal/i18n"
	"github.com/streambinder/foedus/internal/models"
	"github.com/streambinder/foedus/templates"
)

func pickHomepageHeroBackground(backgrounds []models.HomepageHeroBackground) models.HomepageHeroBackground {
	if len(backgrounds) == 0 {
		return models.HomepageHeroBackground{}
	}

	valid := make([]models.HomepageHeroBackground, 0, len(backgrounds))
	for _, bg := range backgrounds {
		if bg.DesktopImage == "" && bg.MobileImage == "" {
			continue
		}
		if bg.DesktopImage == "" {
			bg.DesktopImage = bg.MobileImage
		}
		if bg.MobileImage == "" {
			bg.MobileImage = bg.DesktopImage
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
	baseURL := c.Protocol() + "://" + c.Hostname()
	var ogDescParts []string
	if settings.CeremonyDatetime != "" {
		ogDescParts = append(ogDescParts, i18n.FormatDatetime(settings.CeremonyDatetime, lang))
	}
	if settings.CeremonyLocation != "" {
		ogDescParts = append(ogDescParts, settings.CeremonyLocation)
	}
	ogMeta := BuildOGMeta(
		baseURL,
		baseURL+"/",
		settings.Spouse1Name+" & "+settings.Spouse2Name,
		strings.Join(ogDescParts, " · "),
		settings,
	)
	heroBackground := pickHomepageHeroBackground(settings.HomepageHeroBackgrounds)
	return Render(c, templates.Home(settings, heroBackground, registryItems, claimedAmounts, bankConfigured, chatEnabled, i18n.NewTWithOverrides(lang, settings.HomepageLabels[lang]), lang, ogMeta))
}
