package templates

import (
	"strings"
	"testing"

	"github.com/streambinder/foedus/internal/models"
)

func intPtr(v int) *int { return &v }

func TestLocationDisplayAndMapsURL(t *testing.T) {
	if got := locationDisplay("Villa", "Via Roma 1"); got != "Villa, Via Roma 1" {
		t.Errorf("both = %q", got)
	}
	if got := locationDisplay("Villa", ""); got != "Villa" {
		t.Errorf("name only = %q", got)
	}
	if got := locationDisplay("", "Via Roma 1"); got != "Via Roma 1" {
		t.Errorf("address only = %q", got)
	}
	if got := locationDisplay("", ""); got != "" {
		t.Errorf("empty = %q", got)
	}
	got := mapsURL("Villa", "Via Roma 1")
	if !strings.HasPrefix(got, "https://www.google.com/maps/search/?api=1&query=") {
		t.Errorf("mapsURL = %q", got)
	}
	if !strings.Contains(got, "Villa") {
		t.Errorf("mapsURL missing escaped query: %q", got)
	}
}

func TestPaginationURLs(t *testing.T) {
	if got := paginationURL(2, ""); got != "/dashboard?page=2" {
		t.Errorf("paginationURL without search = %q", got)
	}
	if got := paginationURL(3, "marco"); got != "/dashboard?page=3&q=marco" {
		t.Errorf("paginationURL with search = %q", got)
	}
	if got := tablePaginationURL("rpage", 4); got != "/dashboard?rpage=4" {
		t.Errorf("tablePaginationURL = %q", got)
	}
	if got := invitePaginationURL(2, 1, "", ""); got != "/dashboard?ipage=2" {
		t.Errorf("invitePaginationURL minimal = %q", got)
	}
	if got := invitePaginationURL(2, 3, "", ""); got != "/dashboard?ipage=2&page=3" {
		t.Errorf("invitePaginationURL with guest page = %q", got)
	}
	got := invitePaginationURL(1, 1, "a b", "c&d")
	if !strings.Contains(got, "q=a+b") || !strings.Contains(got, "iq=c%26d") {
		t.Errorf("invitePaginationURL escaping = %q", got)
	}
}

func TestFormatAmounts(t *testing.T) {
	if got := formatGiftAmount(42); got != "42 €" {
		t.Errorf("formatGiftAmount = %q", got)
	}
	if got := formatPrice(0); got != "0 €" {
		t.Errorf("formatPrice = %q", got)
	}
}

func TestRegistryItemName(t *testing.T) {
	items := []models.RegistryItem{{ID: 1, Name: "One"}, {ID: 2, Name: "Two"}}
	if got := registryItemName(items, nil); got != "" {
		t.Errorf("nil id = %q", got)
	}
	if got := registryItemName(items, intPtr(2)); got != "Two" {
		t.Errorf("found = %q", got)
	}
	if got := registryItemName(items, intPtr(99)); got != "" {
		t.Errorf("missing = %q", got)
	}
	if got := registryItemName(nil, intPtr(1)); got != "" {
		t.Errorf("empty items = %q", got)
	}
}

func TestGuestInvitationCode(t *testing.T) {
	invs := []models.Invitation{{ID: 5, Code: "CODE5"}, {ID: 6, Code: "CODE6"}}
	if got := guestInvitationCode(invs, nil); got != "" {
		t.Errorf("nil id = %q", got)
	}
	if got := guestInvitationCode(invs, intPtr(6)); got != "CODE6" {
		t.Errorf("found = %q", got)
	}
	if got := guestInvitationCode(invs, intPtr(7)); got != "" {
		t.Errorf("missing = %q", got)
	}
}

func TestInvitationNamesAndLabels(t *testing.T) {
	inv := models.Invitation{Guests: []models.Guest{{FirstName: "Ada", LastName: "Love"}, {FirstName: "Bob", LastName: "Ray"}}}
	if got := invitationGuestNames(inv); got != "Ada Love, Bob Ray" {
		t.Errorf("names = %q", got)
	}
	if got := invitationGuestNames(models.Invitation{}); got != "" {
		t.Errorf("empty names = %q", got)
	}
	if got := invitationDisplayLabel(models.Invitation{Label: "  Custom  "}); got != "Custom" {
		t.Errorf("custom label = %q", got)
	}
	if got := invitationDisplayLabel(inv); got != "Ada & Bob" {
		t.Errorf("default label = %q", got)
	}
	if got := defaultInvitationLabel(nil); got != "" {
		t.Errorf("no guests = %q", got)
	}
	if got := defaultInvitationLabel([]models.Guest{{FirstName: "Ada"}}); got != "Ada" {
		t.Errorf("single = %q", got)
	}
	if got := defaultInvitationLabel([]models.Guest{{FirstName: "A"}, {FirstName: "B"}, {FirstName: "C"}}); got != "A + 2" {
		t.Errorf("trio = %q", got)
	}
}

func TestSoundtrackHelpers(t *testing.T) {
	if soundtrackHasInviteIDs(nil) {
		t.Error("no events must report false")
	}
	if soundtrackHasInviteIDs([]models.SoundtrackEvent{{InviteID: "  "}}) {
		t.Error("blank invite id must report false")
	}
	if !soundtrackHasInviteIDs([]models.SoundtrackEvent{{InviteID: " x "}}) {
		t.Error("present invite id must report true")
	}

	invs := []models.Invitation{
		{Code: "ABC", Guests: []models.Guest{{FirstName: " Ada "}, {FirstName: ""}, {FirstName: "Bob"}}},
		{Code: "XYZ", Guests: []models.Guest{{FirstName: "Cid"}}},
	}
	if got := soundtrackInviteGuestFirstNames(invs, ""); got != "" {
		t.Errorf("empty invite id = %q", got)
	}
	if got := soundtrackInviteGuestFirstNames(invs, "ABC"); got != "Ada, Bob" {
		t.Errorf("names = %q", got)
	}
	if got := soundtrackInviteGuestFirstNames(invs, "NOPE"); got != "" {
		t.Errorf("unknown code = %q", got)
	}
}

func TestGiftInvitationNames(t *testing.T) {
	invs := []models.Invitation{
		{ID: 3, Guests: []models.Guest{{FirstName: "Ada"}, {FirstName: " "}, {FirstName: "Bob"}}},
	}
	if got := giftInvitationGuestFirstNames(invs, intPtr(3)); got != "Ada, Bob" {
		t.Errorf("names = %q", got)
	}
	if got := giftInvitationGuestFirstNames(invs, intPtr(4)); got != "" {
		t.Errorf("unknown id = %q", got)
	}
}

func TestInvitationActionedAndSelectedAndMediaIDValue(t *testing.T) {
	tr := true
	if invitationActioned(models.Invitation{}) {
		t.Error("no guests must not be actioned")
	}
	if invitationActioned(models.Invitation{Guests: []models.Guest{{}}}) {
		t.Error("unanswered guests must not be actioned")
	}
	if !invitationActioned(models.Invitation{Guests: []models.Guest{{ConfirmedCeremony: &tr}}}) {
		t.Error("answered guest must be actioned")
	}
	if registryItemSelected(nil, 1) {
		t.Error("nil selection must be false")
	}
	if !registryItemSelected(intPtr(2), 2) {
		t.Error("matching selection must be true")
	}
	if registryItemSelected(intPtr(2), 3) {
		t.Error("different selection must be false")
	}
	if got := mediaIDValue(0); got != "" {
		t.Errorf("zero = %q", got)
	}
	if got := mediaIDValue(-1); got != "" {
		t.Errorf("negative = %q", got)
	}
	if got := mediaIDValue(12); got != "12" {
		t.Errorf("positive = %q", got)
	}
}

func TestAppearanceColorOrDefault(t *testing.T) {
	if got := appearanceColorOrDefault("#123abc", "#ffffff"); got != "#123abc" {
		t.Errorf("valid = %q", got)
	}
	if got := appearanceColorOrDefault("red", "#ffffff"); got != "#ffffff" {
		t.Errorf("invalid = %q", got)
	}
}

func TestDonutMath(t *testing.T) {
	if got := donutDashArray(0, 10); got != "0 100" {
		t.Errorf("zero value = %q", got)
	}
	if got := donutDashArray(5, 0); got != "0 100" {
		t.Errorf("zero total = %q", got)
	}
	if got := donutDashArray(1, 4); got != "25.0000 75.0000" {
		t.Errorf("quarter = %q", got)
	}
	if got := donutDashOffset(0, 0); got != "0" {
		t.Errorf("zero total offset = %q", got)
	}
	if got := donutDashOffset(1, 4); got != "-25.0000" {
		t.Errorf("offset = %q", got)
	}
	if got := donutDashOffset(0, 4); got != "-0.0000" {
		t.Errorf("zero offset = %q", got)
	}
}

func TestRemainingAmountAndClaimedPercent(t *testing.T) {
	item := models.RegistryItem{ID: 1, Price: 100}
	if got := remainingAmount(models.RegistryItem{ID: 1, Price: 0}, nil); got != -1 {
		t.Errorf("free item remaining = %d, want -1", got)
	}
	if got := remainingAmount(item, nil); got != 100 {
		t.Errorf("unclaimed remaining = %d", got)
	}
	if got := remainingAmount(item, map[int]int{1: 30}); got != 70 {
		t.Errorf("partial remaining = %d", got)
	}
	if got := remainingAmount(item, map[int]int{1: 100}); got != 0 {
		t.Errorf("full remaining = %d", got)
	}
	if got := remainingAmount(item, map[int]int{1: 150}); got != 0 {
		t.Errorf("over-claimed remaining = %d", got)
	}

	if got := claimedPercent(models.RegistryItem{ID: 1, Price: 0}, nil); got != 0 {
		t.Errorf("free percent = %d", got)
	}
	if got := claimedPercent(item, nil); got != 0 {
		t.Errorf("unclaimed percent = %d", got)
	}
	if got := claimedPercent(item, map[int]int{1: 25}); got != 25 {
		t.Errorf("quarter percent = %d", got)
	}
	if got := claimedPercent(item, map[int]int{1: 100}); got != 100 {
		t.Errorf("full percent = %d", got)
	}
	if got := claimedPercent(item, map[int]int{1: 250}); got != 100 {
		t.Errorf("over percent = %d", got)
	}
}

func TestSpotifyEmbedHelpers(t *testing.T) {
	url := "https://open.spotify.com/playlist/37i9dQZF1DXcBWIGoYBM5M"
	if got := spotifyEmbedURL(url); got != "https://open.spotify.com/embed/playlist/37i9dQZF1DXcBWIGoYBM5M?utm_source=generator&theme=0" {
		t.Errorf("embed url = %q", got)
	}
	if got := spotifyEmbedURL("https://example.com/nope"); got != "" {
		t.Errorf("non-playlist = %q", got)
	}
	if got := spotifyEmbedURL(""); got != "" {
		t.Errorf("empty = %q", got)
	}
	if !hasSpotifyPlaylist(url) {
		t.Error("playlist url must report true")
	}
	if hasSpotifyPlaylist("") {
		t.Error("empty must report false")
	}
}

func TestTimelineAndParkingJSON(t *testing.T) {
	places := []models.Place{
		{Label: "First", Date: "2020-05-01T10:00", Name: "Bar", Address: "Via X", Lat: 1.5, Lng: 2.5, MediaID: 9},
		{Label: "Second", Name: "Hill"},
	}
	got := locationTimelineJSON(places, "en")
	if !strings.Contains(got, `"label":"First"`) || !strings.Contains(got, `"image":"/media/9"`) {
		t.Errorf("timeline json = %s", got)
	}
	if got := locationTimelineJSON(nil, "it"); got != "[]" {
		t.Errorf("empty timeline = %q", got)
	}
	got = parkingJSON([]models.ParkingSpot{{Lat: 43.1, Lng: 12.3}})
	if got != `[{"lat":43.1,"lng":12.3}]` {
		t.Errorf("parking json = %q", got)
	}
	if got := parkingJSON(nil); got != "[]" {
		t.Errorf("empty parking = %q", got)
	}
}

func TestSectionNavigationHelpers(t *testing.T) {
	place := []models.Place{{}}
	playlist := "https://open.spotify.com/playlist/abc123"

	if got := prevSectionBeforeRegistry(nil, place, ""); got != "#honeymoon" {
		t.Errorf("prev registry honeymoon = %q", got)
	}
	if got := prevSectionBeforeRegistry(place, nil, ""); got != "#places" {
		t.Errorf("prev registry places = %q", got)
	}
	if got := prevSectionBeforeRegistry(nil, nil, playlist); got != "#soundtrack" {
		t.Errorf("prev registry soundtrack = %q", got)
	}
	if got := prevSectionBeforeRegistry(nil, nil, ""); got != "#venues" {
		t.Errorf("prev registry venues = %q", got)
	}

	if got := nextSectionAfterVenues(playlist, nil, nil, false); got != "#soundtrack" {
		t.Errorf("after venues soundtrack = %q", got)
	}
	if got := nextSectionAfterVenues("", place, nil, false); got != "#places" {
		t.Errorf("after venues places = %q", got)
	}
	if got := nextSectionAfterVenues("", nil, place, false); got != "#honeymoon" {
		t.Errorf("after venues honeymoon = %q", got)
	}
	if got := nextSectionAfterVenues("", nil, nil, true); got != "#registry" {
		t.Errorf("after venues registry = %q", got)
	}
	if got := nextSectionAfterVenues("", nil, nil, false); got != "" {
		t.Errorf("after venues none = %q", got)
	}

	if got := nextSectionAfterSoundtrack(place, nil, false); got != "#places" {
		t.Errorf("after soundtrack places = %q", got)
	}
	if got := nextSectionAfterSoundtrack(nil, place, false); got != "#honeymoon" {
		t.Errorf("after soundtrack honeymoon = %q", got)
	}
	if got := nextSectionAfterSoundtrack(nil, nil, true); got != "#registry" {
		t.Errorf("after soundtrack registry = %q", got)
	}
	if got := nextSectionAfterSoundtrack(nil, nil, false); got != "" {
		t.Errorf("after soundtrack none = %q", got)
	}

	if got := nextSectionAfterPlaces(place, false); got != "#honeymoon" {
		t.Errorf("after places honeymoon = %q", got)
	}
	if got := nextSectionAfterPlaces(nil, true); got != "#registry" {
		t.Errorf("after places registry = %q", got)
	}
	if got := nextSectionAfterPlaces(nil, false); got != "" {
		t.Errorf("after places none = %q", got)
	}

	if got := prevSectionBeforeHoneymoon(place, ""); got != "#places" {
		t.Errorf("prev honeymoon places = %q", got)
	}
	if got := prevSectionBeforeHoneymoon(nil, playlist); got != "#soundtrack" {
		t.Errorf("prev honeymoon soundtrack = %q", got)
	}
	if got := prevSectionBeforeHoneymoon(nil, ""); got != "#venues" {
		t.Errorf("prev honeymoon venues = %q", got)
	}

	if got := nextSectionAfterHoneymoon(true); got != "#registry" {
		t.Errorf("after honeymoon registry = %q", got)
	}
	if got := nextSectionAfterHoneymoon(false); got != "" {
		t.Errorf("after honeymoon none = %q", got)
	}
}

func TestHeroHelpers(t *testing.T) {
	if got := heroFallbackImage(models.HeroBackground{MobileMediaID: 5, DesktopMediaID: 6}); got != "/media/5" {
		t.Errorf("mobile preferred = %q", got)
	}
	if got := heroFallbackImage(models.HeroBackground{DesktopMediaID: 6}); got != "/media/6" {
		t.Errorf("desktop fallback = %q", got)
	}
	if got := heroFallbackImage(models.HeroBackground{}); got != "" {
		t.Errorf("empty fallback = %q", got)
	}

	if got := homepageHeroPreloads(models.HeroBackground{}); got != nil {
		t.Errorf("no media preloads = %v", got)
	}
	got := homepageHeroPreloads(models.HeroBackground{DesktopMediaID: 6})
	if len(got) != 1 || got[0].Href != "/media/6" || got[0].Media != "" {
		t.Errorf("single-image preloads = %+v", got)
	}
	got = homepageHeroPreloads(models.HeroBackground{MobileMediaID: 5})
	if len(got) != 1 || got[0].Href != "/media/5" {
		t.Errorf("mobile-only preloads = %+v", got)
	}
	got = homepageHeroPreloads(models.HeroBackground{DesktopMediaID: 6, MobileMediaID: 5})
	if len(got) != 2 || got[0].Href != "/media/5" || got[1].Href != "/media/6" {
		t.Errorf("pair preloads = %+v", got)
	}
	if got[0].Media != "(max-width: 767px)" || got[1].Media != "(min-width: 768px)" {
		t.Errorf("pair preload media = %+v", got)
	}
}

func TestLayoutHelpers(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want bool
	}{
		{"/media/1", true},
		{"https://cdn.example.com/x.png", true},
		{"http://example.com/x.png", true},
		{"  /media/2  ", true},
		{"data:image/png;base64,x", false},
		{"", false},
		{"ftp://example.com/x", false},
	} {
		if got := isPreloadableImageURL(tc.in); got != tc.want {
			t.Errorf("isPreloadableImageURL(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
	if got := coordValue(43.1054714); got != "43.1054714" {
		t.Errorf("coordValue = %q", got)
	}
	if got := coordValue(0); got != "0" {
		t.Errorf("coordValue zero = %q", got)
	}
	if got := publicImageURL(0); got != "" {
		t.Errorf("zero media url = %q", got)
	}
	if got := publicImageURL(-1); got != "" {
		t.Errorf("negative media url = %q", got)
	}
	if got := publicImageURL(7); got != "/media/7" {
		t.Errorf("media url = %q", got)
	}
	if got := pageTitle("Title", "Davide", "Agnese"); got != "Title · Davide & Agnese" {
		t.Errorf("full title = %q", got)
	}
	if got := pageTitle("Title", "Davide", ""); got != "Title" {
		t.Errorf("partial title = %q", got)
	}
	if got := pageTitle("Title", "", ""); got != "Title" {
		t.Errorf("bare title = %q", got)
	}
}

func TestInvitationHelpers(t *testing.T) {
	tr, fl := true, false
	guest := models.Guest{PollAnswers: []models.PollAnswer{
		{PollID: 1, Answer: true, Notes: "note one"},
		{PollID: 2, Answer: false, Notes: "note two"},
	}}
	if !guestAnsweredYes(guest, 1) {
		t.Error("poll 1 must be answered yes")
	}
	if guestAnsweredYes(guest, 2) {
		t.Error("poll 2 must not be answered yes")
	}
	if guestAnsweredYes(guest, 3) {
		t.Error("unknown poll must not be answered yes")
	}
	if got := guestPollNotes(guest, 1); got != "note one" {
		t.Errorf("notes = %q", got)
	}
	if got := guestPollNotes(guest, 9); got != "" {
		t.Errorf("unknown poll notes = %q", got)
	}

	if !InvitationAnswered(models.Invitation{}) {
		t.Error("an invitation without guests counts as answered")
	}
	if InvitationAnswered(models.Invitation{Guests: []models.Guest{{}}}) {
		t.Error("an unanswered guest means not answered")
	}
	if InvitationAnswered(models.Invitation{Guests: []models.Guest{{ConfirmedCeremony: &tr}, {}}}) {
		t.Error("one unanswered guest means not answered")
	}
	if !InvitationAnswered(models.Invitation{Guests: []models.Guest{{ConfirmedCeremony: &tr}, {ConfirmedReception: &fl}}}) {
		t.Error("all answered means answered")
	}

	if got := invitationImagePreloads(models.Settings{}, false); len(got) != 1 || got[0].Href != "/static/wax-seal.webp" {
		t.Errorf("fresh preloads = %+v", got)
	}
	if got := invitationImagePreloads(models.Settings{}, true); got != nil {
		t.Errorf("viewed preloads = %v", got)
	}
	if got := invitationImagePreloads(models.Settings{StampMediaID: 4}, false); got != nil {
		t.Errorf("stamp preloads = %v", got)
	}
}
