package handlers

import (
	"strings"

	"github.com/streambinder/foedus/internal/database"
	"github.com/gofiber/fiber/v2"
)

type claimRequest struct {
	RegistryItemID *int   `json:"registry_item_id"`
	Amount         int    `json:"amount"` // cents
	Donor          string `json:"donor"`
}

func ClaimGift(c *fiber.Ctx) error {
	settings, err := database.GetAllSettings()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "internal error"})
	}
	if !settings.IsConfigured() {
		return c.Status(503).JSON(fiber.Map{"error": "not configured"})
	}

	var req claimRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if req.Amount <= 0 {
		return c.Status(400).JSON(fiber.Map{"error": "amount must be positive"})
	}

	// if claiming a registry item, validate it exists and amount doesn't exceed remaining
	if req.RegistryItemID != nil {
		item, err := database.GetRegistryItem(*req.RegistryItemID)
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "item not found"})
		}
		claimed, err := database.GetClaimedAmountsByItem()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "internal error"})
		}
		remaining := item.Price*100 - claimed[item.ID]
		if req.Amount > remaining {
			return c.Status(400).JSON(fiber.Map{"error": "amount exceeds remaining"})
		}
	}

	if err := database.CreateGift(req.Amount, strings.TrimSpace(req.Donor), req.RegistryItemID); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to save gift"})
	}
	return c.JSON(fiber.Map{"ok": true})
}
