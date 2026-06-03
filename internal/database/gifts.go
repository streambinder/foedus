package database

import "github.com/streambinder/foedus/internal/models"

func CreateGift(amount int, donor string, registryItemID *int) error {
	_, err := DB.Exec(
		`INSERT INTO gifts (amount, donor, registry_item_id, confirmed) VALUES (?, ?, ?, 0)`,
		amount, donor, registryItemID,
	)
	return err
}

func GetAllGifts() ([]models.Gift, error) {
	rows, err := DB.Query(`SELECT id, amount, donor, registry_item_id, confirmed, created_at FROM gifts ORDER BY confirmed ASC, created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var gifts []models.Gift
	for rows.Next() {
		var g models.Gift
		if err := rows.Scan(&g.ID, &g.Amount, &g.Donor, &g.RegistryItemID, &g.Confirmed, &g.CreatedAt); err != nil {
			return nil, err
		}
		gifts = append(gifts, g)
	}
	return gifts, nil
}

func GetGift(id int) (models.Gift, error) {
	var g models.Gift
	err := DB.QueryRow(
		`SELECT id, amount, donor, registry_item_id, confirmed, created_at FROM gifts WHERE id = ?`,
		id,
	).Scan(&g.ID, &g.Amount, &g.Donor, &g.RegistryItemID, &g.Confirmed, &g.CreatedAt)
	return g, err
}

func UpdateGift(id, amount int, donor string, registryItemID *int, confirmed bool) error {
	_, err := DB.Exec(
		`UPDATE gifts SET amount = ?, donor = ?, registry_item_id = ?, confirmed = ? WHERE id = ?`,
		amount, donor, registryItemID, confirmed, id,
	)
	return err
}

func DeleteGift(id int) error {
	_, err := DB.Exec(`DELETE FROM gifts WHERE id = ?`, id)
	return err
}

// GetClaimedAmountsByItem returns a map of registry_item_id -> total claimed amount.
func GetClaimedAmountsByItem() (map[int]int, error) {
	rows, err := DB.Query(`SELECT registry_item_id, SUM(amount) FROM gifts WHERE registry_item_id IS NOT NULL GROUP BY registry_item_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	claimed := make(map[int]int)
	for rows.Next() {
		var id, total int
		if err := rows.Scan(&id, &total); err != nil {
			return nil, err
		}
		claimed[id] = total
	}
	return claimed, nil
}
