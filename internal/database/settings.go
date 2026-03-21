package database

import (
	"encoding/json"

	"github.com/streambinder/foedus/internal/models"
)

var settingsKeys = []string{
	"spouse1_name", "spouse2_name", "ceremony_datetime",
	"ceremony_address", "ceremony_location", "ceremony_image",
	"reception_address", "reception_location", "reception_image",
	"bank_account_iban", "bank_account_holder",
	"spotify_playlists", "places",
}

// SeedSettings inserts default empty rows for any missing setting keys.
func SeedSettings() {
	for _, key := range settingsKeys {
		DB.Exec(`INSERT OR IGNORE INTO settings (key, value) VALUES (?, '')`, key)
	}
}

func GetAllSettings() (models.WeddingSettings, error) {
	rows, err := DB.Query(`SELECT key, value FROM settings`)
	if err != nil {
		return models.WeddingSettings{}, err
	}
	defer rows.Close()

	m := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return models.WeddingSettings{}, err
		}
		m[k] = v
	}

	var playlists []string
	if raw := m["spotify_playlists"]; raw != "" {
		json.Unmarshal([]byte(raw), &playlists)
	}

	var places []models.Place
	if raw := m["places"]; raw != "" {
		json.Unmarshal([]byte(raw), &places)
	}

	return models.WeddingSettings{
		Spouse1Name:       m["spouse1_name"],
		Spouse2Name:       m["spouse2_name"],
		CeremonyAddress:   m["ceremony_address"],
		CeremonyLocation:  m["ceremony_location"],
		CeremonyImage:     m["ceremony_image"],
		CeremonyDatetime:  m["ceremony_datetime"],
		ReceptionAddress:  m["reception_address"],
		ReceptionLocation: m["reception_location"],
		ReceptionImage:    m["reception_image"],
		BankAccountIBAN:   m["bank_account_iban"],
		BankAccountHolder: m["bank_account_holder"],
		SpotifyPlaylists:  playlists,
		Places:            places,
	}, nil
}

func UpdateSetting(key, value string) error {
	_, err := DB.Exec(`UPDATE settings SET value = ? WHERE key = ?`, value, key)
	return err
}
