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
		"title.dashboard":  "Dashboard",
		"title.edit_guest": "Edit Guest",

		// dashboard - sections
		"dashboard.title":   "Wedding Dashboard",
		"dashboard.details": "Details",
		"dashboard.guests":  "Guests",

		// dashboard - labels
		"label.spouses":  "Spouses",
		"label.ceremony": "Ceremony",
		"label.reception": "Reception",

		// dashboard - placeholders
		"placeholder.name":              "Name",
		"placeholder.ceremony_address":  "Ceremony address",
		"placeholder.reception_address": "Reception address",
		"placeholder.first_name":        "First name",
		"placeholder.last_name":         "Last name",
		"placeholder.search":            "Search...",

		// dashboard - guest form labels
		"label.first_name": "First Name",
		"label.last_name":  "Last Name",
		// dashboard - table headers
		"th.first_name": "First Name",
		"th.last_name":  "Last Name",
		"th.confirmed":  "Confirmed",
		"th.actions":    "Actions",

		// dashboard - buttons & actions
		"btn.save_settings": "Save",
		"btn.add":           "Add",
		"btn.import_csv":    "Import CSV",
		"btn.update_guest":  "Update Guest",
		"btn.edit":          "Edit",
		"btn.delete":        "Delete",
		"btn.back":          "Back to Dashboard",
		"btn.prev":          "Previous",
		"btn.next":          "Next",

		// dashboard - misc
		"confirm.delete_guest": "Delete this guest?",

		// flash messages
		"flash.settings_saved": "Settings saved.",
		"flash.guest_added":    "Guest added.",
		"flash.guests_imported": "guests imported.",
		"flash.guest_updated":  "Guest updated.",
		"flash.guest_deleted":  "Guest deleted.",

		// home
		"home.getting_married": "We're getting married",
		"home.ceremony":        "Ceremony",
		"home.reception":       "Reception",
		"home.guest_list":      "Guest List",

		// footer
		"link.dashboard": "dashboard",
	},
	"it": {
		// layout
		"title.dashboard":  "Pannello",
		"title.edit_guest": "Modifica Invitato",

		// dashboard - sections
		"dashboard.title":   "Pannello",
		"dashboard.details": "Dettagli",
		"dashboard.guests":  "Invitati",

		// dashboard - labels
		"label.spouses":  "Sposi",
		"label.ceremony": "Cerimonia",
		"label.reception": "Ricevimento",

		// dashboard - placeholders
		"placeholder.name":              "Nome",
		"placeholder.ceremony_address":  "Indirizzo cerimonia",
		"placeholder.reception_address": "Indirizzo ricevimento",
		"placeholder.first_name":        "Nome",
		"placeholder.last_name":         "Cognome",
		"placeholder.search":            "Cerca...",

		// dashboard - guest form labels
		"label.first_name": "Nome",
		"label.last_name":  "Cognome",
		// dashboard - table headers
		"th.first_name": "Nome",
		"th.last_name":  "Cognome",
		"th.confirmed":  "Confermato",
		"th.actions":    "Azioni",

		// dashboard - buttons & actions
		"btn.save_settings": "Salva",
		"btn.add":           "Aggiungi",
		"btn.import_csv":    "Importa CSV",
		"btn.update_guest":  "Aggiorna Invitato",
		"btn.edit":          "Modifica",
		"btn.delete":        "Elimina",
		"btn.back":          "Torna al Pannello",
		"btn.prev":          "Precedente",
		"btn.next":          "Successivo",

		// dashboard - misc
		"confirm.delete_guest": "Eliminare questo invitato?",

		// flash messages
		"flash.settings_saved": "Impostazioni salvate.",
		"flash.guest_added":    "Invitato aggiunto.",
		"flash.guests_imported": "invitati importati.",
		"flash.guest_updated":  "Invitato aggiornato.",
		"flash.guest_deleted":  "Invitato eliminato.",

		// home
		"home.getting_married": "Ci sposiamo",
		"home.ceremony":        "Cerimonia",
		"home.reception":       "Ricevimento",
		"home.guest_list":      "Lista Invitati",

		// footer
		"link.dashboard": "pannello",
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

// FormatDatetime parses a datetime-local string (2006-01-02T15:04) and formats it locale-aware.
// falls back to date-only format (2006-01-02) if datetime parse fails.
// returns the raw string on total parse failure.
func FormatDatetime(datetimeStr, lang string) string {
	t, err := time.Parse("2006-01-02T15:04", datetimeStr)
	if err != nil {
		// try date-only fallback for old data
		t, err = time.Parse("2006-01-02", datetimeStr)
		if err != nil {
			return datetimeStr
		}
	}

	hasTime := strings.Contains(datetimeStr, "T")
	if lang == "it" {
		s := t.Format("2") + " " + italianMonths[t.Month()] + " " + t.Format("2006")
		if hasTime {
			s += ", " + t.Format("15:04")
		}
		return s
	}
	if hasTime {
		return t.Format("January 2, 2006 at 3:04 PM")
	}
	return t.Format("January 2, 2006")
}
