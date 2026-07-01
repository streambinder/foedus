package handlers

import (
	"database/sql"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/streambinder/foedus/internal/database"
	"github.com/streambinder/foedus/internal/i18n"
	"github.com/streambinder/foedus/internal/models"
	"github.com/streambinder/foedus/internal/observability"
	"github.com/streambinder/foedus/templates"
)

func setFlash(c *fiber.Ctx, msg string) {
	c.Cookie(&fiber.Cookie{
		Name:    "flash",
		Value:   url.QueryEscape(msg),
		Expires: time.Now().Add(5 * time.Second),
		Path:    "/dashboard",
	})
}

func getFlash(c *fiber.Ctx) string {
	raw := c.Cookies("flash")
	if raw == "" {
		return ""
	}
	// expire cookie immediately
	c.Cookie(&fiber.Cookie{
		Name:    "flash",
		Value:   "",
		Expires: time.Now().Add(-1 * time.Hour),
		Path:    "/dashboard",
	})
	msg, _ := url.QueryUnescape(raw)
	return msg
}

func getT(c *fiber.Ctx) i18n.T {
	if t, ok := c.Locals("t").(i18n.T); ok {
		return t
	}
	return i18n.NewT("en")
}

func getLang(c *fiber.Ctx) string {
	if lang, ok := c.Locals("lang").(string); ok {
		return lang
	}
	return "en"
}

const (
	guestsPerPage      = 10
	invitationsPerPage = 10
	registryPerPage    = 10
	giftsPerPage       = 10
	soundtrackPerPage  = 10
)

// paginateSlice clamps page to [1,totalPages] and returns the requested window.
// Empty input returns nil + page=1, totalPages=1 so callers don't need guards.
func paginateSlice[T any](items []T, page, perPage int) (window []T, currentPage, totalPages int) {
	totalPages = (len(items) + perPage - 1) / perPage
	if totalPages < 1 {
		totalPages = 1
	}
	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}
	start := (page - 1) * perPage
	end := start + perPage
	if end > len(items) {
		end = len(items)
	}
	return items[start:end], page, totalPages
}

// filterInvitations narrows by case-insensitive substring match against the
// label or any guest's first/last name. Empty query returns input unchanged.
func filterInvitations(invs []models.Invitation, query string) []models.Invitation {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return invs
	}
	out := make([]models.Invitation, 0, len(invs))
	for _, inv := range invs {
		if strings.Contains(strings.ToLower(inv.Label), q) {
			out = append(out, inv)
			continue
		}
		for _, g := range inv.Guests {
			if strings.Contains(strings.ToLower(g.FirstName), q) || strings.Contains(strings.ToLower(g.LastName), q) {
				out = append(out, inv)
				break
			}
		}
	}
	return out
}

func DashboardIndex(c *fiber.Ctx) error {
	logger := handlerLogger(c)
	settings, err := database.GetAllSettings()
	if err != nil {
		logger.Error("dashboard failed to load settings", "error", err.Error())
		return c.Status(500).SendString("failed to load settings")
	}
	confirmedReception, refusedReception, pendingRSVP, invitedGuests, nonVisualizedInvited, totalGuests, err := database.CountConfirmed()
	if err != nil {
		logger.Error("dashboard failed to count guests", "error", err.Error())
		return c.Status(500).SendString("failed to count guests")
	}
	confirmedAdults, confirmedChildren, confirmedInfants, confirmedVendors, err := database.CountConfirmedByType()
	if err != nil {
		logger.Error("dashboard failed to count guests by type", "error", err.Error())
		return c.Status(500).SendString("failed to count guests by type")
	}
	gifts, err := database.GetAllGifts()
	if err != nil {
		logger.Error("dashboard failed to load gifts", "error", err.Error())
		return c.Status(500).SendString("failed to load gifts")
	}
	registryItems, err := database.GetAllRegistryItems()
	if err != nil {
		logger.Error("dashboard failed to load registry items", "error", err.Error())
		return c.Status(500).SendString("failed to load registry items")
	}
	search := strings.TrimSpace(c.Query("q"))
	page, _ := strconv.Atoi(c.Query("page", "1"))
	if page < 1 {
		page = 1
	}
	guests, filteredTotal, err := database.GetGuestsPaginated(page, guestsPerPage, search)
	if err != nil {
		logger.Error("dashboard failed to load guests", "page", page, "search_len", len(search), "error", err.Error())
		return c.Status(500).SendString("failed to load guests")
	}
	totalPages := (filteredTotal + guestsPerPage - 1) / guestsPerPage
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	invitations, err := database.GetAllInvitations()
	if err != nil {
		logger.Error("dashboard failed to load invitations", "error", err.Error())
		return c.Status(500).SendString("failed to load invitations")
	}
	inviteSearch := strings.TrimSpace(c.Query("iq"))
	filteredInvitations := filterInvitations(invitations, inviteSearch)
	invitePage, _ := strconv.Atoi(c.Query("ipage", "1"))
	if invitePage < 1 {
		invitePage = 1
	}
	inviteTotalPages := (len(filteredInvitations) + invitationsPerPage - 1) / invitationsPerPage
	if inviteTotalPages < 1 {
		inviteTotalPages = 1
	}
	if invitePage > inviteTotalPages {
		invitePage = inviteTotalPages
	}
	inviteStart := (invitePage - 1) * invitationsPerPage
	inviteEnd := inviteStart + invitationsPerPage
	if inviteEnd > len(filteredInvitations) {
		inviteEnd = len(filteredInvitations)
	}
	pagedInvitations := filteredInvitations[inviteStart:inviteEnd]
	polls, err := database.GetAllPollsWithCounts()
	if err != nil {
		logger.Error("dashboard failed to load polls", "error", err.Error())
		return c.Status(500).SendString("failed to load polls")
	}
	soundtrackEvents, err := database.GetAllSoundtrackEvents()
	if err != nil {
		logger.Error("dashboard failed to load soundtrack events", "error", err.Error())
		return c.Status(500).SendString("failed to load soundtrack events")
	}

	// table-level pagination — small datasets so slicing in Go is fine
	registryPage, _ := strconv.Atoi(c.Query("rpage", "1"))
	pagedRegistry, registryPage, registryTotalPages := paginateSlice(registryItems, registryPage, registryPerPage)
	giftsPage, _ := strconv.Atoi(c.Query("gpage", "1"))
	pagedGifts, giftsPage, giftsTotalPages := paginateSlice(gifts, giftsPage, giftsPerPage)
	var giftsTotal int
	for _, g := range gifts {
		giftsTotal += g.Amount
	}
	soundtrackPage, _ := strconv.Atoi(c.Query("spage", "1"))
	pagedSoundtrack, soundtrackPage, soundtrackTotalPages := paginateSlice(soundtrackEvents, soundtrackPage, soundtrackPerPage)

	csrfToken, _ := c.Locals("csrf").(string)
	logger.Info(
		"dashboard rendered",
		"guest_count", len(guests),
		"filtered_total", filteredTotal,
		"gift_count", len(gifts),
		"registry_count", len(registryItems),
		"invitation_count", len(invitations),
		"poll_count", len(polls),
		"soundtrack_event_count", len(soundtrackEvents),
		"page", page,
		"search_len", len(search),
		"confirmed_reception", confirmedReception,
		"refused_reception", refusedReception,
		"invited_guests", invitedGuests,
		"pending_rsvp", pendingRSVP,
		"non_visualized_invited", nonVisualizedInvited,
		"total_guests", totalGuests,
	)
	return Render(c, templates.Dashboard(settings, guests, pagedGifts, giftsPage, giftsTotalPages, len(gifts), giftsTotal, pagedRegistry, registryPage, registryTotalPages, registryItems, invitations, pagedInvitations, invitePage, inviteTotalPages, inviteSearch, polls, pagedSoundtrack, soundtrackPage, soundtrackTotalPages, confirmedReception, refusedReception, pendingRSVP, invitedGuests, nonVisualizedInvited, totalGuests, confirmedAdults, confirmedChildren, confirmedInfants, confirmedVendors, page, totalPages, search, csrfToken, getFlash(c), getT(c), getLang(c)))
}

func resolveImageMediaID(q database.Querier, rawImage, rawMediaID string, existingMediaID int, allowedAny bool) (int, error) {
	image := strings.TrimSpace(rawImage)
	if image != "" {
		if allowedAny {
			if err := validateBase64ImageAny(image); err != nil {
				return 0, err
			}
		} else {
			if err := validateBase64Image(image); err != nil {
				return 0, err
			}
		}
		mime, data, err := decodeDataURI(image)
		if err != nil {
			return 0, fiber.NewError(400, "invalid image data")
		}
		newID, err := database.InsertMedia(q, mime, data)
		if err != nil {
			return 0, err
		}
		// orphan cleanup: drop previous media row if it's being replaced
		if existingMediaID > 0 && existingMediaID != newID {
			_ = database.DeleteMedia(q, existingMediaID)
		}
		return newID, nil
	}

	keep := parseFormMediaID(rawMediaID)
	if keep == 0 {
		// cleared field — drop existing media if any
		if existingMediaID > 0 {
			_ = database.DeleteMedia(q, existingMediaID)
		}
		return 0, nil
	}
	if keep != existingMediaID {
		// trying to reference a different media id than what we have stored — refuse
		return 0, fiber.NewError(400, "invalid media id")
	}
	return keep, nil
}

// resolveImageMediaIDFromSet handles the multi-image case (places, hero backgrounds)
// where many fields share the same "existing ids" set: the kept id must match one of
// the previously stored media ids for this settings group.
func resolveImageMediaIDFromSet(q database.Querier, rawImage, rawMediaID string, allowedExistingIDs map[int]struct{}) (int, error) {
	image := strings.TrimSpace(rawImage)
	if image != "" {
		if err := validateBase64ImageAny(image); err != nil {
			return 0, err
		}
		mime, data, err := decodeDataURI(image)
		if err != nil {
			return 0, fiber.NewError(400, "invalid image data")
		}
		return database.InsertMedia(q, mime, data)
	}

	keep := parseFormMediaID(rawMediaID)
	if keep == 0 {
		return 0, nil
	}
	if _, ok := allowedExistingIDs[keep]; !ok {
		return 0, fiber.NewError(400, "invalid media id")
	}
	return keep, nil
}

func parseFormMediaID(raw string) int {
	id, _ := strconv.Atoi(strings.TrimSpace(raw))
	if id < 0 {
		return 0
	}
	return id
}

func decodeDataURI(dataURI string) (string, []byte, error) {
	idx := strings.Index(dataURI, ",")
	if idx == -1 {
		return "", nil, fiber.ErrBadRequest
	}
	header := dataURI[:idx]
	encoded := dataURI[idx+1:]
	mimeType := "image/png"
	if start := strings.Index(header, ":"); start != -1 {
		end := strings.Index(header, ";")
		if end != -1 {
			mimeType = header[start+1 : end]
		}
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", nil, err
	}
	return mimeType, data, nil
}

func collectExistingMediaIDs(settings models.WeddingSettings) map[int]struct{} {
	ids := make(map[int]struct{})
	add := func(id int) {
		if id > 0 {
			ids[id] = struct{}{}
		}
	}
	add(settings.CeremonyMediaID)
	add(settings.ReceptionMediaID)
	add(settings.SharePreviewMediaID)
	for _, place := range settings.Places {
		add(place.MediaID)
	}
	for _, place := range settings.HoneymoonLocations {
		add(place.MediaID)
	}
	for _, bg := range settings.HomepageHeroBackgrounds {
		add(bg.DesktopMediaID)
		add(bg.MobileMediaID)
	}
	return ids
}

func SaveSettings(c *fiber.Ctx) error {
	logger := handlerLogger(c)
	settings, err := database.GetAllSettings()
	if err != nil {
		logger.Error("settings save failed to load existing settings", "error", err.Error())
		return c.Status(500).SendString("failed to load settings")
	}
	existingMediaIDs := collectExistingMediaIDs(settings)

	// counts captured for the success log once the transaction commits.
	var placeCount, honeymoonCount, accommodationCount, impersonationCount, labelLangCount, backgroundLen int
	var playlistConfigured bool

	// everything below runs in one transaction: a rejected save (bad image, contract
	// drift, write failure) rolls back every prior write and every media insert/delete,
	// so the dashboard never ends up in a half-saved state and media bytes are never
	// dropped on a path that didn't actually persist. errors carry their HTTP status
	// via *fiber.Error so the single mapping after WithTx returns the right code.
	saveErr := database.WithTx(func(tx *sql.Tx) error {
		set := func(key, value string) error {
			if err := database.UpdateSetting(tx, key, value); err != nil {
				logger.Error("settings save failed", "key", key, "error", err.Error())
				return fiber.NewError(500, "failed to save settings")
			}
			return nil
		}

		for _, key := range []string{
			"spouse1_name", "spouse2_name", "ceremony_datetime",
			"ceremony_address", "ceremony_location", "ceremony_city",
			"reception_address", "reception_location", "reception_city", "reception_datetime",
			"bank_account_iban", "bank_account_holder",
		} {
			if err := set(key, c.FormValue(key)); err != nil {
				return err
			}
		}

		ceremonyMediaID, err := resolveImageMediaID(tx, c.FormValue("ceremony_image"), c.FormValue("ceremony_media_id"), settings.CeremonyMediaID, true)
		if err != nil {
			logger.Warn("settings save rejected", "field", "ceremony_image", "error", err.Error())
			return err
		}
		if err := set("ceremony_media_id", strconv.Itoa(ceremonyMediaID)); err != nil {
			return err
		}

		receptionMediaID, err := resolveImageMediaID(tx, c.FormValue("reception_image"), c.FormValue("reception_media_id"), settings.ReceptionMediaID, true)
		if err != nil {
			logger.Warn("settings save rejected", "field", "reception_image", "error", err.Error())
			return err
		}
		if err := set("reception_media_id", strconv.Itoa(receptionMediaID)); err != nil {
			return err
		}

		sharePreviewMediaID, err := resolveImageMediaID(tx, c.FormValue("share_preview_image"), c.FormValue("share_preview_media_id"), settings.SharePreviewMediaID, true)
		if err != nil {
			logger.Warn("settings save rejected", "field", "share_preview_image", "error", err.Error())
			return err
		}
		if err := set("share_preview_media_id", strconv.Itoa(sharePreviewMediaID)); err != nil {
			return err
		}

		// spotify playlist: store as a single-entry JSON array for backward compatibility
		playlist := strings.TrimSpace(c.FormValue("spotify_playlist"))
		var playlists []string
		if playlist != "" {
			playlists = append(playlists, playlist)
		}
		playlistConfigured = playlist != ""
		playlistsJSON, _ := json.Marshal(playlists)
		if err := set("spotify_playlists", string(playlistsJSON)); err != nil {
			return err
		}

		// places: collect ordered place entries
		var places []models.Place
		for i := 0; ; i++ {
			label := strings.TrimSpace(c.FormValue(fmt.Sprintf("place_label_%d", i)))
			name := strings.TrimSpace(c.FormValue(fmt.Sprintf("place_name_%d", i)))
			address := strings.TrimSpace(c.FormValue(fmt.Sprintf("place_address_%d", i)))
			date := strings.TrimSpace(c.FormValue(fmt.Sprintf("place_date_%d", i)))
			mediaIDRaw := c.FormValue(fmt.Sprintf("place_media_id_%d", i))
			mediaID, err := resolveImageMediaIDFromSet(tx, c.FormValue(fmt.Sprintf("place_image_%d", i)), mediaIDRaw, existingMediaIDs)
			if err != nil {
				logger.Warn("settings save rejected", "field", fmt.Sprintf("place_image_%d", i), "error", err.Error())
				return err
			}
			if label == "" && name == "" && address == "" && date == "" && mediaID == 0 && mediaIDRaw == "" {
				break
			}
			lat, _ := strconv.ParseFloat(c.FormValue(fmt.Sprintf("place_lat_%d", i)), 64)
			lng, _ := strconv.ParseFloat(c.FormValue(fmt.Sprintf("place_lng_%d", i)), 64)
			places = append(places, models.Place{
				Label:   label,
				Date:    date,
				MediaID: mediaID,
				Name:    name,
				Address: address,
				Lat:     lat,
				Lng:     lng,
			})
		}
		placeCount = len(places)
		placesJSON, _ := json.Marshal(places)
		if err := set("places", string(placesJSON)); err != nil {
			return err
		}

		// honeymoon locations: collect ordered location entries
		var honeymoonLocations []models.Place
		for i := 0; ; i++ {
			label := strings.TrimSpace(c.FormValue(fmt.Sprintf("honeymoon_label_%d", i)))
			name := strings.TrimSpace(c.FormValue(fmt.Sprintf("honeymoon_name_%d", i)))
			address := strings.TrimSpace(c.FormValue(fmt.Sprintf("honeymoon_address_%d", i)))
			date := strings.TrimSpace(c.FormValue(fmt.Sprintf("honeymoon_date_%d", i)))
			mediaIDRaw := c.FormValue(fmt.Sprintf("honeymoon_media_id_%d", i))
			mediaID, err := resolveImageMediaIDFromSet(tx, c.FormValue(fmt.Sprintf("honeymoon_image_%d", i)), mediaIDRaw, existingMediaIDs)
			if err != nil {
				logger.Warn("settings save rejected", "field", fmt.Sprintf("honeymoon_image_%d", i), "error", err.Error())
				return err
			}
			if label == "" && name == "" && address == "" && date == "" && mediaID == 0 && mediaIDRaw == "" {
				break
			}
			lat, _ := strconv.ParseFloat(c.FormValue(fmt.Sprintf("honeymoon_lat_%d", i)), 64)
			lng, _ := strconv.ParseFloat(c.FormValue(fmt.Sprintf("honeymoon_lng_%d", i)), 64)
			honeymoonLocations = append(honeymoonLocations, models.Place{
				Label:   label,
				Date:    date,
				MediaID: mediaID,
				Name:    name,
				Address: address,
				Lat:     lat,
				Lng:     lng,
			})
		}
		honeymoonCount = len(honeymoonLocations)
		honeymoonLocationsJSON, _ := json.Marshal(honeymoonLocations)
		if err := set("honeymoon_locations", string(honeymoonLocationsJSON)); err != nil {
			return err
		}

		// accommodation suggestions: collect name + description + url entries
		var accommodationSuggestions []models.AccommodationSuggestion
		for i := 0; ; i++ {
			name := strings.TrimSpace(c.FormValue(fmt.Sprintf("accommodation_name_%d", i)))
			if name == "" {
				break
			}
			accommodationSuggestions = append(accommodationSuggestions, models.AccommodationSuggestion{
				Name:        name,
				Description: strings.TrimSpace(c.FormValue(fmt.Sprintf("accommodation_description_%d", i))),
				URL:         strings.TrimSpace(c.FormValue(fmt.Sprintf("accommodation_url_%d", i))),
			})
		}
		accommodationCount = len(accommodationSuggestions)
		accommodationSuggestionsJSON, _ := json.Marshal(accommodationSuggestions)
		if err := set("accommodation_suggestions", string(accommodationSuggestionsJSON)); err != nil {
			return err
		}

		// impersonations: collect codename + profile pairs
		var impersonations []models.Impersonation
		for i := 0; ; i++ {
			codename := strings.TrimSpace(c.FormValue(fmt.Sprintf("impersonation_codename_%d", i)))
			if codename == "" {
				break
			}
			impersonations = append(impersonations, models.Impersonation{
				Codename: codename,
				Profile:  strings.TrimSpace(c.FormValue(fmt.Sprintf("impersonation_profile_%d", i))),
			})
		}
		impersonationCount = len(impersonations)
		impersonationsJSON, _ := json.Marshal(impersonations)
		if err := set("impersonations", string(impersonationsJSON)); err != nil {
			return err
		}

		// homepage_labels: collect per-lang per-key overrides
		homepageLabels := make(map[string]map[string]string)
		for _, lang := range []string{"en", "it"} {
			langOverrides := make(map[string]string)
			for _, key := range i18n.HomepageKeys {
				fieldName := "homepage_label_" + lang + "_" + key
				if v := strings.TrimSpace(c.FormValue(fieldName)); v != "" {
					langOverrides[key] = v
				}
			}
			if len(langOverrides) > 0 {
				homepageLabels[lang] = langOverrides
			}
		}
		labelLangCount = len(homepageLabels)
		homepageLabelsJSON, _ := json.Marshal(homepageLabels)
		if err := set("homepage_labels", string(homepageLabelsJSON)); err != nil {
			return err
		}

		// homepage hero backgrounds: collect desktop/mobile pairs
		backgroundCount, _ := strconv.Atoi(c.FormValue("homepage_hero_background_count"))
		var homepageHeroBackgrounds []models.HomepageHeroBackground
		for i := 0; i < backgroundCount; i++ {
			desktopMediaID, err := resolveImageMediaIDFromSet(tx, c.FormValue(fmt.Sprintf("homepage_hero_background_desktop_%d", i)), c.FormValue(fmt.Sprintf("homepage_hero_background_desktop_media_id_%d", i)), existingMediaIDs)
			if err != nil {
				logger.Warn("settings save rejected", "field", fmt.Sprintf("homepage_hero_background_desktop_%d", i), "error", err.Error())
				return err
			}
			mobileMediaID, err := resolveImageMediaIDFromSet(tx, c.FormValue(fmt.Sprintf("homepage_hero_background_mobile_%d", i)), c.FormValue(fmt.Sprintf("homepage_hero_background_mobile_media_id_%d", i)), existingMediaIDs)
			if err != nil {
				logger.Warn("settings save rejected", "field", fmt.Sprintf("homepage_hero_background_mobile_%d", i), "error", err.Error())
				return err
			}
			if desktopMediaID == 0 && mobileMediaID == 0 {
				continue
			}
			homepageHeroBackgrounds = append(homepageHeroBackgrounds, models.HomepageHeroBackground{
				DesktopMediaID: desktopMediaID,
				MobileMediaID:  mobileMediaID,
			})
		}
		// guard against a silent wipe: if the form claimed N background cards but every
		// media id resolved to 0, the submitted field names didn't match what we read
		// here (client/server contract drift) — not a genuine "user cleared all". bail
		// before persisting the empty value, otherwise the orphan cleanup below would
		// delete the still-referenced background media bytes.
		if backgroundCount > 0 && len(homepageHeroBackgrounds) == 0 {
			logger.Error("settings save rejected", "field", "homepage_hero_backgrounds", "reason", "declared cards resolved to zero media ids", "declared_count", backgroundCount)
			return fiber.NewError(400, "background image references missing — not saved")
		}
		backgroundLen = len(homepageHeroBackgrounds)
		homepageHeroBackgroundsJSON, _ := json.Marshal(homepageHeroBackgrounds)
		if err := set("homepage_hero_backgrounds", string(homepageHeroBackgroundsJSON)); err != nil {
			return err
		}

		// orphan cleanup for list-based images: drop media rows that were referenced
		// before the save but are no longer referenced after.
		keptIDs := make(map[int]struct{})
		keep := func(id int) {
			if id > 0 {
				keptIDs[id] = struct{}{}
			}
		}
		keep(ceremonyMediaID)
		keep(receptionMediaID)
		keep(sharePreviewMediaID)
		for _, place := range places {
			keep(place.MediaID)
		}
		for _, place := range honeymoonLocations {
			keep(place.MediaID)
		}
		for _, bg := range homepageHeroBackgrounds {
			keep(bg.DesktopMediaID)
			keep(bg.MobileMediaID)
		}
		for id := range existingMediaIDs {
			if _, stillUsed := keptIDs[id]; !stillUsed {
				if err := database.DeleteMedia(tx, id); err != nil {
					logger.Error("settings save failed", "key", "media_cleanup", "media_id", id, "error", err.Error())
					return fiber.NewError(500, "failed to save settings")
				}
			}
		}
		return nil
	})
	if saveErr != nil {
		if fe, ok := saveErr.(*fiber.Error); ok {
			return c.Status(fe.Code).SendString(fe.Message)
		}
		logger.Error("settings save transaction failed", "error", saveErr.Error())
		return c.Status(500).SendString("failed to save settings")
	}

	logger.Info(
		"settings updated",
		"places", placeCount,
		"honeymoon_locations", honeymoonCount,
		"accommodation_suggestions", accommodationCount,
		"impersonations", impersonationCount,
		"homepage_label_langs", labelLangCount,
		"hero_backgrounds", backgroundLen,
		"spotify_playlist_configured", playlistConfigured,
	)
	setFlash(c, getT(c)("flash.settings_saved"))
	return c.Redirect("/dashboard")
}

func AddGuest(c *fiber.Ctx) error {
	logger := handlerLogger(c)
	firstName := strings.TrimSpace(c.FormValue("first_name"))
	lastName := strings.TrimSpace(c.FormValue("last_name"))
	guestType := normalizeGuestType(c.FormValue("type"))
	if err := database.CreateGuest(
		firstName,
		lastName,
		guestType,
	); err != nil {
		logger.Error("guest create failed", "first_name", firstName, "last_name", lastName, "type", guestType, "error", err.Error())
		return c.Status(500).SendString("failed to add guest")
	}
	logger.Info("guest created", "first_name", firstName, "last_name", lastName, "type", guestType)
	setFlash(c, getT(c)("flash.guest_added"))
	return c.Redirect("/dashboard")
}

// normalizeGuestType maps a form value to a valid CHECK-allowed guest type.
// empty / unknown collapse to "adult" — the DB CHECK constraint enforces the
// closed set; this is just a defensive belt before the SQL layer.
func normalizeGuestType(raw string) string {
	switch strings.TrimSpace(raw) {
	case "child", "infant", "vendor":
		return raw
	default:
		return "adult"
	}
}

func ImportGuestsCSV(c *fiber.Ctx) error {
	logger := handlerLogger(c)
	fh, err := c.FormFile("csv_file")
	if err != nil {
		logger.Warn("guest import rejected", "reason", "missing file")
		return c.Status(400).SendString("no file uploaded")
	}
	f, err := fh.Open()
	if err != nil {
		logger.Error("guest import failed to open file", "filename", fh.Filename, "error", err.Error())
		return c.Status(500).SendString("failed to open file")
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1 // allow variable column count
	r.TrimLeadingSpace = true

	var imported int
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		// last column may be a guest type (adult|child|infant|vendor); peel it
		// off if it matches, otherwise treat the whole row as the name. keeps
		// single-column "Marco Rossi" rows working unchanged.
		guestType := "adult"
		nameCols := record
		if len(record) >= 2 {
			candidate := strings.ToLower(strings.TrimSpace(record[len(record)-1]))
			if candidate == "adult" || candidate == "child" || candidate == "infant" || candidate == "vendor" {
				guestType = candidate
				nameCols = record[:len(record)-1]
			}
		}
		fullName := strings.TrimSpace(strings.Join(nameCols, " "))
		if fullName == "" {
			continue
		}
		// skip header row
		lower := strings.ToLower(fullName)
		if strings.Contains(lower, "first") || strings.Contains(lower, "last") || strings.Contains(lower, "nome") || strings.Contains(lower, "cognome") {
			continue
		}
		// split by space: last token = last name, rest = first name
		parts := strings.Fields(fullName)
		var firstName, lastName string
		if len(parts) == 1 {
			firstName = parts[0]
		} else {
			lastName = parts[len(parts)-1]
			firstName = strings.Join(parts[:len(parts)-1], " ")
		}
		if err := database.CreateGuest(firstName, lastName, guestType); err != nil {
			continue
		}
		imported++
	}
	logger.Info("guest import completed", "filename", fh.Filename, "size_bytes", fh.Size, "imported", imported)
	setFlash(c, strconv.Itoa(imported)+" "+getT(c)("flash.guests_imported"))
	return c.Redirect("/dashboard")
}

func CycleConfirmed(c *fiber.Ctx) error {
	logger := handlerLogger(c)
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		logger.Warn("guest confirmation cycle rejected", "guest_id", c.Params("id"), "error", err.Error())
		return c.Status(400).SendString("invalid id")
	}
	if err := database.CycleConfirmed(id, c.Params("field")); err != nil {
		logger.Warn("guest confirmation cycle rejected", "guest_id", id, "field", c.Params("field"), "error", err.Error())
		return c.Status(400).SendString("invalid field")
	}
	logger.Info("guest confirmation cycled", "guest_id", id, "field", c.Params("field"))
	return c.Redirect("/dashboard")
}

func EditGuestPage(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).SendString("invalid id")
	}
	guest, err := database.GetGuest(id)
	if err != nil {
		return c.Status(404).SendString("guest not found")
	}
	settings, err := database.GetAllSettings()
	if err != nil {
		return c.Status(500).SendString("failed to load settings")
	}
	csrfToken, _ := c.Locals("csrf").(string)
	return Render(c, templates.EditGuest(guest, settings, csrfToken, getT(c), getLang(c)))
}

func UpdateGuest(c *fiber.Ctx) error {
	logger := handlerLogger(c)
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		logger.Warn("guest update rejected", "guest_id", c.Params("id"), "error", err.Error())
		return c.Status(400).SendString("invalid id")
	}
	firstName := strings.TrimSpace(c.FormValue("first_name"))
	lastName := strings.TrimSpace(c.FormValue("last_name"))
	guestType := normalizeGuestType(c.FormValue("type"))
	if err := database.UpdateGuest(
		id,
		firstName,
		lastName,
		guestType,
	); err != nil {
		logger.Error("guest update failed", "guest_id", id, "error", err.Error())
		return c.Status(500).SendString("failed to update guest")
	}
	logger.Info("guest updated", "guest_id", id, "first_name", firstName, "last_name", lastName, "type", guestType)
	setFlash(c, getT(c)("flash.guest_updated"))
	return c.Redirect("/dashboard")
}

func DeleteGuest(c *fiber.Ctx) error {
	logger := handlerLogger(c)
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		logger.Warn("guest delete rejected", "guest_id", c.Params("id"), "error", err.Error())
		return c.Status(400).SendString("invalid id")
	}
	if err := database.DeleteGuest(id); err != nil {
		logger.Error("guest delete failed", "guest_id", id, "error", err.Error())
		return c.Status(500).SendString("failed to delete guest")
	}
	logger.Info("guest deleted", "guest_id", id)
	setFlash(c, getT(c)("flash.guest_deleted"))
	return c.Redirect("/dashboard")
}

const maxImageBytes = 5 * 1024 * 1024 // 5MB

// validateBase64Image checks a base64 data URI for PNG prefix and size
func validateBase64Image(image string) error {
	const prefix = "data:image/png;base64,"
	if !strings.HasPrefix(image, prefix) {
		return fiber.NewError(400, "invalid image format")
	}
	decoded, err := base64.StdEncoding.DecodeString(image[len(prefix):])
	if err != nil {
		return fiber.NewError(400, "invalid image data")
	}
	if len(decoded) > maxImageBytes {
		return fiber.NewError(400, "image too large")
	}
	return nil
}

// validateBase64ImageAny accepts any image/* data URI, not just PNG
func validateBase64ImageAny(image string) error {
	if !strings.HasPrefix(image, "data:image/") {
		return fiber.NewError(400, "invalid image format")
	}
	idx := strings.Index(image, ";base64,")
	if idx == -1 {
		return fiber.NewError(400, "invalid image format")
	}
	decoded, err := base64.StdEncoding.DecodeString(image[idx+8:])
	if err != nil {
		return fiber.NewError(400, "invalid image data")
	}
	if len(decoded) > maxImageBytes {
		return fiber.NewError(400, "image too large")
	}
	return nil
}

func AddRegistryItem(c *fiber.Ctx) error {
	logger := handlerLogger(c)
	name := strings.TrimSpace(c.FormValue("name"))
	if name == "" {
		logger.Warn("registry item create rejected", "reason", "missing name")
		return c.Status(400).SendString("name is required")
	}
	price, err := strconv.Atoi(c.FormValue("price"))
	if err != nil || price < 0 {
		logger.Warn("registry item create rejected", "name", name, "price", c.FormValue("price"))
		return c.Status(400).SendString("invalid price")
	}
	mediaID, err := resolveImageMediaID(database.DB, c.FormValue("image"), "", 0, false)
	if err != nil {
		logger.Warn("registry item create rejected", "name", name, "error", err.Error())
		return c.Status(400).SendString(err.Error())
	}
	if err := database.CreateRegistryItem(name, price, mediaID); err != nil {
		// orphan: media row is now unreferenced, drop it
		if mediaID > 0 {
			_ = database.DeleteMedia(database.DB, mediaID)
		}
		logger.Error("registry item create failed", "name", name, "price", price, "error", err.Error())
		return c.Status(500).SendString("failed to add item")
	}
	logger.Info("registry item created", "name", name, "price", price, "has_image", mediaID > 0)
	setFlash(c, getT(c)("flash.item_added"))
	return c.Redirect("/dashboard")
}

func EditRegistryItemPage(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).SendString("invalid id")
	}
	item, err := database.GetRegistryItem(id)
	if err != nil {
		return c.Status(404).SendString("item not found")
	}
	settings, err := database.GetAllSettings()
	if err != nil {
		return c.Status(500).SendString("failed to load settings")
	}
	csrfToken, _ := c.Locals("csrf").(string)
	return Render(c, templates.EditRegistryItem(item, settings, csrfToken, getT(c), getLang(c)))
}

func UpdateRegistryItem(c *fiber.Ctx) error {
	logger := handlerLogger(c)
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		logger.Warn("registry item update rejected", "item_id", c.Params("id"), "error", err.Error())
		return c.Status(400).SendString("invalid id")
	}
	item, err := database.GetRegistryItem(id)
	if err != nil {
		logger.Warn("registry item update rejected", "item_id", id, "reason", "not found")
		return c.Status(404).SendString("item not found")
	}

	name := strings.TrimSpace(c.FormValue("name"))
	if name == "" {
		logger.Warn("registry item update rejected", "item_id", id, "reason", "missing name")
		return c.Status(400).SendString("name is required")
	}
	price, err := strconv.Atoi(c.FormValue("price"))
	if err != nil || price < 0 {
		logger.Warn("registry item update rejected", "item_id", id, "price", c.FormValue("price"))
		return c.Status(400).SendString("invalid price")
	}
	mediaID, err := resolveImageMediaID(database.DB, c.FormValue("image"), c.FormValue("media_id"), item.MediaID, true)
	if err != nil {
		logger.Warn("registry item update rejected", "item_id", id, "error", err.Error())
		return c.Status(400).SendString(err.Error())
	}
	if err := database.UpdateRegistryItem(id, name, price, mediaID); err != nil {
		logger.Error("registry item update failed", "item_id", id, "error", err.Error())
		return c.Status(500).SendString("failed to update item")
	}
	logger.Info("registry item updated", "item_id", id, "name", name, "price", price, "has_image", mediaID > 0)
	setFlash(c, getT(c)("flash.item_updated"))
	return c.Redirect("/dashboard")
}

func DeleteRegistryItem(c *fiber.Ctx) error {
	logger := handlerLogger(c)
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		logger.Warn("registry item delete rejected", "item_id", c.Params("id"), "error", err.Error())
		return c.Status(400).SendString("invalid id")
	}
	mediaIDToDrop := 0
	if item, err := database.GetRegistryItem(id); err == nil {
		mediaIDToDrop = item.MediaID
	}
	if err := database.DeleteRegistryItem(id); err != nil {
		logger.Error("registry item delete failed", "item_id", id, "error", err.Error())
		return c.Status(500).SendString("failed to delete item")
	}
	if mediaIDToDrop > 0 {
		_ = database.DeleteMedia(database.DB, mediaIDToDrop)
	}
	logger.Info("registry item deleted", "item_id", id)
	setFlash(c, getT(c)("flash.item_deleted"))
	return c.Redirect("/dashboard")
}

func MoveRegistryItem(c *fiber.Ctx) error {
	logger := handlerLogger(c)
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		logger.Warn("registry item reorder rejected", "item_id", c.Params("id"), "error", err.Error())
		return c.Status(400).SendString("invalid id")
	}
	direction := strings.TrimSpace(c.Params("direction"))
	if direction != "up" && direction != "down" {
		logger.Warn("registry item reorder rejected", "item_id", id, "direction", direction)
		return c.Status(400).SendString("invalid direction")
	}
	if err := database.MoveRegistryItem(id, direction); err != nil {
		if err == sql.ErrNoRows {
			logger.Warn("registry item reorder rejected", "item_id", id, "reason", "not found")
			return c.Status(404).SendString("item not found")
		}
		logger.Error("registry item reorder failed", "item_id", id, "direction", direction, "error", err.Error())
		return c.Status(500).SendString("failed to reorder item")
	}
	logger.Info("registry item reordered", "item_id", id, "direction", direction)
	return c.Redirect("/dashboard")
}

func EditGiftPage(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).SendString("invalid id")
	}
	gift, err := database.GetGift(id)
	if err != nil {
		return c.Status(404).SendString("gift not found")
	}
	registryItems, err := database.GetAllRegistryItems()
	if err != nil {
		return c.Status(500).SendString("failed to load registry items")
	}
	settings, err := database.GetAllSettings()
	if err != nil {
		return c.Status(500).SendString("failed to load settings")
	}
	csrfToken, _ := c.Locals("csrf").(string)
	return Render(c, templates.EditGift(gift, registryItems, settings, csrfToken, getT(c), getLang(c)))
}

func UpdateGift(c *fiber.Ctx) error {
	logger := handlerLogger(c)
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		logger.Warn("gift update rejected", "gift_id", c.Params("id"), "error", err.Error())
		return c.Status(400).SendString("invalid id")
	}
	if _, err := database.GetGift(id); err != nil {
		logger.Warn("gift update rejected", "gift_id", id, "reason", "not found")
		return c.Status(404).SendString("gift not found")
	}

	amount, err := strconv.Atoi(c.FormValue("amount"))
	if err != nil || amount < 1 {
		logger.Warn("gift update rejected", "gift_id", id, "amount", c.FormValue("amount"))
		return c.Status(400).SendString("invalid amount")
	}
	registryItemID, err := parseGiftRegistryItemID(c.FormValue("registry_item_id"))
	if err != nil {
		logger.Warn("gift update rejected", "gift_id", id, "error", err.Error())
		return c.Status(400).SendString(err.Error())
	}
	if err := validateGiftAssignment(registryItemID); err != nil {
		logger.Warn("gift update rejected", "gift_id", id, "error", err.Message)
		return c.Status(err.Code).SendString(err.Message)
	}

	donor := strings.TrimSpace(c.FormValue("donor"))
	if err := database.UpdateGift(
		id,
		amount,
		donor,
		registryItemID,
		c.FormValue("confirmed") == "on",
	); err != nil {
		logger.Error("gift update failed", "gift_id", id, "error", err.Error())
		return c.Status(500).SendString("failed to update gift")
	}
	logger.Info("gift updated", "gift_id", id, "amount", amount, "donor", observability.Redact(donor), "registry_item_id", registryItemID, "confirmed", c.FormValue("confirmed") == "on")
	setFlash(c, getT(c)("flash.gift_updated"))
	return c.Redirect("/dashboard")
}

func DeleteGift(c *fiber.Ctx) error {
	logger := handlerLogger(c)
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		logger.Warn("gift delete rejected", "gift_id", c.Params("id"), "error", err.Error())
		return c.Status(400).SendString("invalid id")
	}
	if err := database.DeleteGift(id); err != nil {
		logger.Error("gift delete failed", "gift_id", id, "error", err.Error())
		return c.Status(500).SendString("failed to delete gift")
	}
	logger.Info("gift deleted", "gift_id", id)
	setFlash(c, getT(c)("flash.gift_deleted"))
	return c.Redirect("/dashboard")
}

func parseGiftRegistryItemID(raw string) (*int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	id, err := strconv.Atoi(raw)
	if err != nil {
		return nil, fiber.NewError(400, "invalid registry item")
	}
	return &id, nil
}

// validateGiftAssignment checks the registry item exists.
// amount caps are only enforced on the public ClaimGift endpoint —
// the admin is the authority and can always edit/confirm gifts.
func validateGiftAssignment(registryItemID *int) *fiber.Error {
	if registryItemID == nil {
		return nil
	}
	if _, err := database.GetRegistryItem(*registryItemID); err != nil {
		return fiber.NewError(404, "item not found")
	}
	return nil
}

func CreateInvitation(c *fiber.Ctx) error {
	logger := handlerLogger(c)
	raw := c.FormValue("guest_ids")
	if raw == "" {
		logger.Warn("invitation create skipped", "reason", "missing guest ids")
		return c.Redirect("/dashboard")
	}
	parts := strings.Split(raw, ",")
	var guestIDs []int
	for _, p := range parts {
		id, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			continue
		}
		guestIDs = append(guestIDs, id)
	}
	if len(guestIDs) == 0 {
		logger.Warn("invitation create skipped", "reason", "no valid guest ids")
		return c.Redirect("/dashboard")
	}
	label := strings.TrimSpace(c.FormValue("label"))
	code, err := database.CreateInvitation(guestIDs, label)
	if err != nil {
		logger.Error("invitation create failed", "guest_count", len(guestIDs), "error", err.Error())
		return c.Status(500).SendString("failed to create invitation")
	}
	logger.Info("invitation created", "invitation_code", observability.Redact(code), "guest_count", len(guestIDs), "custom_label", label != "")
	setFlash(c, getT(c)("flash.invitation_created")+" "+code)
	// SPA fetch caller needs the code in a header to build the URL + copy to clipboard;
	// redirect responses would swallow the header after follow, so short-circuit here.
	c.Set("X-Invitation-Code", code)
	if c.Get("X-Requested-With") == "fetch" {
		return c.SendStatus(204)
	}
	return c.Redirect("/dashboard")
}

func AddPoll(c *fiber.Ctx) error {
	logger := handlerLogger(c)
	question := strings.TrimSpace(c.FormValue("question"))
	if question == "" {
		logger.Warn("poll create skipped", "reason", "missing question")
		return c.Redirect("/dashboard")
	}
	if err := database.CreatePoll(question, strings.TrimSpace(c.FormValue("description"))); err != nil {
		logger.Error("poll create failed", "question_len", len(question), "error", err.Error())
		return c.Status(500).SendString("failed to add poll")
	}
	logger.Info("poll created", "question_len", len(question))
	setFlash(c, getT(c)("flash.poll_added"))
	return c.Redirect("/dashboard")
}

func EditPollPage(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).SendString("invalid id")
	}
	poll, err := database.GetPoll(id)
	if err != nil {
		return c.Status(404).SendString("poll not found")
	}
	settings, err := database.GetAllSettings()
	if err != nil {
		return c.Status(500).SendString("failed to load settings")
	}
	csrfToken, _ := c.Locals("csrf").(string)
	return Render(c, templates.EditPoll(poll, settings, csrfToken, getT(c), getLang(c)))
}

func UpdatePoll(c *fiber.Ctx) error {
	logger := handlerLogger(c)
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		logger.Warn("poll update rejected", "poll_id", c.Params("id"), "error", err.Error())
		return c.Status(400).SendString("invalid id")
	}
	question := strings.TrimSpace(c.FormValue("question"))
	if question == "" {
		logger.Warn("poll update skipped", "poll_id", id, "reason", "missing question")
		return c.Redirect("/dashboard")
	}
	if err := database.UpdatePoll(id, question, strings.TrimSpace(c.FormValue("description"))); err != nil {
		logger.Error("poll update failed", "poll_id", id, "error", err.Error())
		return c.Status(500).SendString("failed to update poll")
	}
	logger.Info("poll updated", "poll_id", id, "question_len", len(question))
	setFlash(c, getT(c)("flash.poll_updated"))
	return c.Redirect("/dashboard")
}

func DeletePoll(c *fiber.Ctx) error {
	logger := handlerLogger(c)
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		logger.Warn("poll delete rejected", "poll_id", c.Params("id"), "error", err.Error())
		return c.Status(400).SendString("invalid id")
	}
	if err := database.DeletePoll(id); err != nil {
		logger.Error("poll delete failed", "poll_id", id, "error", err.Error())
		return c.Status(500).SendString("failed to delete poll")
	}
	logger.Info("poll deleted", "poll_id", id)
	setFlash(c, getT(c)("flash.poll_deleted"))
	return c.Redirect("/dashboard")
}

func DeleteSoundtrackEvent(c *fiber.Ctx) error {
	logger := handlerLogger(c)
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		logger.Warn("soundtrack event delete rejected", "event_id", c.Params("id"), "error", err.Error())
		return c.Status(400).SendString("invalid id")
	}
	if err := database.DeleteSoundtrackEvent(id); err != nil {
		logger.Error("soundtrack event delete failed", "event_id", id, "error", err.Error())
		return c.Status(500).SendString("failed to delete soundtrack event")
	}
	logger.Info("soundtrack event deleted", "event_id", id)
	setFlash(c, getT(c)("flash.track_deleted"))
	return c.Redirect("/dashboard")
}

func DeleteInvitation(c *fiber.Ctx) error {
	logger := handlerLogger(c)
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		logger.Warn("invitation delete rejected", "invitation_id", c.Params("id"), "error", err.Error())
		return c.Status(400).SendString("invalid id")
	}
	if err := database.DeleteInvitation(id); err != nil {
		logger.Error("invitation delete failed", "invitation_id", id, "error", err.Error())
		return c.Status(500).SendString("failed to delete invitation")
	}
	logger.Info("invitation deleted", "invitation_id", id)
	setFlash(c, getT(c)("flash.invitation_deleted"))
	return c.Redirect("/dashboard")
}

func ResetInvitationViewed(c *fiber.Ctx) error {
	logger := handlerLogger(c)
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		logger.Warn("invitation reset viewed rejected", "invitation_id", c.Params("id"), "error", err.Error())
		return c.Status(400).SendString("invalid id")
	}
	if err := database.ResetInvitationViewed(id); err != nil {
		logger.Error("invitation reset viewed failed", "invitation_id", id, "error", err.Error())
		return c.Status(500).SendString("failed to reset viewed")
	}
	logger.Info("invitation viewed reset", "invitation_id", id)
	return c.Redirect("/dashboard")
}

func EditInvitationPage(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).SendString("invalid id")
	}
	inv, err := database.GetInvitation(id)
	if err != nil {
		return c.Status(404).SendString("invitation not found")
	}
	settings, err := database.GetAllSettings()
	if err != nil {
		return c.Status(500).SendString("failed to load settings")
	}
	csrfToken, _ := c.Locals("csrf").(string)
	return Render(c, templates.EditInvitation(inv, settings, csrfToken, getT(c), getLang(c)))
}

func UpdateInvitation(c *fiber.Ctx) error {
	logger := handlerLogger(c)
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		logger.Warn("invitation update rejected", "invitation_id", c.Params("id"), "error", err.Error())
		return c.Status(400).SendString("invalid id")
	}
	label := strings.TrimSpace(c.FormValue("label"))
	if err := database.UpdateInvitationLabel(id, label); err != nil {
		logger.Error("invitation update failed", "invitation_id", id, "error", err.Error())
		return c.Status(500).SendString("failed to update invitation")
	}
	logger.Info("invitation updated", "invitation_id", id, "custom_label", label != "")
	setFlash(c, getT(c)("flash.invitation_updated"))
	return c.Redirect("/dashboard")
}

func CounterGuestNames(c *fiber.Ctx) error {
	logger := handlerLogger(c)
	category := c.Params("category")
	groups, err := database.GuestNameGroupsByCounter(category)
	if err == database.ErrUnknownCategory {
		logger.Warn("counter names unknown category", "category", category)
		return c.Status(400).JSON(fiber.Map{"error": "unknown category"})
	}
	if err != nil {
		logger.Error("counter names failed", "category", category, "error", err.Error())
		return c.Status(500).JSON(fiber.Map{"error": "failed to load names"})
	}
	if groups == nil {
		groups = [][]string{}
	}
	total := 0
	for _, g := range groups {
		total += len(g)
	}
	logger.Info("counter names", "category", category, "count", total)
	return c.JSON(fiber.Map{"category": category, "groups": groups})
}
