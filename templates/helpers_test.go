package templates

import (
	"slices"
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

func TestPlanHomeSections(t *testing.T) {
	place := []models.Place{{}}
	playlist := "https://open.spotify.com/playlist/abc123"

	// base order with no customs: every built-in that renders, in canonical order
	plan := planHomeSections(playlist, place, place, true, nil, "en")
	want := []string{"#landing", "#venues", "#soundtrack", "#places", "#honeymoon", "#registry"}
	if got := plan.order; !slices.Equal(got, want) {
		t.Errorf("base order = %v, want %v", got, want)
	}
	if got := prevAnchor(plan.order, "#landing"); got != "" {
		t.Errorf("prev landing = %q, want empty", got)
	}
	if got := nextAnchor(plan.order, "#landing"); got != "#venues" {
		t.Errorf("next landing = %q, want #venues", got)
	}
	if got := prevAnchor(plan.order, "#registry"); got != "#honeymoon" {
		t.Errorf("prev registry = %q, want #honeymoon", got)
	}
	if got := nextAnchor(plan.order, "#registry"); got != "" {
		t.Errorf("next registry = %q, want empty", got)
	}

	// no soundtrack, no places, no honeymoon, no registry: only landing and venues
	plan = planHomeSections("", nil, nil, false, nil, "en")
	want = []string{"#landing", "#venues"}
	if got := plan.order; !slices.Equal(got, want) {
		t.Errorf("minimal order = %v, want %v", got, want)
	}

	// a custom at position 2 renders between landing and venues
	customs := []models.CustomSection{{ID: 7, Position: 2, HeaderEn: "Extra", BodyEn: "More"}}
	plan = planHomeSections("", nil, nil, false, customs, "en")
	want = []string{"#landing", "#custom-7", "#venues"}
	if got := plan.order; !slices.Equal(got, want) {
		t.Errorf("position 2 order = %v, want %v", got, want)
	}
	if got := plan.before["#venues"]; len(got) != 1 || got[0].ID != 7 {
		t.Errorf("before venues = %+v, want the custom section", got)
	}
	if got := nextAnchor(plan.order, "#landing"); got != "#custom-7" {
		t.Errorf("next landing with custom = %q, want #custom-7", got)
	}

	// a custom at position 1 renders before landing; one past the end renders last
	customs = []models.CustomSection{
		{ID: 8, Position: 1, HeaderEn: "First", BodyEn: "First body"},
		{ID: 9, Position: 99, HeaderEn: "Last", BodyEn: "Last body"},
	}
	plan = planHomeSections("", nil, nil, true, customs, "en")
	want = []string{"#custom-8", "#landing", "#venues", "#registry", "#custom-9"}
	if got := plan.order; !slices.Equal(got, want) {
		t.Errorf("edges order = %v, want %v", got, want)
	}
	if got := plan.before["#landing"]; len(got) != 1 || got[0].ID != 8 {
		t.Errorf("before landing = %+v, want the first custom", got)
	}
	if got := plan.before[""]; len(got) != 1 || got[0].ID != 9 {
		t.Errorf("past end = %+v, want the last custom", got)
	}

	// a tie at one position keeps the dashboard form order
	customs = []models.CustomSection{
		{ID: 11, Position: 2, HeaderEn: "Second", BodyEn: "Second body"},
		{ID: 10, Position: 2, HeaderEn: "First", BodyEn: "First body"},
	}
	plan = planHomeSections("", nil, nil, false, customs, "en")
	want = []string{"#landing", "#custom-11", "#custom-10", "#venues"}
	if got := plan.order; !slices.Equal(got, want) {
		t.Errorf("tie order = %v, want %v", got, want)
	}

	// a section invisible in the language leaves the order entirely
	customs = []models.CustomSection{{ID: 12, Position: 2, HeaderIt: "Solo italiano", BodyIt: "Testo"}}
	plan = planHomeSections("", nil, nil, false, customs, "en")
	want = []string{"#landing", "#venues"}
	if got := plan.order; !slices.Equal(got, want) {
		t.Errorf("hidden in English = %v, want %v", got, want)
	}
	plan = planHomeSections("", nil, nil, false, customs, "it")
	want = []string{"#landing", "#custom-12", "#venues"}
	if got := plan.order; !slices.Equal(got, want) {
		t.Errorf("shown in Italian = %v, want %v", got, want)
	}
}

func TestCustomSectionTextAndBackground(t *testing.T) {
	section := models.CustomSection{
		HeaderIt: "Titolo", SubtitleIt: "Sottotitolo", BodyIt: "Corpo",
		HeaderEn: "Title", SubtitleEn: "Subtitle", BodyEn: "Body",
		DesktopMediaID: 5, MobileMediaID: 6,
	}
	header, subtitle, body, ok := customSectionText(section, "it")
	if !ok || header != "Titolo" || subtitle != "Sottotitolo" || body != "Corpo" {
		t.Errorf("italian text = %q %q %q %v", header, subtitle, body, ok)
	}
	header, subtitle, body, ok = customSectionText(section, "en")
	if !ok || header != "Title" || subtitle != "Subtitle" || body != "Body" {
		t.Errorf("english text = %q %q %q %v", header, subtitle, body, ok)
	}

	// no header in the language: the section does not render there
	if _, _, _, ok := customSectionText(models.CustomSection{BodyEn: "Body"}, "en"); ok {
		t.Error("headerless section renders")
	}
	// header but neither body nor image: the section does not render
	if _, _, _, ok := customSectionText(models.CustomSection{HeaderEn: "Title"}, "en"); ok {
		t.Error("bodyless imageless section renders")
	}
	// header plus image but no body: the section renders
	if _, _, _, ok := customSectionText(models.CustomSection{HeaderEn: "Title", DesktopMediaID: 5}, "en"); !ok {
		t.Error("header plus image section does not render")
	}

	// per-viewport fallback, each way, as the hero does
	desktop, mobile := customSectionBackground(models.CustomSection{DesktopMediaID: 5})
	if desktop != 5 || mobile != 5 {
		t.Errorf("desktop-only background = %d %d, want 5 5", desktop, mobile)
	}
	desktop, mobile = customSectionBackground(models.CustomSection{MobileMediaID: 6})
	if desktop != 6 || mobile != 6 {
		t.Errorf("mobile-only background = %d %d, want 6 6", desktop, mobile)
	}
	desktop, mobile = customSectionBackground(models.CustomSection{DesktopMediaID: 5, MobileMediaID: 6})
	if desktop != 5 || mobile != 6 {
		t.Errorf("pair background = %d %d, want 5 6", desktop, mobile)
	}
	desktop, mobile = customSectionBackground(models.CustomSection{})
	if desktop != 0 || mobile != 0 {
		t.Errorf("empty background = %d %d, want 0 0", desktop, mobile)
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
