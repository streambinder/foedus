package database

import (
	"strings"

	"github.com/streambinder/foedus/internal/models"
)

func CreateSoundtrackEvent(title, artist, trackURL, inviteID string) error {
	_, err := DB.Exec(
		`INSERT INTO soundtrack_events (title, artist, url, invite_id) VALUES (?, ?, ?, ?)`,
		strings.TrimSpace(title),
		strings.TrimSpace(artist),
		strings.TrimSpace(trackURL),
		strings.TrimSpace(inviteID),
	)
	return err
}

func GetAllSoundtrackEvents() ([]models.SoundtrackEvent, error) {
	rows, err := DB.Query(
		`SELECT id, title, artist, url, invite_id, created_at
		FROM soundtrack_events
		ORDER BY created_at DESC, id DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.SoundtrackEvent
	for rows.Next() {
		var event models.SoundtrackEvent
		if err := rows.Scan(&event.ID, &event.Title, &event.Artist, &event.URL, &event.InviteID, &event.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}
