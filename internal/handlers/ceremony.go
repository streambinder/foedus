package handlers

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/streambinder/foedus/internal/database"
	"github.com/streambinder/foedus/internal/i18n"
	"github.com/streambinder/foedus/templates"
)

func Ceremony(c *fiber.Ctx) error {
	settings, err := database.GetAllSettings()
	if err != nil {
		return c.Status(500).SendString("failed to load settings")
	}
	if !settings.IsConfigured() {
		return Render(c, templates.SetupGuard(getLang(c), getT(c)))
	}
	lang := getLang(c)
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
		baseURL+"/ceremony",
		settings.Spouse1Name+" & "+settings.Spouse2Name,
		strings.Join(ogDescParts, " · "),
		settings,
	)
	return Render(c, templates.Ceremony(settings, i18n.NewT(lang), lang, ogMeta))
}
