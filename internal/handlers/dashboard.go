package handlers

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/csv"
	"encoding/hex"
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
	settings, err := database.GetAllSettings()
	if err != nil {
		return c.Status(500).SendString("failed to load settings")
	}
	confirmedCeremony, confirmedReception, pendingGuests, totalGuests, err := database.CountConfirmed()
	if err != nil {
		return c.Status(500).SendString("failed to count guests")
	}
	gifts, err := database.GetAllGifts()
	if err != nil {
		return c.Status(500).SendString("failed to load gifts")
	}
	registryItems, err := database.GetAllRegistryItems()
	if err != nil {
		return c.Status(500).SendString("failed to load registry items")
	}
	search := strings.TrimSpace(c.Query("q"))
	page, _ := strconv.Atoi(c.Query("page", "1"))
	if page < 1 {
		page = 1
	}
	guests, filteredTotal, err := database.GetGuestsPaginated(page, guestsPerPage, search)
	if err != nil {
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
		return c.Status(500).SendString("failed to load invitations")
	}
	polls, err := database.GetAllPollsWithCounts()
	if err != nil {
		return c.Status(500).SendString("failed to load polls")
	}
	csrfToken, _ := c.Locals("csrf").(string)
	return Render(c, templates.Dashboard(settings, guests, gifts, registryItems, invitations, polls, confirmedCeremony, confirmedReception, pendingGuests, totalGuests, page, totalPages, search, csrfToken, getFlash(c), getT(c), getLang(c)))
}

func imageToken(image string) string {
	if image == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(image))
	return hex.EncodeToString(sum[:])
}

func resolveExistingImage(rawImage, rawToken, existingImage string) (string, error) {
	image := strings.TrimSpace(rawImage)
	if image != "" {
		if err := validateBase64ImageAny(image); err != nil {
			return "", err
		}
		return image, nil
	}
	if rawToken != "" && rawToken == imageToken(existingImage) {
		return existingImage, nil
	}
	if rawToken != "" {
		return "", fiber.NewError(400, "invalid image token")
	}
	return "", nil
}

func buildExistingImageMap(settings models.WeddingSettings) map[string]string {
	images := make(map[string]string)
	add := func(image string) {
		if image == "" {
			return
		}
		images[imageToken(image)] = image
	}

	add(settings.CeremonyImage)
	add(settings.ReceptionImage)
	add(settings.SharePreviewImage)
	for _, place := range settings.Places {
		add(place.Image)
	}
	for _, place := range settings.HoneymoonLocations {
		add(place.Image)
	}
	for _, bg := range settings.HomepageHeroBackgrounds {
		add(bg.DesktopImage)
		add(bg.MobileImage)
	}
	return images
}

func resolveMappedImage(rawImage, rawToken string, existingImages map[string]string) (string, error) {
	image := strings.TrimSpace(rawImage)
	if image != "" {
		if err := validateBase64ImageAny(image); err != nil {
			return "", err
		}
		return image, nil
	}
	if rawToken == "" {
		return "", nil
	}
	image, ok := existingImages[rawToken]
	if !ok {
		return "", fiber.NewError(400, "invalid image token")
	}
	return image, nil
}

func SaveSettings(c *fiber.Ctx) error {
	settings, err := database.GetAllSettings()
	if err != nil {
		return c.Status(500).SendString("failed to load settings")
	}
	existingImages := buildExistingImageMap(settings)

	keys := []string{
		"spouse1_name", "spouse2_name", "ceremony_datetime",
		"ceremony_address", "ceremony_location",
		"reception_address", "reception_location",
		"bank_account_iban", "bank_account_holder",
	}
	for _, key := range keys {
		val := c.FormValue(key)
		if err := database.UpdateSetting(key, val); err != nil {
			return c.Status(500).SendString("failed to save settings")
		}
	}

	ceremonyImage, err := resolveExistingImage(c.FormValue("ceremony_image"), c.FormValue("ceremony_image_token"), settings.CeremonyImage)
	if err != nil {
		return c.Status(400).SendString(err.Error())
	}
	if err := database.UpdateSetting("ceremony_image", ceremonyImage); err != nil {
		return c.Status(500).SendString("failed to save settings")
	}

	receptionImage, err := resolveExistingImage(c.FormValue("reception_image"), c.FormValue("reception_image_token"), settings.ReceptionImage)
	if err != nil {
		return c.Status(400).SendString(err.Error())
	}
	if err := database.UpdateSetting("reception_image", receptionImage); err != nil {
		return c.Status(500).SendString("failed to save settings")
	}

	sharePreviewImage, err := resolveExistingImage(c.FormValue("share_preview_image"), c.FormValue("share_preview_image_token"), settings.SharePreviewImage)
	if err != nil {
		return c.Status(400).SendString(err.Error())
	}
	if err := database.UpdateSetting("share_preview_image", sharePreviewImage); err != nil {
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
		return c.Status(500).SendString("failed to save settings")
	}

	// places: collect ordered place entries
	var places []models.Place
	for i := 0; ; i++ {
		label := strings.TrimSpace(c.FormValue(fmt.Sprintf("place_label_%d", i)))
		name := strings.TrimSpace(c.FormValue(fmt.Sprintf("place_name_%d", i)))
		address := strings.TrimSpace(c.FormValue(fmt.Sprintf("place_address_%d", i)))
		date := strings.TrimSpace(c.FormValue(fmt.Sprintf("place_date_%d", i)))
		image, err := resolveMappedImage(c.FormValue(fmt.Sprintf("place_image_%d", i)), c.FormValue(fmt.Sprintf("place_image_token_%d", i)), existingImages)
		if err != nil {
			return c.Status(400).SendString(err.Error())
		}
		if label == "" && name == "" && address == "" && date == "" && image == "" && c.FormValue(fmt.Sprintf("place_image_token_%d", i)) == "" {
			break
		}
		lat, _ := strconv.ParseFloat(c.FormValue(fmt.Sprintf("place_lat_%d", i)), 64)
		lng, _ := strconv.ParseFloat(c.FormValue(fmt.Sprintf("place_lng_%d", i)), 64)
		places = append(places, models.Place{
			Label:   label,
			Date:    date,
			Image:   image,
			Name:    name,
			Address: address,
			Lat:     lat,
			Lng:     lng,
		})
	}
	placesJSON, _ := json.Marshal(places)
	if err := database.UpdateSetting("places", string(placesJSON)); err != nil {
		return c.Status(500).SendString("failed to save settings")
	}

	// honeymoon locations: collect ordered location entries
	var honeymoonLocations []models.Place
	for i := 0; ; i++ {
		label := strings.TrimSpace(c.FormValue(fmt.Sprintf("honeymoon_label_%d", i)))
		name := strings.TrimSpace(c.FormValue(fmt.Sprintf("honeymoon_name_%d", i)))
		address := strings.TrimSpace(c.FormValue(fmt.Sprintf("honeymoon_address_%d", i)))
		date := strings.TrimSpace(c.FormValue(fmt.Sprintf("honeymoon_date_%d", i)))
		image, err := resolveMappedImage(c.FormValue(fmt.Sprintf("honeymoon_image_%d", i)), c.FormValue(fmt.Sprintf("honeymoon_image_token_%d", i)), existingImages)
		if err != nil {
			return c.Status(400).SendString(err.Error())
		}
		if label == "" && name == "" && address == "" && date == "" && image == "" && c.FormValue(fmt.Sprintf("honeymoon_image_token_%d", i)) == "" {
			break
		}
		lat, _ := strconv.ParseFloat(c.FormValue(fmt.Sprintf("honeymoon_lat_%d", i)), 64)
		lng, _ := strconv.ParseFloat(c.FormValue(fmt.Sprintf("honeymoon_lng_%d", i)), 64)
		honeymoonLocations = append(honeymoonLocations, models.Place{
			Label:   label,
			Date:    date,
			Image:   image,
			Name:    name,
			Address: address,
			Lat:     lat,
			Lng:     lng,
		})
	}
	honeymoonLocationsJSON, _ := json.Marshal(honeymoonLocations)
	if err := database.UpdateSetting("honeymoon_locations", string(honeymoonLocationsJSON)); err != nil {
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
		return c.Status(500).SendString("failed to save settings")
	}

	// homepage hero backgrounds: collect desktop/mobile pairs
	backgroundCount, _ := strconv.Atoi(c.FormValue("homepage_hero_background_count"))
	var homepageHeroBackgrounds []models.HomepageHeroBackground
	for i := 0; i < backgroundCount; i++ {
		desktopImage, err := resolveMappedImage(c.FormValue(fmt.Sprintf("homepage_hero_background_desktop_%d", i)), c.FormValue(fmt.Sprintf("homepage_hero_background_desktop_token_%d", i)), existingImages)
		if err != nil {
			return c.Status(400).SendString(err.Error())
		}
		mobileImage, err := resolveMappedImage(c.FormValue(fmt.Sprintf("homepage_hero_background_mobile_%d", i)), c.FormValue(fmt.Sprintf("homepage_hero_background_mobile_token_%d", i)), existingImages)
		if err != nil {
			return c.Status(400).SendString(err.Error())
		}
		if desktopImage == "" && mobileImage == "" {
			continue
		}
		homepageHeroBackgrounds = append(homepageHeroBackgrounds, models.HomepageHeroBackground{
			DesktopImage: desktopImage,
			MobileImage:  mobileImage,
		})
	}
	homepageHeroBackgroundsJSON, _ := json.Marshal(homepageHeroBackgrounds)
	if err := database.UpdateSetting("homepage_hero_backgrounds", string(homepageHeroBackgroundsJSON)); err != nil {
		return c.Status(500).SendString("failed to save settings")
	}

	setFlash(c, getT(c)("flash.settings_saved"))
	return c.Redirect("/dashboard")
}

func AddGuest(c *fiber.Ctx) error {
	if err := database.CreateGuest(
		c.FormValue("first_name"),
		c.FormValue("last_name"),
	); err != nil {
		return c.Status(500).SendString("failed to add guest")
	}
	setFlash(c, getT(c)("flash.guest_added"))
	return c.Redirect("/dashboard")
}

func ImportGuestsCSV(c *fiber.Ctx) error {
	fh, err := c.FormFile("csv_file")
	if err != nil {
		return c.Status(400).SendString("no file uploaded")
	}
	f, err := fh.Open()
	if err != nil {
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
	setFlash(c, strconv.Itoa(imported)+" "+getT(c)("flash.guests_imported"))
	return c.Redirect("/dashboard")
}

func CycleConfirmed(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).SendString("invalid id")
	}
	if err := database.CycleConfirmed(id, c.Params("field")); err != nil {
		return c.Status(400).SendString("invalid field")
	}
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
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).SendString("invalid id")
	}
	if err := database.UpdateGuest(
		id,
		c.FormValue("first_name"),
		c.FormValue("last_name"),
	); err != nil {
		return c.Status(500).SendString("failed to update guest")
	}
	setFlash(c, getT(c)("flash.guest_updated"))
	return c.Redirect("/dashboard")
}

func DeleteGuest(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).SendString("invalid id")
	}
	if err := database.DeleteGuest(id); err != nil {
		return c.Status(500).SendString("failed to delete guest")
	}
	setFlash(c, getT(c)("flash.guest_deleted"))
	return c.Redirect("/dashboard")
}

const maxImageBytes = 600 * 1024 // 600KB

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
	name := strings.TrimSpace(c.FormValue("name"))
	if name == "" {
		return c.Status(400).SendString("name is required")
	}
	price, err := strconv.Atoi(c.FormValue("price"))
	if err != nil || price < 0 {
		return c.Status(400).SendString("invalid price")
	}
	image := c.FormValue("image")
	if image != "" {
		if err := validateBase64Image(image); err != nil {
			return c.Status(400).SendString(err.Error())
		}
	}
	if err := database.CreateRegistryItem(name, price, image); err != nil {
		return c.Status(500).SendString("failed to add item")
	}
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
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).SendString("invalid id")
	}
	item, err := database.GetRegistryItem(id)
	if err != nil {
		return c.Status(404).SendString("item not found")
	}

	name := strings.TrimSpace(c.FormValue("name"))
	if name == "" {
		return c.Status(400).SendString("name is required")
	}
	price, err := strconv.Atoi(c.FormValue("price"))
	if err != nil || price < 0 {
		return c.Status(400).SendString("invalid price")
	}
	image := c.FormValue("image")
	if image == "" {
		image = item.Image
	}
	if image != "" {
		if err := validateBase64ImageAny(image); err != nil {
			return c.Status(400).SendString(err.Error())
		}
	}
	if err := database.UpdateRegistryItem(id, name, price, image); err != nil {
		return c.Status(500).SendString("failed to update item")
	}
	setFlash(c, getT(c)("flash.item_updated"))
	return c.Redirect("/dashboard")
}

func DeleteRegistryItem(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).SendString("invalid id")
	}
	if err := database.DeleteRegistryItem(id); err != nil {
		return c.Status(500).SendString("failed to delete item")
	}
	setFlash(c, getT(c)("flash.item_deleted"))
	return c.Redirect("/dashboard")
}

func MoveRegistryItem(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).SendString("invalid id")
	}
	direction := strings.TrimSpace(c.Params("direction"))
	if direction != "up" && direction != "down" {
		return c.Status(400).SendString("invalid direction")
	}
	if err := database.MoveRegistryItem(id, direction); err != nil {
		if err == sql.ErrNoRows {
			return c.Status(404).SendString("item not found")
		}
		return c.Status(500).SendString("failed to reorder item")
	}
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
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).SendString("invalid id")
	}
	if _, err := database.GetGift(id); err != nil {
		return c.Status(404).SendString("gift not found")
	}

	amount, err := strconv.Atoi(c.FormValue("amount"))
	if err != nil || amount < 1 {
		return c.Status(400).SendString("invalid amount")
	}
	registryItemID, err := parseGiftRegistryItemID(c.FormValue("registry_item_id"))
	if err != nil {
		return c.Status(400).SendString(err.Error())
	}
	if err := validateGiftAssignment(amount, registryItemID, id); err != nil {
		return c.Status(err.Code).SendString(err.Message)
	}

	if err := database.UpdateGift(
		id,
		amount,
		strings.TrimSpace(c.FormValue("donor")),
		registryItemID,
		c.FormValue("confirmed") == "on",
	); err != nil {
		return c.Status(500).SendString("failed to update gift")
	}
	setFlash(c, getT(c)("flash.gift_updated"))
	return c.Redirect("/dashboard")
}

func DeleteGift(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).SendString("invalid id")
	}
	if err := database.DeleteGift(id); err != nil {
		return c.Status(500).SendString("failed to delete gift")
	}
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
	raw := c.FormValue("guest_ids")
	if raw == "" {
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
		return c.Redirect("/dashboard")
	}
	code, err := database.CreateInvitation(guestIDs)
	if err != nil {
		return c.Status(500).SendString("failed to create invitation")
	}
	setFlash(c, getT(c)("flash.invitation_created")+" "+code)
	return c.Redirect("/dashboard")
}

func AddPoll(c *fiber.Ctx) error {
	question := strings.TrimSpace(c.FormValue("question"))
	if question == "" {
		return c.Redirect("/dashboard")
	}
	if err := database.CreatePoll(question); err != nil {
		return c.Status(500).SendString("failed to add poll")
	}
	setFlash(c, getT(c)("flash.poll_added"))
	return c.Redirect("/dashboard")
}

func DeletePoll(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).SendString("invalid id")
	}
	if err := database.DeletePoll(id); err != nil {
		return c.Status(500).SendString("failed to delete poll")
	}
	setFlash(c, getT(c)("flash.poll_deleted"))
	return c.Redirect("/dashboard")
}

func DeleteInvitation(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).SendString("invalid id")
	}
	if err := database.DeleteInvitation(id); err != nil {
		return c.Status(500).SendString("failed to delete invitation")
	}
	setFlash(c, getT(c)("flash.invitation_deleted"))
	return c.Redirect("/dashboard")
}
