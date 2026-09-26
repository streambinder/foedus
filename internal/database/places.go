package database

import (
	"database/sql"

	"github.com/streambinder/foedus/internal/models"
)

var placeColumns = []string{"kind", "label", "name", "address", "date", "lat", "lng", "media_id", "sort_order"}

func scanPlace(rows *sql.Rows) (models.Place, error) {
	var place models.Place
	var mediaID sql.NullInt64
	err := rows.Scan(&place.ID, &place.Kind, &place.Label, &place.Name, &place.Address,
		&place.Date, &place.Lat, &place.Lng, &mediaID, &place.SortOrder, &place.CreatedAt)
	place.MediaID = idOrZero(mediaID)
	return place, err
}

// GetPlaces returns one kind of place (models.PlaceKindStory or
// models.PlaceKindHoneymoon) in dashboard order.
func GetPlaces(kind string) ([]models.Place, error) {
	return queryAll(
		`SELECT id, kind, label, name, address, date, lat, lng, media_id, sort_order, created_at
		 FROM places WHERE kind = ? ORDER BY sort_order ASC, id ASC`,
		scanPlace, kind,
	)
}

func ReplacePlaces(q Querier, kind string, places []models.Place) error {
	if _, err := q.Exec(`DELETE FROM places WHERE kind = ?`, kind); err != nil {
		return err
	}
	return insertAll(q, "places", placeColumns, places, func(place models.Place, index int) []any {
		return []any{
			kind, place.Label, place.Name, place.Address, place.Date,
			place.Lat, place.Lng, nullableID(place.MediaID), index,
		}
	})
}
