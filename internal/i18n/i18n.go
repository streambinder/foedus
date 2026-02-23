package i18n

import (
	"strings"
	"time"
)

// T translates a key into the active language's string.
type T func(string) string

var italianMonths = map[time.Month]string{
	time.January:   "gennaio",
	time.February:  "febbraio",
	time.March:     "marzo",
	time.April:     "aprile",
	time.May:       "maggio",
	time.June:      "giugno",
	time.July:      "luglio",
	time.August:    "agosto",
	time.September: "settembre",
	time.October:   "ottobre",
	time.November:  "novembre",
	time.December:  "dicembre",
}

var supported = map[string]bool{"en": true, "it": true}

var translations = map[string]map[string]string{
	"en": {
		// layout
		"title.dashboard": "Dashboard",
		"title.edit_guest": "Edit Guest",

		// dashboard - sections
		"dashboard.title":            "Wedding Dashboard",
		"dashboard.settings":         "Wedding Settings",
		"dashboard.add_guest":        "Add Guest",
		"dashboard.guests":           "Guests",

		// dashboard - settings labels
		"label.spouse1_name":   "Spouse 1 Name",
		"label.spouse2_name":   "Spouse 2 Name",
		"label.ceremony_date":  "Ceremony Date",
		"label.church":         "Church",
		"label.party":          "Party Venue",

		// dashboard - guest form labels
		"label.name":          "Name",
		"label.email":         "Email",
		"label.dietary_notes": "Dietary Notes",
		"label.notes":         "Notes",
		"label.plus_one":      "Plus One",

		// dashboard - table headers
		"th.name":    "Name",
		"th.email":   "Email",
		"th.plus_one": "+1",
		"th.dietary": "Dietary",
		"th.notes":   "Notes",
		"th.actions": "Actions",

		// dashboard - buttons & actions
		"btn.save_settings": "Save Settings",
		"btn.add_guest":     "Add Guest",
		"btn.update_guest":  "Update Guest",
		"btn.edit":          "Edit",
		"btn.delete":        "Delete",
		"btn.back":          "Back to Dashboard",

		// dashboard - misc
		"guests.empty":         "No guests added yet.",
		"confirm.delete_guest": "Delete this guest?",
		"osm.attribution":      "Search powered by OpenStreetMap Nominatim",

		// flash messages
		"flash.settings_saved": "Settings saved.",
		"flash.guest_added":    "Guest added.",
		"flash.guest_updated":  "Guest updated.",
		"flash.guest_deleted":  "Guest deleted.",

		// home
		"home.getting_married": "We're getting married",
		"home.ceremony":        "Ceremony",
		"home.reception":       "Reception",
		"home.guest_list":      "Guest List",

		// plus one values
		"yes": "Yes",
		"no":  "No",
	},
	"it": {
		// layout
		"title.dashboard": "Pannello",
		"title.edit_guest": "Modifica Invitato",

		// dashboard - sections
		"dashboard.title":            "Pannello",
		"dashboard.settings":         "Impostazioni",
		"dashboard.add_guest":        "Aggiungi Invitato",
		"dashboard.guests":           "Invitati",

		// dashboard - settings labels
		"label.spouse1_name":   "Nome Sposo/a 1",
		"label.spouse2_name":   "Nome Sposo/a 2",
		"label.ceremony_date":  "Data Cerimonia",
		"label.church":         "Chiesa",
		"label.party":          "Location Ricevimento",

		// dashboard - guest form labels
		"label.name":          "Nome",
		"label.email":         "Email",
		"label.dietary_notes": "Note Alimentari",
		"label.notes":         "Note",
		"label.plus_one":      "Accompagnatore",

		// dashboard - table headers
		"th.name":    "Nome",
		"th.email":   "Email",
		"th.plus_one": "+1",
		"th.dietary": "Dieta",
		"th.notes":   "Note",
		"th.actions": "Azioni",

		// dashboard - buttons & actions
		"btn.save_settings": "Salva Impostazioni",
		"btn.add_guest":     "Aggiungi Invitato",
		"btn.update_guest":  "Aggiorna Invitato",
		"btn.edit":          "Modifica",
		"btn.delete":        "Elimina",
		"btn.back":          "Torna al Pannello",

		// dashboard - misc
		"guests.empty":         "Nessun invitato aggiunto.",
		"confirm.delete_guest": "Eliminare questo invitato?",
		"osm.attribution":      "Ricerca fornita da OpenStreetMap Nominatim",

		// flash messages
		"flash.settings_saved": "Impostazioni salvate.",
		"flash.guest_added":    "Invitato aggiunto.",
		"flash.guest_updated":  "Invitato aggiornato.",
		"flash.guest_deleted":  "Invitato eliminato.",

		// home
		"home.getting_married": "Ci sposiamo",
		"home.ceremony":        "Cerimonia",
		"home.reception":       "Ricevimento",
		"home.guest_list":      "Lista Invitati",

		// plus one values
		"yes": "Sì",
		"no":  "No",
	},
}

// NewT returns a translator closure for the given language.
// falls back to english for missing keys.
func NewT(lang string) T {
	langMap := translations["en"]
	if m, ok := translations[lang]; ok {
		langMap = m
	}
	fallback := translations["en"]
	return func(key string) string {
		if v, ok := langMap[key]; ok {
			return v
		}
		if v, ok := fallback[key]; ok {
			return v
		}
		return key
	}
}

// DetectLang parses an Accept-Language header and returns the best supported language.
// defaults to "en".
func DetectLang(acceptLang string) string {
	for _, part := range strings.Split(acceptLang, ",") {
		// strip quality value (e.g. "en-US;q=0.9" -> "en-US")
		tag := strings.TrimSpace(strings.SplitN(part, ";", 2)[0])
		if len(tag) >= 2 {
			code := strings.ToLower(tag[:2])
			if supported[code] {
				return code
			}
		}
	}
	return "en"
}

// FormatDate parses a YYYY-MM-DD string and formats it locale-aware.
// returns the raw string on parse failure.
func FormatDate(dateStr, lang string) string {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return dateStr
	}
	if lang == "it" {
		return t.Format("2") + " " + italianMonths[t.Month()] + " " + t.Format("2006")
	}
	return t.Format("January 2, 2006")
}
