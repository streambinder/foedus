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

const guestsPerPage = 10

func DashboardIndex(c *fiber.Ctx) error {
	logger := handlerLogger(c)
	settings, err := database.GetAllSettings()
	if err != nil {
		logger.Error("dashboard failed to load settings", "error", err.Error())
		return c.Status(500).SendString("failed to load settings")
	}
	confirmedCeremony, confirmedReception, pendingGuests, totalGuests, err := database.CountConfirmed()
	if err != nil {
		logger.Error("dashboard failed to count guests", "error", err.Error())
		return c.Status(500).SendString("failed to count guests")
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
		"confirmed_ceremony", confirmedCeremony,
		"confirmed_reception", confirmedReception,
		"pending_guests", pendingGuests,
		"total_guests", totalGuests,
	)
	return Render(c, templates.Dashboard(settings, guests, gifts, registryItems, invitations, polls, soundtrackEvents, confirmedCeremony, confirmedReception, pendingGuests, totalGuests, page, totalPages, search, csrfToken, getFlash(c), getT(c), getLang(c)))
}

func resolveImageMediaID(rawImage, rawMediaID string, existingMediaID int, allowedAny bool) (int, error) {
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
		newID, err := database.InsertMedia(mime, data)
		if err != nil {
			return 0, err
		}
		// orphan cleanup: drop previous media row if it's being replaced
		if existingMediaID > 0 && existingMediaID != newID {
			_ = database.DeleteMedia(existingMediaID)
		}
		return newID, nil
	}

	keep := parseFormMediaID(rawMediaID)
	if keep == 0 {
		// cleared field — drop existing media if any
		if existingMediaID > 0 {
			_ = database.DeleteMedia(existingMediaID)
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
func resolveImageMediaIDFromSet(rawImage, rawMediaID string, allowedExistingIDs map[int]struct{}) (int, error) {
	image := strings.TrimSpace(rawImage)
	if image != "" {
		if err := validateBase64ImageAny(image); err != nil {
			return 0, err
		}
		mime, data, err := decodeDataURI(image)
		if err != nil {
			return 0, fiber.NewError(400, "invalid image data")
		}
		return database.InsertMedia(mime, data)
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

	keys := []string{
		"spouse1_name", "spouse2_name", "ceremony_datetime",
		"ceremony_address", "ceremony_location", "ceremony_city",
		"reception_address", "reception_location", "reception_city", "reception_datetime",
		"bank_account_iban", "bank_account_holder",
	}
	for _, key := range keys {
		val := c.FormValue(key)
		if err := database.UpdateSetting(key, val); err != nil {
			logger.Error("settings save failed", "key", key, "error", err.Error())
			return c.Status(500).SendString("failed to save settings")
		}
	}

	ceremonyMediaID, err := resolveImageMediaID(c.FormValue("ceremony_image"), c.FormValue("ceremony_media_id"), settings.CeremonyMediaID, true)
	if err != nil {
		logger.Warn("settings save rejected", "field", "ceremony_image", "error", err.Error())
		return c.Status(400).SendString(err.Error())
	}
	if err := database.UpdateSetting("ceremony_media_id", strconv.Itoa(ceremonyMediaID)); err != nil {
		logger.Error("settings save failed", "key", "ceremony_media_id", "error", err.Error())
		return c.Status(500).SendString("failed to save settings")
	}

	receptionMediaID, err := resolveImageMediaID(c.FormValue("reception_image"), c.FormValue("reception_media_id"), settings.ReceptionMediaID, true)
	if err != nil {
		logger.Warn("settings save rejected", "field", "reception_image", "error", err.Error())
		return c.Status(400).SendString(err.Error())
	}
	if err := database.UpdateSetting("reception_media_id", strconv.Itoa(receptionMediaID)); err != nil {
		logger.Error("settings save failed", "key", "reception_media_id", "error", err.Error())
		return c.Status(500).SendString("failed to save settings")
	}

	sharePreviewMediaID, err := resolveImageMediaID(c.FormValue("share_preview_image"), c.FormValue("share_preview_media_id"), settings.SharePreviewMediaID, true)
	if err != nil {
		logger.Warn("settings save rejected", "field", "share_preview_image", "error", err.Error())
		return c.Status(400).SendString(err.Error())
	}
	if err := database.UpdateSetting("share_preview_media_id", strconv.Itoa(sharePreviewMediaID)); err != nil {
		logger.Error("settings save failed", "key", "share_preview_media_id", "error", err.Error())
		return c.Status(500).SendString("failed to save settings")
	}

	// spotify playlist: store as a single-entry JSON array for backward compatibility
	playlist := strings.TrimSpace(c.FormValue("spotify_playlist"))
	var playlists []string
	if playlist != "" {
		playlists = append(playlists, playlist)
	}
	playlistsJSON, _ := json.Marshal(playlists)
	if err := database.UpdateSetting("spotify_playlists", string(playlistsJSON)); err != nil {
		logger.Error("settings save failed", "key", "spotify_playlists", "error", err.Error())
		return c.Status(500).SendString("failed to save settings")
	}

	// places: collect ordered place entries
	var places []models.Place
	for i := 0; ; i++ {
		label := strings.TrimSpace(c.FormValue(fmt.Sprintf("place_label_%d", i)))
		name := strings.TrimSpace(c.FormValue(fmt.Sprintf("place_name_%d", i)))
		address := strings.TrimSpace(c.FormValue(fmt.Sprintf("place_address_%d", i)))
		date := strings.TrimSpace(c.FormValue(fmt.Sprintf("place_date_%d", i)))
		mediaIDRaw := c.FormValue(fmt.Sprintf("place_media_id_%d", i))
		mediaID, err := resolveImageMediaIDFromSet(c.FormValue(fmt.Sprintf("place_image_%d", i)), mediaIDRaw, existingMediaIDs)
		if err != nil {
			logger.Warn("settings save rejected", "field", fmt.Sprintf("place_image_%d", i), "error", err.Error())
			return c.Status(400).SendString(err.Error())
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
	placesJSON, _ := json.Marshal(places)
	if err := database.UpdateSetting("places", string(placesJSON)); err != nil {
		logger.Error("settings save failed", "key", "places", "error", err.Error())
		return c.Status(500).SendString("failed to save settings")
	}

	// honeymoon locations: collect ordered location entries
	var honeymoonLocations []models.Place
	for i := 0; ; i++ {
		label := strings.TrimSpace(c.FormValue(fmt.Sprintf("honeymoon_label_%d", i)))
		name := strings.TrimSpace(c.FormValue(fmt.Sprintf("honeymoon_name_%d", i)))
		address := strings.TrimSpace(c.FormValue(fmt.Sprintf("honeymoon_address_%d", i)))
		date := strings.TrimSpace(c.FormValue(fmt.Sprintf("honeymoon_date_%d", i)))
		mediaIDRaw := c.FormValue(fmt.Sprintf("honeymoon_media_id_%d", i))
		mediaID, err := resolveImageMediaIDFromSet(c.FormValue(fmt.Sprintf("honeymoon_image_%d", i)), mediaIDRaw, existingMediaIDs)
		if err != nil {
			logger.Warn("settings save rejected", "field", fmt.Sprintf("honeymoon_image_%d", i), "error", err.Error())
			return c.Status(400).SendString(err.Error())
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
	honeymoonLocationsJSON, _ := json.Marshal(honeymoonLocations)
	if err := database.UpdateSetting("honeymoon_locations", string(honeymoonLocationsJSON)); err != nil {
		logger.Error("settings save failed", "key", "honeymoon_locations", "error", err.Error())
		return c.Status(500).SendString("failed to save settings")
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
	accommodationSuggestionsJSON, _ := json.Marshal(accommodationSuggestions)
	if err := database.UpdateSetting("accommodation_suggestions", string(accommodationSuggestionsJSON)); err != nil {
		logger.Error("settings save failed", "key", "accommodation_suggestions", "error", err.Error())
		return c.Status(500).SendString("failed to save settings")
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
	impersonationsJSON, _ := json.Marshal(impersonations)
	if err := database.UpdateSetting("impersonations", string(impersonationsJSON)); err != nil {
		logger.Error("settings save failed", "key", "impersonations", "error", err.Error())
		return c.Status(500).SendString("failed to save settings")
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
	homepageLabelsJSON, _ := json.Marshal(homepageLabels)
	if err := database.UpdateSetting("homepage_labels", string(homepageLabelsJSON)); err != nil {
		logger.Error("settings save failed", "key", "homepage_labels", "error", err.Error())
		return c.Status(500).SendString("failed to save settings")
	}

	// homepage hero backgrounds: collect desktop/mobile pairs
	backgroundCount, _ := strconv.Atoi(c.FormValue("homepage_hero_background_count"))
	var homepageHeroBackgrounds []models.HomepageHeroBackground
	for i := 0; i < backgroundCount; i++ {
		desktopMediaID, err := resolveImageMediaIDFromSet(c.FormValue(fmt.Sprintf("homepage_hero_background_desktop_%d", i)), c.FormValue(fmt.Sprintf("homepage_hero_background_desktop_media_id_%d", i)), existingMediaIDs)
		if err != nil {
			logger.Warn("settings save rejected", "field", fmt.Sprintf("homepage_hero_background_desktop_%d", i), "error", err.Error())
			return c.Status(400).SendString(err.Error())
		}
		mobileMediaID, err := resolveImageMediaIDFromSet(c.FormValue(fmt.Sprintf("homepage_hero_background_mobile_%d", i)), c.FormValue(fmt.Sprintf("homepage_hero_background_mobile_media_id_%d", i)), existingMediaIDs)
		if err != nil {
			logger.Warn("settings save rejected", "field", fmt.Sprintf("homepage_hero_background_mobile_%d", i), "error", err.Error())
			return c.Status(400).SendString(err.Error())
		}
		if desktopMediaID == 0 && mobileMediaID == 0 {
			continue
		}
		homepageHeroBackgrounds = append(homepageHeroBackgrounds, models.HomepageHeroBackground{
			DesktopMediaID: desktopMediaID,
			MobileMediaID:  mobileMediaID,
		})
	}
	homepageHeroBackgroundsJSON, _ := json.Marshal(homepageHeroBackgrounds)
	if err := database.UpdateSetting("homepage_hero_backgrounds", string(homepageHeroBackgroundsJSON)); err != nil {
		logger.Error("settings save failed", "key", "homepage_hero_backgrounds", "error", err.Error())
		return c.Status(500).SendString("failed to save settings")
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
			_ = database.DeleteMedia(id)
		}
	}

	logger.Info(
		"settings updated",
		"places", len(places),
		"honeymoon_locations", len(honeymoonLocations),
		"accommodation_suggestions", len(accommodationSuggestions),
		"impersonations", len(impersonations),
		"homepage_label_langs", len(homepageLabels),
		"hero_backgrounds", len(homepageHeroBackgrounds),
		"spotify_playlist_configured", playlist != "",
	)
	setFlash(c, getT(c)("flash.settings_saved"))
	return c.Redirect("/dashboard")
}

func AddGuest(c *fiber.Ctx) error {
	logger := handlerLogger(c)
	firstName := strings.TrimSpace(c.FormValue("first_name"))
	lastName := strings.TrimSpace(c.FormValue("last_name"))
	if err := database.CreateGuest(
		firstName,
		lastName,
	); err != nil {
		logger.Error("guest create failed", "first_name", firstName, "last_name", lastName, "error", err.Error())
		return c.Status(500).SendString("failed to add guest")
	}
	logger.Info("guest created", "first_name", firstName, "last_name", lastName)
	setFlash(c, getT(c)("flash.guest_added"))
	return c.Redirect("/dashboard")
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
		// join all columns into a single full name
		fullName := strings.TrimSpace(strings.Join(record, " "))
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
		if err := database.CreateGuest(firstName, lastName); err != nil {
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
	if err := database.UpdateGuest(
		id,
		firstName,
		lastName,
	); err != nil {
		logger.Error("guest update failed", "guest_id", id, "error", err.Error())
		return c.Status(500).SendString("failed to update guest")
	}
	logger.Info("guest updated", "guest_id", id, "first_name", firstName, "last_name", lastName)
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
	mediaID, err := resolveImageMediaID(c.FormValue("image"), "", 0, false)
	if err != nil {
		logger.Warn("registry item create rejected", "name", name, "error", err.Error())
		return c.Status(400).SendString(err.Error())
	}
	if err := database.CreateRegistryItem(name, price, mediaID); err != nil {
		// orphan: media row is now unreferenced, drop it
		if mediaID > 0 {
			_ = database.DeleteMedia(mediaID)
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
	mediaID, err := resolveImageMediaID(c.FormValue("image"), c.FormValue("media_id"), item.MediaID, true)
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
		_ = database.DeleteMedia(mediaIDToDrop)
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
	if err := validateGiftAssignment(amount, registryItemID, id); err != nil {
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

func validateGiftAssignment(amount int, registryItemID *int, excludeGiftID int) *fiber.Error {
	if registryItemID == nil {
		return nil
	}
	item, err := database.GetRegistryItem(*registryItemID)
	if err != nil {
		return fiber.NewError(404, "item not found")
	}
	claimed, err := database.GetClaimedAmountsByItemExcludingGift(excludeGiftID)
	if err != nil {
		return fiber.NewError(500, "failed to validate gift")
	}
	if amount > item.Price-claimed[item.ID] {
		return fiber.NewError(400, "amount exceeds remaining")
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
	return c.Redirect("/dashboard#dashboard-soundtrack-events")
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
