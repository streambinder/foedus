package database

import (
	"database/sql"

	"github.com/dpucci/foedus/internal/models"
)

func CreateGuest(firstName, lastName string) error {
	_, err := DB.Exec(
		`INSERT INTO guests (first_name, last_name) VALUES (?, ?)`,
		firstName, lastName,
	)
	return err
}

func GetAllGuests() ([]models.Guest, error) {
	rows, err := DB.Query(`SELECT id, first_name, last_name, confirmed, created_at, updated_at FROM guests ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var guests []models.Guest
	for rows.Next() {
		var g models.Guest
		if err := rows.Scan(&g.ID, &g.FirstName, &g.LastName, &g.Confirmed, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, err
		}
		guests = append(guests, g)
	}
	return guests, nil
}

func GetGuest(id int) (models.Guest, error) {
	var g models.Guest
	err := DB.QueryRow(
		`SELECT id, first_name, last_name, confirmed, created_at, updated_at FROM guests WHERE id = ?`, id,
	).Scan(&g.ID, &g.FirstName, &g.LastName, &g.Confirmed, &g.CreatedAt, &g.UpdatedAt)
	return g, err
}

func UpdateGuest(id int, firstName, lastName string) error {
	_, err := DB.Exec(
		`UPDATE guests SET first_name = ?, last_name = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		firstName, lastName, id,
	)
	return err
}

func DeleteGuest(id int) error {
	_, err := DB.Exec(`DELETE FROM guests WHERE id = ?`, id)
	return err
}

func ToggleConfirmed(id int) error {
	_, err := DB.Exec(
		`UPDATE guests SET confirmed = NOT confirmed, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, id,
	)
	return err
}

func CountConfirmed() (confirmed int, total int, err error) {
	err = DB.QueryRow(`SELECT COALESCE(SUM(confirmed), 0), COUNT(*) FROM guests`).Scan(&confirmed, &total)
	return
}

func GetGuestsPaginated(page, perPage int, search string) ([]models.Guest, int, error) {
	offset := (page - 1) * perPage
	var rows *sql.Rows
	var err error
	var total int

	if search == "" {
		err = DB.QueryRow(`SELECT COUNT(*) FROM guests`).Scan(&total)
		if err != nil {
			return nil, 0, err
		}
		rows, err = DB.Query(
			`SELECT id, first_name, last_name, confirmed, created_at, updated_at FROM guests ORDER BY id DESC LIMIT ? OFFSET ?`,
			perPage, offset,
		)
	} else {
		pattern := "%" + search + "%"
		err = DB.QueryRow(
			`SELECT COUNT(*) FROM guests WHERE first_name LIKE ? OR last_name LIKE ?`,
			pattern, pattern,
		).Scan(&total)
		if err != nil {
			return nil, 0, err
		}
		rows, err = DB.Query(
			`SELECT id, first_name, last_name, confirmed, created_at, updated_at FROM guests WHERE first_name LIKE ? OR last_name LIKE ? ORDER BY id DESC LIMIT ? OFFSET ?`,
			pattern, pattern, perPage, offset,
		)
	}
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var guests []models.Guest
	for rows.Next() {
		var g models.Guest
		if err := rows.Scan(&g.ID, &g.FirstName, &g.LastName, &g.Confirmed, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, 0, err
		}
		guests = append(guests, g)
	}
	return guests, total, nil
}
