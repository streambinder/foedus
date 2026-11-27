package database

import (
	"database/sql"

	"github.com/streambinder/foedus/internal/models"
)

func scanCustomSection(rows *sql.Rows) (models.CustomSection, error) {
	var section models.CustomSection
	var desktopMediaID, mobileMediaID, bodyMediaID sql.NullInt64
	err := rows.Scan(&section.ID, &section.Position, &section.HeaderIt, &section.HeaderEn,
		&section.SubtitleIt, &section.SubtitleEn, &section.BodyIt, &section.BodyEn,
		&desktopMediaID, &mobileMediaID, &bodyMediaID, &section.SortOrder, &section.CreatedAt)
	section.DesktopMediaID = idOrZero(desktopMediaID)
	section.MobileMediaID = idOrZero(mobileMediaID)
	section.BodyMediaID = idOrZero(bodyMediaID)
	return section, err
}

func GetCustomSections() ([]models.CustomSection, error) {
	return queryAll(
		`SELECT id, position, header_it, header_en, subtitle_it, subtitle_en,
		 body_it, body_en, desktop_media_id, mobile_media_id, body_media_id, sort_order, created_at
		 FROM custom_sections ORDER BY sort_order ASC, id ASC`,
		scanCustomSection,
	)
}

func ReplaceCustomSections(q Querier, sections []models.CustomSection) error {
	if _, err := q.Exec(`DELETE FROM custom_sections`); err != nil {
		return err
	}
	return insertAll(q, "custom_sections", []string{"position", "header_it", "header_en", "subtitle_it", "subtitle_en", "body_it", "body_en", "desktop_media_id", "mobile_media_id", "body_media_id", "sort_order"}, sections,
		func(section models.CustomSection, index int) []any {
			return []any{
				section.Position, section.HeaderIt, section.HeaderEn,
				section.SubtitleIt, section.SubtitleEn, section.BodyIt, section.BodyEn,
				nullableID(section.DesktopMediaID), nullableID(section.MobileMediaID), nullableID(section.BodyMediaID), index,
			}
		})
}
