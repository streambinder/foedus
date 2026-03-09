package database

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"

	"github.com/streambinder/foedus/internal/models"
)

const base62Chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func GenerateCode() string {
	b := make([]byte, 8)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(base62Chars))))
		b[i] = base62Chars[n.Int64()]
	}
	return string(b)
}

func CreateInvitation(guestIDs []int) (string, error) {
	code := GenerateCode()

	tx, err := DB.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	result, err := tx.Exec(`INSERT INTO invitations (code) VALUES (?)`, code)
	if err != nil {
		return "", err
	}
	invitationID, err := result.LastInsertId()
	if err != nil {
		return "", err
	}

	// build a single UPDATE with IN clause
	placeholders := make([]string, len(guestIDs))
	args := make([]any, 0, len(guestIDs)+1)
	args = append(args, invitationID)
	for i, id := range guestIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}
	_, err = tx.Exec(
		fmt.Sprintf(`UPDATE guests SET invitation_id = ? WHERE id IN (%s)`, strings.Join(placeholders, ",")),
		args...,
	)
	if err != nil {
		return "", err
	}

	return code, tx.Commit()
}

func GetAllInvitations() ([]models.Invitation, error) {
	rows, err := DB.Query(`SELECT id, code, viewed_at, created_at FROM invitations ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invitations []models.Invitation
	for rows.Next() {
		var inv models.Invitation
		if err := rows.Scan(&inv.ID, &inv.Code, &inv.ViewedAt, &inv.CreatedAt); err != nil {
			return nil, err
		}
		invitations = append(invitations, inv)
	}

	// load guests per invitation
	for i := range invitations {
		guestRows, err := DB.Query(
			`SELECT id, first_name, last_name, confirmed_ceremony, confirmed_reception, invitation_id, created_at, updated_at FROM guests WHERE invitation_id = ? ORDER BY id`,
			invitations[i].ID,
		)
		if err != nil {
			return nil, err
		}
		for guestRows.Next() {
			var g models.Guest
			if err := guestRows.Scan(&g.ID, &g.FirstName, &g.LastName, &g.ConfirmedCeremony, &g.ConfirmedReception, &g.InvitationID, &g.CreatedAt, &g.UpdatedAt); err != nil {
				guestRows.Close()
				return nil, err
			}
			invitations[i].Guests = append(invitations[i].Guests, g)
		}
		guestRows.Close()
	}

	return invitations, nil
}

func DeleteInvitation(id int) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`UPDATE guests SET invitation_id = NULL WHERE invitation_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM invitations WHERE id = ?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

func MarkInvitationViewed(id int) error {
	_, err := DB.Exec(`UPDATE invitations SET viewed_at = CURRENT_TIMESTAMP WHERE id = ? AND viewed_at IS NULL`, id)
	return err
}

func SetGuestRSVP(id int, ceremony, reception *bool) error {
	_, err := DB.Exec(
		`UPDATE guests SET confirmed_ceremony = ?, confirmed_reception = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		boolToNullableInt(ceremony), boolToNullableInt(reception), id,
	)
	return err
}

func boolToNullableInt(b *bool) *int {
	if b == nil {
		return nil
	}
	v := 0
	if *b {
		v = 1
	}
	return &v
}

func GetInvitationByCode(code string) (models.Invitation, error) {
	var inv models.Invitation
	err := DB.QueryRow(
		`SELECT id, code, viewed_at, created_at FROM invitations WHERE code = ?`, code,
	).Scan(&inv.ID, &inv.Code, &inv.ViewedAt, &inv.CreatedAt)
	if err != nil {
		return inv, err
	}

	rows, err := DB.Query(
		`SELECT id, first_name, last_name, confirmed_ceremony, confirmed_reception, invitation_id, created_at, updated_at FROM guests WHERE invitation_id = ? ORDER BY id`,
		inv.ID,
	)
	if err != nil {
		return inv, err
	}
	defer rows.Close()
	for rows.Next() {
		var g models.Guest
		if err := rows.Scan(&g.ID, &g.FirstName, &g.LastName, &g.ConfirmedCeremony, &g.ConfirmedReception, &g.InvitationID, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return inv, err
		}
		inv.Guests = append(inv.Guests, g)
	}

	// load poll answers per guest
	var guestIDs []int
	for _, g := range inv.Guests {
		guestIDs = append(guestIDs, g.ID)
	}
	answersMap, err := GetPollAnswersForGuests(guestIDs)
	if err != nil {
		return inv, err
	}
	for i := range inv.Guests {
		inv.Guests[i].PollAnswers = answersMap[inv.Guests[i].ID]
	}

	return inv, nil
}
