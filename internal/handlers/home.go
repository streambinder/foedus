package handlers

import (
	"strings"

	"github.com/streambinder/foedus/internal/database"
	"github.com/streambinder/foedus/internal/i18n"
	"github.com/streambinder/foedus/templates"
	"github.com/gofiber/fiber/v2"
)

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
	ogMeta := templates.OGMeta{
		Title:       settings.Spouse1Name + " & " + settings.Spouse2Name,
		Description: strings.Join(ogDescParts, " · "),
		URL:         baseURL + "/",
		ImageURL:    baseURL + "/og-image",
	}
	return Render(c, templates.Home(settings, registryItems, claimedAmounts, bankConfigured, chatEnabled, i18n.NewTWithOverrides(lang, settings.HomepageLabels[lang]), lang, ogMeta))
}
