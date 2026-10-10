package database

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/streambinder/foedus/internal/models"
)

func boolPtr(v bool) *bool { return &v }

func TestGuestLifecycle(t *testing.T) {
	newTestDB(t)

	if err := CreateGuest("Ada", "Lovelace", "adult"); err != nil {
		t.Fatal(err)
	}
	if err := CreateGuest("Bad", "Type", "alien"); err == nil {
		t.Fatal("expected the CHECK constraint to reject an unknown guest type")
	}

	guests, err := GetAllGuests()
	if err != nil {
		t.Fatal(err)
	}
	if len(guests) != 1 {
		t.Fatalf("got %d guests, want 1", len(guests))
	}
	guest := guests[0]
	if guest.FirstName != "Ada" || guest.ConfirmedCeremony != nil || guest.InvitationID != nil {
		t.Errorf("guest = %+v", guest)
	}

	fetched, err := GetGuest(guest.ID)
	if err != nil {
		t.Fatal(err)
	}
	if fetched.LastName != "Lovelace" {
		t.Errorf("fetched guest = %+v", fetched)
	}
	if _, err := GetGuest(9999); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("GetGuest(9999) error = %v, want ErrNoRows", err)
	}

	if err := UpdateGuest(guest.ID, "Ada", "Byron", "child"); err != nil {
		t.Fatal(err)
	}
	fetched, _ = GetGuest(guest.ID)
	if fetched.LastName != "Byron" || fetched.Type != "child" {
		t.Errorf("updated guest = %+v", fetched)
	}

	if err := DeleteGuest(guest.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := GetGuest(guest.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("deleted guest error = %v, want ErrNoRows", err)
	}
}

func TestCycleConfirmed(t *testing.T) {
	newTestDB(t)

	if err := CreateGuest("Cy", "Cle", "adult"); err != nil {
		t.Fatal(err)
	}
	guests, _ := GetAllGuests()
	id := guests[0].ID

	// NULL -> 1 -> 0 -> NULL
	if err := CycleConfirmed(id, "ceremony"); err != nil {
		t.Fatal(err)
	}
	guest, _ := GetGuest(id)
	if guest.ConfirmedCeremony == nil || !*guest.ConfirmedCeremony {
		t.Fatalf("after first cycle confirmed_ceremony = %v, want true", guest.ConfirmedCeremony)
	}
	if err := CycleConfirmed(id, "ceremony"); err != nil {
		t.Fatal(err)
	}
	guest, _ = GetGuest(id)
	if guest.ConfirmedCeremony == nil || *guest.ConfirmedCeremony {
		t.Fatalf("after second cycle confirmed_ceremony = %v, want false", guest.ConfirmedCeremony)
	}
	if err := CycleConfirmed(id, "ceremony"); err != nil {
		t.Fatal(err)
	}
	guest, _ = GetGuest(id)
	if guest.ConfirmedCeremony != nil {
		t.Fatalf("after third cycle confirmed_ceremony = %v, want nil", guest.ConfirmedCeremony)
	}
	if err := CycleConfirmed(id, "reception"); err != nil {
		t.Fatal(err)
	}
	guest, _ = GetGuest(id)
	if guest.ConfirmedReception == nil || !*guest.ConfirmedReception {
		t.Fatalf("reception cycle = %v, want true", guest.ConfirmedReception)
	}
	if err := CycleConfirmed(id, "bogus"); err == nil {
		t.Fatal("expected an error for an invalid field")
	}
}

func TestCounts(t *testing.T) {
	newTestDB(t)

	// empty database counts are all zero
	confirmed, refused, pending, invited, nonVisualized, total, err := CountConfirmed()
	if err != nil {
		t.Fatal(err)
	}
	if confirmed+refused+pending+invited+nonVisualized+total != 0 {
		t.Errorf("empty counts = %d %d %d %d %d %d, want all zero", confirmed, refused, pending, invited, nonVisualized, total)
	}
	adult, child, infant, vendor, err := CountConfirmedByType()
	if err != nil {
		t.Fatal(err)
	}
	if adult+child+infant+vendor != 0 {
		t.Errorf("empty type counts = %d %d %d %d", adult, child, infant, vendor)
	}

	for _, g := range []struct{ first, typ string }{
		{"Adult", "adult"}, {"Child", "child"}, {"Infant", "infant"}, {"Vendor", "vendor"}, {"Pending", "adult"}, {"Refused", "adult"}, {"Hermit", "adult"},
	} {
		if err := CreateGuest(g.first, "X", g.typ); err != nil {
			t.Fatal(err)
		}
	}
	guests, _ := GetAllGuests()
	byName := make(map[string]int)
	for _, g := range guests {
		byName[g.FirstName] = g.ID
	}
	// invite the first six; Hermit stays uninvited but confirms anyway
	code, err := CreateInvitation([]int{byName["Adult"], byName["Child"], byName["Infant"], byName["Vendor"], byName["Pending"], byName["Refused"]}, "party")
	if err != nil {
		t.Fatal(err)
	}
	inv, err := GetInvitationByCode(code)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Adult", "Child", "Infant", "Vendor"} {
		if err := SetGuestRSVP(byName[name], boolPtr(true), boolPtr(true)); err != nil {
			t.Fatal(err)
		}
	}
	if err := SetGuestRSVP(byName["Refused"], boolPtr(false), boolPtr(false)); err != nil {
		t.Fatal(err)
	}
	if err := SetGuestRSVP(byName["Hermit"], nil, boolPtr(true)); err != nil {
		t.Fatal(err)
	}
	_ = inv

	confirmed, refused, pending, invited, nonVisualized, total, err = CountConfirmed()
	if err != nil {
		t.Fatal(err)
	}
	// reception confirmed: Adult, Child, Infant, Vendor, Hermit = 5
	if confirmed != 5 {
		t.Errorf("confirmed = %d, want 5", confirmed)
	}
	if refused != 1 {
		t.Errorf("refused = %d, want 1", refused)
	}
	if pending != 1 {
		t.Errorf("pending = %d, want 1 (the invited guest who has not answered)", pending)
	}
	if invited != 6 {
		t.Errorf("invited = %d, want 6", invited)
	}
	if nonVisualized != 6 {
		t.Errorf("nonVisualized = %d, want 6 (invitation never viewed)", nonVisualized)
	}
	if total != 7 {
		t.Errorf("total = %d, want 7", total)
	}

	adult, child, infant, vendor, err = CountConfirmedByType()
	if err != nil {
		t.Fatal(err)
	}
	// the Hermit is an adult too
	if adult != 2 || child != 1 || infant != 1 || vendor != 1 {
		t.Errorf("type counts = %d %d %d %d, want 2 1 1 1", adult, child, infant, vendor)
	}

	// marking the invitation viewed flips nonVisualized to zero
	if err := MarkInvitationViewed(inv.ID); err != nil {
		t.Fatal(err)
	}
	_, _, _, _, nonVisualized, _, err = CountConfirmed()
	if err != nil {
		t.Fatal(err)
	}
	if nonVisualized != 0 {
		t.Errorf("nonVisualized after view = %d, want 0", nonVisualized)
	}
}

func TestGuestsPaginated(t *testing.T) {
	newTestDB(t)

	for _, name := range []string{"Marco Rossi", "Anna Bianchi", "Marco Verdi", "Lucia Neri"} {
		if err := CreateGuest(name, "", "adult"); err != nil {
			t.Fatal(err)
		}
	}

	page, total, err := GetGuestsPaginated(1, 2, "")
	if err != nil {
		t.Fatal(err)
	}
	if total != 4 || len(page) != 2 {
		t.Errorf("page 1: total = %d len = %d, want 4 and 2", total, len(page))
	}
	page, total, err = GetGuestsPaginated(2, 2, "")
	if err != nil {
		t.Fatal(err)
	}
	if total != 4 || len(page) != 2 {
		t.Errorf("page 2: total = %d len = %d, want 4 and 2", total, len(page))
	}
	// search matches against the first name column as stored
	page, total, err = GetGuestsPaginated(1, 10, "Marco")
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(page) != 2 {
		t.Errorf("search Marco: total = %d len = %d, want 2 and 2", total, len(page))
	}
	page, total, err = GetGuestsPaginated(1, 10, "Nobody")
	if err != nil {
		t.Fatal(err)
	}
	if total != 0 || len(page) != 0 {
		t.Errorf("search Nobody: total = %d len = %d, want 0 and 0", total, len(page))
	}
}

func TestGuestNameGroupsByCounter(t *testing.T) {
	newTestDB(t)

	if _, err := GuestNameGroupsByCounter("nonsense"); !errors.Is(err, ErrUnknownCategory) {
		t.Fatalf("unknown category error = %v, want ErrUnknownCategory", err)
	}

	if err := CreateGuest("Solo", "Guest", "adult"); err != nil {
		t.Fatal(err)
	}
	if err := CreateGuest("Anna", "One", "adult"); err != nil {
		t.Fatal(err)
	}
	if err := CreateGuest("Beppe", "Two", "adult"); err != nil {
		t.Fatal(err)
	}
	guests, _ := GetAllGuests()
	ids := make(map[string]int)
	for _, g := range guests {
		ids[g.FirstName] = g.ID
	}
	code, err := CreateInvitation([]int{ids["Anna"], ids["Beppe"]}, "pair")
	if err != nil {
		t.Fatal(err)
	}
	inv, err := GetInvitationByCode(code)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []int{ids["Solo"], ids["Anna"], ids["Beppe"]} {
		if err := SetGuestRSVP(id, nil, boolPtr(true)); err != nil {
			t.Fatal(err)
		}
	}
	_ = inv

	groups, err := GuestNameGroupsByCounter("confirmed_reception")
	if err != nil {
		t.Fatal(err)
	}
	// one group for the invited pair, one singleton for the uninvited guest
	if len(groups) != 2 {
		t.Fatalf("got %d groups, want 2: %v", len(groups), groups)
	}
	var pair []string
	var solo []string
	for _, g := range groups {
		if len(g) == 2 {
			pair = g
		} else {
			solo = g
		}
	}
	if pair[0] != "Anna One" || pair[1] != "Beppe Two" {
		t.Errorf("invited group = %v, want [Anna One Beppe Two]", pair)
	}
	if len(solo) != 1 || solo[0] != "Solo Guest" {
		t.Errorf("uninvited group = %v, want [Solo Guest]", solo)
	}

	// every known category must run without error
	for category := range counterWhereClauses {
		if _, err := GuestNameGroupsByCounter(category); err != nil {
			t.Errorf("category %q: %v", category, err)
		}
	}

	uninvited, err := GuestNameGroupsByCounter("uninvited")
	if err != nil {
		t.Fatal(err)
	}
	if len(uninvited) != 1 || uninvited[0][0] != "Solo Guest" {
		t.Errorf("uninvited groups = %v", uninvited)
	}
}

func TestInvitationLifecycle(t *testing.T) {
	newTestDB(t)

	for _, name := range []string{"A", "B", "C"} {
		if err := CreateGuest(name, "Guest", "adult"); err != nil {
			t.Fatal(err)
		}
	}
	guests, _ := GetAllGuests()
	ids := []int{guests[0].ID, guests[1].ID, guests[2].ID}

	code, err := CreateInvitation(ids, "  My Label  ")
	if err != nil {
		t.Fatal(err)
	}
	if len(code) != 8 {
		t.Errorf("code %q has length %d, want 8", code, len(code))
	}

	inv, err := GetInvitationByCode(code)
	if err != nil {
		t.Fatal(err)
	}
	if inv.Label != "My Label" {
		t.Errorf("label = %q, want trimmed %q", inv.Label, "My Label")
	}
	if len(inv.Guests) != 3 {
		t.Fatalf("invitation has %d guests, want 3", len(inv.Guests))
	}
	if inv.Guests[0].InvitationOrder == nil || *inv.Guests[0].InvitationOrder != 0 {
		t.Errorf("first guest order = %v, want 0", inv.Guests[0].InvitationOrder)
	}

	byID, err := GetInvitation(inv.ID)
	if err != nil {
		t.Fatal(err)
	}
	if byID.Code != code {
		t.Errorf("GetInvitation code = %q, want %q", byID.Code, code)
	}
	if _, err := GetInvitation(9999); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("GetInvitation(9999) error = %v, want ErrNoRows", err)
	}
	if _, err := GetInvitationByCode("missing"); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("GetInvitationByCode(missing) error = %v, want ErrNoRows", err)
	}

	if err := UpdateInvitationLabel(inv.ID, "  Renamed "); err != nil {
		t.Fatal(err)
	}
	byID, _ = GetInvitation(inv.ID)
	if byID.Label != "Renamed" {
		t.Errorf("renamed label = %q", byID.Label)
	}

	all, err := GetAllInvitations()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || len(all[0].Guests) != 3 {
		t.Fatalf("GetAllInvitations = %+v", all)
	}

	// deleting the invitation unlinks the guests instead of deleting them
	if err := DeleteInvitation(inv.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := GetInvitation(inv.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("deleted invitation error = %v, want ErrNoRows", err)
	}
	guest, err := GetGuest(ids[0])
	if err != nil {
		t.Fatal(err)
	}
	if guest.InvitationID != nil {
		t.Errorf("guest still linked to deleted invitation: %v", guest.InvitationID)
	}
}

func TestInvitationRanking(t *testing.T) {
	// rank: actioned (2) sorts last, viewed (1) middle, fresh (0) first in the
	// stable sort used by GetAllInvitations — verify through the helpers.
	fresh := models.Invitation{}
	if got := invitationDashboardRank(fresh); got != 0 {
		t.Errorf("fresh rank = %d, want 0", got)
	}
	ts := models.Timestamp{}
	viewed := models.Invitation{ViewedAt: &ts}
	if got := invitationDashboardRank(viewed); got != 1 {
		t.Errorf("viewed rank = %d, want 1", got)
	}
	answered := models.Invitation{Guests: []models.Guest{{ConfirmedReception: boolPtr(true)}}}
	if got := invitationDashboardRank(answered); got != 2 {
		t.Errorf("actioned rank = %d, want 2", got)
	}
	if invitationActioned(models.Invitation{Guests: []models.Guest{{}}}) {
		t.Error("guest without answers must not count as actioned")
	}
	if !invitationActioned(models.Invitation{Guests: []models.Guest{{ConfirmedCeremony: boolPtr(false)}}}) {
		t.Error("guest with a ceremony answer must count as actioned")
	}
}

func TestGenerateCodeAndDefaultLabel(t *testing.T) {
	seen := make(map[string]bool)
	for range 50 {
		code := GenerateCode()
		if len(code) != 8 {
			t.Fatalf("code %q length = %d, want 8", code, len(code))
		}
		for _, r := range code {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
				t.Fatalf("code %q contains a non-base62 character", code)
			}
		}
		seen[code] = true
	}
	if len(seen) < 45 {
		t.Errorf("only %d distinct codes in 50 draws", len(seen))
	}

	if got := DefaultInvitationLabel(nil); got != "" {
		t.Errorf("empty label = %q", got)
	}
	if got := DefaultInvitationLabel([]models.Guest{{FirstName: "Ada"}}); got != "Ada" {
		t.Errorf("single label = %q", got)
	}
	if got := DefaultInvitationLabel([]models.Guest{{FirstName: "Ada"}, {FirstName: "Bob"}}); got != "Ada & Bob" {
		t.Errorf("pair label = %q", got)
	}
	if got := DefaultInvitationLabel([]models.Guest{{FirstName: "Ada"}, {FirstName: "Bob"}, {FirstName: "Cid"}, {FirstName: "Dee"}}); got != "Ada + 3" {
		t.Errorf("group label = %q", got)
	}
}

func TestPollsLifecycle(t *testing.T) {
	newTestDB(t)

	if err := CreatePoll("Coming?", "Let us know"); err != nil {
		t.Fatal(err)
	}
	if err := CreatePoll("Bus?", ""); err != nil {
		t.Fatal(err)
	}
	polls, err := GetAllPolls()
	if err != nil {
		t.Fatal(err)
	}
	if len(polls) != 2 {
		t.Fatalf("got %d polls, want 2", len(polls))
	}
	poll := polls[0]

	fetched, err := GetPoll(poll.ID)
	if err != nil {
		t.Fatal(err)
	}
	if fetched.Question != "Coming?" {
		t.Errorf("poll = %+v", fetched)
	}
	if _, err := GetPoll(9999); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("GetPoll(9999) error = %v, want ErrNoRows", err)
	}

	if err := UpdatePoll(poll.ID, "Attending?", "Updated"); err != nil {
		t.Fatal(err)
	}
	fetched, _ = GetPoll(poll.ID)
	if fetched.Question != "Attending?" || fetched.Description != "Updated" {
		t.Errorf("updated poll = %+v", fetched)
	}

	// answers: empty map is a no-op
	if err := CreateGuest("Voter", "One", "adult"); err != nil {
		t.Fatal(err)
	}
	guests, _ := GetAllGuests()
	guestID := guests[0].ID
	if err := SavePollAnswers(guestID, map[int]models.PollAnswer{}); err != nil {
		t.Fatal(err)
	}
	if err := SavePollAnswers(guestID, map[int]models.PollAnswer{
		poll.ID:     {PollID: poll.ID, Answer: true, Notes: "plus one"},
		polls[1].ID: {PollID: polls[1].ID, Answer: false},
	}); err != nil {
		t.Fatal(err)
	}
	// re-saving replaces the previous answer
	if err := SavePollAnswers(guestID, map[int]models.PollAnswer{
		poll.ID: {PollID: poll.ID, Answer: true, Notes: "changed"},
	}); err != nil {
		t.Fatal(err)
	}

	answers, err := GetPollAnswersForGuests([]int{guestID})
	if err != nil {
		t.Fatal(err)
	}
	if len(answers[guestID]) != 2 {
		t.Fatalf("got %d answers, want 2", len(answers[guestID]))
	}
	empty, err := GetPollAnswersForGuests(nil)
	if err != nil {
		t.Fatal(err)
	}
	if empty != nil {
		t.Errorf("empty guest list answers = %v, want nil", empty)
	}

	withCounts, err := GetAllPollsWithCounts()
	if err != nil {
		t.Fatal(err)
	}
	if len(withCounts) != 2 {
		t.Fatalf("got %d polls with counts, want 2", len(withCounts))
	}
	for _, p := range withCounts {
		if p.ID == poll.ID {
			if p.TotalCount != 1 {
				t.Errorf("poll total count = %d, want 1", p.TotalCount)
			}
			if len(p.YesVoters) != 1 || p.YesVoters[0].Name != "Voter One" || p.YesVoters[0].Notes != "changed" {
				t.Errorf("yes voters = %+v", p.YesVoters)
			}
		}
	}

	// deleting a guest purges their poll answers first
	if err := DeleteGuest(guestID); err != nil {
		t.Fatal(err)
	}
	answers, _ = GetPollAnswersForGuests([]int{guestID})
	if len(answers) != 0 {
		t.Errorf("answers survived guest deletion: %v", answers)
	}

	// deleting a poll purges its answers
	if err := DeletePoll(poll.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := GetPoll(poll.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("deleted poll error = %v, want ErrNoRows", err)
	}
}

func TestRegistryLifecycle(t *testing.T) {
	newTestDB(t)

	if err := CreateRegistryItem("Toaster", 50, 0); err != nil {
		t.Fatal(err)
	}
	mediaID, err := InsertMedia(DB, "image/png", []byte("img"))
	if err != nil {
		t.Fatal(err)
	}
	if err := CreateRegistryItem("Lamp", 80, mediaID); err != nil {
		t.Fatal(err)
	}
	if err := CreateRegistryItem("Vase", 30, 0); err != nil {
		t.Fatal(err)
	}
	items, err := GetAllRegistryItems()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("got %d items, want 3", len(items))
	}
	if items[0].Name != "Toaster" || items[1].Name != "Lamp" || items[2].Name != "Vase" {
		t.Errorf("sort order not preserved: %v", items)
	}
	if items[1].MediaID != mediaID {
		t.Errorf("lamp media id = %d, want %d", items[1].MediaID, mediaID)
	}
	if items[0].MediaID != 0 {
		t.Errorf("toaster media id = %d, want 0", items[0].MediaID)
	}

	item, err := GetRegistryItem(items[1].ID)
	if err != nil {
		t.Fatal(err)
	}
	if item.Name != "Lamp" {
		t.Errorf("item = %+v", item)
	}
	if _, err := GetRegistryItem(9999); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("GetRegistryItem(9999) error = %v, want ErrNoRows", err)
	}

	if err := UpdateRegistryItem(item.ID, "Lamp XL", 90, 0); err != nil {
		t.Fatal(err)
	}
	item, _ = GetRegistryItem(item.ID)
	if item.Name != "Lamp XL" || item.Price != 90 || item.MediaID != 0 {
		t.Errorf("updated item = %+v", item)
	}

	// moving the last item down is a no-op that still commits
	if err := MoveRegistryItem(items[2].ID, "down"); err != nil {
		t.Fatal(err)
	}
	// moving the first item up is a no-op too
	if err := MoveRegistryItem(items[0].ID, "up"); err != nil {
		t.Fatal(err)
	}
	// swap the first two
	if err := MoveRegistryItem(items[0].ID, "down"); err != nil {
		t.Fatal(err)
	}
	items, _ = GetAllRegistryItems()
	if items[0].Name != "Lamp XL" || items[1].Name != "Toaster" {
		t.Errorf("after move down: %v", items)
	}
	// moving the second item up swaps the pair back
	if err := MoveRegistryItem(items[1].ID, "up"); err != nil {
		t.Fatal(err)
	}
	items, _ = GetAllRegistryItems()
	if items[0].Name != "Toaster" {
		t.Errorf("after move up: %v", items)
	}
	if err := MoveRegistryItem(items[0].ID, "sideways"); err == nil {
		t.Error("expected an error for an invalid direction")
	}
	if err := MoveRegistryItem(9999, "down"); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("MoveRegistryItem(9999) error = %v, want ErrNoRows", err)
	}

	// a gift keeps its row when its item is deleted, detached instead
	if err := CreateGift(20, "Donor", &items[0].ID, nil); err != nil {
		t.Fatal(err)
	}
	if err := DeleteRegistryItem(items[0].ID); err != nil {
		t.Fatal(err)
	}
	gifts, err := GetAllGifts()
	if err != nil {
		t.Fatal(err)
	}
	if len(gifts) != 1 || gifts[0].RegistryItemID != nil {
		t.Errorf("gift after item deletion = %+v", gifts)
	}
	if _, err := GetRegistryItem(items[0].ID); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("deleted item error = %v, want ErrNoRows", err)
	}
}

func TestGiftsLifecycle(t *testing.T) {
	newTestDB(t)

	if err := CreateRegistryItem("Item", 100, 0); err != nil {
		t.Fatal(err)
	}
	items, _ := GetAllRegistryItems()
	itemID := items[0].ID

	if err := CreateGift(30, "Alice", &itemID, nil); err != nil {
		t.Fatal(err)
	}
	if err := CreateGift(20, "Bob", &itemID, nil); err != nil {
		t.Fatal(err)
	}
	if err := CreateGift(10, "Carol", nil, nil); err != nil {
		t.Fatal(err)
	}

	claimed, err := GetClaimedAmountsByItem()
	if err != nil {
		t.Fatal(err)
	}
	if claimed[itemID] != 50 {
		t.Errorf("claimed = %d, want 50", claimed[itemID])
	}
	if len(claimed) != 1 {
		t.Errorf("claimed map = %v", claimed)
	}

	gifts, err := GetAllGifts()
	if err != nil {
		t.Fatal(err)
	}
	if len(gifts) != 3 {
		t.Fatalf("got %d gifts, want 3", len(gifts))
	}
	gift := gifts[0]
	fetched, err := GetGift(gift.ID)
	if err != nil {
		t.Fatal(err)
	}
	if fetched.Donor == "" {
		t.Errorf("gift = %+v", fetched)
	}
	if _, err := GetGift(9999); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("GetGift(9999) error = %v, want ErrNoRows", err)
	}

	if err := UpdateGift(gift.ID, 99, "Updated Donor", nil, true); err != nil {
		t.Fatal(err)
	}
	fetched, _ = GetGift(gift.ID)
	if fetched.Amount != 99 || fetched.Donor != "Updated Donor" || !fetched.Confirmed || fetched.RegistryItemID != nil {
		t.Errorf("updated gift = %+v", fetched)
	}

	if err := DeleteGift(gift.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := GetGift(gift.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("deleted gift error = %v, want ErrNoRows", err)
	}
}

func TestMediaLifecycle(t *testing.T) {
	newTestDB(t)

	id, err := InsertMedia(DB, "image/jpeg", []byte("jpeg-bytes"))
	if err != nil {
		t.Fatal(err)
	}
	if id <= 0 {
		t.Fatalf("inserted media id = %d", id)
	}
	media, err := GetMedia(id)
	if err != nil {
		t.Fatal(err)
	}
	if media.Mime != "image/jpeg" || string(media.Bytes) != "jpeg-bytes" {
		t.Errorf("media = %+v", media)
	}
	mime, length, err := GetMediaMeta(id)
	if err != nil {
		t.Fatal(err)
	}
	if mime != "image/jpeg" || length != len("jpeg-bytes") {
		t.Errorf("meta = %q %d", mime, length)
	}
	if _, _, err := GetMediaMeta(9999); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("GetMediaMeta(9999) error = %v, want ErrNoRows", err)
	}
	if _, err := GetMedia(9999); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("GetMedia(9999) error = %v, want ErrNoRows", err)
	}

	// deleting a non-positive id is a no-op
	if err := DeleteMedia(DB, 0); err != nil {
		t.Fatal(err)
	}
	if err := DeleteMedia(DB, -3); err != nil {
		t.Fatal(err)
	}
	if err := DeleteMedia(DB, id); err != nil {
		t.Fatal(err)
	}
	if _, err := GetMedia(id); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("deleted media error = %v, want ErrNoRows", err)
	}
}

func TestImpersonationsRoundTrip(t *testing.T) {
	newTestDB(t)

	if err := ReplaceImpersonations(DB, []models.Impersonation{
		{Codename: "zio", Profile: "warm"},
		{Codename: "nonna", Profile: "dry"},
	}); err != nil {
		t.Fatal(err)
	}
	got, err := GetImpersonations()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Codename != "zio" || got[1].Codename != "nonna" {
		t.Fatalf("impersonations = %+v", got)
	}
	if got[0].Profile != "warm" || got[0].SortOrder != 0 || got[1].SortOrder != 1 {
		t.Errorf("impersonation fields = %+v", got)
	}
}

func TestAllInvitationsSorting(t *testing.T) {
	newTestDB(t)

	mkGuest := func(name string) int {
		if err := CreateGuest(name, "G", "adult"); err != nil {
			t.Fatal(err)
		}
		guests, _ := GetAllGuests()
		for _, g := range guests {
			if g.FirstName == name {
				return g.ID
			}
		}
		t.Fatalf("guest %q not found", name)
		return 0
	}
	mkInv := func(name string) models.Invitation {
		code, err := CreateInvitation([]int{mkGuest(name)}, name)
		if err != nil {
			t.Fatal(err)
		}
		inv, err := GetInvitationByCode(code)
		if err != nil {
			t.Fatal(err)
		}
		return inv
	}

	fresh := mkInv("Fresh")
	viewed := mkInv("Viewed")
	actioned := mkInv("Actioned")
	if err := MarkInvitationViewed(viewed.ID); err != nil {
		t.Fatal(err)
	}
	if err := SetGuestRSVP(actioned.Guests[0].ID, boolPtr(true), nil); err != nil {
		t.Fatal(err)
	}

	all, err := GetAllInvitations()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Fatalf("got %d invitations, want 3", len(all))
	}
	// rank order: fresh (0) first, viewed (1) next, actioned (2) last
	if all[0].ID != fresh.ID || all[1].ID != viewed.ID || all[2].ID != actioned.ID {
		t.Errorf("sorted order = %d %d %d, want %d %d %d",
			all[0].ID, all[1].ID, all[2].ID, fresh.ID, viewed.ID, actioned.ID)
	}
}

func TestSoundtrackEventsLifecycle(t *testing.T) {
	newTestDB(t)

	if err := CreateSoundtrackEvent("  Song One ", " Artist ", " https://x ", " inv1 "); err != nil {
		t.Fatal(err)
	}
	if err := CreateSoundtrackEvent("Song Two", "Artist", "https://y", ""); err != nil {
		t.Fatal(err)
	}
	events, err := GetAllSoundtrackEvents()
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}
	// newest first, values trimmed on write
	if events[0].Title != "Song Two" || events[1].Title != "Song One" {
		t.Errorf("events = %+v", events)
	}
	if events[1].Artist != "Artist" || events[1].URL != "https://x" || events[1].InviteID != "inv1" {
		t.Errorf("trimmed event = %+v", events[1])
	}
	if err := DeleteSoundtrackEvent(events[0].ID); err != nil {
		t.Fatal(err)
	}
	events, _ = GetAllSoundtrackEvents()
	if len(events) != 1 {
		t.Errorf("after delete got %d events, want 1", len(events))
	}
}

func TestSetGuestRSVPAndBoolConversion(t *testing.T) {
	newTestDB(t)

	if got := boolToNullableInt(nil); got != nil {
		t.Errorf("boolToNullableInt(nil) = %v, want nil", got)
	}
	if got := boolToNullableInt(boolPtr(true)); got == nil || *got != 1 {
		t.Errorf("boolToNullableInt(true) = %v, want 1", got)
	}
	if got := boolToNullableInt(boolPtr(false)); got == nil || *got != 0 {
		t.Errorf("boolToNullableInt(false) = %v, want 0", got)
	}

	if err := CreateGuest("R", "SVP", "adult"); err != nil {
		t.Fatal(err)
	}
	guests, _ := GetAllGuests()
	id := guests[0].ID
	if err := SetGuestRSVP(id, boolPtr(true), nil); err != nil {
		t.Fatal(err)
	}
	guest, _ := GetGuest(id)
	if guest.ConfirmedCeremony == nil || !*guest.ConfirmedCeremony || guest.ConfirmedReception != nil {
		t.Errorf("guest after RSVP = %+v", guest)
	}
	if err := SetGuestRSVP(id, nil, nil); err != nil {
		t.Fatal(err)
	}
	guest, _ = GetGuest(id)
	if guest.ConfirmedCeremony != nil || guest.ConfirmedReception != nil {
		t.Errorf("guest after clearing RSVP = %+v", guest)
	}
}
