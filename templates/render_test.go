package templates

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/a-h/templ"
	"github.com/streambinder/foedus/internal/i18n"
	"github.com/streambinder/foedus/internal/models"
)

func render(t *testing.T, component templ.Component) string {
	t.Helper()
	var buf bytes.Buffer
	if err := component.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render failed: %v", err)
	}
	return buf.String()
}

func boolPtr(v bool) *bool { return &v }

func fullSettings() models.Settings {
	return models.Settings{
		GroomName: "Davide", BrideName: "Agnese",
		CeremonyAddress: "Via Roma 1", CeremonyLocation: "Chiesa di San Francesco", CeremonyCity: "Terni",
		CeremonyDatetime: "2027-07-17T15:30", CeremonyLat: 42.56, CeremonyLng: 12.64, CeremonyMediaID: 11,
		ReceptionAddress: "Strada Verde 2", ReceptionLocation: "Villa Verde", ReceptionCity: "Terni",
		ReceptionDatetime: "2027-07-17T19:30", ReceptionLat: 42.57, ReceptionLng: 12.65, ReceptionMediaID: 12,
		BankAccountIBAN: "IT60X0542811101000000123456", BankAccountHolder: "Davide Pucci",
		SpotifyPlaylist:     "https://open.spotify.com/playlist/37i9dQZF1DXcBWIGoYBM5M",
		SharePreviewMediaID: 13,
		EnvelopeColor:       "#123abc", PageBgColor: "#f0e6d8", HomeBgColor: "#e8f0e0",
		StampMediaID: 14, FloraBlMediaID: 15, FloraTrMediaID: 16,
	}
}

func TestRenderLayoutVariants(t *testing.T) {
	translator := i18n.NewT("en")

	// bare layout: no OG meta, no leaflet, no preloads
	out := render(t, SetupGuard("en", translator))
	if !strings.Contains(out, "Setup Required") {
		t.Errorf("setup guard output missing title: %.200s", out)
	}

	// full OG meta with every optional field, leaflet and media preloads
	t.Setenv("CARTO_API_KEY", "pk.test")
	ogMeta := OGMeta{
		Title: "T", Description: "D", URL: "https://example.com/",
		ImageURL: "https://example.com/og.png", ImageType: "image/png",
		ImageWidth: "1200", ImageHeight: "630",
	}
	out = render(t, Layout("Title", "en", translator, ogMeta, true, []ImagePreload{
		{Href: "/media/1", Media: "(max-width: 767px)"},
		{Href: "/media/2"},
		{Href: ""},
	}, "--x:1;"))
	for _, want := range []string{"carto-api-key", "og:image:width", "og:image:height", "media=\"(max-width: 767px)\""} {
		if !strings.Contains(out, want) {
			t.Errorf("full layout missing %q", want)
		}
	}

	// OG title without description or image dimensions
	out = render(t, Layout("Title", "it", i18n.NewT("it"), OGMeta{Title: "Solo titolo"}, false, nil, ""))
	if strings.Contains(out, "og:image:width") {
		t.Error("layout rendered image width for an empty OG meta field")
	}
	if !strings.Contains(out, "og:title") {
		t.Error("layout missing og:title")
	}
}

func TestRenderFlora(t *testing.T) {
	settings := fullSettings()
	out := render(t, Flora(settings, "bl"))
	if !strings.Contains(out, "/media/15") {
		t.Errorf("flora bl with override = %.200s", out)
	}
	out = render(t, Flora(settings, "tr"))
	if !strings.Contains(out, "/media/16") {
		t.Errorf("flora tr with override = %.200s", out)
	}
	out = render(t, Flora(models.Settings{}, "bl"))
	if strings.Contains(out, "background-image") {
		t.Errorf("flora without override must not set a background image: %.200s", out)
	}
	// any corner other than bl uses the top-right media id
	out = render(t, Flora(settings, "xx"))
	if !strings.Contains(out, "/media/16") {
		t.Errorf("flora unknown corner = %.200s", out)
	}
}

func TestRenderCeremony(t *testing.T) {
	ogMeta := OGMeta{Title: "Ceremony", ImageType: "image/png", ImageWidth: "1200", ImageHeight: "630"}

	out := render(t, Ceremony(fullSettings(), i18n.NewT("en"), "en", ogMeta))
	if !strings.Contains(out, "Chiesa") && !strings.Contains(out, "Via Roma 1") {
		t.Errorf("full ceremony missing venue: %.300s", out)
	}

	// no media, location only, date-only datetime
	settings := models.Settings{
		GroomName: "Davide", BrideName: "Agnese",
		CeremonyLocation: "Chiesa", CeremonyDatetime: "2026-07-17",
	}
	out = render(t, Ceremony(settings, i18n.NewT("it"), "it", OGMeta{}))
	if !strings.Contains(out, "Chiesa") {
		t.Errorf("location-only ceremony missing venue: %.300s", out)
	}

	// address only, no location, no datetime, no names beyond the couple
	settings = models.Settings{GroomName: "A", BrideName: "B", CeremonyAddress: "Via Sola 9"}
	out = render(t, Ceremony(settings, i18n.NewT("en"), "en", OGMeta{}))
	if !strings.Contains(out, "Via Sola 9") {
		t.Errorf("address-only ceremony missing address: %.300s", out)
	}

	// nothing configured at all
	out = render(t, Ceremony(models.Settings{}, i18n.NewT("en"), "en", OGMeta{}))
	if strings.Contains(out, "venues-pair") {
		t.Error("empty ceremony rendered the venues block")
	}
}

func homeFixtures() (models.Settings, HomeContent, []models.RegistryItem, map[int]int) {
	settings := fullSettings()
	content := HomeContent{
		Places: []models.Place{
			{Label: "First date", Date: "2020-05-01T10:00", Name: "Bar Lume", Address: "Via A 1", Lat: 42.1, Lng: 12.1, MediaID: 21},
			{Label: "Proposal", Date: "2024-12-24", Name: "Hill", Address: "", Lat: 0, Lng: 0, MediaID: 0},
		},
		Honeymoon: []models.Place{
			{Label: "Safari", Name: "Serengeti", Address: "Tanzania", Lat: -2.3, Lng: 34.8, MediaID: 22},
		},
		ParkingSpots: []models.ParkingSpot{{Lat: 42.56, Lng: 12.64}},
		Accommodations: []models.Accommodation{
			{Name: "Hotel Uno", Description: "Near the venue", URL: "https://hotel.example.com"},
			{Name: "BB Due", Description: "", URL: ""},
			{Name: "Casa Uno", Description: "Quiet", URL: ""},
		},
		HeroBackground: models.HeroBackground{DesktopMediaID: 51, MobileMediaID: 52},
	}
	items := []models.RegistryItem{
		{ID: 1, Name: "Toaster", Price: 100, MediaID: 61},
		{ID: 2, Name: "Lamp", Price: 100},
		{ID: 3, Name: "Vase", Price: 50},
		{ID: 4, Name: "Open gift", Price: 0},
	}
	claimed := map[int]int{1: 100, 2: 40, 3: 50, 4: 10}
	return settings, content, items, claimed
}

func TestRenderHome(t *testing.T) {
	settings, content, items, claimed := homeFixtures()
	ogMeta := OGMeta{Title: "Davide & Agnese", Description: "Wedding", ImageType: "image/png", ImageWidth: "1200", ImageHeight: "630"}

	out := render(t, Home(settings, content, items, claimed, true, true, true, "/CODE1?no_redirect=1", true, i18n.NewT("en"), "en", ogMeta))
	for _, want := range []string{"Davide", "Bar Lume", "Serengeti", "Hotel Uno", "Toaster"} {
		if !strings.Contains(out, want) {
			t.Errorf("full home missing %q", want)
		}
	}

	out = render(t, Home(settings, content, items, claimed, true, false, true, "", false, i18n.NewT("it"), "it", ogMeta))
	if !strings.Contains(out, "Davide") {
		t.Errorf("italian home render failed: %.200s", out)
	}

	// soundtrack enabled but no invite update URL (the anonymous branch)
	out = render(t, Home(settings, content, items, claimed, false, true, false, "", false, i18n.NewT("en"), "en", OGMeta{}))
	if !strings.Contains(out, "Davide") {
		t.Errorf("home without bank config failed: %.200s", out)
	}
}

func TestRenderHomeNoMediaVenues(t *testing.T) {
	// venues with addresses but no uploaded media take the text-only layout,
	// parking button included
	settings := fullSettings()
	settings.CeremonyMediaID = 0
	settings.ReceptionMediaID = 0
	content := HomeContent{
		ParkingSpots: []models.ParkingSpot{{Lat: 42.56, Lng: 12.64}},
	}
	out := render(t, Home(settings, content, nil, nil, false, false, false, "", false, i18n.NewT("en"), "en", OGMeta{}))
	if !strings.Contains(out, "Via Roma 1") {
		t.Errorf("no-media home missing ceremony address: %.300s", out)
	}
	if !strings.Contains(out, "Strada Verde 2") {
		t.Errorf("no-media home missing reception address: %.300s", out)
	}
}

func TestRenderHomeLabelOverrides(t *testing.T) {
	// the section descriptions default to empty in both languages, so the
	// subtitle blocks only render when a dashboard override supplies text
	settings, content, items, claimed := homeFixtures()
	translator := i18n.NewTWithOverrides("en", map[string]string{
		"home.venues_description":    "Where it happens",
		"home.places_description":    "Our story so far",
		"home.honeymoon_description": "After the party",
	})
	out := render(t, Home(settings, content, items, claimed, true, false, false, "", false, translator, "en", OGMeta{}))
	for _, want := range []string{"Where it happens", "Our story so far", "After the party"} {
		if !strings.Contains(out, want) {
			t.Errorf("home with overrides missing %q", want)
		}
	}
}

func TestRenderHomePlacesWithoutPlaylist(t *testing.T) {
	// with places but no playlist, the places section links back to the
	// venues section instead of the soundtrack section
	settings, content, items, claimed := homeFixtures()
	settings.SpotifyPlaylist = ""
	out := render(t, Home(settings, content, items, claimed, true, false, false, "", false, i18n.NewT("en"), "en", OGMeta{}))
	if !strings.Contains(out, `href="#venues"`) {
		t.Errorf("places without playlist missing the venues arrow: %.300s", out)
	}
}

func TestRenderHomeEmptyAndPassed(t *testing.T) {
	// bare minimum: couple names only
	settings := models.Settings{GroomName: "A", BrideName: "B"}
	out := render(t, Home(settings, HomeContent{}, nil, nil, false, false, false, "", false, i18n.NewT("en"), "en", OGMeta{}))
	if !strings.Contains(out, "A") {
		t.Errorf("empty home failed: %.200s", out)
	}

	// the event has passed: contributions close but the page still renders
	settings, content, items, claimed := homeFixtures()
	settings.CeremonyDatetime = "2020-01-01T10:00"
	out = render(t, Home(settings, content, items, claimed, true, true, true, "", false, i18n.NewT("en"), "en", OGMeta{}))
	if !strings.Contains(out, "Davide") {
		t.Errorf("passed-event home failed: %.200s", out)
	}

	// no playlist, no places, reception only
	settings = models.Settings{
		GroomName: "A", BrideName: "B",
		ReceptionAddress: "Via Ricezione 3", ReceptionDatetime: "2027-01-01",
		BankAccountIBAN: "IT00", BankAccountHolder: "Holder",
	}
	out = render(t, Home(settings, HomeContent{HeroBackground: models.HeroBackground{MobileMediaID: 9}},
		[]models.RegistryItem{{ID: 1, Name: "Solo", Price: 10}}, nil, true, false, false, "", false, i18n.NewT("en"), "en", OGMeta{}))
	if !strings.Contains(out, "Via Ricezione 3") {
		t.Errorf("reception-only home missing venue: %.300s", out)
	}
}

func invitationFixtures() (models.Invitation, []models.Poll) {
	viewed := models.Timestamp{Time: time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)}
	_ = viewed
	tr, fl := true, false
	inv := models.Invitation{
		ID: 7, Code: "CODE7", Label: "Famiglia Pucci",
		Guests: []models.Guest{
			{
				ID: 1, FirstName: "Ada", LastName: "Love", ConfirmedCeremony: &tr, ConfirmedReception: &fl,
				PollAnswers: []models.PollAnswer{{PollID: 1, Answer: true, Notes: "window seat"}, {PollID: 2, Answer: false}},
			},
			{ID: 2, FirstName: "Bob", LastName: "Ray"},
			{
				ID: 3, FirstName: "Cid", LastName: "Ray", ConfirmedCeremony: &fl, ConfirmedReception: &tr,
				PollAnswers: []models.PollAnswer{{PollID: 1, Answer: false}, {PollID: 2, Answer: true, Notes: "aisle"}},
			},
		},
	}
	polls := []models.Poll{
		{ID: 1, Question: "Need a bus?", Description: "From the station"},
		{ID: 2, Question: "Vegetarian?"},
	}
	return inv, polls
}

func TestRenderInvitation(t *testing.T) {
	inv, polls := invitationFixtures()
	settings := fullSettings()
	ogMeta := OGMeta{Title: "Invitation", ImageType: "image/png", ImageWidth: "1200", ImageHeight: "630"}

	// partially answered, not viewed: the RSVP form renders with every
	// confirmation combination across the three guests
	out := render(t, Invitation(inv, settings, polls, false, false, i18n.NewT("en"), "en", ogMeta, "Title"))
	if !strings.Contains(out, "Ada") {
		t.Errorf("invitation missing guest: %.300s", out)
	}

	// already viewed: the envelope renders complete
	out = render(t, Invitation(inv, settings, polls, true, true, i18n.NewT("it"), "it", ogMeta, "Titolo"))
	if !strings.Contains(out, "is-complete") {
		t.Errorf("viewed invitation missing complete class: %.300s", out)
	}

	// fully answered guests take the answered branch
	answered := models.Invitation{Guests: []models.Guest{
		{FirstName: "Ada", ConfirmedCeremony: boolPtr(true), ConfirmedReception: boolPtr(true)},
	}}
	out = render(t, Invitation(answered, settings, nil, false, false, i18n.NewT("en"), "en", OGMeta{}, "T"))
	if !strings.Contains(out, "Ada") {
		t.Errorf("answered invitation failed: %.200s", out)
	}

	// no guests at all counts as answered too
	out = render(t, Invitation(models.Invitation{}, settings, nil, false, false, i18n.NewT("en"), "en", OGMeta{}, "T"))
	if out == "" {
		t.Error("guestless invitation rendered empty output")
	}

	// an event in the past closes the RSVP
	passed := fullSettings()
	passed.CeremonyDatetime = "2020-01-01T10:00"
	passed.StampMediaID = 0
	out = render(t, Invitation(inv, passed, polls, false, false, i18n.NewT("en"), "en", OGMeta{}, "T"))
	if !strings.Contains(out, "This event has already taken place") {
		t.Errorf("passed-event invitation missing the locked notice: %.200s", out)
	}

	// venues without uploaded media take the text-only layout
	noMedia := fullSettings()
	noMedia.CeremonyMediaID = 0
	noMedia.ReceptionMediaID = 0
	out = render(t, Invitation(inv, noMedia, polls, false, false, i18n.NewT("en"), "en", OGMeta{}, "T"))
	if !strings.Contains(out, "Via Roma 1") || !strings.Contains(out, "Strada Verde 2") {
		t.Errorf("no-media invitation missing venue addresses: %.300s", out)
	}

	// minimal settings: no venues, no stamp, invalid colours fall back
	minimal := models.Settings{GroomName: "A", BrideName: "B", EnvelopeColor: "red", PageBgColor: ""}
	out = render(t, Invitation(inv, minimal, polls, false, false, i18n.NewT("en"), "en", OGMeta{}, "T"))
	if !strings.Contains(out, models.DefaultEnvelopeColor) {
		t.Errorf("invalid envelope colour did not fall back: %.300s", out)
	}
}

func TestRenderDonut(t *testing.T) {
	out := render(t, Donut("RSVP", 0, 0, nil))
	if !strings.Contains(out, "RSVP") {
		t.Errorf("empty donut = %.200s", out)
	}
	slices := []DonutSlice{
		{Value: 0, ClassName: "zero", Label: "Zero", Category: "cat_zero"},
		{Value: 5, ClassName: "five", Label: "Five", Category: "cat_five"},
		{Value: 3, ClassName: "three", Label: "Three"},
	}
	out = render(t, Donut("Types", 8, 8, slices))
	if !strings.Contains(out, "counter-names-trigger") {
		t.Errorf("donut with category missing trigger: %.300s", out)
	}
	if !strings.Contains(out, "8<small>/8</small>") {
		t.Errorf("donut center missing totals: %.300s", out)
	}
}

func dashboardFixtures() (models.Settings, SettingsContent, []models.Guest, []models.Invitation) {
	settings := fullSettings()
	settings.EnvelopeColor = "not-a-color" // exercises the colour fallback inputs
	invID := 7
	tr, fl := true, false
	viewedAt := &models.Timestamp{Time: time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)}
	invitations := []models.Invitation{
		{ID: 7, Code: "CODE7", Label: "Famiglia", ViewedAt: viewedAt, Guests: []models.Guest{
			{FirstName: "Ada", LastName: "Love", ConfirmedCeremony: &tr},
			{FirstName: "Bob", LastName: "Ray"},
		}},
		{ID: 8, Code: "CODE8", Guests: []models.Guest{
			{FirstName: "Cid", LastName: "Ray"},
		}},
	}
	guests := []models.Guest{
		{ID: 1, FirstName: "Ada", LastName: "Love", Type: "adult", InvitationID: &invID},
		{ID: 2, FirstName: "Bob", LastName: "Ray", Type: "child", ConfirmedCeremony: &tr, ConfirmedReception: &fl},
		{ID: 3, FirstName: "Cid", LastName: "Ray", Type: "vendor", ConfirmedCeremony: &fl, ConfirmedReception: &tr},
	}
	content := SettingsContent{
		Places: []models.Place{
			{Label: "First", Name: "Bar", Address: "Via A", Date: "2020-05-01", Lat: 42.1, Lng: 12.1, MediaID: 21},
			{Label: "Second", Name: "Hill"},
		},
		Honeymoon: []models.Place{
			{Label: "Safari", Name: "Serengeti", Lat: -2.3, Lng: 34.8, MediaID: 23},
		},
		ParkingSpots:   []models.ParkingSpot{{Lat: 42.5, Lng: 12.6}, {}},
		Accommodations: []models.Accommodation{{Name: "Hotel", Description: "Desc", URL: "https://h.example.com"}},
		Impersonations: []models.Impersonation{{Codename: "zio", Profile: "warm and funny"}},
		HeroBackgrounds: []models.HeroBackground{
			{DesktopMediaID: 31, MobileMediaID: 32},
			{DesktopMediaID: 33},
			{MobileMediaID: 34},
			{},
		},
		HomepageLabels: map[string]map[string]string{
			"en": {"home.ceremony": "The Rite"},
			"it": {"home.ceremony": "Il Rito"},
		},
	}
	return settings, content, guests, invitations
}

func TestRenderDashboardFull(t *testing.T) {
	settings, content, guests, invitations := dashboardFixtures()
	tr := true
	_ = tr
	gifts := []models.Gift{
		{ID: 1, Amount: 50, Donor: "Alice", InvitationID: intPtr(7), Confirmed: true},
		{ID: 2, Amount: 30, Donor: "Bob"},
	}
	registry := []models.RegistryItem{
		{ID: 1, Name: "Toaster", Price: 100, MediaID: 61},
		{ID: 2, Name: "Lamp", Price: 0},
	}
	polls := []models.Poll{
		{ID: 1, Question: "Bus?", Description: "d", TotalCount: 2, YesVoters: []models.PollVoter{{Name: "Ada Love", Notes: "note"}, {Name: "Bob Ray"}}},
		{ID: 2, Question: "Veg?", TotalCount: 0},
	}
	soundtrack := []models.SoundtrackEvent{
		{ID: 1, Title: "Song", Artist: "Artist", URL: "https://open.spotify.com/track/x", InviteID: "CODE7"},
		{ID: 2, Title: "NoURL", Artist: "Artist", InviteID: "UNKNOWN"},
		{ID: 3, Title: "Plain", Artist: "Artist"},
	}
	out := render(t, Dashboard(settings, content, guests,
		gifts, 2, 3, len(gifts), 80,
		registry, 2, 3, registry,
		invitations, invitations, 2, 3, "fam",
		polls,
		soundtrack, 2, 3,
		5, 2, 3, 8, 4, 10,
		4, 1, 0, 0,
		2, 3, "ada", "csrf-token", "Settings saved", i18n.NewT("en"), "en"))
	for _, want := range []string{"Settings saved", "Ada", "Toaster", "Famiglia", "Bus?", "Song"} {
		if !strings.Contains(out, want) {
			t.Errorf("full dashboard missing %q", want)
		}
	}
	// first page of the guests table (prev link off, next link on) and the
	// last page of the invitations table (next link off)
	out = render(t, Dashboard(settings, content, guests,
		gifts, 3, 3, len(gifts), 80,
		registry, 1, 3, registry,
		invitations, invitations, 3, 3, "",
		polls,
		soundtrack, 1, 3,
		0, 0, 0, 0, 0, 0,
		0, 0, 0, 0,
		1, 3, "", "csrf-token", "", i18n.NewT("it"), "it"))
	if !strings.Contains(out, "Ada") {
		t.Errorf("dashboard pagination variant failed: %.200s", out)
	}
}

func TestRenderDashboardPaginationExtremes(t *testing.T) {
	settings, content, guests, invitations := dashboardFixtures()
	gifts := []models.Gift{{ID: 1, Amount: 50, Donor: "Alice"}}
	registry := []models.RegistryItem{{ID: 1, Name: "Toaster", Price: 100}}
	polls := []models.Poll{{ID: 1, Question: "Bus?"}}
	soundtrack := []models.SoundtrackEvent{{ID: 1, Title: "Song"}}
	// last guests page, first invitations page, first gifts page, last
	// soundtrack and registry pages: every prev/next disabled span renders.
	out := render(t, Dashboard(settings, content, guests,
		gifts, 1, 3, len(gifts), 50,
		registry, 3, 3, registry,
		invitations, invitations, 1, 3, "",
		polls, soundtrack, 3, 3,
		0, 0, 0, 0, 0, 0,
		0, 0, 0, 0,
		3, 3, "", "csrf", "", i18n.NewT("en"), "en"))
	if !strings.Contains(out, "disabled") {
		t.Error("pagination extremes render missing a disabled span")
	}
}

func TestRenderDashboardPlaceMediaVariants(t *testing.T) {
	settings, content, _, _ := dashboardFixtures()
	// story place without media and honeymoon place without media take the
	// hidden-preview branch of each card; the reverse pairing is covered by
	// the full fixture
	content.Places = []models.Place{
		{Label: "NoPic", Name: "Field"},
		{Label: "Pic", Name: "Bar", MediaID: 44},
	}
	content.Honeymoon = []models.Place{{Label: "NoPic", Name: "Beach"}}
	out := render(t, Dashboard(settings, content, nil,
		nil, 1, 1, 0, 0,
		nil, 1, 1, nil,
		nil, nil, 1, 1, "",
		nil, nil, 1, 1,
		0, 0, 0, 0, 0, 0,
		0, 0, 0, 0,
		1, 1, "", "", "", i18n.NewT("en"), "en"))
	if !strings.Contains(out, "display:none") {
		t.Error("place media variants render missing a hidden preview")
	}
}

func TestRenderDashboardEmpty(t *testing.T) {
	out := render(t, Dashboard(models.Settings{}, SettingsContent{}, nil,
		nil, 1, 1, 0, 0,
		nil, 1, 1, nil,
		nil, nil, 1, 1, "",
		nil,
		nil, 1, 1,
		0, 0, 0, 0, 0, 0,
		0, 0, 0, 0,
		1, 1, "", "", "", i18n.NewT("en"), "en"))
	if out == "" {
		t.Fatal("empty dashboard rendered no output")
	}
	if strings.Contains(out, "flash-success") {
		t.Error("empty dashboard rendered a flash block without a message")
	}
}

func TestRenderEditPages(t *testing.T) {
	settings := fullSettings()
	translator := i18n.NewT("en")

	out := render(t, EditGuest(models.Guest{ID: 1, FirstName: "Ada", LastName: "Love", Type: "child"}, settings, "csrf", translator, "en"))
	if !strings.Contains(out, "Ada") {
		t.Errorf("edit guest = %.200s", out)
	}
	// every guest type must mark its own option as selected
	for _, guestType := range []string{"adult", "infant", "vendor"} {
		out = render(t, EditGuest(models.Guest{ID: 1, FirstName: "Ada", Type: guestType}, settings, "csrf", translator, "en"))
		if !strings.Contains(out, "selected") {
			t.Errorf("edit guest type %q: no selected option rendered", guestType)
		}
	}
	out = render(t, EditRegistryItem(models.RegistryItem{ID: 2, Name: "Lamp", Price: 80, MediaID: 61}, settings, "csrf", translator, "en"))
	if !strings.Contains(out, "Lamp") {
		t.Errorf("edit registry item = %.200s", out)
	}
	out = render(t, EditRegistryItem(models.RegistryItem{ID: 3, Name: "NoPic", Price: 10}, settings, "csrf", translator, "en"))
	if !strings.Contains(out, "NoPic") {
		t.Errorf("edit registry item without media = %.200s", out)
	}
	items := []models.RegistryItem{{ID: 1, Name: "Toaster"}, {ID: 2, Name: "Lamp"}}
	out = render(t, EditGift(models.Gift{ID: 4, Amount: 50, Donor: "Alice", RegistryItemID: intPtr(2), Confirmed: true}, items, settings, "csrf", translator, "en"))
	if !strings.Contains(out, "Alice") {
		t.Errorf("edit gift = %.200s", out)
	}
	out = render(t, EditGift(models.Gift{ID: 5, Amount: 20, Donor: "Bob"}, items, settings, "csrf", translator, "en"))
	if !strings.Contains(out, "Bob") {
		t.Errorf("edit unconfirmed gift = %.200s", out)
	}
	out = render(t, EditPoll(models.Poll{ID: 6, Question: "Bus?", Description: "From station"}, settings, "csrf", translator, "en"))
	if !strings.Contains(out, "Bus?") {
		t.Errorf("edit poll = %.200s", out)
	}
	out = render(t, EditInvitation(models.Invitation{ID: 7, Code: "CODE7", Label: "Famiglia"}, settings, "csrf", translator, "en"))
	if !strings.Contains(out, "Famiglia") {
		t.Errorf("edit invitation = %.200s", out)
	}
	out = render(t, EditInvitation(models.Invitation{ID: 8, Code: "CODE8"}, settings, "csrf", translator, "en"))
	if out == "" {
		t.Error("edit invitation without label rendered empty output")
	}
}
