package database

import (
	"database/sql"
	"log/slog"

	"github.com/streambinder/foedus/internal/models"
)

const settingsColumns = `groom_name, bride_name,
	ceremony_datetime, ceremony_address, ceremony_location, ceremony_city, ceremony_lat, ceremony_lng, ceremony_media_id,
	reception_datetime, reception_address, reception_location, reception_city, reception_lat, reception_lng, reception_media_id,
	bank_account_iban, bank_account_holder, spotify_playlist, playlist_readonly, share_preview_media_id`

// seedSettings materialises the single settings row. Every read assumes it
// exists, so this runs on every boot rather than only on a fresh database.
func seedSettings() {
	if _, err := DB.Exec(`INSERT OR IGNORE INTO settings (id) VALUES (1)`); err != nil {
		slog.Error("failed to seed settings row", "error", err.Error())
		return
	}
	// idempotent column migration for databases created before the
	// playlist_readonly setting existed: CREATE TABLE is IF NOT EXISTS, so
	// old databases would otherwise miss the column forever.
	var colCount int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('settings') WHERE name = 'playlist_readonly'`).Scan(&colCount); err != nil {
		slog.Error("failed to inspect settings columns", "error", err.Error())
		return
	}
	if colCount == 0 {
		if _, err := DB.Exec(`ALTER TABLE settings ADD COLUMN playlist_readonly INTEGER NOT NULL DEFAULT 0`); err != nil {
			slog.Error("failed to add playlist_readonly column", "error", err.Error())
			return
		}
		slog.Info("playlist_readonly column added to settings")
	}
	slog.Info("settings row ensured")
}

func GetSettings() (models.Settings, error) {
	var settings models.Settings
	var ceremonyMediaID, receptionMediaID, sharePreviewMediaID sql.NullInt64
	var playlistReadOnly sql.NullBool
	err := DB.QueryRow(`SELECT `+settingsColumns+` FROM settings WHERE id = 1`).Scan(
		&settings.GroomName, &settings.BrideName,
		&settings.CeremonyDatetime, &settings.CeremonyAddress, &settings.CeremonyLocation,
		&settings.CeremonyCity, &settings.CeremonyLat, &settings.CeremonyLng, &ceremonyMediaID,
		&settings.ReceptionDatetime, &settings.ReceptionAddress, &settings.ReceptionLocation,
		&settings.ReceptionCity, &settings.ReceptionLat, &settings.ReceptionLng, &receptionMediaID,
		&settings.BankAccountIBAN, &settings.BankAccountHolder,
		&settings.SpotifyPlaylist, &playlistReadOnly, &sharePreviewMediaID,
	)
	if err != nil {
		return models.Settings{}, err
	}
	settings.CeremonyMediaID = idOrZero(ceremonyMediaID)
	settings.ReceptionMediaID = idOrZero(receptionMediaID)
	settings.SharePreviewMediaID = idOrZero(sharePreviewMediaID)
	settings.PlaylistReadOnly = playlistReadOnly.Valid && playlistReadOnly.Bool
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
			bank_account_iban = ?, bank_account_holder = ?, spotify_playlist = ?,
			playlist_readonly = ?, share_preview_media_id = ?
		WHERE id = 1`,
		settings.GroomName, settings.BrideName,
		settings.CeremonyDatetime, settings.CeremonyAddress, settings.CeremonyLocation, settings.CeremonyCity,
		settings.CeremonyLat, settings.CeremonyLng, nullableID(settings.CeremonyMediaID),
		settings.ReceptionDatetime, settings.ReceptionAddress, settings.ReceptionLocation, settings.ReceptionCity,
		settings.ReceptionLat, settings.ReceptionLng, nullableID(settings.ReceptionMediaID),
		settings.BankAccountIBAN, settings.BankAccountHolder,
		settings.SpotifyPlaylist, settings.PlaylistReadOnly, nullableID(settings.SharePreviewMediaID),
	)
	return err
}
