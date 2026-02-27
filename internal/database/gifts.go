package database

import "github.com/dpucci/foedus/internal/models"

func CreateGift(amount int, currency, donor, message, sessionID string) error {
	_, err := DB.Exec(
		`INSERT OR IGNORE INTO gifts (amount, currency, donor, message, session_id) VALUES (?, ?, ?, ?, ?)`,
		amount, currency, donor, message, sessionID,
	)
	return err
}

func GetAllGifts() ([]models.Gift, error) {
	rows, err := DB.Query(`SELECT id, amount, currency, donor, message, session_id, created_at FROM gifts ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var gifts []models.Gift
	for rows.Next() {
		var g models.Gift
		if err := rows.Scan(&g.ID, &g.Amount, &g.Currency, &g.Donor, &g.Message, &g.SessionID, &g.CreatedAt); err != nil {
			return nil, err
		}
		gifts = append(gifts, g)
	}
	return gifts, nil
}
