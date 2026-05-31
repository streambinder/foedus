package database

import (
	"encoding/json"
	"log/slog"
	"strconv"
	"strings"

	"github.com/streambinder/foedus/internal/models"
)

var settingsKeys = []string{
	"spouse1_name", "spouse2_name", "ceremony_datetime",
	"ceremony_address", "ceremony_location", "ceremony_city", "ceremony_media_id",
	"reception_address", "reception_location", "reception_city", "reception_datetime", "reception_media_id",
	"bank_account_iban", "bank_account_holder",
	"spotify_playlists", "places", "honeymoon_locations", "accommodation_suggestions", "impersonations", "homepage_labels",
	"homepage_hero_backgrounds",
	"share_preview_media_id",
}

// SeedSettings inserts default empty rows for any missing setting keys.
func SeedSettings() {
	inserted := 0
	for _, key := range settingsKeys {
		result, err := DB.Exec(`INSERT OR IGNORE INTO settings (key, value) VALUES (?, '')`, key)
		if err != nil {
			slog.Error("failed to seed setting", "key", key, "error", err.Error())
			continue
		}
		if rows, err := result.RowsAffected(); err == nil && rows > 0 {
			inserted++
		}
	}
	slog.Info("settings seeded", "keys", len(settingsKeys), "inserted", inserted)
}

func parseMediaID(raw string) int {
	id, _ := strconv.Atoi(strings.TrimSpace(raw))
	if id < 0 {
		return 0
	}
	return id
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

	var playlist string
	if raw := m["spotify_playlists"]; raw != "" {
		raw = strings.TrimSpace(raw)
		if strings.HasPrefix(raw, "[") {
			var playlists []string
			json.Unmarshal([]byte(raw), &playlists)
			for _, candidate := range playlists {
				candidate = strings.TrimSpace(candidate)
				if candidate != "" {
					playlist = candidate
					break
				}
			}
		} else {
			playlist = raw
		}
	}

	var places []models.Place
	if raw := m["places"]; raw != "" {
		json.Unmarshal([]byte(raw), &places)
	}

	var honeymoonLocations []models.Place
	if raw := m["honeymoon_locations"]; raw != "" {
		json.Unmarshal([]byte(raw), &honeymoonLocations)
	}

	var accommodationSuggestions []models.AccommodationSuggestion
	if raw := m["accommodation_suggestions"]; raw != "" {
		json.Unmarshal([]byte(raw), &accommodationSuggestions)
	}

	var impersonations []models.Impersonation
	if raw := m["impersonations"]; raw != "" {
		json.Unmarshal([]byte(raw), &impersonations)
	}

	var homepageLabels map[string]map[string]string
	if raw := m["homepage_labels"]; raw != "" {
		json.Unmarshal([]byte(raw), &homepageLabels)
	}

	var homepageHeroBackgrounds []models.HomepageHeroBackground
	if raw := m["homepage_hero_backgrounds"]; raw != "" {
		json.Unmarshal([]byte(raw), &homepageHeroBackgrounds)
	}

	return models.WeddingSettings{
		Spouse1Name:              m["spouse1_name"],
		Spouse2Name:              m["spouse2_name"],
		CeremonyAddress:          m["ceremony_address"],
		CeremonyLocation:         m["ceremony_location"],
		CeremonyCity:             m["ceremony_city"],
		CeremonyMediaID:          parseMediaID(m["ceremony_media_id"]),
		CeremonyDatetime:         m["ceremony_datetime"],
		ReceptionAddress:         m["reception_address"],
		ReceptionLocation:        m["reception_location"],
		ReceptionCity:            m["reception_city"],
		ReceptionDatetime:        m["reception_datetime"],
		ReceptionMediaID:         parseMediaID(m["reception_media_id"]),
		BankAccountIBAN:          m["bank_account_iban"],
		BankAccountHolder:        m["bank_account_holder"],
		SpotifyPlaylist:          playlist,
		Places:                   places,
		HoneymoonLocations:       honeymoonLocations,
		AccommodationSuggestions: accommodationSuggestions,
		Impersonations:           impersonations,
		HomepageLabels:           homepageLabels,
		HomepageHeroBackgrounds:  homepageHeroBackgrounds,
		SharePreviewMediaID:      parseMediaID(m["share_preview_media_id"]),
	}, nil
}

func UpdateSetting(q Querier, key, value string) error {
	_, err := q.Exec(`UPDATE settings SET value = ? WHERE key = ?`, value, key)
	return err
}
