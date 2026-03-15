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
		"title.setup":      "Setup Required",

		// dashboard - sections
		"dashboard.title":   "Wedding Dashboard",
		"dashboard.details": "Settings",
		"dashboard.guests":  "Guests",

		// dashboard - labels
		"label.spouses":  "Couple's Names",
		"label.ceremony": "Ceremony",
		"label.reception": "Celebration",

		// dashboard - counters
		"counter.ceremony":  "Ceremony",
		"counter.reception": "Celebration",
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
		"home.getting_married": "Together, forever",
		"home.ceremony":        "The Ceremony",
		"home.reception":       "The Celebration",
		"home.guest_list":      "Guest List",
		"home.gift":            "Gift",
		"home.gift_description": "If you'd like to honour us with a gift, choose an amount",

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
		"dashboard.gifts":   "Gifts Received",
		"th.donor":          "Donor",
		"th.amount":         "Amount",
		"th.message":        "Message",
		"th.date":           "Date",

		// registry
		"dashboard.registry":      "Wish List",
		"home.registry":           "A Gift for Us",
		"home.registry_description": "If you'd like to join us on this new journey",
		"placeholder.item_name":   "Item name",
		"placeholder.item_price":  "Price",
		"btn.add_item":            "Add Item",
		"btn.buy":                 "Give",
		"badge.sold":              "Gifted",
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
		"home.generic_gift":             "A Gift of Your Choice",
		"home.generic_gift_description": "Choose your amount",
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

		// polls
		"dashboard.polls":       "Polls",
		"placeholder.question":  "Question",
		"th.question":           "Question",
		"th.yes":                "Yes",
		"th.no":                 "No",
		"flash.poll_added":      "Poll added.",
		"flash.poll_deleted":    "Poll deleted.",
		"confirm.delete_poll":   "Delete this poll?",

		// invitations
		"dashboard.invitations":      "Invitations",
		"th.code":                    "Code",
		"th.guests":                  "Guests",
		"th.viewed":                  "Viewed",
		"th.created":                 "Created",
		"btn.create_invitation":      "Generate Invitation",
		"flash.invitation_created":   "Invitation created:",
		"flash.invitation_deleted":   "Invitation deleted.",
		"confirm.delete_invitation":  "Delete this invitation?",

		// invitation public page
		"invitation.title":            "Invitation",
		"invitation.rsvp":             "We Look Forward to Seeing You",
		"invitation.rsvp_description": "Let us know if you can join us on this special day",
		"invitation.confirm":          "Send Confirmation",
		"invitation.already_answered": "You have already sent your confirmation.",
		"invitation.change_answer":    "Change my answer",

		// setup guard
		"setup.not_ready":     "This wedding hasn't been set up yet.",
		"setup.go_to_dashboard": "Go to the dashboard to get started.",

		// footer
		"link.dashboard": "dashboard",
		"link.home":      "home",
	},
	"it": {
		// layout
		"title.dashboard":  "Pannello",
		"title.edit_guest": "Modifica Invitato",
		"title.setup":      "Configurazione necessaria",

		// dashboard - sections
		"dashboard.title":   "Pannello",
		"dashboard.details": "Impostazioni",
		"dashboard.guests":  "Invitati",

		// dashboard - labels
		"label.spouses":  "Nomi degli sposi",
		"label.ceremony": "Rito",
		"label.reception": "Festa",

		// dashboard - counters
		"counter.ceremony":  "Rito",
		"counter.reception": "Festa",
		"counter.pending":   "In attesa",

		// dashboard - placeholders
		"placeholder.name":              "Nome",
		"placeholder.ceremony_address":  "Indirizzo cerimonia",
		"placeholder.reception_address": "Indirizzo ricevimento",
		"placeholder.first_name":        "Nome",
		"placeholder.last_name":         "Cognome",
		"placeholder.search":            "Cerca tra gli invitati...",

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
		"home.getting_married": "Insieme per sempre",
		"home.ceremony":        "Il Rito",
		"home.reception":       "I Festeggiamenti",
		"home.guest_list":      "Lista Invitati",
		"home.gift":            "Regalo",
		"home.gift_description": "Se desiderate onorarci con un dono, scegliete un importo",

		// gift form
		"placeholder.donor":        "Il vostro nome (facoltativo)",
		"placeholder.gift_message": "Lasciate un messaggio (facoltativo)",
		"placeholder.amount":       "Importo",
		"btn.send_gift":            "Invia Regalo",

		// gift success
		"gift.thank_you":      "Grazie!",
		"gift.thanks_message": "Il vostro dono è stato registrato. Grazie per la vostra generosità!",
		"gift.thanks_donor":   "Grazie",
		"gift.back_home":      "Torna alla home",

		// dashboard - gifts
		"dashboard.gifts":   "Doni ricevuti",
		"th.donor":          "Da",
		"th.amount":         "Importo",
		"th.message":        "Messaggio",
		"th.date":           "Data",

		// registry
		"dashboard.registry":      "Lista dei desideri",
		"home.registry":           "Un Pensiero per Noi",
		"home.registry_description": "Se desiderate accompagnarci in questo nuovo cammino",
		"placeholder.item_name":   "Nome articolo",
		"placeholder.item_price":  "Prezzo",
		"btn.add_item":            "Aggiungi",
		"btn.buy":                 "Dona",
		"badge.sold":              "Offerto",
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
		"home.generic_gift":             "Dono Libero",
		"home.generic_gift_description": "Scegliete voi l'importo",
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

		// polls
		"dashboard.polls":       "Sondaggi",
		"placeholder.question":  "Domanda",
		"th.question":           "Domanda",
		"th.yes":                "Sì",
		"th.no":                 "No",
		"flash.poll_added":      "Sondaggio aggiunto.",
		"flash.poll_deleted":    "Sondaggio eliminato.",
		"confirm.delete_poll":   "Eliminare questo sondaggio?",

		// invitations
		"dashboard.invitations":      "Inviti",
		"th.code":                    "Codice",
		"th.guests":                  "Invitati",
		"th.viewed":                  "Visualizzato",
		"th.created":                 "Creato",
		"btn.create_invitation":      "Genera invito",
		"flash.invitation_created":   "Invito creato:",
		"flash.invitation_deleted":   "Invito eliminato.",
		"confirm.delete_invitation":  "Eliminare questo invito?",

		// invitation public page
		"invitation.title":            "Invito",
		"invitation.rsvp":             "Vi aspettiamo",
		"invitation.rsvp_description": "Fateci sapere se potrete essere con noi in questo giorno speciale",
		"invitation.confirm":          "Invio conferma",
		"invitation.already_answered": "Avete già inviato la vostra conferma.",
		"invitation.change_answer":    "Modifica risposta",

		// setup guard
		"setup.not_ready":       "Questo matrimonio non è ancora stato configurato.",
		"setup.go_to_dashboard": "Vai al pannello per iniziare.",

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
