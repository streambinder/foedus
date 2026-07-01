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
		"title.edit_item":  "Edit Item",
		"title.edit_gift":  "Edit Gift",
		"title.edit_poll":  "Edit Poll",
		"title.setup":      "Setup Required",

		// dashboard - sections
		"dashboard.title":           "Wedding Dashboard",
		"dashboard.details":         "Settings",
		"dashboard.guests":          "Guests",
		"dashboard.track_additions": "Track Additions",

		// dashboard - labels
		"label.spouses":   "Couple's Names",
		"label.ceremony":  "Ceremony",
		"label.reception": "Celebration",

		// dashboard - counters
		"counter.rsvp":              "RSVP",
		"counter.visualization":     "Invitation Views",
		"counter.reach":             "Invitation Reach",
		"counter.confirmed":         "Confirmed",
		"counter.refused":           "Refused",
		"counter.pending_rsvp":      "Pending",
		"counter.viewed":            "Viewed",
		"counter.nonvisualized":     "Not Viewed",
		"counter.invited":           "Invited",
		"counter.uninvited":         "Not Invited",
		"counter.confirmed_by_type": "Confirmed by Type",

		// dashboard - placeholders
		"placeholder.name":              "Name",
		"placeholder.ceremony_address":  "Ceremony address",
		"placeholder.reception_address": "Reception address",
		"placeholder.first_name":        "First name",
		"placeholder.last_name":         "Last name",
		"placeholder.search":            "Search...",
		"placeholder.invitation_search": "Search invitations...",

		// dashboard - guest form labels
		"label.first_name":    "First Name",
		"label.last_name":     "Last Name",
		"label.type":          "Type",
		"label.amount":        "Amount",
		"label.donor":         "Donor",
		"label.registry_item": "Registry Item",
		"label.confirmed":     "Confirmed",
		// guest type enum (matches CHECK on guests.type)
		"guest_type.adult":  "Adult",
		"guest_type.child":  "Child",
		"guest_type.infant": "Infant",
		"guest_type.vendor": "Vendor",
		// dashboard - table headers
		"th.first_name":    "First Name",
		"th.last_name":     "Last Name",
		"th.type":          "Type",
		"th.ceremony":      "Ceremony",
		"th.reception":     "Reception",
		"th.actions":       "Actions",
		"th.confirmed":     "Confirmed",
		"th.track_title":   "Title",
		"th.track_artist":  "Artist",
		"th.track_url":     "Track URL",
		"th.invite_id":     "Invite ID",
		"th.invite_guests": "Guests",

		// rsvp radio labels
		"rsvp.yes": "Yes",
		"rsvp.no":  "No",

		// dashboard - buttons & actions
		"btn.save_settings": "Save",
		"btn.add":           "Add",
		"btn.import_csv":    "Import CSV",
		"btn.update_guest":  "Update Guest",
		"btn.update_item":   "Update Item",
		"btn.update_gift":   "Update Gift",
		"btn.update_poll":   "Update Poll",
		"btn.edit":          "Edit",
		"btn.delete":        "Delete",
		"btn.reset_viewed":  "Reset viewed",
		"btn.move_up":       "Move Up",
		"btn.move_down":     "Move Down",
		"btn.back":          "Back to Dashboard",
		"btn.prev":          "Previous",
		"btn.next":          "Next",

		// dashboard - misc
		"confirm.delete_guest": "Delete this guest?",

		// flash messages
		"flash.settings_saved":  "Settings saved.",
		"flash.guest_added":     "Guest added.",
		"flash.guests_imported": "guests imported.",
		"flash.guest_updated":   "Guest updated.",
		"flash.guest_deleted":   "Guest deleted.",
		"flash.item_updated":    "Item updated.",
		"flash.gift_updated":    "Gift updated.",
		"flash.gift_deleted":    "Gift deleted.",

		// home
		"home.getting_married":    "We are announcing our marriage",
		"home.venues":             "Location",
		"home.venues_description": "",
		"home.directions":         "directions",
		"home.ceremony":           "The Ceremony",
		"home.reception":          "The Celebration",
		"home.guest_list":         "Guest List",
		"home.gift":               "Gift",
		"home.gift_description":   "If you'd like to honour us with a gift, choose an amount",

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
		"dashboard.gifts": "Gifts Received",
		"th.donor":        "Donor",
		"th.amount":       "Amount",
		"th.message":      "Message",
		"th.date":         "Date",

		// registry
		"dashboard.registry":          "Wish List",
		"home.registry":               "Wedding List",
		"home.registry_description":   "no ideas? got you covered.",
		"placeholder.item_name":       "Item name",
		"placeholder.item_price":      "Price",
		"btn.add_item":                "Add Item",
		"btn.buy":                     "Give",
		"badge.sold":                  "Gifted",
		"th.item_name":                "Name",
		"th.item_price":               "Price",
		"th.item_image":               "Image",
		"th.item_order":               "Order",
		"flash.item_added":            "Item added.",
		"flash.item_deleted":          "Item deleted.",
		"confirm.delete_item":         "Delete this item?",
		"confirm.delete_gift":         "Delete this gift?",
		"status.confirmed":            "Yes",
		"status.pending_confirmation": "No",
		"option.no_registry_item":     "No linked registry item",

		// bank transfer modal
		"modal.transfer_intro":          "Make a bank transfer with the details below to claim this gift.",
		"modal.iban":                    "IBAN",
		"modal.owner":                   "Account holder",
		"modal.reason":                  "Reason",
		"modal.transfer_confirm_text":   "Once the transfer is done, confirm below.",
		"btn.cancel":                    "Cancel",
		"btn.confirm":                   "Confirm",
		"btn.copy":                      "Copy",
		"btn.copied":                    "Copied!",
		"btn.close":                     "Close",
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
		"dashboard.polls":         "Polls",
		"placeholder.question":    "Question",
		"placeholder.description": "Description (optional)",
		"th.question":             "Question",
		"th.description":          "Description",
		"th.yes":                  "Yes",
		"th.no":                   "No",
		"flash.poll_added":        "Poll added.",
		"flash.poll_updated":      "Poll updated.",
		"flash.poll_deleted":      "Poll deleted.",
		"confirm.delete_poll":     "Delete this poll?",
		"flash.track_deleted":     "Track removed from history.",
		"confirm.delete_track":    "Remove this track from history? Spotify playlist not affected.",

		// invitations
		"dashboard.invitations":        "Invitations",
		"th.code":                      "Code",
		"th.guests":                    "Guests",
		"th.actioned":                  "Actioned",
		"th.viewed":                    "Viewed",
		"th.created":                   "Created",
		"btn.create_invitation":        "Generate Invitation",
		"btn.update_invitation":        "Update Invitation",
		"flash.invitation_created":     "Invitation created:",
		"flash.invitation_updated":     "Invitation updated.",
		"flash.invitation_deleted":     "Invitation deleted.",
		"confirm.delete_invitation":    "Delete this invitation?",
		"th.label":                     "Label",
		"label.invitation_label":       "Label",
		"placeholder.invitation_label": "Label",
		"hint.invitation_label":        "Used as the OG title. Leave empty to compute from guests.",
		"title.edit_invitation":        "Edit Invitation",

		// invitation public page
		"invitation.title":             "Invitation",
		"invitation.rsvp":              "We Look Forward to Seeing You",
		"invitation.rsvp_description":  "Let us know if you can join us",
		"invitation.confirm":           "Confirm",
		"invitation.already_answered":  "You have already sent your confirmation.",
		"invitation.change_answer":     "change my answer",
		"invitation.go_home":           "go to homepage",
		"invitation.submitted_message": "Thank you! Your RSVP has been recorded.",
		"invitation.replay_animation":  "Replay envelope animation",
		"invitation.close_envelope":    "close envelope",
		"invitation.brought_to_you_by": "Created by Davide Pucci",
		"invitation.poll_notes":        "Notes",
		"placeholder.poll_notes":       "Add a note",

		// ceremony page
		"ceremony.title": "Ceremony",

		// setup guard
		"setup.not_ready":       "This wedding hasn't been set up yet.",
		"setup.go_to_dashboard": "Go to the dashboard to get started.",

		// spotify playlists
		"home.soundtrack":                 "Soundtrack",
		"home.soundtrack_description":     "Help us out building the playlist for the big day by collaborating on Spotify!",
		"soundtrack.search_placeholder":   "Search for a song to add...",
		"soundtrack.invite_required":      "Open the page with your invite link to contribute",
		"soundtrack.added":                "Added to playlist!",
		"soundtrack.error":                "Could not add track, try again",
		"soundtrack.rate_limited":         "Too many requests, slow down",
		"dashboard.playlists":             "Collaborative Playlist",
		"dashboard.playlists_description": "Set the Spotify collaborative playlist embedded in the homepage soundtrack section",
		"label.playlist_url":              "Spotify playlist URL",
		"placeholder.playlist_url":        "https://open.spotify.com/playlist/...",

		// places
		"home.places":                     "Our places",
		"home.places_description":         "",
		"home.honeymoon":                  "Honeymoon",
		"home.honeymoon_description":      "",
		"dashboard.places":                "Places",
		"dashboard.places_description":    "Photos and locations that mark your story, shown on the homepage map",
		"dashboard.honeymoon":             "Honeymoon",
		"dashboard.honeymoon_description": "Locations, labels, and photos for the honeymoon journey shown on the homepage map",
		"label.place_label":               "Label",
		"label.place_address":             "Address",
		"label.place_date":                "Date",
		"label.place_image":               "Image",
		"placeholder.place_label":         "e.g. First date",
		"placeholder.place_address":       "Search for an address...",
		"btn.add_place":                   "Add place",
		"btn.add_honeymoon_location":      "Add honeymoon stop",

		// accommodation suggestions
		"home.accommodations":                   "Accommodations",
		"home.accommodations_description":       "Looking for a place to stay and not sure where to start? You can start with these suggestions below.",
		"dashboard.accommodations":              "Guest Accommodation Suggestions",
		"dashboard.accommodations_description":  "Suggestions shown at the end of the homepage for guests looking for a place to stay",
		"label.accommodation_name":              "Name",
		"label.accommodation_description":       "Description",
		"label.accommodation_url":               "Link",
		"placeholder.accommodation_name":        "e.g. Agriturismo Il Gelsomino",
		"placeholder.accommodation_description": "Optional note for guests",
		"placeholder.accommodation_url":         "https://...",
		"btn.add_accommodation":                 "Add suggestion",

		// impersonations
		"dashboard.impersonations":             "Impersonations",
		"dashboard.impersonations_description": "Define personas the chatbot can use to reply as",
		"placeholder.impersonation_codename":   "e.g. Anna",
		"placeholder.impersonation_profile":    "Describe how this person writes...",
		"btn.add_impersonation":                "Add impersonation",
		"chat.placeholder":                     "Ask us anything...",
		"chat.title":                           "Chat",
		"chat.send":                            "Send",
		"chat.error":                           "Something went wrong, try again",
		"chat.rate_limited":                    "Too many messages, slow down",
		"home.update_invitation":               "Invite",

		// footer
		"link.dashboard": "dashboard",
		"link.home":      "home",
	},
	"it": {
		// layout
		"title.dashboard":  "Pannello",
		"title.edit_guest": "Modifica Invitato",
		"title.edit_item":  "Modifica Articolo",
		"title.edit_gift":  "Modifica Dono",
		"title.edit_poll":  "Modifica Sondaggio",
		"title.setup":      "Configurazione necessaria",

		// dashboard - sections
		"dashboard.title":           "Pannello",
		"dashboard.details":         "Impostazioni",
		"dashboard.guests":          "Invitati",
		"dashboard.track_additions": "Brani aggiunti",

		// dashboard - labels
		"label.spouses":   "Nomi degli sposi",
		"label.ceremony":  "Rito",
		"label.reception": "Festa",

		// dashboard - counters
		"counter.rsvp":              "RSVP",
		"counter.visualization":     "Visualizzazioni",
		"counter.reach":             "Copertura inviti",
		"counter.confirmed":         "Confermati",
		"counter.refused":           "Rifiutati",
		"counter.pending_rsvp":      "In attesa",
		"counter.viewed":            "Visti",
		"counter.nonvisualized":     "Non visti",
		"counter.invited":           "Invitati",
		"counter.uninvited":         "Non invitati",
		"counter.confirmed_by_type": "Confermati per tipo",

		// dashboard - placeholders
		"placeholder.name":              "Nome",
		"placeholder.ceremony_address":  "Indirizzo cerimonia",
		"placeholder.reception_address": "Indirizzo ricevimento",
		"placeholder.first_name":        "Nome",
		"placeholder.last_name":         "Cognome",
		"placeholder.search":            "Cerca tra gli invitati...",
		"placeholder.invitation_search": "Cerca tra gli inviti...",

		// dashboard - guest form labels
		"label.first_name":    "Nome",
		"label.last_name":     "Cognome",
		"label.type":          "Tipo",
		"label.amount":        "Importo",
		"label.donor":         "Da",
		"label.registry_item": "Articolo lista",
		"label.confirmed":     "Confermato",
		// guest type enum (matches CHECK on guests.type)
		"guest_type.adult":  "Adulto",
		"guest_type.child":  "Bambino",
		"guest_type.infant": "Neonato",
		"guest_type.vendor": "Fornitore",
		// dashboard - table headers
		"th.first_name":    "Nome",
		"th.last_name":     "Cognome",
		"th.type":          "Tipo",
		"th.ceremony":      "Cerimonia",
		"th.reception":     "Ricevimento",
		"th.actions":       "Azioni",
		"th.confirmed":     "Confermato",
		"th.track_title":   "Titolo",
		"th.track_artist":  "Artista",
		"th.track_url":     "URL brano",
		"th.invite_id":     "ID invito",
		"th.invite_guests": "Invitati",

		// rsvp radio labels
		"rsvp.yes": "Sì",
		"rsvp.no":  "No",

		// dashboard - buttons & actions
		"btn.save_settings": "Salva",
		"btn.add":           "Aggiungi",
		"btn.import_csv":    "Importa CSV",
		"btn.update_guest":  "Aggiorna Invitato",
		"btn.update_item":   "Aggiorna Articolo",
		"btn.update_gift":   "Aggiorna Dono",
		"btn.update_poll":   "Aggiorna Sondaggio",
		"btn.edit":          "Modifica",
		"btn.delete":        "Elimina",
		"btn.reset_viewed":  "Reset visione",
		"btn.move_up":       "Sposta Su",
		"btn.move_down":     "Sposta Giù",
		"btn.back":          "Torna al Pannello",
		"btn.prev":          "Precedente",
		"btn.next":          "Successivo",

		// dashboard - misc
		"confirm.delete_guest": "Eliminare questo invitato?",

		// flash messages
		"flash.settings_saved":  "Impostazioni salvate.",
		"flash.guest_added":     "Invitato aggiunto.",
		"flash.guests_imported": "invitati importati.",
		"flash.guest_updated":   "Invitato aggiornato.",
		"flash.guest_deleted":   "Invitato eliminato.",
		"flash.item_updated":    "Articolo aggiornato.",
		"flash.gift_updated":    "Dono aggiornato.",
		"flash.gift_deleted":    "Dono eliminato.",

		// home
		"home.getting_married":    "Annunciamo il nostro matrimonio",
		"home.venues":             "Location",
		"home.venues_description": "",
		"home.directions":         "indicazioni",
		"home.ceremony":           "Il Rito",
		"home.reception":          "I Festeggiamenti",
		"home.guest_list":         "Lista Invitati",
		"home.gift":               "Regalo",
		"home.gift_description":   "Se desiderate onorarci con un dono, scegliete un importo",

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
		"dashboard.gifts": "Doni ricevuti",
		"th.donor":        "Da",
		"th.amount":       "Importo",
		"th.message":      "Messaggio",
		"th.date":         "Data",

		// registry
		"dashboard.registry":          "Lista dei desideri",
		"home.registry":               "Lista nozze",
		"home.registry_description":   "Nessuna idea? Ci pensiamo noi.",
		"placeholder.item_name":       "Nome articolo",
		"placeholder.item_price":      "Prezzo",
		"btn.add_item":                "Aggiungi",
		"btn.buy":                     "Dona",
		"badge.sold":                  "Offerto",
		"th.item_name":                "Nome",
		"th.item_price":               "Prezzo",
		"th.item_image":               "Immagine",
		"th.item_order":               "Ordine",
		"flash.item_added":            "Articolo aggiunto.",
		"flash.item_deleted":          "Articolo eliminato.",
		"confirm.delete_item":         "Eliminare questo articolo?",
		"confirm.delete_gift":         "Eliminare questo dono?",
		"status.confirmed":            "Sì",
		"status.pending_confirmation": "No",
		"option.no_registry_item":     "Nessun articolo collegato",

		// bank transfer modal
		"modal.transfer_intro":          "Effettua un bonifico con i dati seguenti per prenotare questo regalo.",
		"modal.iban":                    "IBAN",
		"modal.owner":                   "Intestatario",
		"modal.reason":                  "Causale",
		"modal.transfer_confirm_text":   "Una volta effettuato il bonifico, conferma qui sotto.",
		"btn.cancel":                    "Annulla",
		"btn.confirm":                   "Conferma",
		"btn.copy":                      "Copia",
		"btn.copied":                    "Copiato!",
		"btn.close":                     "Chiudi",
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
		"dashboard.polls":         "Sondaggi",
		"placeholder.question":    "Domanda",
		"placeholder.description": "Descrizione (facoltativa)",
		"th.question":             "Domanda",
		"th.description":          "Descrizione",
		"th.yes":                  "Sì",
		"th.no":                   "No",
		"flash.poll_added":        "Sondaggio aggiunto.",
		"flash.poll_updated":      "Sondaggio aggiornato.",
		"flash.poll_deleted":      "Sondaggio eliminato.",
		"confirm.delete_poll":     "Eliminare questo sondaggio?",
		"flash.track_deleted":     "Brano rimosso dallo storico.",
		"confirm.delete_track":    "Rimuovere questo brano dallo storico? La playlist Spotify non verrà modificata.",

		// invitations
		"dashboard.invitations":        "Inviti",
		"th.code":                      "Codice",
		"th.guests":                    "Invitati",
		"th.actioned":                  "Confermato",
		"th.viewed":                    "Visualizzato",
		"th.created":                   "Creato",
		"btn.create_invitation":        "Genera invito",
		"btn.update_invitation":        "Aggiorna invito",
		"flash.invitation_created":     "Invito creato:",
		"flash.invitation_updated":     "Invito aggiornato.",
		"flash.invitation_deleted":     "Invito eliminato.",
		"confirm.delete_invitation":    "Eliminare questo invito?",
		"th.label":                     "Etichetta",
		"label.invitation_label":       "Etichetta",
		"placeholder.invitation_label": "Etichetta",
		"hint.invitation_label":        "Usata come titolo OG. Lascia vuoto per generarla dagli invitati.",
		"title.edit_invitation":        "Modifica invito",

		// invitation public page
		"invitation.title":             "Invito",
		"invitation.rsvp":              "Vi aspettiamo",
		"invitation.rsvp_description":  "Fateci sapere se potrete essere con noi",
		"invitation.confirm":           "Conferma",
		"invitation.already_answered":  "Avete già inviato la vostra conferma.",
		"invitation.change_answer":     "modifica risposta",
		"invitation.go_home":           "vai alla homepage",
		"invitation.submitted_message": "Grazie! La vostra risposta è stata registrata.",
		"invitation.replay_animation":  "Rivedi animazione busta",
		"invitation.close_envelope":    "chiudi busta",
		"invitation.brought_to_you_by": "Creato da Davide Pucci",
		"invitation.poll_notes":        "Note",
		"placeholder.poll_notes":       "Aggiungi una nota",

		// ceremony page
		"ceremony.title": "Cerimonia",

		// setup guard
		"setup.not_ready":       "Questo matrimonio non è ancora stato configurato.",
		"setup.go_to_dashboard": "Vai al pannello per iniziare.",

		// spotify playlists
		"home.soundtrack":                 "Colonna sonora",
		"home.soundtrack_description":     "Dateci una mano a creare la playlist del grande giorno collaborando su Spotify!",
		"soundtrack.search_placeholder":   "Cerca un brano da aggiungere...",
		"soundtrack.invite_required":      "Apri la pagina con il tuo invito per contribuire",
		"soundtrack.added":                "Aggiunto alla playlist!",
		"soundtrack.error":                "Impossibile aggiungere il brano, riprova",
		"soundtrack.rate_limited":         "Troppe richieste, rallenta",
		"dashboard.playlists":             "Playlist Collaborativa",
		"dashboard.playlists_description": "Imposta la playlist collaborativa di Spotify incorporata nella sezione soundtrack della homepage",
		"label.playlist_url":              "URL playlist Spotify",
		"placeholder.playlist_url":        "https://open.spotify.com/playlist/...",

		// places
		"home.places":                     "I nostri luoghi",
		"home.places_description":         "",
		"home.honeymoon":                  "Viaggio di nozze",
		"home.honeymoon_description":      "",
		"dashboard.places":                "Luoghi",
		"dashboard.places_description":    "Foto e luoghi che segnano la vostra storia, mostrati nella mappa in homepage",
		"dashboard.honeymoon":             "Viaggio di nozze",
		"dashboard.honeymoon_description": "Luoghi, etichette e foto del viaggio di nozze mostrati nella mappa in homepage",
		"label.place_label":               "Etichetta",
		"label.place_address":             "Indirizzo",
		"label.place_date":                "Data",
		"label.place_image":               "Immagine",
		"placeholder.place_label":         "es. Primo appuntamento",
		"placeholder.place_address":       "Cerca un indirizzo...",
		"btn.add_place":                   "Aggiungi luogo",
		"btn.add_honeymoon_location":      "Aggiungi tappa",

		// accommodation suggestions
		"home.accommodations":                   "Alloggi",
		"home.accommodations_description":       "Se state cercando un alloggio e non sapete da dove partire, potete iniziare da questi suggerimenti.",
		"dashboard.accommodations":              "Suggerimenti alloggi per gli ospiti",
		"dashboard.accommodations_description":  "Suggerimenti mostrati alla fine della homepage per gli ospiti che cercano dove dormire",
		"label.accommodation_name":              "Nome",
		"label.accommodation_description":       "Descrizione",
		"label.accommodation_url":               "Link",
		"placeholder.accommodation_name":        "es. Agriturismo Il Gelsomino",
		"placeholder.accommodation_description": "Nota facoltativa per gli ospiti",
		"placeholder.accommodation_url":         "https://...",
		"btn.add_accommodation":                 "Aggiungi suggerimento",

		// impersonations
		"dashboard.impersonations":             "Impersonazioni",
		"dashboard.impersonations_description": "Definisci i personaggi che il chatbot può usare per rispondere",
		"placeholder.impersonation_codename":   "es. Anna",
		"placeholder.impersonation_profile":    "Descrivi come scrive questa persona...",
		"btn.add_impersonation":                "Aggiungi impersonazione",
		"chat.placeholder":                     "Chiedici qualcosa...",
		"chat.title":                           "Chat",
		"chat.send":                            "Invia",
		"chat.error":                           "Qualcosa è andato storto, riprova",
		"chat.rate_limited":                    "Troppi messaggi, rallenta",
		"home.update_invitation":               "Invito",

		// footer
		"link.dashboard": "pannello",
		"link.home":      "home",
	},
}

// HomepageKeys is the ordered list of i18n keys overridable from the dashboard.
var HomepageKeys = []string{
	"home.getting_married", "home.venues", "home.venues_description", "home.directions", "home.ceremony", "home.reception",
	"home.soundtrack", "home.soundtrack_description",
	"soundtrack.search_placeholder", "soundtrack.added", "soundtrack.error", "soundtrack.rate_limited",
	"soundtrack.invite_required",
	"home.places", "home.places_description",
	"home.honeymoon", "home.honeymoon_description",
	"home.accommodations", "home.accommodations_description",
	"home.registry", "home.registry_description",
	"home.generic_gift", "home.generic_gift_description", "home.remaining",
	"invitation.title", "invitation.rsvp", "invitation.rsvp_description",
	"invitation.confirm", "invitation.already_answered", "invitation.change_answer", "invitation.go_home", "invitation.submitted_message", "invitation.replay_animation", "invitation.close_envelope", "invitation.brought_to_you_by",
	"gift.thank_you", "gift.thanks_message",
	"badge.sold",
	"btn.buy", "btn.copy", "btn.copied", "btn.cancel", "btn.confirm", "btn.close",
	"modal.transfer_intro", "modal.iban", "modal.owner", "modal.reason", "modal.transfer_confirm_text",
	"placeholder.custom_amount", "placeholder.donor",
	"chat.title", "chat.placeholder", "chat.send", "chat.error", "chat.rate_limited",
	"home.update_invitation",
	"rsvp.yes", "rsvp.no",
	"th.amount", "th.ceremony", "th.reception",
	"link.dashboard",
}

// Defaults returns the default translations for the given language for all HomepageKeys.
func Defaults(lang string) map[string]string {
	t := NewT(lang)
	result := make(map[string]string, len(HomepageKeys))
	for _, key := range HomepageKeys {
		result[key] = t(key)
	}
	return result
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

// NewTWithOverrides returns a translator that checks overrides first, then falls back to NewT.
func NewTWithOverrides(lang string, overrides map[string]string) T {
	base := NewT(lang)
	return func(key string) string {
		if overrides != nil {
			if v, ok := overrides[key]; ok && v != "" {
				return v
			}
		}
		return base(key)
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

func parseDatetime(datetimeStr string) (time.Time, bool, bool) {
	t, err := time.Parse("2006-01-02T15:04", datetimeStr)
	hasTime := strings.Contains(datetimeStr, "T")
	if err != nil {
		t, err = time.Parse("2006-01-02", datetimeStr)
		if err != nil {
			return time.Time{}, false, hasTime
		}
	}
	return t, true, hasTime
}

// FormatDatetimeUniversal formats a datetime-local string as dd/mm/yyyy HH:MM.
func FormatDatetimeUniversal(datetimeStr string) string {
	t, ok, hasTime := parseDatetime(datetimeStr)
	if !ok {
		return datetimeStr
	}
	if hasTime {
		return t.Format("02/01/2006 15:04")
	}
	return t.Format("02/01/2006")
}

// FormatDatetime parses a datetime-local string (2006-01-02T15:04) and formats it locale-aware.
// falls back to date-only format (2006-01-02) if datetime parse fails.
// returns the raw string on total parse failure.
func FormatDatetime(datetimeStr, lang string) string {
	t, ok, hasTime := parseDatetime(datetimeStr)
	if !ok {
		return datetimeStr
	}

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

func FormatDatetimeDateLine(datetimeStr, lang string) string {
	t, ok, _ := parseDatetime(datetimeStr)
	if !ok {
		return datetimeStr
	}
	return FormatDate(t, lang)
}

func FormatDatetimeTimeLine(datetimeStr, lang string) string {
	t, ok, hasTime := parseDatetime(datetimeStr)
	if !ok || !hasTime {
		return ""
	}
	if lang == "it" {
		return t.Format("15:04")
	}
	return t.Format("3:04 PM")
}
