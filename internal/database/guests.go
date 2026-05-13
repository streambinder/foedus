package database

import (
	"database/sql"
	"fmt"

	"github.com/streambinder/foedus/internal/models"
)

func CreateGuest(firstName, lastName, guestType string) error {
	_, err := DB.Exec(
		`INSERT INTO guests (first_name, last_name, type) VALUES (?, ?, ?)`,
		firstName, lastName, guestType,
	)
	return err
}

func GetAllGuests() ([]models.Guest, error) {
	rows, err := DB.Query(`SELECT id, first_name, last_name, type, confirmed_ceremony, confirmed_reception, invitation_id, created_at, updated_at FROM guests ORDER BY id DESC`)
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
		`SELECT id, first_name, last_name, type, confirmed_ceremony, confirmed_reception, invitation_id, created_at, updated_at FROM guests WHERE id = ?`, id,
	).Scan(&g.ID, &g.FirstName, &g.LastName, &g.Type, &g.ConfirmedCeremony, &g.ConfirmedReception, &g.InvitationID, &g.CreatedAt, &g.UpdatedAt)
	return g, err
}

func UpdateGuest(id int, firstName, lastName, guestType string) error {
	_, err := DB.Exec(
		`UPDATE guests SET first_name = ?, last_name = ?, type = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		firstName, lastName, guestType, id,
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
	// confirmedReception: any guest with confirmed_reception=1 (invitation
	//   not required — orphan confirmations still count toward headcount)
	// refusedReception: any guest with confirmed_reception=0 (same)
	// pendingRSVP: invited but reception not yet actioned (invitation
	//   required — can only be "pending" if you got an invite)
	// invited: linked to any invitation
	// nonVisualizedInvited: invited but invitation never viewed
	err = DB.QueryRow(`
		SELECT
			COALESCE(SUM(CASE WHEN g.confirmed_reception = 1 THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN g.confirmed_reception = 0 THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN g.invitation_id IS NOT NULL AND g.confirmed_reception IS NULL THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN g.invitation_id IS NOT NULL THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN g.invitation_id IS NOT NULL AND i.viewed_at IS NULL THEN 1 ELSE 0 END), 0),
			COUNT(*)
		FROM guests g
		LEFT JOIN invitations i ON i.id = g.invitation_id
	`).Scan(&confirmedReception, &refusedReception, &pendingRSVP, &invited, &nonVisualizedInvited, &total)
	return
}

// CountConfirmedByType breaks down confirmed-reception guests by their type.
// drives the per-type donut so we can see how many of the confirmed headcount
// are paying-adults vs children vs non-counted infants/vendors. counts from
// any guest with confirmed_reception=1 — orphans (no invitation) included
// since the question is "who are we feeding", not "who got invited".
func CountConfirmedByType() (adult, child, infant, vendor int, err error) {
	err = DB.QueryRow(`
		SELECT
			COALESCE(SUM(CASE WHEN type = 'adult'  THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN type = 'child'  THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN type = 'infant' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN type = 'vendor' THEN 1 ELSE 0 END), 0)
		FROM guests
		WHERE confirmed_reception = 1
	`).Scan(&adult, &child, &infant, &vendor)
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
			`SELECT id, first_name, last_name, type, confirmed_ceremony, confirmed_reception, invitation_id, created_at, updated_at FROM guests ORDER BY id DESC LIMIT ? OFFSET ?`,
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
			`SELECT id, first_name, last_name, type, confirmed_ceremony, confirmed_reception, invitation_id, created_at, updated_at FROM guests WHERE first_name LIKE ? OR last_name LIKE ? OR TRIM(COALESCE(first_name, '') || ' ' || COALESCE(last_name, '')) LIKE ? ORDER BY id DESC LIMIT ? OFFSET ?`,
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
	err := rows.Scan(&g.ID, &g.FirstName, &g.LastName, &g.Type, &g.ConfirmedCeremony, &g.ConfirmedReception, &g.InvitationID, &g.CreatedAt, &g.UpdatedAt)
	return g, err
}

// GuestNamesByCounter returns guest "First Last" names matching a counter
// category. Categories mirror the dashboard donut slices in CountConfirmed.
// Empty/unknown category returns ErrUnknownCategory.
func GuestNamesByCounter(category string) ([]string, error) {
	clause, ok := counterWhereClauses[category]
	if !ok {
		return nil, ErrUnknownCategory
	}
	rows, err := DB.Query(`
		SELECT TRIM(COALESCE(g.first_name, '') || ' ' || COALESCE(g.last_name, ''))
		FROM guests g
		LEFT JOIN invitations i ON i.id = g.invitation_id
		WHERE ` + clause + `
		ORDER BY g.first_name, g.last_name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

// counterWhereClauses pins each clickable counter to the predicate that
// produced its number in CountConfirmed — keep these two in sync if the
// definition of a category ever changes.
var counterWhereClauses = map[string]string{
	"confirmed_reception":        "g.confirmed_reception = 1",
	"refused_reception":          "g.confirmed_reception = 0",
	"pending_rsvp":               "g.invitation_id IS NOT NULL AND g.confirmed_reception IS NULL",
	"viewed":                     "g.invitation_id IS NOT NULL AND i.viewed_at IS NOT NULL",
	"nonvisualized":              "g.invitation_id IS NOT NULL AND i.viewed_at IS NULL",
	"invited":                    "g.invitation_id IS NOT NULL",
	"uninvited":                  "g.invitation_id IS NULL",
	"confirmed_reception_adult":  "g.confirmed_reception = 1 AND g.type = 'adult'",
	"confirmed_reception_child":  "g.confirmed_reception = 1 AND g.type = 'child'",
	"confirmed_reception_infant": "g.confirmed_reception = 1 AND g.type = 'infant'",
	"confirmed_reception_vendor": "g.confirmed_reception = 1 AND g.type = 'vendor'",
}

var ErrUnknownCategory = fmt.Errorf("unknown counter category")
