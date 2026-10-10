package templates

import (
	"bytes"
	"context"
	"errors"
	"io"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/a-h/templ"
	templruntime "github.com/a-h/templ/runtime"
	"github.com/streambinder/foedus/internal/i18n"
	"github.com/streambinder/foedus/internal/models"
)

// The generated templ code checks the error of every single write and
// returns early on failure. Those return statements are the majority of the
// generated code, and the only way to execute them is to render into a
// writer that fails mid-stream. A small runtime buffer makes the failure
// land inside the render instead of only at the final flush, and sweeping
// the failure point across the whole output walks the abort through every
// section of the component.
func init() {
	templruntime.DefaultBufferSize = 16
}

// failAfterWriter accepts budget bytes and fails every write after that.
type failAfterWriter struct {
	budget int
	wrote  int
}

func (w *failAfterWriter) Write(p []byte) (int, error) {
	if w.wrote >= w.budget {
		return 0, errors.New("writer failed")
	}
	remaining := w.budget - w.wrote
	if len(p) > remaining {
		w.wrote += remaining
		return remaining, errors.New("writer failed")
	}
	w.wrote += len(p)
	return len(p), nil
}

func sweepComponent(t *testing.T, name string, component templ.Component) {
	t.Helper()
	var buf bytes.Buffer
	if err := component.Render(context.Background(), &buf); err != nil {
		t.Fatalf("%s: baseline render failed: %v", name, err)
	}
	total := buf.Len()
	step := total / 4000
	if step > 12 {
		step = 12
	}
	if step < 4 {
		step = 4
	}
	budgets := make([]int, 0, total/step+4)
	for budget := 0; budget <= total; budget += step {
		budgets = append(budgets, budget)
	}
	// failures inside the final buffered window: every write succeeds and
	// only the flush on release fails, which is the one path that assigns
	// the buffer error to the render result
	for _, budget := range []int{total - 1, total - 2, total - 3} {
		if budget >= 0 {
			budgets = append(budgets, budget)
		}
	}
	// the renders are independent, so fan them out over the available
	// cores: the full sweep is millions of small renders and would take
	// minutes single-threaded under the race detector
	workers := runtime.GOMAXPROCS(0)
	jobs := make(chan int)
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for budget := range jobs {
				_ = component.Render(context.Background(), &failAfterWriter{budget: budget})
			}
		}()
	}
	for _, budget := range budgets {
		jobs <- budget
	}
	close(jobs)
	wg.Wait()

	// a cancelled context aborts the render before the first write
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := component.Render(ctx, &bytes.Buffer{}); err == nil {
		t.Errorf("%s: render with a cancelled context succeeded, want an error", name)
	}
}

func TestRenderWriteErrorSweeps(t *testing.T) {
	settings, content, guests, invitations := dashboardFixtures()
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
	sweepComponent(t, "dashboard full", Dashboard(settings, content, guests,
		gifts, 2, 3, len(gifts), 80,
		registry, 2, 3, registry,
		invitations, invitations, 2, 3, "fam",
		polls, soundtrack, 2, 3,
		5, 2, 3, 8, 4, 10,
		4, 1, 0, 0,
		2, 3, "ada", "csrf-token", "Settings saved", i18n.NewT("en"), "en"))
	sweepComponent(t, "dashboard empty", Dashboard(models.Settings{}, SettingsContent{}, nil,
		nil, 1, 1, 0, 0,
		nil, 1, 1, nil,
		nil, nil, 1, 1, "",
		nil, nil, 1, 1,
		0, 0, 0, 0, 0, 0,
		0, 0, 0, 0,
		1, 1, "", "", "", i18n.NewT("en"), "en"))

	homeSettings, homeContent, homeItems, homeClaimed := homeFixtures()
	ogMeta := OGMeta{Title: "Davide & Agnese", Description: "Wedding", ImageType: "image/png", ImageWidth: "1200", ImageHeight: "630"}
	sweepComponent(t, "home full", Home(homeSettings, homeContent, homeItems, homeClaimed, true, true, true, "/CODE1?no_redirect=1", true, i18n.NewT("en"), "en", ogMeta))
	sweepComponent(t, "home bare", Home(models.Settings{GroomName: "A", BrideName: "B"}, HomeContent{}, nil, nil, false, false, false, "", false, i18n.NewT("en"), "en", OGMeta{}))
	passedSettings := homeSettings
	passedSettings.CeremonyDatetime = "2020-01-01T10:00"
	sweepComponent(t, "home passed", Home(passedSettings, homeContent, homeItems, homeClaimed, true, true, true, "", false, i18n.NewT("en"), "en", OGMeta{}))

	inv, invPolls := invitationFixtures()
	sweepComponent(t, "invitation form", Invitation(inv, fullSettings(), invPolls, false, false, i18n.NewT("en"), "en", ogMeta, "Title"))
	sweepComponent(t, "invitation viewed", Invitation(inv, fullSettings(), invPolls, true, true, i18n.NewT("it"), "it", ogMeta, "Titolo"))
	sweepComponent(t, "invitation minimal", Invitation(inv, models.Settings{GroomName: "A", BrideName: "B"}, invPolls, false, false, i18n.NewT("en"), "en", OGMeta{}, "T"))

	sweepComponent(t, "ceremony full", Ceremony(fullSettings(), i18n.NewT("en"), "en", ogMeta))
	sweepComponent(t, "ceremony empty", Ceremony(models.Settings{}, i18n.NewT("en"), "en", OGMeta{}))

	sweepComponent(t, "setup guard", SetupGuard("en", i18n.NewT("en")))
	sweepComponent(t, "layout full", Layout("Title", "en", i18n.NewT("en"), ogMeta, true,
		[]ImagePreload{{Href: "/media/1", Media: "(max-width: 767px)"}, {Href: "/media/2"}}, "--x:1;"))
	sweepComponent(t, "dashboard layout", DashboardLayout("Title", "en"))

	sweepComponent(t, "donut", Donut("Types", 8, 8, []DonutSlice{
		{Value: 0, ClassName: "zero", Label: "Zero", Category: "cat_zero"},
		{Value: 5, ClassName: "five", Label: "Five", Category: "cat_five"},
		{Value: 3, ClassName: "three", Label: "Three"},
	}))
	sweepComponent(t, "flora", Flora(fullSettings(), "bl"))

	translator := i18n.NewT("en")
	sweepComponent(t, "edit guest", EditGuest(models.Guest{ID: 1, FirstName: "Ada", LastName: "Love", Type: "child"}, fullSettings(), "csrf", translator, "en"))
	sweepComponent(t, "edit registry item", EditRegistryItem(models.RegistryItem{ID: 2, Name: "Lamp", Price: 80, MediaID: 61}, fullSettings(), "csrf", translator, "en"))
	sweepComponent(t, "edit gift", EditGift(models.Gift{ID: 4, Amount: 50, Donor: "Alice", RegistryItemID: intPtr(2), Confirmed: true},
		[]models.RegistryItem{{ID: 1, Name: "Toaster"}, {ID: 2, Name: "Lamp"}}, fullSettings(), "csrf", translator, "en"))
	sweepComponent(t, "edit poll", EditPoll(models.Poll{ID: 6, Question: "Bus?", Description: "From station"}, fullSettings(), "csrf", translator, "en"))
	sweepComponent(t, "edit invitation", EditInvitation(models.Invitation{ID: 7, Code: "CODE7", Label: "Famiglia"}, fullSettings(), "csrf", translator, "en"))

	// pagination extremes: the disabled prev/next spans are separate
	// generated sections with their own writes
	sweepComponent(t, "dashboard pagination extremes", Dashboard(settings, content, guests,
		gifts, 1, 3, len(gifts), 80,
		registry, 3, 3, registry,
		invitations, invitations, 1, 3, "",
		polls, soundtrack, 3, 3,
		0, 0, 0, 0, 0, 0,
		0, 0, 0, 0,
		3, 3, "", "csrf-token", "", i18n.NewT("en"), "en"))

	// venues without media and label overrides take generated branches the
	// full fixtures never reach
	noMediaSettings := fullSettings()
	noMediaSettings.CeremonyMediaID = 0
	noMediaSettings.ReceptionMediaID = 0
	sweepComponent(t, "home no media", Home(noMediaSettings, HomeContent{
		ParkingSpots: []models.ParkingSpot{{Lat: 42.56, Lng: 12.64}},
	}, nil, nil, false, false, false, "", false, i18n.NewT("en"), "en", OGMeta{}))
	overrideTranslator := i18n.NewTWithOverrides("en", map[string]string{
		"home.venues_description":    "Where it happens",
		"home.places_description":    "Our story so far",
		"home.honeymoon_description": "After the party",
	})
	sweepComponent(t, "home overrides", Home(homeSettings, homeContent, homeItems, homeClaimed, true, false, false, "", false, overrideTranslator, "en", OGMeta{}))
	sweepComponent(t, "invitation no media", Invitation(inv, noMediaSettings, invPolls, false, false, i18n.NewT("en"), "en", OGMeta{}, "T"))

	// the two form building blocks only ever render as dashboard children;
	// sweeping them directly covers their own guards and error returns
	sweepComponent(t, "setting input", settingInput("groom_name", "Groom", "Davide"))
	sweepComponent(t, "homepage label group", homepageLabelGroup("en", "Homepage", []string{"home.ceremony", "btn.buy"},
		map[string]map[string]string{"en": {"home.ceremony": "The Rite"}},
		map[string]string{"home.ceremony": "The Ceremony", "btn.buy": "Give"}))
	sweepComponent(t, "homepage label group empty", homepageLabelGroup("it", "Homepage", nil, nil, nil))
}

// pad stretches a value so expression writes in the generated code are
// large enough to flush on their own: a write that never flushes can never
// surface a writer error, so short values leave those error returns
// unreachable during a sweep.
func pad(value string) string {
	if value == "" {
		return value
	}
	return value + " " + strings.Repeat("extra padding text ", 6)
}

// Long values change which generated writes flush mid-render, so sweeping a
// long-data variant reaches error returns that short values cannot.
func TestRenderWriteErrorSweepsLongData(t *testing.T) {
	settings, content, guests, invitations := dashboardFixtures()
	settings.GroomName = pad(settings.GroomName)
	settings.BrideName = pad(settings.BrideName)
	settings.CeremonyAddress = pad(settings.CeremonyAddress)
	settings.CeremonyLocation = pad(settings.CeremonyLocation)
	settings.CeremonyCity = pad(settings.CeremonyCity)
	settings.ReceptionAddress = pad(settings.ReceptionAddress)
	settings.ReceptionLocation = pad(settings.ReceptionLocation)
	settings.BankAccountIBAN = pad(settings.BankAccountIBAN)
	settings.BankAccountHolder = pad(settings.BankAccountHolder)
	settings.SpotifyPlaylist = pad(settings.SpotifyPlaylist)
	for i := range content.Places {
		content.Places[i].Label = pad(content.Places[i].Label)
		content.Places[i].Name = pad(content.Places[i].Name)
		content.Places[i].Address = pad(content.Places[i].Address)
	}
	for i := range content.Honeymoon {
		content.Honeymoon[i].Label = pad(content.Honeymoon[i].Label)
		content.Honeymoon[i].Name = pad(content.Honeymoon[i].Name)
	}
	for i := range content.Accommodations {
		content.Accommodations[i].Name = pad(content.Accommodations[i].Name)
		content.Accommodations[i].Description = pad(content.Accommodations[i].Description)
		content.Accommodations[i].URL = pad(content.Accommodations[i].URL)
	}
	for i := range content.Impersonations {
		content.Impersonations[i].Codename = pad(content.Impersonations[i].Codename)
		content.Impersonations[i].Profile = pad(content.Impersonations[i].Profile)
	}
	for i := range guests {
		guests[i].FirstName = pad(guests[i].FirstName)
		guests[i].LastName = pad(guests[i].LastName)
	}
	for i := range invitations {
		invitations[i].Label = pad(invitations[i].Label)
		invitations[i].Code = pad(invitations[i].Code)
		for j := range invitations[i].Guests {
			invitations[i].Guests[j].FirstName = pad(invitations[i].Guests[j].FirstName)
			invitations[i].Guests[j].LastName = pad(invitations[i].Guests[j].LastName)
		}
	}
	gifts := []models.Gift{
		{ID: 1, Amount: 50, Donor: pad("Alice"), InvitationID: intPtr(7), Confirmed: true},
		{ID: 2, Amount: 30, Donor: pad("Bob")},
	}
	registry := []models.RegistryItem{
		{ID: 1, Name: pad("Toaster"), Price: 100, MediaID: 61},
		{ID: 2, Name: pad("Lamp"), Price: 0},
	}
	polls := []models.Poll{
		{ID: 1, Question: pad("Bus?"), Description: pad("d"), TotalCount: 2, YesVoters: []models.PollVoter{{Name: pad("Ada Love"), Notes: pad("note")}, {Name: pad("Bob Ray")}}},
		{ID: 2, Question: pad("Veg?"), TotalCount: 0},
	}
	soundtrack := []models.SoundtrackEvent{
		{ID: 1, Title: pad("Song"), Artist: pad("Artist"), URL: pad("https://open.spotify.com/track/x"), InviteID: invitations[0].Code},
		{ID: 2, Title: pad("NoURL"), Artist: pad("Artist"), InviteID: pad("UNKNOWN")},
	}
	sweepComponent(t, "dashboard long", Dashboard(settings, content, guests,
		gifts, 1, 3, len(gifts), 80,
		registry, 1, 3, registry,
		invitations, invitations, 1, 3, pad("fam"),
		polls, soundtrack, 1, 3,
		5, 2, 3, 8, 4, 10,
		4, 1, 0, 0,
		1, 3, pad("ada"), "csrf-token", pad("Settings saved"), i18n.NewT("it"), "it"))

	homeSettings, homeContent, homeItems, homeClaimed := homeFixtures()
	homeSettings.GroomName = pad(homeSettings.GroomName)
	homeSettings.BrideName = pad(homeSettings.BrideName)
	homeSettings.CeremonyAddress = pad(homeSettings.CeremonyAddress)
	homeSettings.CeremonyLocation = pad(homeSettings.CeremonyLocation)
	homeSettings.ReceptionAddress = pad(homeSettings.ReceptionAddress)
	homeSettings.ReceptionLocation = pad(homeSettings.ReceptionLocation)
	homeSettings.BankAccountIBAN = pad(homeSettings.BankAccountIBAN)
	homeSettings.BankAccountHolder = pad(homeSettings.BankAccountHolder)
	for i := range homeContent.Places {
		homeContent.Places[i].Label = pad(homeContent.Places[i].Label)
		homeContent.Places[i].Name = pad(homeContent.Places[i].Name)
		homeContent.Places[i].Address = pad(homeContent.Places[i].Address)
	}
	for i := range homeContent.Honeymoon {
		homeContent.Honeymoon[i].Label = pad(homeContent.Honeymoon[i].Label)
		homeContent.Honeymoon[i].Name = pad(homeContent.Honeymoon[i].Name)
		homeContent.Honeymoon[i].Address = pad(homeContent.Honeymoon[i].Address)
	}
	for i := range homeContent.Accommodations {
		homeContent.Accommodations[i].Name = pad(homeContent.Accommodations[i].Name)
		homeContent.Accommodations[i].Description = pad(homeContent.Accommodations[i].Description)
		homeContent.Accommodations[i].URL = pad(homeContent.Accommodations[i].URL)
	}
	for i := range homeItems {
		homeItems[i].Name = pad(homeItems[i].Name)
	}
	ogMeta := OGMeta{Title: pad("Davide & Agnese"), Description: pad("Wedding"), ImageType: "image/png", ImageWidth: "1200", ImageHeight: "630"}
	sweepComponent(t, "home long", Home(homeSettings, homeContent, homeItems, homeClaimed, true, true, true, pad("/CODE1?no_redirect=1"), true, i18n.NewT("en"), "en", ogMeta))

	inv, invPolls := invitationFixtures()
	for i := range inv.Guests {
		inv.Guests[i].FirstName = pad(inv.Guests[i].FirstName)
		inv.Guests[i].LastName = pad(inv.Guests[i].LastName)
		for j := range inv.Guests[i].PollAnswers {
			inv.Guests[i].PollAnswers[j].Notes = pad(inv.Guests[i].PollAnswers[j].Notes)
		}
	}
	for i := range invPolls {
		invPolls[i].Question = pad(invPolls[i].Question)
		invPolls[i].Description = pad(invPolls[i].Description)
	}
	invSettings := fullSettings()
	invSettings.CeremonyAddress = pad(invSettings.CeremonyAddress)
	invSettings.CeremonyLocation = pad(invSettings.CeremonyLocation)
	invSettings.ReceptionAddress = pad(invSettings.ReceptionAddress)
	invSettings.ReceptionLocation = pad(invSettings.ReceptionLocation)
	sweepComponent(t, "invitation long", Invitation(inv, invSettings, invPolls, false, false, i18n.NewT("en"), "en", ogMeta, pad("Title")))
	sweepComponent(t, "ceremony long", Ceremony(invSettings, i18n.NewT("it"), "it", ogMeta))
}

// Which write of a component can surface a writer error depends on the
// flush phase: a write only fails when it is the one that fills the buffer
// past the failure point. A second pass with a different buffer size shifts
// every flush window and makes a different set of writes fail-able. The
// pool must be dropped first, because pooled buffers keep the size they
// were created with.
func TestRenderWriteErrorSweepsPhaseShift(t *testing.T) {
	defer func() {
		runtime.GC()
		runtime.GC()
		templruntime.DefaultBufferSize = 16
	}()
	for _, size := range []int{7, 3, 11, 5, 13, 2, 9, 4, 6} {
		runtime.GC()
		runtime.GC()
		templruntime.DefaultBufferSize = size
		sweepPhaseFixtures(t, size)
	}
}

func sweepPhaseFixtures(t *testing.T, size int) {
	t.Helper()
	settings, content, guests, invitations := dashboardFixtures()
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
	sweepComponent(t, "dashboard full phase", Dashboard(settings, content, guests,
		gifts, 2, 3, len(gifts), 80,
		registry, 2, 3, registry,
		invitations, invitations, 2, 3, "fam",
		polls, soundtrack, 2, 3,
		5, 2, 3, 8, 4, 10,
		4, 1, 0, 0,
		2, 3, "ada", "csrf-token", "Settings saved", i18n.NewT("en"), "en"))

	homeSettings, homeContent, homeItems, homeClaimed := homeFixtures()
	ogMeta := OGMeta{Title: "Davide & Agnese", Description: "Wedding", ImageType: "image/png", ImageWidth: "1200", ImageHeight: "630"}
	sweepComponent(t, "home full phase", Home(homeSettings, homeContent, homeItems, homeClaimed, true, true, true, "/CODE1?no_redirect=1", true, i18n.NewT("en"), "en", ogMeta))

	inv, invPolls := invitationFixtures()
	sweepComponent(t, "invitation form phase", Invitation(inv, fullSettings(), invPolls, false, false, i18n.NewT("en"), "en", ogMeta, "Title"))
	sweepComponent(t, "dashboard empty phase", Dashboard(models.Settings{}, SettingsContent{}, nil,
		nil, 1, 1, 0, 0,
		nil, 1, 1, nil,
		nil, nil, 1, 1, "",
		nil, nil, 1, 1,
		0, 0, 0, 0, 0, 0,
		0, 0, 0, 0,
		1, 1, "", "", "", i18n.NewT("en"), "en"))
	sweepComponent(t, "ceremony phase", Ceremony(fullSettings(), i18n.NewT("en"), "en", ogMeta))
	sweepComponent(t, "invitation viewed phase", Invitation(inv, fullSettings(), invPolls, true, true, i18n.NewT("it"), "it", ogMeta, "Titolo"))
	translator := i18n.NewT("en")
	sweepComponent(t, "edit gift phase", EditGift(models.Gift{ID: 4, Amount: 50, Donor: "Alice", RegistryItemID: intPtr(2), Confirmed: true},
		[]models.RegistryItem{{ID: 1, Name: "Toaster"}, {ID: 2, Name: "Lamp"}}, fullSettings(), "csrf", translator, "en"))
	sweepComponent(t, "layout phase", Layout("Title", "en", translator, ogMeta, true,
		[]ImagePreload{{Href: "/media/1", Media: "(max-width: 767px)"}, {Href: "/media/2"}}, "--x:1;"))
}

// A child component that fails must surface its error through the layout
// instead of producing a half-rendered page.
func TestRenderChildError(t *testing.T) {
	broken := templ.ComponentFunc(func(_ context.Context, _ io.Writer) error {
		return errors.New("child blew up")
	})

	var buf bytes.Buffer
	ctx := templ.WithChildren(context.Background(), broken)
	if err := Layout("Title", "en", i18n.NewT("en"), OGMeta{}, false, nil, "").Render(ctx, &buf); err == nil {
		t.Error("layout with a failing child succeeded, want an error")
	}

	buf.Reset()
	ctx = templ.WithChildren(context.Background(), broken)
	if err := DashboardLayout("Title", "en").Render(ctx, &buf); err == nil {
		t.Error("dashboard layout with a failing child succeeded, want an error")
	}

	// a working child renders its content inside the layout
	working := templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		_, err := io.WriteString(w, "child-content")
		return err
	})
	buf.Reset()
	ctx = templ.WithChildren(context.Background(), working)
	if err := Layout("Title", "en", i18n.NewT("en"), OGMeta{}, false, nil, "").Render(ctx, &buf); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("child-content")) {
		t.Error("layout did not render its child content")
	}
}
