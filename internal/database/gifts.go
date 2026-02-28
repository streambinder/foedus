package database

import "github.com/dpucci/foedus/internal/models"

func CreateGift(amount int, currency, donor, message, sessionID string, registryItemID *int) error {
	_, err := DB.Exec(
		`INSERT OR IGNORE INTO gifts (amount, currency, donor, message, session_id, registry_item_id) VALUES (?, ?, ?, ?, ?, ?)`,
		amount, currency, donor, message, sessionID, registryItemID,
	)
	return err
}

func GetAllGifts() ([]models.Gift, error) {
	rows, err := DB.Query(`SELECT id, amount, currency, donor, message, session_id, registry_item_id, created_at FROM gifts ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var gifts []models.Gift
	for rows.Next() {
		var g models.Gift
		if err := rows.Scan(&g.ID, &g.Amount, &g.Currency, &g.Donor, &g.Message, &g.SessionID, &g.RegistryItemID, &g.CreatedAt); err != nil {
			return nil, err
		}
		gifts = append(gifts, g)
	}
	return gifts, nil
}

// GetSoldRegistryItemIDs returns a set of registry_item IDs that have been purchased.
func GetSoldRegistryItemIDs() (map[int]bool, error) {
	rows, err := DB.Query(`SELECT DISTINCT registry_item_id FROM gifts WHERE registry_item_id IS NOT NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sold := make(map[int]bool)
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		sold[id] = true
	}
	return sold, nil
}
