package database

import (
	"database/sql"

	"github.com/streambinder/foedus/internal/models"
)

func scanHeroBackground(rows *sql.Rows) (models.HeroBackground, error) {
	var background models.HeroBackground
	var desktopMediaID, mobileMediaID sql.NullInt64
	err := rows.Scan(&background.ID, &desktopMediaID, &mobileMediaID, &background.SortOrder, &background.CreatedAt)
	background.DesktopMediaID = idOrZero(desktopMediaID)
	background.MobileMediaID = idOrZero(mobileMediaID)
	return background, err
}

func GetHeroBackgrounds() ([]models.HeroBackground, error) {
	return queryAll(
		`SELECT id, desktop_media_id, mobile_media_id, sort_order, created_at
		 FROM hero_backgrounds ORDER BY sort_order ASC, id ASC`,
		scanHeroBackground,
	)
}

func ReplaceHeroBackgrounds(q Querier, backgrounds []models.HeroBackground) error {
	if _, err := q.Exec(`DELETE FROM hero_backgrounds`); err != nil {
		return err
	}
	return insertAll(q, "hero_backgrounds", []string{"desktop_media_id", "mobile_media_id", "sort_order"}, backgrounds,
		func(background models.HeroBackground, index int) []any {
			return []any{nullableID(background.DesktopMediaID), nullableID(background.MobileMediaID), index}
		})
}
