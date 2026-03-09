package database

import "github.com/streambinder/foedus/internal/models"

var settingsKeys = []string{
	"spouse1_name", "spouse2_name", "ceremony_datetime",
	"ceremony_address", "ceremony_location",
	"reception_address", "reception_location",
	"bank_account_iban", "bank_account_holder",
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
		Spouse1Name:       m["spouse1_name"],
		Spouse2Name:       m["spouse2_name"],
		CeremonyAddress:   m["ceremony_address"],
		CeremonyLocation:  m["ceremony_location"],
		CeremonyDatetime:  m["ceremony_datetime"],
		ReceptionAddress:  m["reception_address"],
		ReceptionLocation: m["reception_location"],
		BankAccountIBAN:   m["bank_account_iban"],
		BankAccountHolder: m["bank_account_holder"],
	}, nil
}

func UpdateSetting(key, value string) error {
	_, err := DB.Exec(`UPDATE settings SET value = ? WHERE key = ?`, value, key)
	return err
}
