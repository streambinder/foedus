package database

import "github.com/dpucci/foedus/internal/models"

var settingsKeys = []string{
	"spouse1_name", "spouse2_name", "ceremony_date",
	"church_name", "church_address",
	"party_venue", "party_address",
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

	return models.WeddingSettings{
		Spouse1Name:   m["spouse1_name"],
		Spouse2Name:   m["spouse2_name"],
		CeremonyDate:  m["ceremony_date"],
		ChurchName:    m["church_name"],
		ChurchAddress: m["church_address"],
		PartyVenue:    m["party_venue"],
		PartyAddress:  m["party_address"],
	}, nil
}

func UpdateSetting(key, value string) error {
	_, err := DB.Exec(`UPDATE settings SET value = ? WHERE key = ?`, value, key)
	return err
}
