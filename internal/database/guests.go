package database

import "github.com/dpucci/foedus/internal/models"

func CreateGuest(name, email string, plusOne bool, dietaryNotes, notes string) error {
	po := 0
	if plusOne {
		po = 1
	}
	_, err := DB.Exec(
		`INSERT INTO guests (name, email, plus_one, dietary_notes, notes) VALUES (?, ?, ?, ?, ?)`,
		name, email, po, dietaryNotes, notes,
	)
	return err
}

func GetAllGuests() ([]models.Guest, error) {
	rows, err := DB.Query(`SELECT id, name, email, plus_one, dietary_notes, notes, created_at, updated_at FROM guests ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var guests []models.Guest
	for rows.Next() {
		var g models.Guest
		var po int
		if err := rows.Scan(&g.ID, &g.Name, &g.Email, &po, &g.DietaryNotes, &g.Notes, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, err
		}
		g.PlusOne = po == 1
		guests = append(guests, g)
	}
	return guests, nil
}

func GetGuest(id int) (models.Guest, error) {
	var g models.Guest
	var po int
	err := DB.QueryRow(
		`SELECT id, name, email, plus_one, dietary_notes, notes, created_at, updated_at FROM guests WHERE id = ?`, id,
	).Scan(&g.ID, &g.Name, &g.Email, &po, &g.DietaryNotes, &g.Notes, &g.CreatedAt, &g.UpdatedAt)
	g.PlusOne = po == 1
	return g, err
}

func UpdateGuest(id int, name, email string, plusOne bool, dietaryNotes, notes string) error {
	po := 0
	if plusOne {
		po = 1
	}
	_, err := DB.Exec(
		`UPDATE guests SET name = ?, email = ?, plus_one = ?, dietary_notes = ?, notes = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		name, email, po, dietaryNotes, notes, id,
	)
	return err
}

func DeleteGuest(id int) error {
	_, err := DB.Exec(`DELETE FROM guests WHERE id = ?`, id)
	return err
}
