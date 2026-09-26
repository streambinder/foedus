package database

import (
	"database/sql"

	"github.com/streambinder/foedus/internal/models"
)

func scanParkingSpot(rows *sql.Rows) (models.ParkingSpot, error) {
	var spot models.ParkingSpot
	err := rows.Scan(&spot.ID, &spot.Lat, &spot.Lng, &spot.SortOrder, &spot.CreatedAt)
	return spot, err
}

func GetParkingSpots() ([]models.ParkingSpot, error) {
	return queryAll(
		`SELECT id, lat, lng, sort_order, created_at FROM parking_spots ORDER BY sort_order ASC, id ASC`,
		scanParkingSpot,
	)
}

func ReplaceParkingSpots(q Querier, spots []models.ParkingSpot) error {
	if _, err := q.Exec(`DELETE FROM parking_spots`); err != nil {
		return err
	}
	return insertAll(q, "parking_spots", []string{"lat", "lng", "sort_order"}, spots,
		func(spot models.ParkingSpot, index int) []any {
			return []any{spot.Lat, spot.Lng, index}
		})
}
