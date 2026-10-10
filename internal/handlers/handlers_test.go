package handlers

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/streambinder/foedus/internal/database"
	"github.com/streambinder/foedus/internal/i18n"
	"github.com/streambinder/foedus/internal/middleware"
	"github.com/streambinder/foedus/internal/models"
	"github.com/streambinder/foedus/internal/spotify"
)

func setupDB(t *testing.T) {
	t.Helper()
	database.Init(filepath.Join(t.TempDir(), "test.db"))
	t.Cleanup(func() { database.DB.Close() })
}

// newTestApp mirrors the route table of main.go without the CSRF and auth
// middleware, which are not part of the handlers under test.
func newTestApp() *fiber.App {
	app := fiber.New(fiber.Config{ErrorHandler: middleware.ErrorHandler})
	app.Use(middleware.RequestContext())
	app.Use(middleware.LangDetect())
	app.Get("/", Home)
	app.Get("/ceremony", Ceremony)
	app.Get("/media/:id", MediaImage)
	app.Get("/og-image", OGImage)
	app.Post("/gift/claim", ClaimGift)
	app.Post("/chat", ChatStream)
	app.Get("/soundtrack/search", SoundtrackSearch)
	app.Post("/soundtrack/add", SoundtrackAdd)
	app.Get("/dashboard/", DashboardIndex)
	app.Get("/dashboard/counters/:category", CounterGuestNames)
	app.Post("/dashboard/settings", SaveSettings)
	app.Post("/dashboard/guests", AddGuest)
	app.Post("/dashboard/guests/import", ImportGuestsCSV)
	app.Get("/dashboard/guests/:id/edit", EditGuestPage)
	app.Post("/dashboard/guests/:id", UpdateGuest)
	app.Post("/dashboard/guests/:id/delete", DeleteGuest)
	app.Post("/dashboard/guests/:id/confirm/:field", CycleConfirmed)
	app.Post("/dashboard/registry", AddRegistryItem)
	app.Get("/dashboard/registry/:id/edit", EditRegistryItemPage)
	app.Post("/dashboard/registry/:id", UpdateRegistryItem)
	app.Post("/dashboard/registry/:id/move/:direction", MoveRegistryItem)
	app.Post("/dashboard/registry/:id/delete", DeleteRegistryItem)
	app.Get("/dashboard/gifts/:id/edit", EditGiftPage)
	app.Post("/dashboard/gifts/:id", UpdateGift)
	app.Post("/dashboard/gifts/:id/delete", DeleteGift)
	app.Post("/dashboard/invitations", CreateInvitation)
	app.Get("/dashboard/invitations/:id/edit", EditInvitationPage)
	app.Post("/dashboard/invitations/:id", UpdateInvitation)
	app.Post("/dashboard/invitations/:id/delete", DeleteInvitation)
	app.Post("/dashboard/invitations/:id/viewed/reset", ResetInvitationViewed)
	app.Post("/dashboard/polls", AddPoll)
	app.Get("/dashboard/polls/:id/edit", EditPollPage)
	app.Post("/dashboard/polls/:id", UpdatePoll)
	app.Post("/dashboard/polls/:id/delete", DeletePoll)
	app.Post("/dashboard/soundtrack/:id/delete", DeleteSoundtrackEvent)
	app.Get("/:code", ViewInvitation)
	app.Post("/:code/viewed", MarkInvitationViewed)
	app.Post("/:code/rsvp", UpdateInvitationRSVP)
	return app
}

func doReq(t *testing.T, app *fiber.App, req *http.Request) (int, http.Header, string) {
	t.Helper()
	resp, err := app.Test(req, 60000)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, resp.Header, string(body)
}

func get(t *testing.T, app *fiber.App, path string) (int, http.Header, string) {
	t.Helper()
	return doReq(t, app, httptest.NewRequest(fiber.MethodGet, path, nil))
}

func postForm(t *testing.T, app *fiber.App, path string, form url.Values) (int, http.Header, string) {
	t.Helper()
	req := httptest.NewRequest(fiber.MethodPost, path, strings.NewReader(form.Encode()))
	req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationForm)
	return doReq(t, app, req)
}

func postJSON(t *testing.T, app *fiber.App, path string, payload any) (int, http.Header, string) {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(fiber.MethodPost, path, strings.NewReader(string(raw)))
	req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	return doReq(t, app, req)
}

func postRaw(t *testing.T, app *fiber.App, path, contentType, body string) (int, http.Header, string) {
	t.Helper()
	req := httptest.NewRequest(fiber.MethodPost, path, strings.NewReader(body))
	if contentType != "" {
		req.Header.Set(fiber.HeaderContentType, contentType)
	}
	return doReq(t, app, req)
}

func configureSettings(t *testing.T, mutate func(*models.Settings)) models.Settings {
	t.Helper()
	settings := models.Settings{
		GroomName: "Davide", BrideName: "Agnese",
		CeremonyDatetime: "2027-07-17T15:30", CeremonyAddress: "Via Roma 1",
		CeremonyLocation: "Chiesa", CeremonyCity: "Terni",
		ReceptionDatetime: "2027-07-17T19:30", ReceptionAddress: "Strada Verde 2",
		ReceptionLocation: "Villa Verde", ReceptionCity: "Terni",
	}
	if mutate != nil {
		mutate(&settings)
	}
	if err := database.UpdateSettings(database.DB, settings); err != nil {
		t.Fatal(err)
	}
	return settings
}

func mustCreateGuest(t *testing.T, first, last, typ string) models.Guest {
	t.Helper()
	if err := database.CreateGuest(first, last, typ); err != nil {
		t.Fatal(err)
	}
	guests, err := database.GetAllGuests()
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range guests {
		if g.FirstName == first && g.LastName == last {
			return g
		}
	}
	t.Fatalf("guest %s %s not found", first, last)
	return models.Guest{}
}

func pngDataURI(t *testing.T, content string) string {
	t.Helper()
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte(content))
}

func TestHomeAndCeremony(t *testing.T) {
	setupDB(t)
	app := newTestApp()
	defer app.Shutdown()

	// unconfigured site: both pages render the setup guard
	status, _, body := get(t, app, "/")
	if status != 200 || !strings.Contains(body, "Setup Required") {
		t.Errorf("unconfigured home: status %d", status)
	}
	status, _, _ = get(t, app, "/ceremony")
	if status != 200 {
		t.Errorf("unconfigured ceremony: status %d", status)
	}

	configureSettings(t, nil)
	status, _, body = get(t, app, "/")
	if status != 200 || !strings.Contains(body, "Davide") {
		t.Errorf("configured home: status %d", status)
	}
	status, _, body = get(t, app, "/ceremony")
	if status != 200 || !strings.Contains(body, "Via Roma 1") {
		t.Errorf("configured ceremony: status %d", status)
	}

	// a valid invite code in the query wires the update URL into the page
	guest := mustCreateGuest(t, "Ada", "Love", "adult")
	code, err := database.CreateInvitation([]int{guest.ID}, "Ada")
	if err != nil {
		t.Fatal(err)
	}
	status, _, _ = get(t, app, "/?invite="+code+"&submitted=1")
	if status != 200 {
		t.Errorf("home with invite: status %d", status)
	}
	// an unknown invite code is ignored, not an error
	status, _, _ = get(t, app, "/?invite=missing")
	if status != 200 {
		t.Errorf("home with unknown invite: status %d", status)
	}
}

func TestHomeErrors(t *testing.T) {
	setupDB(t)
	configureSettings(t, nil)
	app := newTestApp()
	defer app.Shutdown()

	if _, err := database.DB.Exec(`DROP TABLE registry_items`); err != nil {
		t.Fatal(err)
	}
	status, _, body := get(t, app, "/")
	if status != 500 || !strings.Contains(body, "registry") {
		t.Errorf("home without registry table: status %d body %q", status, body)
	}
}

func TestHomeErrorBranches(t *testing.T) {
	for _, tc := range []struct {
		name string
		drop string
		want string
	}{
		{"claimed amounts", "gifts", "claimed amounts"},
		{"homepage content", "places", "homepage content"},
		{"labels", "homepage_labels", "labels"},
		{"impersonation count", "impersonations", "impersonations"},
		{"invitation lookup failure", "invitations", "invitation"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setupDB(t)
			configureSettings(t, nil)
			app := newTestApp()
			defer app.Shutdown()
			if _, err := database.DB.Exec(`DROP TABLE ` + tc.drop); err != nil {
				t.Fatal(err)
			}
			path := "/"
			if tc.drop == "invitations" {
				path = "/?invite=whatever"
			}
			status, _, body := get(t, app, path)
			if status != 500 || !strings.Contains(body, tc.want) {
				t.Errorf("status %d body %q, want 500 mentioning %q", status, body, tc.want)
			}
		})
	}

	t.Run("settings failure", func(t *testing.T) {
		setupDB(t)
		app := newTestApp()
		defer app.Shutdown()
		database.DB.Close()
		status, _, body := get(t, app, "/")
		if status != 500 || !strings.Contains(body, "settings") {
			t.Errorf("status %d body %q", status, body)
		}
		status, _, _ = get(t, app, "/ceremony")
		if status != 500 {
			t.Errorf("ceremony with closed DB: status %d", status)
		}
	})
}

func TestViewInvitation(t *testing.T) {
	setupDB(t)
	app := newTestApp()
	defer app.Shutdown()

	status, _, _ := get(t, app, "/missing")
	if status != 404 {
		t.Errorf("missing invitation: status %d, want 404", status)
	}

	guest := mustCreateGuest(t, "Ada", "Love", "adult")
	code, err := database.CreateInvitation([]int{guest.ID}, "")
	if err != nil {
		t.Fatal(err)
	}

	// site not configured yet: the setup guard wins over the invitation
	status, _, body := get(t, app, "/"+code)
	if status != 200 || !strings.Contains(body, "Setup Required") {
		t.Errorf("unconfigured invitation view: status %d", status)
	}

	configureSettings(t, nil)
	status, _, body = get(t, app, "/"+code)
	if status != 200 || !strings.Contains(body, "Ada") {
		t.Errorf("invitation view: status %d", status)
	}

	// answered guests redirect to the homepage unless no_redirect is set
	yes := true
	if err := database.SetGuestRSVP(guest.ID, &yes, &yes); err != nil {
		t.Fatal(err)
	}
	status, headers, _ := get(t, app, "/"+code)
	if status != 302 || !strings.Contains(headers.Get("Location"), "invite="+code) {
		t.Errorf("answered invitation: status %d location %q", status, headers.Get("Location"))
	}
	status, _, _ = get(t, app, "/"+code+"?no_redirect=1")
	if status != 200 {
		t.Errorf("answered invitation with no_redirect: status %d", status)
	}

	// italian visitor gets the italian chrome
	req := httptest.NewRequest(fiber.MethodGet, "/"+code+"?no_redirect=1", nil)
	req.Header.Set("Accept-Language", "it")
	status, _, _ = doReq(t, app, req)
	if status != 200 {
		t.Errorf("italian invitation view: status %d", status)
	}
}

func TestInvitationViewErrors(t *testing.T) {
	setupDB(t)
	configureSettings(t, nil)
	guest := mustCreateGuest(t, "Ada", "Love", "adult")
	code, err := database.CreateInvitation([]int{guest.ID}, "x")
	if err != nil {
		t.Fatal(err)
	}
	app := newTestApp()
	defer app.Shutdown()

	if _, err := database.DB.Exec(`DROP TABLE polls`); err != nil {
		t.Fatal(err)
	}
	status, _, _ := get(t, app, "/"+code)
	if status != 500 {
		t.Errorf("view without polls table: status %d, want 500", status)
	}
}

func TestMarkAndRSVP(t *testing.T) {
	setupDB(t)
	configureSettings(t, nil)
	app := newTestApp()
	defer app.Shutdown()

	status, _, _ := postForm(t, app, "/missing/viewed", url.Values{})
	if status != 404 {
		t.Errorf("mark missing viewed: status %d, want 404", status)
	}
	status, _, _ = postForm(t, app, "/missing/rsvp", url.Values{})
	if status != 404 {
		t.Errorf("rsvp missing: status %d, want 404", status)
	}

	guest := mustCreateGuest(t, "Ada", "Love", "adult")
	other := mustCreateGuest(t, "Bob", "Ray", "adult")
	code, err := database.CreateInvitation([]int{guest.ID, other.ID}, "pair")
	if err != nil {
		t.Fatal(err)
	}
	if err := database.CreatePoll("Bus?", "From the station"); err != nil {
		t.Fatal(err)
	}
	polls, _ := database.GetAllPolls()
	pollID := polls[0].ID

	status, _, _ = postForm(t, app, "/"+code+"/viewed", url.Values{})
	if status != 204 {
		t.Errorf("mark viewed: status %d, want 204", status)
	}

	form := url.Values{}
	form.Set("ceremony_"+strconv.Itoa(guest.ID), "1")
	form.Set("reception_"+strconv.Itoa(guest.ID), "0")
	form.Set("ceremony_"+strconv.Itoa(other.ID), "0")
	form.Set("reception_"+strconv.Itoa(other.ID), "1")
	form.Set("poll_"+strconv.Itoa(pollID)+"_"+strconv.Itoa(guest.ID), "1")
	form.Set("poll_notes_"+strconv.Itoa(pollID)+"_"+strconv.Itoa(guest.ID), "  window seat ")
	status, headers, _ := postForm(t, app, "/"+code+"/rsvp", form)
	if status != 302 || !strings.Contains(headers.Get("Location"), "submitted=1") {
		t.Errorf("rsvp submit: status %d location %q", status, headers.Get("Location"))
	}

	updated, err := database.GetGuest(guest.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.ConfirmedCeremony == nil || !*updated.ConfirmedCeremony {
		t.Errorf("guest ceremony confirmation = %v, want true", updated.ConfirmedCeremony)
	}
	if updated.ConfirmedReception == nil || *updated.ConfirmedReception {
		t.Errorf("guest reception confirmation = %v, want false", updated.ConfirmedReception)
	}
	answers, err := database.GetPollAnswersForGuests([]int{guest.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(answers[guest.ID]) != 1 || answers[guest.ID][0].Notes != "window seat" {
		t.Errorf("poll answers = %+v", answers)
	}

	// after the event the RSVP endpoint closes
	configureSettings(t, func(s *models.Settings) { s.CeremonyDatetime = "2020-01-01T10:00" })
	status, _, _ = postForm(t, app, "/"+code+"/rsvp", form)
	if status != 403 {
		t.Errorf("rsvp after event: status %d, want 403", status)
	}
}

func TestRSVPErrorBranches(t *testing.T) {
	setupDB(t)
	configureSettings(t, nil)
	guest := mustCreateGuest(t, "Ada", "Love", "adult")
	code, err := database.CreateInvitation([]int{guest.ID}, "x")
	if err != nil {
		t.Fatal(err)
	}
	if err := database.CreatePoll("Bus?", ""); err != nil {
		t.Fatal(err)
	}
	app := newTestApp()
	defer app.Shutdown()

	// without the answers table the handler cannot even load the current
	// answers, so the save path is never reached
	if _, err := database.DB.Exec(`DROP TABLE poll_answers`); err != nil {
		t.Fatal(err)
	}
	form := url.Values{}
	form.Set("ceremony_"+strconv.Itoa(guest.ID), "1")
	status, _, body := postForm(t, app, "/"+code+"/rsvp", form)
	if status != 500 || !strings.Contains(body, "failed to load invitation") {
		t.Errorf("rsvp without answers table: status %d body %q", status, body)
	}
}

func TestClaimGift(t *testing.T) {
	setupDB(t)
	app := newTestApp()
	defer app.Shutdown()

	// not configured yet
	status, _, _ := postJSON(t, app, "/gift/claim", map[string]any{"amount": 10, "donor": "X"})
	if status != 503 {
		t.Errorf("claim while unconfigured: status %d, want 503", status)
	}

	configureSettings(t, nil)

	// malformed body and invalid amounts
	status, _, _ = postRaw(t, app, "/gift/claim", fiber.MIMEApplicationJSON, "{not json")
	if status != 400 {
		t.Errorf("claim with bad json: status %d, want 400", status)
	}
	status, _, _ = postJSON(t, app, "/gift/claim", map[string]any{"amount": 0, "donor": "X"})
	if status != 400 {
		t.Errorf("claim with zero amount: status %d, want 400", status)
	}
	status, _, _ = postJSON(t, app, "/gift/claim", map[string]any{"amount": -5, "donor": "X"})
	if status != 400 {
		t.Errorf("claim with negative amount: status %d, want 400", status)
	}

	// unknown registry item
	missing := 999
	status, _, _ = postJSON(t, app, "/gift/claim", map[string]any{"amount": 10, "registry_item_id": missing})
	if status != 404 {
		t.Errorf("claim for missing item: status %d, want 404", status)
	}

	// generic claim, tied to an invitation via its code
	guest := mustCreateGuest(t, "Ada", "Love", "adult")
	code, err := database.CreateInvitation([]int{guest.ID}, "Ada")
	if err != nil {
		t.Fatal(err)
	}
	status, _, body := postJSON(t, app, "/gift/claim", map[string]any{"amount": 25, "donor": "  Alice ", "invite_code": code})
	if status != 200 || !strings.Contains(body, `"ok":true`) {
		t.Errorf("generic claim: status %d body %q", status, body)
	}
	// unknown invite code: the claim is still recorded, unlinked
	status, _, _ = postJSON(t, app, "/gift/claim", map[string]any{"amount": 5, "donor": "Bob", "invite_code": "nope"})
	if status != 200 {
		t.Errorf("claim with unknown invite: status %d", status)
	}
	gifts, err := database.GetAllGifts()
	if err != nil {
		t.Fatal(err)
	}
	if len(gifts) != 2 {
		t.Fatalf("got %d gifts, want 2", len(gifts))
	}
	var linked, unlinked int
	for _, g := range gifts {
		if g.InvitationID != nil {
			linked++
		} else {
			unlinked++
		}
		if g.Donor == "Alice" && g.Amount != 25 {
			t.Errorf("linked gift = %+v", g)
		}
	}
	if linked != 1 || unlinked != 1 {
		t.Errorf("linked %d unlinked %d, want 1 and 1", linked, unlinked)
	}

	// registry item claims are capped at the remaining amount
	if err := database.CreateRegistryItem("Toaster", 100, 0); err != nil {
		t.Fatal(err)
	}
	items, _ := database.GetAllRegistryItems()
	itemID := items[0].ID
	status, _, _ = postJSON(t, app, "/gift/claim", map[string]any{"amount": 60, "registry_item_id": itemID})
	if status != 200 {
		t.Errorf("first item claim: status %d", status)
	}
	status, _, _ = postJSON(t, app, "/gift/claim", map[string]any{"amount": 41, "registry_item_id": itemID})
	if status != 400 {
		t.Errorf("claim exceeding remaining: status %d, want 400", status)
	}
	status, _, _ = postJSON(t, app, "/gift/claim", map[string]any{"amount": 40, "registry_item_id": itemID})
	if status != 200 {
		t.Errorf("claim of exact remaining: status %d", status)
	}
	// an open-ended item has no cap
	if err := database.CreateRegistryItem("Open", 0, 0); err != nil {
		t.Fatal(err)
	}
	items, _ = database.GetAllRegistryItems()
	for _, item := range items {
		if item.Name == "Open" {
			status, _, _ = postJSON(t, app, "/gift/claim", map[string]any{"amount": 10000, "registry_item_id": item.ID})
			if status != 200 {
				t.Errorf("open-ended claim: status %d", status)
			}
		}
	}

	// after the event all claims close — and the passed check runs before
	// the configured check
	configureSettings(t, func(s *models.Settings) {
		s.CeremonyDatetime = "2020-01-01T10:00"
		s.GroomName = ""
		s.BrideName = ""
	})
	status, _, _ = postJSON(t, app, "/gift/claim", map[string]any{"amount": 10})
	if status != 403 {
		t.Errorf("claim after event: status %d, want 403", status)
	}
}

func TestClaimGiftErrors(t *testing.T) {
	setupDB(t)
	configureSettings(t, nil)
	app := newTestApp()
	defer app.Shutdown()

	if err := database.CreateRegistryItem("Toaster", 100, 0); err != nil {
		t.Fatal(err)
	}
	items, _ := database.GetAllRegistryItems()
	if _, err := database.DB.Exec(`DROP TABLE gifts`); err != nil {
		t.Fatal(err)
	}
	// the claimed-amounts lookup fails first for item claims
	status, _, _ := postJSON(t, app, "/gift/claim", map[string]any{"amount": 10, "registry_item_id": items[0].ID})
	if status != 500 {
		t.Errorf("item claim without gifts table: status %d, want 500", status)
	}
	// a generic claim reaches the insert and fails there
	status, _, _ = postJSON(t, app, "/gift/claim", map[string]any{"amount": 10, "donor": "X"})
	if status != 500 {
		t.Errorf("generic claim without gifts table: status %d, want 500", status)
	}
}

func TestMediaImage(t *testing.T) {
	setupDB(t)
	app := newTestApp()
	defer app.Shutdown()
	resetMediaCache()

	for _, path := range []string{"/media/abc", "/media/0", "/media/-1", "/media/999"} {
		status, _, _ := get(t, app, path)
		if status != 404 {
			t.Errorf("GET %s: status %d, want 404", path, status)
		}
	}

	id, err := database.InsertMedia(database.DB, "image/png", []byte("png-bytes"))
	if err != nil {
		t.Fatal(err)
	}
	path := "/media/" + strconv.Itoa(id)
	status, headers, body := get(t, app, path)
	if status != 200 || body != "png-bytes" {
		t.Errorf("media fetch: status %d body %q", status, body)
	}
	etag := headers.Get(fiber.HeaderETag)
	if etag == "" {
		t.Fatal("media response has no ETag")
	}
	// the second fetch is served from the cache
	status, _, body = get(t, app, path)
	if status != 200 || body != "png-bytes" {
		t.Errorf("cached media fetch: status %d body %q", status, body)
	}
	// a matching If-None-Match short-circuits before the cache
	req := httptest.NewRequest(fiber.MethodGet, path, nil)
	req.Header.Set(fiber.HeaderIfNoneMatch, etag)
	status, _, _ = doReq(t, app, req)
	if status != 304 {
		t.Errorf("conditional media fetch: status %d, want 304", status)
	}
	// a stale ETag gets the full body
	req = httptest.NewRequest(fiber.MethodGet, path, nil)
	req.Header.Set(fiber.HeaderIfNoneMatch, `"other"`)
	status, _, _ = doReq(t, app, req)
	if status != 200 {
		t.Errorf("stale etag media fetch: status %d, want 200", status)
	}
}

func resetMediaCache() {
	mediaCacheMu.Lock()
	defer mediaCacheMu.Unlock()
	mediaCache = make(map[int]cachedMedia)
	mediaCacheBytes = 0
}

func TestMediaCacheEviction(t *testing.T) {
	setupDB(t)
	app := newTestApp()
	defer app.Shutdown()
	resetMediaCache()

	id, err := database.InsertMedia(database.DB, "image/png", []byte("small"))
	if err != nil {
		t.Fatal(err)
	}
	// pretend the cache is already at capacity: the next insert evicts all
	mediaCacheMu.Lock()
	mediaCacheBytes = mediaCacheMaxBytes
	mediaCacheMu.Unlock()

	status, _, body := get(t, app, "/media/"+strconv.Itoa(id))
	if status != 200 || body != "small" {
		t.Errorf("evicting fetch: status %d body %q", status, body)
	}
	mediaCacheMu.RLock()
	defer mediaCacheMu.RUnlock()
	if len(mediaCache) != 1 || mediaCacheBytes != len("small") {
		t.Errorf("cache after eviction: %d entries, %d bytes", len(mediaCache), mediaCacheBytes)
	}
}

func TestOGImageAndMeta(t *testing.T) {
	setupDB(t)
	configureSettings(t, nil)
	app := newTestApp()
	defer app.Shutdown()

	// no share preview configured: falls back to the static default file,
	// which does not exist relative to the test working directory
	status, _, _ := get(t, app, "/og-image")
	if status == 200 {
		t.Error("og-image without preview media unexpectedly succeeded")
	}

	mediaID, err := database.InsertMedia(database.DB, "image/jpeg", []byte("og-bytes"))
	if err != nil {
		t.Fatal(err)
	}
	configureSettings(t, func(s *models.Settings) { s.SharePreviewMediaID = mediaID })
	status, headers, body := get(t, app, "/og-image")
	if status != 200 || body != "og-bytes" || headers.Get(fiber.HeaderContentType) != "image/jpeg" {
		t.Errorf("og-image with media: status %d body %q type %q", status, body, headers.Get(fiber.HeaderContentType))
	}

	// BuildOGMeta: the media mime replaces the default type and drops the
	// dimensions, a missing media keeps the defaults
	meta := BuildOGMeta("https://x", "https://x/", "T", "D", models.Settings{SharePreviewMediaID: mediaID})
	if meta.ImageType != "image/jpeg" || meta.ImageWidth != "" || meta.ImageHeight != "" {
		t.Errorf("meta with media = %+v", meta)
	}
	if meta.ImageURL != "https://x/og-image" || meta.URL != "https://x/" || meta.Title != "T" {
		t.Errorf("meta fields = %+v", meta)
	}
	meta = BuildOGMeta("https://x", "https://x/", "T", "D", models.Settings{SharePreviewMediaID: 9999})
	if meta.ImageType != "image/png" || meta.ImageWidth != "1200" || meta.ImageHeight != "630" {
		t.Errorf("meta with missing media = %+v", meta)
	}
	meta = BuildOGMeta("https://x", "https://x/", "T", "D", models.Settings{})
	if meta.ImageType != "image/png" {
		t.Errorf("meta without media = %+v", meta)
	}
	// an empty stored mime keeps the defaults as well
	emptyMimeID, err := database.InsertMedia(database.DB, "", []byte("x"))
	if err != nil {
		t.Fatal(err)
	}
	meta = BuildOGMeta("https://x", "https://x/", "T", "D", models.Settings{SharePreviewMediaID: emptyMimeID})
	if meta.ImageType != "image/png" {
		t.Errorf("meta with empty mime = %+v", meta)
	}
}

func TestOGCeremonyLocation(t *testing.T) {
	for _, tc := range []struct {
		name     string
		settings models.Settings
		want     string
	}{
		{"address and city", models.Settings{CeremonyAddress: "Via A", CeremonyCity: "Terni", CeremonyLocation: "Chiesa"}, "Via A, Terni"},
		{"address only", models.Settings{CeremonyAddress: "Via A", CeremonyLocation: "Chiesa"}, "Via A"},
		{"city only", models.Settings{CeremonyCity: "Terni", CeremonyLocation: "Chiesa"}, "Terni"},
		{"location fallback", models.Settings{CeremonyLocation: "Chiesa"}, "Chiesa"},
		{"nothing", models.Settings{}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := ogCeremonyLocation(tc.settings); got != tc.want {
				t.Errorf("ogCeremonyLocation = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPickHeroBackground(t *testing.T) {
	setupDB(t)

	if got := pickHeroBackground(nil); got != (models.HeroBackground{}) {
		t.Errorf("no backgrounds = %+v", got)
	}
	dead := models.HeroBackground{DesktopMediaID: 9998, MobileMediaID: 9999}
	if got := pickHeroBackground([]models.HeroBackground{dead}); got != (models.HeroBackground{}) {
		t.Errorf("all dead backgrounds = %+v", got)
	}
	desktopID, err := database.InsertMedia(database.DB, "image/jpeg", []byte("d"))
	if err != nil {
		t.Fatal(err)
	}
	mobileID, err := database.InsertMedia(database.DB, "image/jpeg", []byte("m"))
	if err != nil {
		t.Fatal(err)
	}
	// a dead desktop side falls back to the live mobile image, and vice versa
	got := pickHeroBackground([]models.HeroBackground{{DesktopMediaID: 9999, MobileMediaID: mobileID}})
	if got.DesktopMediaID != mobileID || got.MobileMediaID != mobileID {
		t.Errorf("desktop fallback = %+v", got)
	}
	got = pickHeroBackground([]models.HeroBackground{{DesktopMediaID: desktopID, MobileMediaID: 9999}})
	if got.DesktopMediaID != desktopID || got.MobileMediaID != desktopID {
		t.Errorf("mobile fallback = %+v", got)
	}
	got = pickHeroBackground([]models.HeroBackground{{DesktopMediaID: desktopID, MobileMediaID: mobileID}})
	if got.DesktopMediaID != desktopID || got.MobileMediaID != mobileID {
		t.Errorf("intact pair = %+v", got)
	}
	if mediaExists(0) || mediaExists(-1) {
		t.Error("non-positive media ids must not exist")
	}
	if !mediaExists(desktopID) {
		t.Error("inserted media must exist")
	}
	if mediaExists(9999) {
		t.Error("missing media must not exist")
	}
}

func TestHelperFunctions(t *testing.T) {
	// parseRSVPField
	if got := parseRSVPField("1"); got == nil || !*got {
		t.Errorf("parseRSVPField(1) = %v", got)
	}
	if got := parseRSVPField("0"); got == nil || *got {
		t.Errorf("parseRSVPField(0) = %v", got)
	}
	if got := parseRSVPField(""); got != nil {
		t.Errorf("parseRSVPField(empty) = %v", got)
	}
	if got := parseRSVPField("yes"); got != nil {
		t.Errorf("parseRSVPField(yes) = %v", got)
	}

	// invitationTitle
	translator := i18n.NewT("en")
	inv := models.Invitation{Label: " Famiglia "}
	if got := invitationTitle(translator, inv, "Davide", "Agnese"); !strings.Contains(got, "Famiglia") || !strings.Contains(got, "Davide & Agnese") {
		t.Errorf("title with label = %q", got)
	}
	inv = models.Invitation{Guests: []models.Guest{{FirstName: "Ada"}, {FirstName: "Bob"}}}
	if got := invitationTitle(translator, inv, "", ""); !strings.Contains(got, "Ada & Bob") {
		t.Errorf("title from guests = %q", got)
	}
	inv = models.Invitation{}
	if got := invitationTitle(translator, inv, "Davide", ""); strings.Contains(got, "·") {
		t.Errorf("title without both names = %q", got)
	}

	// spotifyPlaylistID
	if got := spotifyPlaylistID("https://open.spotify.com/playlist/abc123XYZ"); got != "abc123XYZ" {
		t.Errorf("playlist id = %q", got)
	}
	if got := spotifyPlaylistID("https://example.com/nothing"); got != "" {
		t.Errorf("playlist id from other url = %q", got)
	}
	if got := spotifyPlaylistID(""); got != "" {
		t.Errorf("playlist id empty = %q", got)
	}

	// errString
	if got := errString(nil); got != "" {
		t.Errorf("errString(nil) = %q", got)
	}
	if got := errString(io.EOF); got != "EOF" {
		t.Errorf("errString(EOF) = %q", got)
	}

	// paginateSlice
	window, page, total := paginateSlice([]int{1, 2, 3, 4, 5}, 2, 2)
	if len(window) != 2 || window[0] != 3 || page != 2 || total != 3 {
		t.Errorf("paginate middle = %v %d %d", window, page, total)
	}
	window, page, total = paginateSlice([]int{1, 2, 3}, 99, 2)
	if len(window) != 1 || window[0] != 3 || page != 2 || total != 2 {
		t.Errorf("paginate clamped end = %v %d %d", window, page, total)
	}
	window, page, total = paginateSlice([]int{1, 2, 3}, 0, 2)
	if len(window) != 2 || page != 1 || total != 2 {
		t.Errorf("paginate clamped start = %v %d %d", window, page, total)
	}
	window, page, total = paginateSlice([]int{}, 1, 10)
	if len(window) != 0 || page != 1 || total != 1 {
		t.Errorf("paginate empty = %v %d %d", window, page, total)
	}
	window, _, _ = paginateSlice([]int{1, 2, 3}, 1, 10)
	if len(window) != 3 {
		t.Errorf("paginate oversized page = %v", window)
	}

	// filterInvitations
	invs := []models.Invitation{
		{Label: "Famiglia Pucci", Guests: []models.Guest{{FirstName: "Ada", LastName: "Love"}}},
		{Label: "", Guests: []models.Guest{{FirstName: "Bob", LastName: "Ray"}}},
		{Label: "Colleghi", Guests: []models.Guest{{FirstName: "Cid", LastName: "Ray"}}},
	}
	if got := filterInvitations(invs, "  "); len(got) != 3 {
		t.Errorf("blank filter = %d, want all", len(got))
	}
	if got := filterInvitations(invs, "pucci"); len(got) != 1 {
		t.Errorf("label filter = %d", len(got))
	}
	if got := filterInvitations(invs, "RAY"); len(got) != 2 {
		t.Errorf("last name filter = %d", len(got))
	}
	if got := filterInvitations(invs, "ada"); len(got) != 1 {
		t.Errorf("first name filter = %d", len(got))
	}
	if got := filterInvitations(invs, "zzz"); len(got) != 0 {
		t.Errorf("no match filter = %d", len(got))
	}

	// normalizeGuestType: note it returns the raw value for known types
	// (the caller trims first in the handlers that pass form values through
	// TrimSpace — AddGuest trims into guestType via normalize on the raw
	// form value which fiber already decoded)
	for raw, want := range map[string]string{
		"child": "child", "infant": "infant", "vendor": "vendor",
		"adult": "adult", "": "adult", " alien ": "adult",
	} {
		if got := normalizeGuestType(raw); got != want {
			t.Errorf("normalizeGuestType(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestGetTAndLangFallbacks(t *testing.T) {
	app := fiber.New()
	defer app.Shutdown()
	app.Get("/x", func(c *fiber.Ctx) error {
		// no LangDetect middleware here: both helpers must fall back
		if got := getLang(c); got != "en" {
			t.Errorf("getLang fallback = %q", got)
		}
		translator := getT(c)
		if got := translator("title.setup"); got != "Setup Required" {
			t.Errorf("getT fallback = %q", got)
		}
		c.Locals("lang", "it")
		c.Locals("t", i18n.NewT("it"))
		if got := getLang(c); got != "it" {
			t.Errorf("getLang local = %q", got)
		}
		if got := getT(c)("title.setup"); got != "Configurazione necessaria" {
			t.Errorf("getT local = %q", got)
		}
		// mistyped locals fall back too
		c.Locals("lang", 3)
		c.Locals("t", "nope")
		if got := getLang(c); got != "en" {
			t.Errorf("getLang mistyped = %q", got)
		}
		if got := getT(c)("title.setup"); got != "Setup Required" {
			t.Errorf("getT mistyped = %q", got)
		}
		return c.SendString("ok")
	})
	status, _, _ := doReq(t, app, httptest.NewRequest(fiber.MethodGet, "/x", nil))
	if status != 200 {
		t.Errorf("status %d", status)
	}
}

func TestFlashHelpers(t *testing.T) {
	app := fiber.New()
	defer app.Shutdown()
	app.Get("/set", func(c *fiber.Ctx) error {
		setFlash(c, "hello world")
		return c.SendString("set")
	})
	app.Get("/read", func(c *fiber.Ctx) error {
		return c.SendString(getFlash(c))
	})
	_, headers, _ := doReq(t, app, httptest.NewRequest(fiber.MethodGet, "/set", nil))
	cookies := headers.Get(fiber.HeaderSetCookie)
	if !strings.Contains(cookies, "flash=hello+world") {
		t.Errorf("flash cookie = %q", cookies)
	}
	// no cookie: empty flash
	_, _, body := doReq(t, app, httptest.NewRequest(fiber.MethodGet, "/read", nil))
	if body != "" {
		t.Errorf("flash without cookie = %q", body)
	}
	// escaped cookie value round-trips
	req := httptest.NewRequest(fiber.MethodGet, "/read", nil)
	req.AddCookie(&http.Cookie{Name: "flash", Value: "hello+world"})
	_, _, body = doReq(t, app, req)
	if body != "hello world" {
		t.Errorf("flash from cookie = %q", body)
	}
	// a malformed escape decodes to an empty message instead of failing;
	// the raw header form is required because http.Cookie sanitizes it
	req = httptest.NewRequest(fiber.MethodGet, "/read", nil)
	req.Header.Set("Cookie", "flash=%zz")
	_, _, body = doReq(t, app, req)
	if body != "" {
		t.Errorf("flash malformed = %q", body)
	}
}

func TestSpotifyInitDefaults(t *testing.T) {
	// keep the spotify package in a known state for the soundtrack tests:
	// other tests re-initialise it as needed
	spotify.Init("", "", "")
	if SoundtrackEnabled() {
		t.Error("soundtrack must be disabled without spotify credentials")
	}
}
