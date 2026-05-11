package database

import (
	"database/sql"
	"fmt"

	"github.com/streambinder/foedus/internal/models"
)

func CreateGuest(firstName, lastName string) error {
	_, err := DB.Exec(
		`INSERT INTO guests (first_name, last_name) VALUES (?, ?)`,
		firstName, lastName,
	)
	return err
}

func GetAllGuests() ([]models.Guest, error) {
	rows, err := DB.Query(`SELECT id, first_name, last_name, confirmed_ceremony, confirmed_reception, invitation_id, created_at, updated_at FROM guests ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var guests []models.Guest
	for rows.Next() {
		g, err := scanGuest(rows)
		if err != nil {
			return nil, err
		}
		guests = append(guests, g)
	}
	return guests, nil
}

func GetGuest(id int) (models.Guest, error) {
	var g models.Guest
	err := DB.QueryRow(
		`SELECT id, first_name, last_name, confirmed_ceremony, confirmed_reception, invitation_id, created_at, updated_at FROM guests WHERE id = ?`, id,
	).Scan(&g.ID, &g.FirstName, &g.LastName, &g.ConfirmedCeremony, &g.ConfirmedReception, &g.InvitationID, &g.CreatedAt, &g.UpdatedAt)
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
	// poll_answers.guest_id has a FK to guests(id); with foreign_keys=ON we
	// must purge dependents first or the DELETE fails with constraint error.
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM poll_answers WHERE guest_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM guests WHERE id = ?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

// CycleConfirmed cycles a confirmation field through NULL → 1 → 0 → NULL.
// field must be "ceremony" or "reception".
func CycleConfirmed(id int, field string) error {
	var col string
	switch field {
	case "ceremony":
		col = "confirmed_ceremony"
	case "reception":
		col = "confirmed_reception"
	default:
		return fmt.Errorf("invalid field: %s", field)
	}
	_, err := DB.Exec(
		fmt.Sprintf(`UPDATE guests SET %s = CASE WHEN %s IS NULL THEN 1 WHEN %s = 1 THEN 0 ELSE NULL END, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, col, col, col),
		id,
	)
	return err
}

func CountConfirmed() (confirmedReception, refusedReception, pendingRSVP, invited, nonVisualizedInvited, total int, err error) {
	// reception-only metrics (we don't pay for ceremony-only guests)
	// confirmedReception: said yes to reception
	// refusedReception: said no to reception
	// pendingRSVP: invited but reception not yet actioned
	// invited: linked to any invitation
	// nonVisualizedInvited: invited but invitation never viewed
	err = DB.QueryRow(`
		SELECT
			COALESCE(SUM(CASE WHEN g.invitation_id IS NOT NULL AND g.confirmed_reception = 1 THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN g.invitation_id IS NOT NULL AND g.confirmed_reception = 0 THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN g.invitation_id IS NOT NULL AND g.confirmed_reception IS NULL THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN g.invitation_id IS NOT NULL THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN g.invitation_id IS NOT NULL AND i.viewed_at IS NULL THEN 1 ELSE 0 END), 0),
			COUNT(*)
		FROM guests g
		LEFT JOIN invitations i ON i.id = g.invitation_id
	`).Scan(&confirmedReception, &refusedReception, &pendingRSVP, &invited, &nonVisualizedInvited, &total)
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
			`SELECT id, first_name, last_name, confirmed_ceremony, confirmed_reception, invitation_id, created_at, updated_at FROM guests ORDER BY id DESC LIMIT ? OFFSET ?`,
			perPage, offset,
		)
	} else {
		pattern := "%" + search + "%"
		err = DB.QueryRow(
			`SELECT COUNT(*) FROM guests WHERE first_name LIKE ? OR last_name LIKE ? OR TRIM(COALESCE(first_name, '') || ' ' || COALESCE(last_name, '')) LIKE ?`,
			pattern, pattern, pattern,
		).Scan(&total)
		if err != nil {
			return nil, 0, err
		}
		rows, err = DB.Query(
			`SELECT id, first_name, last_name, confirmed_ceremony, confirmed_reception, invitation_id, created_at, updated_at FROM guests WHERE first_name LIKE ? OR last_name LIKE ? OR TRIM(COALESCE(first_name, '') || ' ' || COALESCE(last_name, '')) LIKE ? ORDER BY id DESC LIMIT ? OFFSET ?`,
			pattern, pattern, pattern, perPage, offset,
		)
	}
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var guests []models.Guest
	for rows.Next() {
		g, err := scanGuest(rows)
		if err != nil {
			return nil, 0, err
		}
		guests = append(guests, g)
	}
	return guests, total, nil
}

// scanGuest scans a guest row from the standard column set
func scanGuest(rows *sql.Rows) (models.Guest, error) {
	var g models.Guest
	err := rows.Scan(&g.ID, &g.FirstName, &g.LastName, &g.ConfirmedCeremony, &g.ConfirmedReception, &g.InvitationID, &g.CreatedAt, &g.UpdatedAt)
	return g, err
}
