package database

import (
	"database/sql"

	"github.com/streambinder/foedus/internal/models"
)

func scanAccommodation(rows *sql.Rows) (models.Accommodation, error) {
	var accommodation models.Accommodation
	err := rows.Scan(&accommodation.ID, &accommodation.Name, &accommodation.Description,
		&accommodation.URL, &accommodation.SortOrder, &accommodation.CreatedAt)
	return accommodation, err
}

func GetAccommodations() ([]models.Accommodation, error) {
	return queryAll(
		`SELECT id, name, description, url, sort_order, created_at FROM accommodations ORDER BY sort_order ASC, id ASC`,
		scanAccommodation,
	)
}

func ReplaceAccommodations(q Querier, accommodations []models.Accommodation) error {
	if _, err := q.Exec(`DELETE FROM accommodations`); err != nil {
		return err
	}
	return insertAll(q, "accommodations", []string{"name", "description", "url", "sort_order"}, accommodations,
		func(accommodation models.Accommodation, index int) []any {
			return []any{accommodation.Name, accommodation.Description, accommodation.URL, index}
		})
}
