package handlers

import (
	"encoding/base64"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/streambinder/foedus/internal/database"
	"github.com/streambinder/foedus/internal/models"
	"github.com/streambinder/foedus/templates"
)

func baseSettingsForm() url.Values {
	form := url.Values{}
	form.Set("groom_name", " Davide ")
	form.Set("bride_name", "Agnese")
	form.Set("ceremony_datetime", "2027-07-17T15:30")
	form.Set("ceremony_address", "Via Roma 1")
	form.Set("ceremony_location", "Chiesa")
	form.Set("ceremony_city", "Terni")
	form.Set("ceremony_lat", "42.563")
	form.Set("ceremony_lng", "12.643")
	form.Set("reception_datetime", "2027-07-17T19:30")
	form.Set("reception_address", "Strada Verde 2")
	form.Set("reception_location", "Villa Verde")
	form.Set("reception_city", "Terni")
	form.Set("reception_lat", "42.6")
	form.Set("reception_lng", "12.7")
	form.Set("bank_account_iban", "IT60X0542811101000000123456")
	form.Set("bank_account_holder", "Davide Pucci")
	form.Set("spotify_playlist", "https://open.spotify.com/playlist/abc123XYZ")
	form.Set("envelope_color", "#A1B2C3")
	form.Set("page_bg_color", "")
	form.Set("home_bg_color", "#000000")
	return form
}

func TestSaveSettingsFull(t *testing.T) {
	setupDB(t)
	app := newTestApp()
	defer app.Shutdown()

	ceremonyMedia, err := database.InsertMedia(database.DB, "image/png", []byte("ceremony"))
	if err != nil {
		t.Fatal(err)
	}
	stampMedia, err := database.InsertMedia(database.DB, "image/png", []byte("stamp"))
	if err != nil {
		t.Fatal(err)
	}
	configureSettings(t, func(s *models.Settings) {
		s.CeremonyMediaID = ceremonyMedia
		s.StampMediaID = stampMedia
		s.PageBgColor = "#123456"
	})

	form := baseSettingsForm()
	// keep the ceremony image, clear the stamp, upload a reception image
	form.Set("ceremony_media_id", strconv.Itoa(ceremonyMedia))
	form.Set("reception_image", "data:image/jpeg;base64,"+base64.StdEncoding.EncodeToString([]byte("reception")))
	// one story place with a fresh upload, one that reuses the ceremony media
	form.Set("place_label_0", "2019")
	form.Set("place_name_0", "Bar Centrale")
	form.Set("place_image_0", pngDataURI(t, "place-img"))
	form.Set("place_label_1", "2021")
	form.Set("place_name_1", "Lago")
	form.Set("place_media_id_1", strconv.Itoa(ceremonyMedia))
	form.Set("place_lat_1", "42.1")
	form.Set("place_lng_1", "12.1")
	// one honeymoon stop without an image
	form.Set("honeymoon_label_0", "Viaggio")
	form.Set("honeymoon_name_0", "Serengeti")
	// parking: one real spot, one 0,0 row that is skipped
	form.Set("parking_lat_0", "42.5")
	form.Set("parking_lng_0", "12.6")
	form.Set("parking_lat_1", "0")
	form.Set("parking_lng_1", "0")
	// one accommodation and two impersonations
	form.Set("accommodation_name_0", "Hotel Uno")
	form.Set("accommodation_description_0", "Near the venue")
	form.Set("accommodation_url_0", "https://example.com")
	form.Set("impersonation_codename_0", "Nonna")
	form.Set("impersonation_profile_0", "Grandmother of the groom")
	form.Set("impersonation_codename_1", "Zio")
	// one hero background: new desktop upload, mobile reuses ceremony media
	form.Set("homepage_hero_background_count", "1")
	form.Set("homepage_hero_background_desktop_0", pngDataURI(t, "hero-desktop"))
	form.Set("homepage_hero_background_mobile_media_id_0", strconv.Itoa(ceremonyMedia))
	// one label override
	form.Set("homepage_label_en_home.ceremony", "The Rite")

	status, headers, _ := postForm(t, app, "/dashboard/settings", form)
	if status != 302 || headers.Get("Location") != "/dashboard" {
		t.Fatalf("save settings: status %d location %q", status, headers.Get("Location"))
	}

	saved, err := database.GetSettings()
	if err != nil {
		t.Fatal(err)
	}
	if saved.GroomName != "Davide" || saved.EnvelopeColor != "#a1b2c3" || saved.PageBgColor != "#123456" || saved.HomeBgColor != "#000000" {
		t.Errorf("saved scalars = %+v", saved)
	}
	if saved.CeremonyMediaID != ceremonyMedia {
		t.Errorf("ceremony media = %d, want kept %d", saved.CeremonyMediaID, ceremonyMedia)
	}
	if saved.StampMediaID != 0 {
		t.Errorf("stamp media = %d, want cleared", saved.StampMediaID)
	}
	if saved.ReceptionMediaID == 0 {
		t.Error("reception media was not uploaded")
	}
	if _, err := database.GetMedia(stampMedia); err == nil {
		t.Error("cleared stamp media row still exists")
	}

	story, err := database.GetPlaces(models.PlaceKindStory)
	if err != nil {
		t.Fatal(err)
	}
	if len(story) != 2 || story[0].MediaID == 0 || story[1].MediaID != ceremonyMedia {
		t.Errorf("story places = %+v", story)
	}
	honeymoon, _ := database.GetPlaces(models.PlaceKindHoneymoon)
	if len(honeymoon) != 1 || honeymoon[0].Name != "Serengeti" {
		t.Errorf("honeymoon places = %+v", honeymoon)
	}
	spots, _ := database.GetParkingSpots()
	if len(spots) != 1 || spots[0].Lat != 42.5 {
		t.Errorf("parking spots = %+v", spots)
	}
	stays, _ := database.GetAccommodations()
	if len(stays) != 1 || stays[0].Name != "Hotel Uno" {
		t.Errorf("accommodations = %+v", stays)
	}
	personas, _ := database.GetImpersonations()
	if len(personas) != 2 || personas[0].Codename != "Nonna" {
		t.Errorf("impersonations = %+v", personas)
	}
	heroes, _ := database.GetHeroBackgrounds()
	if len(heroes) != 1 || heroes[0].DesktopMediaID == 0 || heroes[0].MobileMediaID != ceremonyMedia {
		t.Errorf("hero backgrounds = %+v", heroes)
	}
	labels, _ := database.GetAllHomepageLabels()
	if labels["en"]["home.ceremony"] != "The Rite" {
		t.Errorf("homepage labels = %+v", labels)
	}
}

func TestSaveSettingsRejections(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(form url.Values)
	}{
		{"invalid colour", func(f url.Values) { f.Set("envelope_color", "red") }},
		{"invalid bg colour", func(f url.Values) { f.Set("home_bg_color", "#12345") }},
		{"invalid ceremony image", func(f url.Values) { f.Set("ceremony_image", "data:image/png;base64,%%%") }},
		{"foreign ceremony media", func(f url.Values) { f.Set("ceremony_media_id", "987") }},
		{"place foreign media", func(f url.Values) { f.Set("place_name_0", "X"); f.Set("place_media_id_0", "987") }},
		{"place invalid image", func(f url.Values) { f.Set("place_name_0", "X"); f.Set("place_image_0", "data:text/plain;base64,AAAA") }},
		{"honeymoon invalid image", func(f url.Values) { f.Set("honeymoon_name_0", "X"); f.Set("honeymoon_image_0", "not-a-data-uri") }},
		{"hero declared but empty", func(f url.Values) { f.Set("homepage_hero_background_count", "1") }},
		{"hero invalid desktop", func(f url.Values) {
			f.Set("homepage_hero_background_count", "1")
			f.Set("homepage_hero_background_desktop_0", "data:image/png;base64,%%%")
		}},
		{"hero invalid mobile", func(f url.Values) {
			f.Set("homepage_hero_background_count", "1")
			f.Set("homepage_hero_background_desktop_0", pngDataURI(t, "ok"))
			f.Set("homepage_hero_background_mobile_0", "data:image/png;base64,%%%")
		}},
		{"hero foreign media", func(f url.Values) {
			f.Set("homepage_hero_background_count", "1")
			f.Set("homepage_hero_background_desktop_media_id_0", "987")
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setupDB(t)
			configureSettings(t, nil)
			app := newTestApp()
			defer app.Shutdown()
			form := baseSettingsForm()
			tc.mutate(form)
			status, _, _ := postForm(t, app, "/dashboard/settings", form)
			if status != 400 {
				t.Errorf("status %d, want 400", status)
			}
			// a rejected save persists nothing
			saved, err := database.GetSettings()
			if err != nil {
				t.Fatal(err)
			}
			if saved.GroomName != "Davide" || saved.CeremonyAddress != "Via Roma 1" {
				t.Errorf("settings changed after rejection: %+v", saved)
			}
		})
	}
}

func TestSaveSettingsWriteFailure(t *testing.T) {
	setupDB(t)
	configureSettings(t, nil)
	app := newTestApp()
	defer app.Shutdown()
	// reads keep working but every insert into the labels table aborts, so
	// the transaction fails at the write stage and rolls everything back
	if _, err := database.DB.Exec(`CREATE TRIGGER fail_labels BEFORE INSERT ON homepage_labels
		BEGIN SELECT RAISE(ABORT, 'boom'); END`); err != nil {
		t.Fatal(err)
	}
	form := baseSettingsForm()
	form.Set("groom_name", "Changed")
	form.Set("homepage_label_en_home.ceremony", "The Rite")
	status, _, body := postForm(t, app, "/dashboard/settings", form)
	if status != 500 || !strings.Contains(body, "failed to save settings") {
		t.Errorf("status %d body %q", status, body)
	}
	saved, err := database.GetSettings()
	if err != nil {
		t.Fatal(err)
	}
	if saved.GroomName != "Davide" {
		t.Errorf("groom name = %q after rollback, want the original", saved.GroomName)
	}
}

func TestSaveSettingsLoadFailures(t *testing.T) {
	t.Run("content load", func(t *testing.T) {
		setupDB(t)
		configureSettings(t, nil)
		app := newTestApp()
		defer app.Shutdown()
		if _, err := database.DB.Exec(`DROP TABLE accommodations`); err != nil {
			t.Fatal(err)
		}
		status, _, body := postForm(t, app, "/dashboard/settings", baseSettingsForm())
		if status != 500 || !strings.Contains(body, "failed to load settings") {
			t.Errorf("status %d body %q", status, body)
		}
	})
	t.Run("settings load", func(t *testing.T) {
		setupDB(t)
		app := newTestApp()
		defer app.Shutdown()
		database.DB.Close()
		status, _, body := postForm(t, app, "/dashboard/settings", baseSettingsForm())
		if status != 500 || !strings.Contains(body, "failed to load settings") {
			t.Errorf("status %d body %q", status, body)
		}
	})
}

func TestSaveSettingsHeroMobileOnly(t *testing.T) {
	setupDB(t)
	configureSettings(t, nil)
	app := newTestApp()
	defer app.Shutdown()

	form := baseSettingsForm()
	form.Set("homepage_hero_background_count", "2")
	// first card mobile-only, second card desktop-only
	form.Set("homepage_hero_background_mobile_0", pngDataURI(t, "m0"))
	form.Set("homepage_hero_background_desktop_1", pngDataURI(t, "d1"))
	status, _, _ := postForm(t, app, "/dashboard/settings", form)
	if status != 302 {
		t.Fatalf("status %d, want 302", status)
	}
	heroes, err := database.GetHeroBackgrounds()
	if err != nil {
		t.Fatal(err)
	}
	if len(heroes) != 2 || heroes[0].MobileMediaID == 0 || heroes[1].DesktopMediaID == 0 {
		t.Errorf("heroes = %+v", heroes)
	}
}

func TestImageHelpers(t *testing.T) {
	setupDB(t)

	// parseFormMediaID
	for raw, want := range map[string]int{"": 0, " 12 ": 12, "abc": 0, "-4": 0, "0": 0} {
		if got := parseFormMediaID(raw); got != want {
			t.Errorf("parseFormMediaID(%q) = %d, want %d", raw, got, want)
		}
	}

	// parseCoord
	if got := parseCoord("42.5"); got != 42.5 {
		t.Errorf("parseCoord = %v", got)
	}
	if got := parseCoord("north"); got != 0 {
		t.Errorf("parseCoord invalid = %v", got)
	}
	if got := parseCoord("  "); got != 0 {
		t.Errorf("parseCoord blank = %v", got)
	}

	// validateBase64Image (PNG only) and validateBase64ImageAny
	good := pngDataURI(t, "img")
	if err := validateBase64Image(good); err != nil {
		t.Errorf("valid png rejected: %v", err)
	}
	if err := validateBase64Image("data:image/jpeg;base64,AAAA"); err == nil {
		t.Error("jpeg accepted by the PNG-only validator")
	}
	if err := validateBase64Image("data:image/png;base64,%%%"); err == nil {
		t.Error("broken base64 accepted")
	}
	big := "data:image/png;base64," + base64.StdEncoding.EncodeToString(make([]byte, maxImageBytes+1))
	if err := validateBase64Image(big); err == nil {
		t.Error("oversized image accepted")
	}
	if err := validateBase64ImageAny("data:image/jpeg;base64,AAAA"); err != nil {
		t.Errorf("jpeg rejected by the any validator: %v", err)
	}
	if err := validateBase64ImageAny("data:text/plain;base64,AAAA"); err == nil {
		t.Error("non-image accepted by the any validator")
	}
	if err := validateBase64ImageAny("data:image/png,AAAA"); err == nil {
		t.Error("missing base64 marker accepted")
	}
	if err := validateBase64ImageAny("data:image/png;base64,%%%"); err == nil {
		t.Error("broken base64 accepted by the any validator")
	}
	if err := validateBase64ImageAny("data:image/png;base64," + base64.StdEncoding.EncodeToString(make([]byte, maxImageBytes+1))); err == nil {
		t.Error("oversized image accepted by the any validator")
	}

	// decodeDataURI
	mime, data, err := decodeDataURI(good)
	if err != nil || mime != "image/png" || string(data) != "img" {
		t.Errorf("decodeDataURI = %q %q %v", mime, data, err)
	}
	mime, _, err = decodeDataURI("data:weird,AAAA")
	if err != nil || mime != "image/png" {
		t.Errorf("decodeDataURI without mime = %q %v", mime, err)
	}
	mime, _, err = decodeDataURI("plainprefix,AAAA")
	if err != nil || mime != "image/png" {
		t.Errorf("decodeDataURI without colon = %q %v", mime, err)
	}
	if _, _, err := decodeDataURI("no-comma-here"); err == nil {
		t.Error("comma-less data URI accepted")
	}
	if _, _, err := decodeDataURI("data:image/png;base64,%%%"); err == nil {
		t.Error("broken payload accepted")
	}
}

func TestResolveImageMediaID(t *testing.T) {
	setupDB(t)
	q := database.DB

	// fresh upload replaces and drops the previous row (any format allowed)
	oldID, err := database.InsertMedia(q, "image/png", []byte("old"))
	if err != nil {
		t.Fatal(err)
	}
	newID, err := resolveImageMediaID(q, "data:image/jpeg;base64,"+base64.StdEncoding.EncodeToString([]byte("new")), "", oldID, true)
	if err != nil {
		t.Fatal(err)
	}
	if newID == 0 || newID == oldID {
		t.Errorf("uploaded id = %d", newID)
	}
	if _, err := database.GetMedia(oldID); err == nil {
		t.Error("replaced media row still exists")
	}
	// PNG-only mode rejects a jpeg upload
	if _, err := resolveImageMediaID(q, "data:image/jpeg;base64,AAAA", "", 0, false); err == nil {
		t.Error("jpeg accepted in PNG-only mode")
	}
	// keep: the media id must match the stored one
	kept, err := resolveImageMediaID(q, "", strconv.Itoa(newID), newID, true)
	if err != nil || kept != newID {
		t.Errorf("keep = %d, %v", kept, err)
	}
	if _, err := resolveImageMediaID(q, "", "12345", newID, true); err == nil {
		t.Error("foreign media id accepted")
	}
	// clear: drops the stored row
	cleared, err := resolveImageMediaID(q, "", "", newID, true)
	if err != nil || cleared != 0 {
		t.Errorf("clear = %d, %v", cleared, err)
	}
	if _, err := database.GetMedia(newID); err == nil {
		t.Error("cleared media row still exists")
	}
	// clear with nothing stored is a no-op
	if id, err := resolveImageMediaID(q, "", "", 0, true); err != nil || id != 0 {
		t.Errorf("empty clear = %d, %v", id, err)
	}
	// a decodable-header but broken payload fails at decode time
	if _, err := resolveImageMediaID(q, "data:image/png;base64,%%%", "", 0, true); err == nil {
		t.Error("broken payload accepted")
	}

	// set-based resolution (places and hero backgrounds)
	set := map[int]struct{}{newID: {}}
	_ = set
	seedID, err := database.InsertMedia(q, "image/png", []byte("seed"))
	if err != nil {
		t.Fatal(err)
	}
	set = map[int]struct{}{seedID: {}}
	upID, err := resolveImageMediaIDFromSet(q, pngDataURI(t, "fresh"), "", set)
	if err != nil || upID == 0 {
		t.Errorf("set upload = %d, %v", upID, err)
	}
	kept, err = resolveImageMediaIDFromSet(q, "", strconv.Itoa(seedID), set)
	if err != nil || kept != seedID {
		t.Errorf("set keep = %d, %v", kept, err)
	}
	if _, err := resolveImageMediaIDFromSet(q, "", "999", set); err == nil {
		t.Error("unknown set reference accepted")
	}
	if id, err := resolveImageMediaIDFromSet(q, "", "", set); err != nil || id != 0 {
		t.Errorf("set empty = %d, %v", id, err)
	}
	if _, err := resolveImageMediaIDFromSet(q, "data:text/plain;base64,AAAA", "", set); err == nil {
		t.Error("non-image accepted by set resolution")
	}
}

func TestCollectExistingMediaIDs(t *testing.T) {
	settings := models.Settings{
		CeremonyMediaID: 1, ReceptionMediaID: 2, SharePreviewMediaID: 3,
		StampMediaID: 4, FloraBlMediaID: 0, FloraTrMediaID: 5,
	}
	content := templates.SettingsContent{
		Places:          []models.Place{{MediaID: 6}, {MediaID: 0}, {MediaID: 6}},
		Honeymoon:       []models.Place{{MediaID: 7}},
		HeroBackgrounds: []models.HeroBackground{{DesktopMediaID: 8, MobileMediaID: 9}, {DesktopMediaID: 8}},
	}
	ids := collectExistingMediaIDs(settings, content)
	for _, want := range []int{1, 2, 3, 4, 5, 6, 7, 8, 9} {
		if _, ok := ids[want]; !ok {
			t.Errorf("id %d missing from %v", want, ids)
		}
	}
	if len(ids) != 9 {
		t.Errorf("collected %d ids, want 9 unique", len(ids))
	}
	empty := collectExistingMediaIDs(models.Settings{}, templates.SettingsContent{})
	if len(empty) != 0 {
		t.Errorf("empty collection = %v", empty)
	}
}
