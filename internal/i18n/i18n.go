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

		// dashboard - counters
		"counter.ceremony":  "Ceremony",
		"counter.reception": "Reception",
		"counter.pending":   "Pending",

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
		"th.ceremony":   "Ceremony",
		"th.reception":  "Reception",
		"th.actions":    "Actions",

		// rsvp radio labels
		"rsvp.yes": "Yes",
		"rsvp.no":  "No",

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
		"home.gift":            "Gift",
		"home.gift_description": "If you'd like to send us a gift, choose an amount below.",

		// gift form
		"placeholder.donor":        "Your name (optional)",
		"placeholder.gift_message": "Leave a message (optional)",
		"placeholder.amount":       "Amount",
		"btn.send_gift":            "Send Gift",

		// gift success
		"gift.thank_you":      "Thank You!",
		"gift.thanks_message": "Your gift has been recorded. Thank you for your generosity!",
		"gift.thanks_donor":   "Thank you",
		"gift.back_home":      "Back to home",

		// dashboard - gifts
		"dashboard.gifts":   "Gifts",
		"th.donor":          "Donor",
		"th.amount":         "Amount",
		"th.message":        "Message",
		"th.date":           "Date",

		// registry
		"dashboard.registry":      "Registry",
		"home.registry":           "Registry",
		"home.registry_description": "If you'd like to gift us something specific, pick an item below.",
		"placeholder.item_name":   "Item name",
		"placeholder.item_price":  "Price",
		"btn.add_item":            "Add Item",
		"btn.buy":                 "Buy",
		"badge.sold":              "Sold",
		"th.item_name":            "Name",
		"th.item_price":           "Price",
		"th.item_image":           "Image",
		"flash.item_added":        "Item added.",
		"flash.item_deleted":      "Item deleted.",
		"confirm.delete_item":     "Delete this item?",

		// bank transfer modal
		"modal.transfer_intro":        "Make a bank transfer with the details below to claim this gift.",
		"modal.iban":                  "IBAN",
		"modal.owner":                 "Account holder",
		"modal.reason":                "Reason",
		"modal.transfer_confirm_text": "Once the transfer is done, confirm below.",
		"btn.cancel":                  "Cancel",
		"btn.confirm":                 "Confirm",
		"btn.copy":                    "Copy",
		"btn.copied":                  "Copied!",
		"btn.close":                   "Close",
		"home.generic_gift":             "Free Gift",
		"home.generic_gift_description": "Send us a gift of any amount",
		"placeholder.custom_amount":     "Custom amount",
		"home.remaining":                "remaining",

		// bank settings
		"label.bank_account_iban":         "IBAN",
		"label.bank_account_holder":       "Account holder",
		"placeholder.bank_account_iban":   "IT60 X054 2811 1010 0000 0123 456",
		"placeholder.bank_account_holder": "Account holder name",
		"label.bank":                      "Bank Transfer",

		// registry item link in gifts table
		"th.registry_item": "Item",

		// invitations
		"dashboard.invitations":      "Invitations",
		"th.code":                    "Code",
		"th.guests":                  "Guests",
		"th.viewed":                  "Viewed",
		"th.created":                 "Created",
		"btn.create_invitation":      "Create Invitation",
		"flash.invitation_created":   "Invitation created:",
		"flash.invitation_deleted":   "Invitation deleted.",
		"confirm.delete_invitation":  "Delete this invitation?",

		// invitation public page
		"invitation.title":            "Invitation",
		"invitation.rsvp":             "RSVP",
		"invitation.rsvp_description": "Please confirm attendance for each guest.",
		"invitation.confirm":          "Confirm",

		// footer
		"link.dashboard": "dashboard",
		"link.home":      "home",
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

		// dashboard - counters
		"counter.ceremony":  "Cerimonia",
		"counter.reception": "Ricevimento",
		"counter.pending":   "In attesa",

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
		"th.ceremony":   "Cerimonia",
		"th.reception":  "Ricevimento",
		"th.actions":    "Azioni",

		// rsvp radio labels
		"rsvp.yes": "Sì",
		"rsvp.no":  "No",

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
		"home.gift":            "Regalo",
		"home.gift_description": "Se desideri farci un regalo, scegli un importo qui sotto.",

		// gift form
		"placeholder.donor":        "Il tuo nome (facoltativo)",
		"placeholder.gift_message": "Lascia un messaggio (facoltativo)",
		"placeholder.amount":       "Importo",
		"btn.send_gift":            "Invia Regalo",

		// gift success
		"gift.thank_you":      "Grazie!",
		"gift.thanks_message": "Il tuo regalo è stato registrato. Grazie per la tua generosità!",
		"gift.thanks_donor":   "Grazie",
		"gift.back_home":      "Torna alla home",

		// dashboard - gifts
		"dashboard.gifts":   "Regali",
		"th.donor":          "Da",
		"th.amount":         "Importo",
		"th.message":        "Messaggio",
		"th.date":           "Data",

		// registry
		"dashboard.registry":      "Lista nozze",
		"home.registry":           "Lista nozze",
		"home.registry_description": "Se vuoi regalarci qualcosa di specifico, scegli un articolo qui sotto.",
		"placeholder.item_name":   "Nome articolo",
		"placeholder.item_price":  "Prezzo",
		"btn.add_item":            "Aggiungi",
		"btn.buy":                 "Acquista",
		"badge.sold":              "Venduto",
		"th.item_name":            "Nome",
		"th.item_price":           "Prezzo",
		"th.item_image":           "Immagine",
		"flash.item_added":        "Articolo aggiunto.",
		"flash.item_deleted":      "Articolo eliminato.",
		"confirm.delete_item":     "Eliminare questo articolo?",

		// bank transfer modal
		"modal.transfer_intro":        "Effettua un bonifico con i dati seguenti per prenotare questo regalo.",
		"modal.iban":                  "IBAN",
		"modal.owner":                 "Intestatario",
		"modal.reason":                "Causale",
		"modal.transfer_confirm_text": "Una volta effettuato il bonifico, conferma qui sotto.",
		"btn.cancel":                  "Annulla",
		"btn.confirm":                 "Conferma",
		"btn.copy":                    "Copia",
		"btn.copied":                  "Copiato!",
		"btn.close":                   "Chiudi",
		"home.generic_gift":             "Regalo libero",
		"home.generic_gift_description": "Inviaci un regalo di qualsiasi importo",
		"placeholder.custom_amount":     "Importo personalizzato",
		"home.remaining":                "rimanenti",

		// bank settings
		"label.bank_account_iban":         "IBAN",
		"label.bank_account_holder":       "Intestatario",
		"placeholder.bank_account_iban":   "IT60 X054 2811 1010 0000 0123 456",
		"placeholder.bank_account_holder": "Nome intestatario conto",
		"label.bank":                      "Bonifico Bancario",

		// registry item link in gifts table
		"th.registry_item": "Articolo",

		// invitations
		"dashboard.invitations":      "Inviti",
		"th.code":                    "Codice",
		"th.guests":                  "Invitati",
		"th.viewed":                  "Visualizzato",
		"th.created":                 "Creato",
		"btn.create_invitation":      "Crea Invito",
		"flash.invitation_created":   "Invito creato:",
		"flash.invitation_deleted":   "Invito eliminato.",
		"confirm.delete_invitation":  "Eliminare questo invito?",

		// invitation public page
		"invitation.title":            "Invito",
		"invitation.rsvp":             "RSVP",
		"invitation.rsvp_description": "Conferma la presenza per ciascun invitato.",
		"invitation.confirm":          "Conferma",

		// footer
		"link.dashboard": "pannello",
		"link.home":      "home",
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

// FormatDate formats a time.Time as a short date string.
func FormatDate(t time.Time, lang string) string {
	if lang == "it" {
		return t.Format("2") + " " + italianMonths[t.Month()] + " " + t.Format("2006")
	}
	return t.Format("Jan 2, 2006")
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
