package handlers

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/streambinder/foedus/internal/database"
	"github.com/streambinder/foedus/internal/models"
)

func TestDashboardIndex(t *testing.T) {
	setupDB(t)
	configureSettings(t, nil)
	app := newTestApp()
	defer app.Shutdown()

	guest := mustCreateGuest(t, "Ada", "Love", "adult")
	mustCreateGuest(t, "Bob", "Ray", "child")
	if _, err := database.CreateInvitation([]int{guest.ID}, "Ada"); err != nil {
		t.Fatal(err)
	}
	if err := database.CreateRegistryItem("Toaster", 100, 0); err != nil {
		t.Fatal(err)
	}
	if err := database.CreateGift(50, "Alice", nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := database.CreatePoll("Bus?", "desc"); err != nil {
		t.Fatal(err)
	}
	if err := database.SavePollAnswers(guest.ID, map[int]models.PollAnswer{1: {Answer: true}}); err != nil {
		t.Fatal(err)
	}
	if err := database.CreateSoundtrackEvent("Song", "Artist", "https://open.spotify.com/track/x", ""); err != nil {
		t.Fatal(err)
	}

	status, _, body := get(t, app, "/dashboard/?guest_page=1&invite_page=1&gifts_page=1&registry_page=1&soundtrack_page=1&search=ada")
	if status != 200 || !strings.Contains(body, "Ada") {
		t.Errorf("dashboard: status %d", status)
	}
	// out-of-range pages clamp instead of erroring
	status, _, _ = get(t, app, "/dashboard/?guest_page=99&invite_page=-2&gifts_page=abc")
	if status != 200 {
		t.Errorf("dashboard clamped pages: status %d", status)
	}
	// flash cookie message renders and clears
	req := httptest.NewRequest(fiber.MethodGet, "/dashboard/", nil)
	req.AddCookie(&http.Cookie{Name: "flash", Value: "Saved+ok"})
	status, _, body = doReq(t, app, req)
	if status != 200 || !strings.Contains(body, "Saved ok") {
		t.Errorf("dashboard with flash: status %d", status)
	}
}

func TestDashboardIndexErrors(t *testing.T) {
	// note: dropping invitations is not in this table because the guest
	// counters query the invitations table first, so the paginated
	// invitation query cannot fail on its own through table drops
	for _, tc := range []struct {
		name string
		drop string
		want string
	}{
		{"content load", "places", "settings content"},
		{"guest counts", "guests", "failed to count guests"},
		{"gift list", "gifts", "failed to load gifts"},
		{"registry list", "registry_items", "failed to load registry items"},
		{"poll list", "polls", "failed to load polls"},
		{"soundtrack list", "soundtrack_events", "failed to load soundtrack events"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setupDB(t)
			configureSettings(t, nil)
			app := newTestApp()
			defer app.Shutdown()
			if _, err := database.DB.Exec(`DROP TABLE ` + tc.drop); err != nil {
				t.Fatal(err)
			}
			status, _, body := get(t, app, "/dashboard/")
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
		status, _, body := get(t, app, "/dashboard/")
		if status != 500 || !strings.Contains(body, "settings") {
			t.Errorf("status %d body %q", status, body)
		}
	})
}

func TestCounterGuestNames(t *testing.T) {
	setupDB(t)
	app := newTestApp()
	defer app.Shutdown()

	ada := mustCreateGuest(t, "Ada", "Love", "adult")
	bob := mustCreateGuest(t, "Bob", "Ray", "adult")
	cid := mustCreateGuest(t, "Cid", "Ray", "child")
	dot := mustCreateGuest(t, "Dot", "Ray", "infant")
	mustCreateGuest(t, "Eve", "Solo", "adult")
	yes, no := true, false
	if err := database.SetGuestRSVP(ada.ID, &yes, &yes); err != nil {
		t.Fatal(err)
	}
	if err := database.SetGuestRSVP(bob.ID, &no, &no); err != nil {
		t.Fatal(err)
	}
	if err := database.SetGuestRSVP(cid.ID, &yes, &yes); err != nil {
		t.Fatal(err)
	}
	// ada, bob, cid and dot are invited; ada's invitation has been viewed
	viewedCode, err := database.CreateInvitation([]int{ada.ID}, "ada")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.CreateInvitation([]int{bob.ID, cid.ID, dot.ID}, "group"); err != nil {
		t.Fatal(err)
	}
	viewed, err := database.GetInvitationByCode(viewedCode)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.MarkInvitationViewed(viewed.ID); err != nil {
		t.Fatal(err)
	}

	cases := map[string]int{
		"confirmed_reception":        2,
		"refused_reception":          1,
		"pending_rsvp":               1,
		"viewed":                     1,
		"nonvisualized":              3,
		"invited":                    4,
		"uninvited":                  1,
		"confirmed_reception_adult":  1,
		"confirmed_reception_child":  1,
		"confirmed_reception_infant": 0,
		"confirmed_reception_vendor": 0,
	}
	for category, want := range cases {
		status, _, body := get(t, app, "/dashboard/counters/"+category)
		var payload struct {
			Category string     `json:"category"`
			Groups   [][]string `json:"groups"`
		}
		if err := json.Unmarshal([]byte(body), &payload); err != nil {
			t.Fatalf("counter %s: %v (body %q)", category, err, body)
		}
		total := 0
		for _, group := range payload.Groups {
			total += len(group)
		}
		if status != 200 || payload.Category != category || total != want {
			t.Errorf("counter %s = %d names (status %d), want %d", category, total, status, want)
		}
	}
	status, _, _ := get(t, app, "/dashboard/counters/bogus")
	if status != 400 {
		t.Errorf("unknown counter: status %d, want 400", status)
	}
}

func TestGuestCRUD(t *testing.T) {
	setupDB(t)
	app := newTestApp()
	defer app.Shutdown()

	// add: names are trimmed, an unknown type falls back to adult
	form := url.Values{
		"first_name": {" Ada "}, "last_name": {"Love"}, "type": {"child"},
	}
	status, headers, _ := postForm(t, app, "/dashboard/guests", form)
	if status != 302 || headers.Get("Location") != "/dashboard" {
		t.Errorf("add guest: status %d location %q", status, headers.Get("Location"))
	}
	guests, _ := database.GetAllGuests()
	if len(guests) != 1 || guests[0].FirstName != "Ada" || guests[0].Type != "child" {
		t.Fatalf("guests after add = %+v", guests)
	}
	guest := mustCreateGuest(t, "Bob", "Ray", "adult")

	// edit page: invalid id, missing guest, happy path
	status, _, _ = get(t, app, "/dashboard/guests/abc/edit")
	if status != 400 {
		t.Errorf("edit invalid id: status %d", status)
	}
	status, _, _ = get(t, app, "/dashboard/guests/999/edit")
	if status != 404 {
		t.Errorf("edit missing guest: status %d", status)
	}
	status, _, body := get(t, app, "/dashboard/guests/"+strconv.Itoa(guest.ID)+"/edit")
	if status != 200 || !strings.Contains(body, "Bob") {
		t.Errorf("edit page: status %d", status)
	}

	// update: renames and retypes; a missing guest is a silent no-op
	form = url.Values{"first_name": {"Robert"}, "last_name": {"Ray"}, "type": {"vendor"}}
	status, _, _ = postForm(t, app, "/dashboard/guests/"+strconv.Itoa(guest.ID), form)
	if status != 302 {
		t.Errorf("update guest: status %d", status)
	}
	updated, err := database.GetGuest(guest.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.FirstName != "Robert" || updated.Type != "vendor" {
		t.Errorf("updated guest = %+v", updated)
	}
	status, _, _ = postForm(t, app, "/dashboard/guests/999", url.Values{"first_name": {"Ghost"}})
	if status != 302 {
		t.Errorf("update missing guest: status %d", status)
	}
	status, _, _ = postForm(t, app, "/dashboard/guests/abc", url.Values{"first_name": {"X"}})
	if status != 400 {
		t.Errorf("update invalid id: status %d, want 400", status)
	}

	// confirm cycling: an unknown field and a malformed id are plain 400s,
	// a valid field cycles the stored tri-state
	status, _, _ = postForm(t, app, "/dashboard/guests/"+strconv.Itoa(guest.ID)+"/confirm/bogus", url.Values{})
	if status != 400 {
		t.Errorf("cycle bogus field: status %d, want 400", status)
	}
	status, _, _ = postForm(t, app, "/dashboard/guests/"+strconv.Itoa(guest.ID)+"/confirm/ceremony", url.Values{})
	if status != 302 {
		t.Errorf("cycle ceremony: status %d", status)
	}
	updated, _ = database.GetGuest(guest.ID)
	if updated.ConfirmedCeremony == nil || !*updated.ConfirmedCeremony {
		t.Errorf("after cycle: confirmed ceremony = %v", updated.ConfirmedCeremony)
	}
	status, _, _ = postForm(t, app, "/dashboard/guests/abc/confirm/ceremony", url.Values{})
	if status != 400 {
		t.Errorf("cycle invalid id: status %d, want 400", status)
	}

	// delete
	status, _, _ = postForm(t, app, "/dashboard/guests/"+strconv.Itoa(guest.ID)+"/delete", url.Values{})
	if status != 302 {
		t.Errorf("delete guest: status %d", status)
	}
	if _, err := database.GetGuest(guest.ID); err == nil {
		t.Error("guest still present after delete")
	}
	status, _, _ = postForm(t, app, "/dashboard/guests/abc/delete", url.Values{})
	if status != 400 {
		t.Errorf("delete invalid id: status %d, want 400", status)
	}
}

func TestGuestCRUDErrorsClosedDB(t *testing.T) {
	setupDB(t)
	app := newTestApp()
	defer app.Shutdown()
	guest := mustCreateGuest(t, "Ada", "Love", "adult")
	database.DB.Close()

	status, _, body := postForm(t, app, "/dashboard/guests", url.Values{"first_name": {"X"}})
	if status != 500 || !strings.Contains(body, "failed to add guest") {
		t.Errorf("add with closed DB: status %d body %q", status, body)
	}
	status, _, body = postForm(t, app, "/dashboard/guests/"+strconv.Itoa(guest.ID), url.Values{"first_name": {"Y"}})
	if status != 500 || !strings.Contains(body, "failed to update guest") {
		t.Errorf("update with closed DB: status %d body %q", status, body)
	}
	status, _, body = postForm(t, app, "/dashboard/guests/"+strconv.Itoa(guest.ID)+"/delete", url.Values{})
	if status != 500 || !strings.Contains(body, "failed to delete guest") {
		t.Errorf("delete with closed DB: status %d body %q", status, body)
	}
	status, _, body = get(t, app, "/dashboard/counters/invited")
	if status != 500 {
		t.Errorf("counter with closed DB: status %d body %q", status, body)
	}
}

func TestEditGuestPageSettingsError(t *testing.T) {
	setupDB(t)
	app := newTestApp()
	defer app.Shutdown()
	guest := mustCreateGuest(t, "Ada", "Love", "adult")
	if _, err := database.DB.Exec(`DROP TABLE settings`); err != nil {
		t.Fatal(err)
	}
	status, _, body := get(t, app, "/dashboard/guests/"+strconv.Itoa(guest.ID)+"/edit")
	if status != 500 || !strings.Contains(body, "settings") {
		t.Errorf("status %d body %q", status, body)
	}
}

func TestImportGuestsCSV(t *testing.T) {
	setupDB(t)
	app := newTestApp()
	defer app.Shutdown()

	importCSV := func(t *testing.T, content string) (int, http.Header, string) {
		t.Helper()
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		fw, err := mw.CreateFormFile("csv_file", "guests.csv")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
		if err := mw.Close(); err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(fiber.MethodPost, "/dashboard/guests/import", &buf)
		req.Header.Set(fiber.HeaderContentType, mw.FormDataContentType())
		return doReq(t, app, req)
	}

	// no file at all: a plain 400
	req := httptest.NewRequest(fiber.MethodPost, "/dashboard/guests/import", strings.NewReader(""))
	req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationForm)
	status, _, _ := doReq(t, app, req)
	if status != 400 {
		t.Errorf("import without file: status %d, want 400", status)
	}

	csv := "First Name,Last Name,Type\r\n" +
		"Ada,Love,adult\r\n" +
		"Bob,Ray,child\r\n" +
		"Solo\r\n" +
		",Rossi,adult\r\n" +
		"Bad\"Quote,Row,adult\r\n" +
		"Eve,Verdi,alien\r\n"
	status, _, _ = importCSV(t, csv)
	if status != 302 {
		t.Errorf("import: status %d", status)
	}
	guests, err := database.GetAllGuests()
	if err != nil {
		t.Fatal(err)
	}
	// the header row and the bare-quote record are skipped; a trailing
	// column that is not a known type becomes part of the name, so the
	// Eve row lands as first name "Eve Verdi", last name "alien"
	if len(guests) != 5 {
		t.Fatalf("imported %d guests: %+v", len(guests), guests)
	}
	types := map[string]string{}
	for _, g := range guests {
		types[g.FirstName] = g.Type
	}
	if types["Eve Verdi"] != "adult" || types["Bob"] != "child" || types["Solo"] != "adult" || types["Rossi"] != "adult" {
		t.Errorf("imported types = %v", types)
	}

	// a second import with duplicates: CreateGuest ignores them silently
	// through the database uniqueness rules — the endpoint still redirects
	status, _, _ = importCSV(t, "Ada,Love,adult\r\n")
	if status != 302 {
		t.Errorf("reimport: status %d", status)
	}
}

func TestRegistryCRUD(t *testing.T) {
	setupDB(t)
	app := newTestApp()
	defer app.Shutdown()

	// invalid submissions are rejected with a plain 400
	for name, form := range map[string]url.Values{
		"no name":     {"price": {"10"}},
		"bad price":   {"name": {"X"}, "price": {"abc"}},
		"fractional":  {"name": {"X"}, "price": {"9.99"}},
		"negative":    {"name": {"X"}, "price": {"-1"}},
		"bad image":   {"name": {"X"}, "price": {"1"}, "image": {"data:image/png;base64,!!!"}},
		"wrong image": {"name": {"X"}, "price": {"1"}, "image": {"data:image/gif;base64,AAAA"}},
	} {
		status, _, _ := postForm(t, app, "/dashboard/registry", form)
		if status != 400 {
			t.Errorf("add registry (%s): status %d, want 400", name, status)
		}
	}
	items, _ := database.GetAllRegistryItems()
	if len(items) != 0 {
		t.Fatalf("invalid adds created %d items", len(items))
	}

	// valid add with image
	form := url.Values{"name": {"Toaster"}, "price": {"99"}, "image": {pngDataURI(t, "toast-img")}}
	status, _, _ := postForm(t, app, "/dashboard/registry", form)
	if status != 302 {
		t.Errorf("add registry with image: status %d", status)
	}
	items, _ = database.GetAllRegistryItems()
	if len(items) != 1 || items[0].MediaID == 0 {
		t.Fatalf("items after add = %+v", items)
	}
	item := items[0]

	// edit page variants
	status, _, _ = get(t, app, "/dashboard/registry/abc/edit")
	if status != 400 {
		t.Errorf("edit registry invalid id: status %d", status)
	}
	status, _, _ = get(t, app, "/dashboard/registry/999/edit")
	if status != 404 {
		t.Errorf("edit registry missing: status %d", status)
	}
	status, _, body := get(t, app, "/dashboard/registry/"+strconv.Itoa(item.ID)+"/edit")
	if status != 200 || !strings.Contains(body, "Toaster") {
		t.Errorf("edit registry page: status %d", status)
	}

	// update keeping the stored media (media_id matches), then replacing it
	keepForm := url.Values{
		"name": {"Toaster Pro"}, "price": {"120"},
		"media_id": {strconv.Itoa(item.MediaID)}, "image": {""},
	}
	status, _, _ = postForm(t, app, "/dashboard/registry/"+strconv.Itoa(item.ID), keepForm)
	if status != 302 {
		t.Errorf("update keep media: status %d", status)
	}
	updated, err := database.GetRegistryItem(item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Toaster Pro" || updated.MediaID != item.MediaID || updated.Price != 120 {
		t.Errorf("kept update = %+v", updated)
	}
	replaceForm := url.Values{
		"name": {"Toaster Pro"}, "price": {"120"},
		"media_id": {strconv.Itoa(item.MediaID)}, "image": {pngDataURI(t, "new-img")},
	}
	status, _, _ = postForm(t, app, "/dashboard/registry/"+strconv.Itoa(item.ID), replaceForm)
	if status != 302 {
		t.Errorf("update replace media: status %d", status)
	}
	updated, _ = database.GetRegistryItem(item.ID)
	if updated.MediaID == item.MediaID || updated.MediaID == 0 {
		t.Errorf("replaced update media = %d, want a new id", updated.MediaID)
	}
	firstMedia := updated.MediaID
	// clearing the image removes the media row
	clearForm := url.Values{"name": {"Toaster Pro"}, "price": {"120"}, "media_id": {"0"}, "image": {""}}
	status, _, _ = postForm(t, app, "/dashboard/registry/"+strconv.Itoa(item.ID), clearForm)
	if status != 302 {
		t.Errorf("update clear media: status %d", status)
	}
	updated, _ = database.GetRegistryItem(item.ID)
	if updated.MediaID != 0 {
		t.Errorf("cleared update media = %d", updated.MediaID)
	}
	// the replaced media row survives as an orphan: the eager delete in
	// the resolver runs while the item still references the row, so the
	// foreign key rejects it and the error is deliberately ignored
	if _, err := database.GetMedia(firstMedia); err != nil {
		t.Error("replaced media row should survive as an orphan")
	}
	// a media_id that matches neither the stored value nor an upload is rejected
	status, _, _ = postForm(t, app, "/dashboard/registry/"+strconv.Itoa(item.ID),
		url.Values{"name": {"X"}, "price": {"1"}, "media_id": {"4242"}})
	if status != 400 {
		t.Errorf("update with foreign media id: status %d, want 400", status)
	}
	// update of a missing item, invalid price and missing name
	status, _, _ = postForm(t, app, "/dashboard/registry/999", url.Values{"name": {"X"}, "price": {"1"}})
	if status != 404 {
		t.Errorf("update missing item: status %d, want 404", status)
	}
	status, _, _ = postForm(t, app, "/dashboard/registry/"+strconv.Itoa(item.ID), url.Values{"name": {"X"}, "price": {"nah"}})
	if status != 400 {
		t.Errorf("update bad price: status %d, want 400", status)
	}
	status, _, _ = postForm(t, app, "/dashboard/registry/"+strconv.Itoa(item.ID), url.Values{"price": {"1"}})
	if status != 400 {
		t.Errorf("update no name: status %d, want 400", status)
	}

	// move: order swaps, invalid direction rejected, ends are no-ops
	if err := database.CreateRegistryItem("Second", 10, 0); err != nil {
		t.Fatal(err)
	}
	items, _ = database.GetAllRegistryItems()
	var second models.RegistryItem
	for _, it := range items {
		if it.Name == "Second" {
			second = it
		}
	}
	status, _, _ = postForm(t, app, "/dashboard/registry/"+strconv.Itoa(second.ID)+"/move/sideways", url.Values{})
	if status != 400 {
		t.Errorf("move bad direction: status %d, want 400", status)
	}
	status, _, _ = postForm(t, app, "/dashboard/registry/999/move/up", url.Values{})
	if status != 404 {
		t.Errorf("move missing: status %d, want 404", status)
	}
	status, _, _ = postForm(t, app, "/dashboard/registry/"+strconv.Itoa(second.ID)+"/move/up", url.Values{})
	if status != 302 {
		t.Errorf("move up: status %d", status)
	}
	items, _ = database.GetAllRegistryItems()
	if items[0].Name != "Second" {
		t.Errorf("after move up, first item = %q", items[0].Name)
	}
	status, _, _ = postForm(t, app, "/dashboard/registry/"+strconv.Itoa(second.ID)+"/move/up", url.Values{})
	if status != 302 {
		t.Errorf("move past top: status %d", status)
	}
	status, _, _ = postForm(t, app, "/dashboard/registry/"+strconv.Itoa(second.ID)+"/move/down", url.Values{})
	if status != 302 {
		t.Errorf("move down: status %d", status)
	}

	// delete removes the item and any attached media
	withMedia := url.Values{"name": {"WithPic"}, "price": {"1"}, "image": {pngDataURI(t, "pic")}}
	_, _, _ = postForm(t, app, "/dashboard/registry", withMedia)
	items, _ = database.GetAllRegistryItems()
	var picItem models.RegistryItem
	for _, it := range items {
		if it.Name == "WithPic" {
			picItem = it
		}
	}
	status, _, _ = postForm(t, app, "/dashboard/registry/"+strconv.Itoa(picItem.ID)+"/delete", url.Values{})
	if status != 302 {
		t.Errorf("delete item: status %d", status)
	}
	if _, err := database.GetRegistryItem(picItem.ID); err == nil {
		t.Error("item still present after delete")
	}
	if _, err := database.GetMedia(picItem.MediaID); err == nil {
		t.Error("media row survived item delete")
	}
	// deleting a missing id still redirects (nothing to drop)
	status, _, _ = postForm(t, app, "/dashboard/registry/999/delete", url.Values{})
	if status != 302 {
		t.Errorf("delete missing item: status %d", status)
	}
	status, _, _ = postForm(t, app, "/dashboard/registry/abc/delete", url.Values{})
	if status != 400 {
		t.Errorf("delete invalid id: status %d, want 400", status)
	}
}

func TestRegistryErrors(t *testing.T) {
	setupDB(t)
	app := newTestApp()
	defer app.Shutdown()

	// create fails after the image upload: the orphan media row is cleaned
	if _, err := database.DB.Exec(`DROP TABLE registry_items`); err != nil {
		t.Fatal(err)
	}
	form := url.Values{"name": {"X"}, "price": {"1"}, "image": {pngDataURI(t, "orphan")}}
	status, _, body := postForm(t, app, "/dashboard/registry", form)
	if status != 500 || !strings.Contains(body, "failed to add item") {
		t.Errorf("create without table: status %d body %q", status, body)
	}
	var mediaCount int
	if err := database.DB.QueryRow(`SELECT COUNT(*) FROM media`).Scan(&mediaCount); err != nil {
		t.Fatal(err)
	}
	if mediaCount != 0 {
		t.Errorf("orphan media rows = %d, want 0", mediaCount)
	}

	// edit page: the registry lookup fails inside the handler
	setupDB(t)
	item := seedRegistryItem(t)
	if _, err := database.DB.Exec(`DROP TABLE settings`); err != nil {
		t.Fatal(err)
	}
	status, _, body = get(t, app, "/dashboard/registry/"+strconv.Itoa(item.ID)+"/edit")
	if status != 500 || !strings.Contains(body, "settings") {
		t.Errorf("edit without settings: status %d body %q", status, body)
	}
}

func seedRegistryItem(t *testing.T) models.RegistryItem {
	t.Helper()
	if err := database.CreateRegistryItem("Seeded", 10, 0); err != nil {
		t.Fatal(err)
	}
	items, err := database.GetAllRegistryItems()
	if err != nil {
		t.Fatal(err)
	}
	return items[0]
}

func TestGiftEditUpdateDelete(t *testing.T) {
	setupDB(t)
	configureSettings(t, nil)
	app := newTestApp()
	defer app.Shutdown()

	if err := database.CreateRegistryItem("Toaster", 100, 0); err != nil {
		t.Fatal(err)
	}
	items, _ := database.GetAllRegistryItems()
	itemID := items[0].ID
	if err := database.CreateGift(50, "Alice", &itemID, nil); err != nil {
		t.Fatal(err)
	}
	gifts, _ := database.GetAllGifts()
	gift := gifts[0]

	status, _, _ := get(t, app, "/dashboard/gifts/abc/edit")
	if status != 400 {
		t.Errorf("edit gift invalid id: status %d", status)
	}
	status, _, _ = get(t, app, "/dashboard/gifts/999/edit")
	if status != 404 {
		t.Errorf("edit gift missing: status %d", status)
	}
	status, _, body := get(t, app, "/dashboard/gifts/"+strconv.Itoa(gift.ID)+"/edit")
	if status != 200 || !strings.Contains(body, "Alice") {
		t.Errorf("edit gift page: status %d", status)
	}

	// update: amount, donor trim, unlink (empty item field), confirmed flag
	form := url.Values{
		"amount": {"75"}, "donor": {"  Alice B "}, "registry_item_id": {""}, "confirmed": {"on"},
	}
	status, _, _ = postForm(t, app, "/dashboard/gifts/"+strconv.Itoa(gift.ID), form)
	if status != 302 {
		t.Errorf("update gift: status %d", status)
	}
	updated, err := database.GetGift(gift.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Amount != 75 || updated.Donor != "Alice B" || updated.RegistryItemID != nil || !updated.Confirmed {
		t.Errorf("updated gift = %+v", updated)
	}
	// relink to the item, unconfirmed
	form = url.Values{"amount": {"10"}, "donor": {"Alice"}, "registry_item_id": {strconv.Itoa(itemID)}}
	status, _, _ = postForm(t, app, "/dashboard/gifts/"+strconv.Itoa(gift.ID), form)
	if status != 302 {
		t.Errorf("relink gift: status %d", status)
	}
	updated, _ = database.GetGift(gift.ID)
	if updated.RegistryItemID == nil || *updated.RegistryItemID != itemID || updated.Confirmed {
		t.Errorf("relinked gift = %+v", updated)
	}
	// invalid inputs: zero, fractional and negative amounts are rejected
	for _, bad := range []string{"0", "-2", "7.5", "nah"} {
		status, _, _ = postForm(t, app, "/dashboard/gifts/"+strconv.Itoa(gift.ID), url.Values{"amount": {bad}})
		if status != 400 {
			t.Errorf("update amount %q: status %d, want 400", bad, status)
		}
	}
	status, _, _ = postForm(t, app, "/dashboard/gifts/"+strconv.Itoa(gift.ID), url.Values{"amount": {"1"}, "registry_item_id": {"abc"}})
	if status != 400 {
		t.Errorf("update bad item id: status %d", status)
	}
	status, _, _ = postForm(t, app, "/dashboard/gifts/"+strconv.Itoa(gift.ID), url.Values{"amount": {"1"}, "registry_item_id": {"999"}})
	if status != 404 {
		t.Errorf("update missing item: status %d", status)
	}
	status, _, _ = postForm(t, app, "/dashboard/gifts/999", url.Values{"amount": {"1"}})
	if status != 404 {
		t.Errorf("update missing gift: status %d", status)
	}
	status, _, _ = postForm(t, app, "/dashboard/gifts/abc", url.Values{"amount": {"1"}})
	if status != 400 {
		t.Errorf("update invalid id: status %d", status)
	}

	// delete
	status, _, _ = postForm(t, app, "/dashboard/gifts/"+strconv.Itoa(gift.ID)+"/delete", url.Values{})
	if status != 302 {
		t.Errorf("delete gift: status %d", status)
	}
	if _, err := database.GetGift(gift.ID); err == nil {
		t.Error("gift still present after delete")
	}
	status, _, _ = postForm(t, app, "/dashboard/gifts/abc/delete", url.Values{})
	if status != 400 {
		t.Errorf("delete gift invalid id: status %d", status)
	}
}

func TestGiftEditErrors(t *testing.T) {
	setupDB(t)
	configureSettings(t, nil)
	app := newTestApp()
	defer app.Shutdown()
	if err := database.CreateGift(50, "Alice", nil, nil); err != nil {
		t.Fatal(err)
	}
	gifts, _ := database.GetAllGifts()
	gift := gifts[0]

	if _, err := database.DB.Exec(`DROP TABLE registry_items`); err != nil {
		t.Fatal(err)
	}
	status, _, body := get(t, app, "/dashboard/gifts/"+strconv.Itoa(gift.ID)+"/edit")
	if status != 500 || !strings.Contains(body, "failed to load registry items") {
		t.Errorf("status %d body %q", status, body)
	}
}

func TestInvitationDashboardCRUD(t *testing.T) {
	setupDB(t)
	configureSettings(t, nil)
	app := newTestApp()
	defer app.Shutdown()

	ada := mustCreateGuest(t, "Ada", "Love", "adult")
	bob := mustCreateGuest(t, "Bob", "Ray", "adult")

	// create without guests and with only unparsable ids: plain redirects
	status, _, _ := postForm(t, app, "/dashboard/invitations", url.Values{})
	if status != 302 {
		t.Errorf("create empty invitation: status %d", status)
	}
	status, _, _ = postForm(t, app, "/dashboard/invitations", url.Values{"guest_ids": {"abc", ""}})
	if status != 302 {
		t.Errorf("create unparsable guests: status %d", status)
	}
	// real create: standard form gets a redirect; a fetch caller gets a
	// 204 with the new code in the X-Invitation-Code header instead
	form := url.Values{"guest_ids": {strconv.Itoa(ada.ID), strconv.Itoa(bob.ID)}, "label": {"Pair"}}
	status, headers, _ := postForm(t, app, "/dashboard/invitations", form)
	if status != 302 || headers.Get("Location") != "/dashboard" {
		t.Errorf("create invitation: status %d location %q", status, headers.Get("Location"))
	}
	if headers.Get("X-Invitation-Code") == "" {
		t.Error("create invitation missing the code header")
	}
	req := httptest.NewRequest(fiber.MethodPost, "/dashboard/invitations", strings.NewReader(form.Encode()))
	req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationForm)
	req.Header.Set("X-Requested-With", "fetch")
	status, headers, _ = doReq(t, app, req)
	if status != 204 || headers.Get("X-Invitation-Code") == "" {
		t.Errorf("fetch create: status %d code header %q", status, headers.Get("X-Invitation-Code"))
	}

	invs, err := database.GetAllInvitations()
	if err != nil {
		t.Fatal(err)
	}
	if len(invs) != 2 {
		t.Fatalf("invitations = %d, want 2", len(invs))
	}
	inv := invs[0]

	// edit page + update guest set and label
	status, _, _ = get(t, app, "/dashboard/invitations/abc/edit")
	if status != 400 {
		t.Errorf("edit invitation invalid id: status %d", status)
	}
	status, _, _ = get(t, app, "/dashboard/invitations/999/edit")
	if status != 404 {
		t.Errorf("edit invitation missing: status %d", status)
	}
	status, _, _ = get(t, app, "/dashboard/invitations/"+strconv.Itoa(inv.ID)+"/edit")
	if status != 200 {
		t.Errorf("edit invitation page: status %d", status)
	}
	form = url.Values{"guest_ids": {strconv.Itoa(ada.ID)}, "label": {"Solo"}}
	status, _, _ = postForm(t, app, "/dashboard/invitations/"+strconv.Itoa(inv.ID), form)
	if status != 302 {
		t.Errorf("update invitation: status %d", status)
	}
	updated, err := database.GetInvitation(inv.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Guests) != 1 || updated.Label != "Solo" {
		t.Errorf("updated invitation = %+v", updated)
	}
	// updating a missing invitation redirects with a flash; a malformed
	// id is a plain 400
	status, _, _ = postForm(t, app, "/dashboard/invitations/999", form)
	if status != 302 {
		t.Errorf("update missing invitation: status %d", status)
	}
	status, _, _ = postForm(t, app, "/dashboard/invitations/abc", form)
	if status != 400 {
		t.Errorf("update invalid id: status %d, want 400", status)
	}

	// viewed reset
	if err := database.MarkInvitationViewed(inv.ID); err != nil {
		t.Fatal(err)
	}
	status, _, _ = postForm(t, app, "/dashboard/invitations/"+strconv.Itoa(inv.ID)+"/viewed/reset", url.Values{})
	if status != 302 {
		t.Errorf("reset viewed: status %d", status)
	}
	updated, _ = database.GetInvitation(inv.ID)
	if updated.ViewedAt != nil {
		t.Error("viewed flag still set after reset")
	}
	status, _, _ = postForm(t, app, "/dashboard/invitations/abc/viewed/reset", url.Values{})
	if status != 400 {
		t.Errorf("reset invalid id: status %d, want 400", status)
	}

	// delete
	status, _, _ = postForm(t, app, "/dashboard/invitations/"+strconv.Itoa(inv.ID)+"/delete", url.Values{})
	if status != 302 {
		t.Errorf("delete invitation: status %d", status)
	}
	if _, err := database.GetInvitation(inv.ID); err == nil {
		t.Error("invitation still present after delete")
	}
	status, _, _ = postForm(t, app, "/dashboard/invitations/abc/delete", url.Values{})
	if status != 400 {
		t.Errorf("delete invalid id: status %d, want 400", status)
	}
}

func TestPollCRUD(t *testing.T) {
	setupDB(t)
	configureSettings(t, nil)
	app := newTestApp()
	defer app.Shutdown()

	status, headers, _ := postForm(t, app, "/dashboard/polls", url.Values{"question": {" "}})
	if status != 302 || headers.Get("Location") != "/dashboard" {
		t.Errorf("add blank poll: status %d location %q", status, headers.Get("Location"))
	}
	status, _, _ = postForm(t, app, "/dashboard/polls", url.Values{"question": {"Bus?"}, "description": {"From the station"}})
	if status != 302 {
		t.Errorf("add poll: status %d", status)
	}
	polls, _ := database.GetAllPolls()
	if len(polls) != 1 {
		t.Fatalf("polls = %d", len(polls))
	}
	poll := polls[0]

	status, _, _ = get(t, app, "/dashboard/polls/abc/edit")
	if status != 400 {
		t.Errorf("edit poll invalid id: status %d", status)
	}
	status, _, _ = get(t, app, "/dashboard/polls/999/edit")
	if status != 404 {
		t.Errorf("edit poll missing: status %d", status)
	}
	status, _, body := get(t, app, "/dashboard/polls/"+strconv.Itoa(poll.ID)+"/edit")
	if status != 200 || !strings.Contains(body, "Bus?") {
		t.Errorf("edit poll page: status %d", status)
	}
	status, _, _ = postForm(t, app, "/dashboard/polls/"+strconv.Itoa(poll.ID), url.Values{"question": {"Veg menu?"}, "description": {""}})
	if status != 302 {
		t.Errorf("update poll: status %d", status)
	}
	polls, _ = database.GetAllPolls()
	if polls[0].Question != "Veg menu?" {
		t.Errorf("updated poll = %+v", polls[0])
	}
	status, _, _ = postForm(t, app, "/dashboard/polls/"+strconv.Itoa(poll.ID), url.Values{"question": {""}})
	if status != 302 {
		t.Errorf("update poll blank: status %d", status)
	}
	status, _, _ = postForm(t, app, "/dashboard/polls/999", url.Values{"question": {"X"}})
	if status != 302 {
		t.Errorf("update missing poll: status %d", status)
	}
	status, _, _ = postForm(t, app, "/dashboard/polls/"+strconv.Itoa(poll.ID)+"/delete", url.Values{})
	if status != 302 {
		t.Errorf("delete poll: status %d", status)
	}
	polls, _ = database.GetAllPolls()
	if len(polls) != 0 {
		t.Errorf("polls after delete = %d", len(polls))
	}
	status, _, _ = postForm(t, app, "/dashboard/polls/abc/delete", url.Values{})
	if status != 400 {
		t.Errorf("delete poll invalid id: status %d, want 400", status)
	}
}

func TestDeleteSoundtrackEvent(t *testing.T) {
	setupDB(t)
	app := newTestApp()
	defer app.Shutdown()
	if err := database.CreateSoundtrackEvent("Song", "Artist", "", ""); err != nil {
		t.Fatal(err)
	}
	events, _ := database.GetAllSoundtrackEvents()
	status, _, _ := postForm(t, app, "/dashboard/soundtrack/"+strconv.Itoa(events[0].ID)+"/delete", url.Values{})
	if status != 302 {
		t.Errorf("delete soundtrack event: status %d", status)
	}
	events, _ = database.GetAllSoundtrackEvents()
	if len(events) != 0 {
		t.Errorf("events after delete = %d", len(events))
	}
	status, _, _ = postForm(t, app, "/dashboard/soundtrack/abc/delete", url.Values{})
	if status != 400 {
		t.Errorf("delete soundtrack invalid id: status %d, want 400", status)
	}

	database.DB.Close()
	status, _, body := postForm(t, app, "/dashboard/soundtrack/1/delete", url.Values{})
	if status != 500 || !strings.Contains(body, "delete soundtrack") {
		t.Errorf("delete with closed DB: status %d body %q", status, body)
	}
}
