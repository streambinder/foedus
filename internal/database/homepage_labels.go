package database

// Only overridden labels are stored. A key with no row falls through to the
// compiled-in i18n default, so an empty result is the normal case, not an error.

// GetHomepageLabels returns the overrides for one language — what the public
// pages need. The dashboard wants every language and uses GetAllHomepageLabels.
func GetHomepageLabels(lang string) (map[string]string, error) {
	rows, err := DB.Query(`SELECT key, value FROM homepage_labels WHERE lang = ?`, lang)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	overrides := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		overrides[key] = value
	}
	return overrides, rows.Err()
}

func GetAllHomepageLabels() (map[string]map[string]string, error) {
	rows, err := DB.Query(`SELECT lang, key, value FROM homepage_labels`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	labels := make(map[string]map[string]string)
	for rows.Next() {
		var lang, key, value string
		if err := rows.Scan(&lang, &key, &value); err != nil {
			return nil, err
		}
		if labels[lang] == nil {
			labels[lang] = make(map[string]string)
		}
		labels[lang][key] = value
	}
	return labels, rows.Err()
}

func ReplaceHomepageLabels(q Querier, labels map[string]map[string]string) error {
	if _, err := q.Exec(`DELETE FROM homepage_labels`); err != nil {
		return err
	}
	for lang, overrides := range labels {
		for key, value := range overrides {
			if _, err := q.Exec(`INSERT INTO homepage_labels (lang, key, value) VALUES (?, ?, ?)`, lang, key, value); err != nil {
				return err
			}
		}
	}
	return nil
}
