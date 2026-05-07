package handlers

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/streambinder/foedus/internal/database"
	"github.com/streambinder/foedus/internal/observability"
)

type claimRequest struct {
	RegistryItemID *int   `json:"registry_item_id"`
	Amount         int    `json:"amount"` // whole currency units (e.g. euros)
	Donor          string `json:"donor"`
}

func ClaimGift(c *fiber.Ctx) error {
	logger := handlerLogger(c)

	settings, err := database.GetAllSettings()
	if err != nil {
		logger.Error("gift claim failed to load settings", "error", err.Error())
		return c.Status(500).JSON(fiber.Map{"error": "internal error"})
	}
	if !settings.IsConfigured() {
		logger.Warn("gift claim rejected", "reason", "site not configured")
		return c.Status(503).JSON(fiber.Map{"error": "not configured"})
	}

	var req claimRequest
	if err := c.BodyParser(&req); err != nil {
		logger.Warn("gift claim invalid request", "error", err.Error())
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if req.Amount <= 0 {
		logger.Warn("gift claim invalid amount", "amount", req.Amount)
		return c.Status(400).JSON(fiber.Map{"error": "amount must be positive"})
	}

	// if claiming a registry item, validate it exists and amount doesn't exceed remaining
	if req.RegistryItemID != nil {
		item, err := database.GetRegistryItem(*req.RegistryItemID)
		if err != nil {
			logger.Warn("gift claim rejected", "reason", "registry item not found", "registry_item_id", *req.RegistryItemID)
			return c.Status(404).JSON(fiber.Map{"error": "item not found"})
		}
		// price=0 means open-ended (free gift style), no cap on amount
		if item.Price > 0 {
			claimed, err := database.GetClaimedAmountsByItem()
			if err != nil {
				logger.Error("gift claim failed to load claimed amounts", "registry_item_id", item.ID, "error", err.Error())
				return c.Status(500).JSON(fiber.Map{"error": "internal error"})
			}
			remaining := item.Price - claimed[item.ID]
			if req.Amount > remaining {
				logger.Warn("gift claim exceeds remaining amount", "registry_item_id", item.ID, "amount", req.Amount, "remaining", remaining)
				return c.Status(400).JSON(fiber.Map{"error": "amount exceeds remaining"})
			}
		}
	}

	if err := database.CreateGift(req.Amount, strings.TrimSpace(req.Donor), req.RegistryItemID); err != nil {
		logger.Error("gift claim save failed", "amount", req.Amount, "donor", observability.Redact(req.Donor), "error", err.Error())
		return c.Status(500).JSON(fiber.Map{"error": "failed to save gift"})
	}
	logger.Info(
		"gift claim recorded",
		"amount", req.Amount,
		"donor", observability.Redact(req.Donor),
		"registry_item_id", req.RegistryItemID,
	)
	return c.JSON(fiber.Map{"ok": true})
}
