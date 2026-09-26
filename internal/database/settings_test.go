package database

import (
	"testing"

	"github.com/streambinder/foedus/internal/models"
)

// The point of splitting the settings blob was to put these values under the
// schema's control: real types instead of TEXT, and real FKs on every media
// reference. These tests pin both.

func TestSettingsRowIsSeededAndTyped(t *testing.T) {
	newTestDB(t)

	settings, err := GetSettings()
	if err != nil {
		t.Fatalf("settings row missing after Init: %v", err)
	}
	if settings.IsConfigured() {
		t.Error("a freshly seeded settings row reports itself configured")
	}

	mediaID, err := InsertMedia(DB, "image/png", []byte("bytes"))
	if err != nil {
		t.Fatal(err)
	}
	want := models.Settings{
		GroomName: "Davide", BrideName: "Agnese",
		CeremonyDatetime: "2026-07-17T15:00", CeremonyLat: 43.1054714, CeremonyLng: 12.3509192,
		CeremonyMediaID: mediaID, SpotifyPlaylist: "https://open.spotify.com/playlist/x",
	}
	if err := UpdateSettings(DB, want); err != nil {
		t.Fatal(err)
	}

	got, err := GetSettings()
	if err != nil {
		t.Fatal(err)
	}
	// float64 straight out of a REAL column — the KV table stored "43.1054714"
	// as text and re-parsed it on every read.
	if got.CeremonyLat != want.CeremonyLat || got.CeremonyLng != want.CeremonyLng {
		t.Errorf("coords round-tripped as %v,%v want %v,%v", got.CeremonyLat, got.CeremonyLng, want.CeremonyLat, want.CeremonyLng)
	}
	if got.CeremonyMediaID != mediaID {
		t.Errorf("ceremony media id %d, want %d", got.CeremonyMediaID, mediaID)
	}
	if !got.IsConfigured() {
		t.Error("settings with both names set report unconfigured")
	}
}

// An absent media id must reach the column as NULL: 0 has no matching media row
// and would trip the FK.
func TestSettingsAbsentMediaIDIsNull(t *testing.T) {
	newTestDB(t)

	if err := UpdateSettings(DB, models.Settings{GroomName: "A", BrideName: "B"}); err != nil {
		t.Fatalf("settings with no media ids rejected: %v", err)
	}
	settings, err := GetSettings()
	if err != nil {
		t.Fatal(err)
	}
	if settings.CeremonyMediaID != 0 || settings.SharePreviewMediaID != 0 {
		t.Errorf("NULL media ids read back as %d/%d, want 0/0", settings.CeremonyMediaID, settings.SharePreviewMediaID)
	}
}

func TestSettingsRejectsDanglingMediaReference(t *testing.T) {
	newTestDB(t)

	if err := UpdateSettings(DB, models.Settings{CeremonyMediaID: 9999}); err == nil {
		t.Fatal("expected the FK to reject a ceremony_media_id with no media row")
	}
}

func TestPlacesAreScopedByKindAndOrdered(t *testing.T) {
	newTestDB(t)

	story := []models.Place{
		{Label: "First date", Name: "Bar", Lat: 43.1, Lng: 12.3},
		{Label: "Proposal", Name: "Hill", Lat: 43.2, Lng: 12.4},
	}
	if err := ReplacePlaces(DB, models.PlaceKindStory, story); err != nil {
		t.Fatal(err)
	}
	if err := ReplacePlaces(DB, models.PlaceKindHoneymoon, []models.Place{{Label: "Tanzania"}}); err != nil {
		t.Fatal(err)
	}

	got, err := GetPlaces(models.PlaceKindStory)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d story places, want 2", len(got))
	}
	// slice order is the form order and must survive the round trip
	if got[0].Label != "First date" || got[1].Label != "Proposal" {
		t.Errorf("story places came back as %q, %q", got[0].Label, got[1].Label)
	}
	if got[0].SortOrder != 0 || got[1].SortOrder != 1 {
		t.Errorf("sort_order %d,%d want 0,1", got[0].SortOrder, got[1].SortOrder)
	}
	if got[0].Kind != models.PlaceKindStory {
		t.Errorf("kind %q, want %q", got[0].Kind, models.PlaceKindStory)
	}

	// replacing one kind must leave the other alone
	if err := ReplacePlaces(DB, models.PlaceKindStory, nil); err != nil {
		t.Fatal(err)
	}
	honeymoon, err := GetPlaces(models.PlaceKindHoneymoon)
	if err != nil {
		t.Fatal(err)
	}
	if len(honeymoon) != 1 {
		t.Fatalf("clearing story places left %d honeymoon places, want 1", len(honeymoon))
	}
}

// The blob could hold a media id for bytes that no longer existed; the column
// cannot. This is the regression the split was for.
func TestPlaceRejectsDanglingMediaReference(t *testing.T) {
	newTestDB(t)

	if err := ReplacePlaces(DB, models.PlaceKindStory, []models.Place{{Label: "x", MediaID: 9999}}); err == nil {
		t.Fatal("expected the FK to reject a place media_id with no media row")
	}
}

func TestHeroBackgroundsRoundTrip(t *testing.T) {
	newTestDB(t)

	desktopID, err := InsertMedia(DB, "image/jpeg", []byte("wide"))
	if err != nil {
		t.Fatal(err)
	}
	// mobile side left empty — the pair is valid with only one image
	if err := ReplaceHeroBackgrounds(DB, []models.HeroBackground{{DesktopMediaID: desktopID}}); err != nil {
		t.Fatal(err)
	}
	got, err := GetHeroBackgrounds()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].DesktopMediaID != desktopID || got[0].MobileMediaID != 0 {
		t.Fatalf("got %+v, want one pair with desktop=%d mobile=0", got, desktopID)
	}
}

func TestCollectionsReplaceWholesale(t *testing.T) {
	newTestDB(t)

	if err := ReplaceAccommodations(DB, []models.Accommodation{{Name: "One"}, {Name: "Two"}}); err != nil {
		t.Fatal(err)
	}
	if err := ReplaceAccommodations(DB, []models.Accommodation{{Name: "Only"}}); err != nil {
		t.Fatal(err)
	}
	accommodations, err := GetAccommodations()
	if err != nil {
		t.Fatal(err)
	}
	if len(accommodations) != 1 || accommodations[0].Name != "Only" {
		t.Fatalf("got %+v, want a single \"Only\"", accommodations)
	}

	if err := ReplaceImpersonations(DB, []models.Impersonation{{Codename: "Davide", Profile: "warm"}}); err != nil {
		t.Fatal(err)
	}
	count, err := CountImpersonations()
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("CountImpersonations = %d, want 1", count)
	}

	if err := ReplaceParkingSpots(DB, []models.ParkingSpot{{Lat: 43.1, Lng: 12.3}}); err != nil {
		t.Fatal(err)
	}
	spots, err := GetParkingSpots()
	if err != nil {
		t.Fatal(err)
	}
	if len(spots) != 1 || spots[0].Lat != 43.1 {
		t.Fatalf("got %+v, want one spot at 43.1", spots)
	}
}

func TestHomepageLabelsRoundTripPerLanguage(t *testing.T) {
	newTestDB(t)

	if err := ReplaceHomepageLabels(DB, map[string]map[string]string{
		"en": {"home.ceremony": "Ceremony"},
		"it": {"home.ceremony": "Cerimonia", "btn.buy": "Regala"},
	}); err != nil {
		t.Fatal(err)
	}

	italian, err := GetHomepageLabels("it")
	if err != nil {
		t.Fatal(err)
	}
	if len(italian) != 2 || italian["home.ceremony"] != "Cerimonia" {
		t.Fatalf("got %v, want the two italian overrides", italian)
	}

	all, err := GetAllHomepageLabels()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 || all["en"]["home.ceremony"] != "Ceremony" {
		t.Fatalf("got %v, want overrides for both languages", all)
	}

	// an unknown language is empty, not an error — every key falls through to
	// the compiled-in default
	missing, err := GetHomepageLabels("de")
	if err != nil {
		t.Fatal(err)
	}
	if len(missing) != 0 {
		t.Errorf("got %v for an unconfigured language, want empty", missing)
	}
}
