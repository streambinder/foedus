package database

import (
	"database/sql"
	"log/slog"

	"github.com/streambinder/foedus/internal/models"
)

const settingsColumns = `groom_name, bride_name,
	ceremony_datetime, ceremony_address, ceremony_location, ceremony_city, ceremony_lat, ceremony_lng, ceremony_media_id,
	reception_datetime, reception_address, reception_location, reception_city, reception_lat, reception_lng, reception_media_id,
	bank_account_iban, bank_account_holder, spotify_playlist, share_preview_media_id`

// seedSettings materialises the single settings row. Every read assumes it
// exists, so this runs on every boot rather than only on a fresh database.
func seedSettings() {
	if _, err := DB.Exec(`INSERT OR IGNORE INTO settings (id) VALUES (1)`); err != nil {
		slog.Error("failed to seed settings row", "error", err.Error())
		return
	}
	slog.Info("settings row ensured")
}

func GetSettings() (models.Settings, error) {
	var settings models.Settings
	var ceremonyMediaID, receptionMediaID, sharePreviewMediaID sql.NullInt64
	err := DB.QueryRow(`SELECT `+settingsColumns+` FROM settings WHERE id = 1`).Scan(
		&settings.GroomName, &settings.BrideName,
		&settings.CeremonyDatetime, &settings.CeremonyAddress, &settings.CeremonyLocation,
		&settings.CeremonyCity, &settings.CeremonyLat, &settings.CeremonyLng, &ceremonyMediaID,
		&settings.ReceptionDatetime, &settings.ReceptionAddress, &settings.ReceptionLocation,
		&settings.ReceptionCity, &settings.ReceptionLat, &settings.ReceptionLng, &receptionMediaID,
		&settings.BankAccountIBAN, &settings.BankAccountHolder,
		&settings.SpotifyPlaylist, &sharePreviewMediaID,
	)
	if err != nil {
		return models.Settings{}, err
	}
	settings.CeremonyMediaID = idOrZero(ceremonyMediaID)
	settings.ReceptionMediaID = idOrZero(receptionMediaID)
	settings.SharePreviewMediaID = idOrZero(sharePreviewMediaID)
	return settings, nil
}

func UpdateSettings(q Querier, settings models.Settings) error {
	_, err := q.Exec(
		`UPDATE settings SET
			groom_name = ?, bride_name = ?,
			ceremony_datetime = ?, ceremony_address = ?, ceremony_location = ?, ceremony_city = ?,
			ceremony_lat = ?, ceremony_lng = ?, ceremony_media_id = ?,
			reception_datetime = ?, reception_address = ?, reception_location = ?, reception_city = ?,
			reception_lat = ?, reception_lng = ?, reception_media_id = ?,
			bank_account_iban = ?, bank_account_holder = ?, spotify_playlist = ?, share_preview_media_id = ?
		WHERE id = 1`,
		settings.GroomName, settings.BrideName,
		settings.CeremonyDatetime, settings.CeremonyAddress, settings.CeremonyLocation, settings.CeremonyCity,
		settings.CeremonyLat, settings.CeremonyLng, nullableID(settings.CeremonyMediaID),
		settings.ReceptionDatetime, settings.ReceptionAddress, settings.ReceptionLocation, settings.ReceptionCity,
		settings.ReceptionLat, settings.ReceptionLng, nullableID(settings.ReceptionMediaID),
		settings.BankAccountIBAN, settings.BankAccountHolder,
		settings.SpotifyPlaylist, nullableID(settings.SharePreviewMediaID),
	)
	return err
}
