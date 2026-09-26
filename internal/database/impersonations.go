package database

import (
	"database/sql"

	"github.com/streambinder/foedus/internal/models"
)

func scanImpersonation(rows *sql.Rows) (models.Impersonation, error) {
	var impersonation models.Impersonation
	err := rows.Scan(&impersonation.ID, &impersonation.Codename, &impersonation.Profile,
		&impersonation.SortOrder, &impersonation.CreatedAt)
	return impersonation, err
}

func GetImpersonations() ([]models.Impersonation, error) {
	return queryAll(
		`SELECT id, codename, profile, sort_order, created_at FROM impersonations ORDER BY sort_order ASC, id ASC`,
		scanImpersonation,
	)
}

// CountImpersonations answers "is the chat usable" without dragging every
// persona profile — kilobytes of prompt text — into the homepage render.
func CountImpersonations() (int, error) {
	var count int
	err := DB.QueryRow(`SELECT COUNT(*) FROM impersonations`).Scan(&count)
	return count, err
}

func ReplaceImpersonations(q Querier, impersonations []models.Impersonation) error {
	if _, err := q.Exec(`DELETE FROM impersonations`); err != nil {
		return err
	}
	return insertAll(q, "impersonations", []string{"codename", "profile", "sort_order"}, impersonations,
		func(impersonation models.Impersonation, index int) []any {
			return []any{impersonation.Codename, impersonation.Profile, index}
		})
}
