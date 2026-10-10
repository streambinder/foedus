package database

import (
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/streambinder/foedus/internal/models"
)

func TestHelpers(t *testing.T) {
	if got := nullableID(0); got != nil {
		t.Errorf("nullableID(0) = %v, want nil", got)
	}
	if got := nullableID(-2); got != nil {
		t.Errorf("nullableID(-2) = %v, want nil", got)
	}
	if got := nullableID(7); got != 7 {
		t.Errorf("nullableID(7) = %v, want 7", got)
	}
	if got := idOrZero(sql.NullInt64{}); got != 0 {
		t.Errorf("idOrZero(invalid) = %d, want 0", got)
	}
	if got := idOrZero(sql.NullInt64{Int64: 42, Valid: true}); got != 42 {
		t.Errorf("idOrZero(42) = %d, want 42", got)
	}
	if got := dsnWithPragmas("file.db"); !strings.HasPrefix(got, "file.db?") {
		t.Errorf("dsnWithPragmas without query = %q", got)
	}
	if got := dsnWithPragmas("file.db?mode=ro"); !strings.Contains(got, "?mode=ro&_pragma=") {
		t.Errorf("dsnWithPragmas with query = %q", got)
	}
}

func TestWithTx(t *testing.T) {
	newTestDB(t)

	// success commits
	if err := WithTx(func(tx *sql.Tx) error {
		_, err := tx.Exec(`INSERT INTO guests (first_name) VALUES ('TxGuest')`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	guests, _ := GetAllGuests()
	if len(guests) != 1 {
		t.Fatalf("got %d guests after commit, want 1", len(guests))
	}

	// an error rolls back
	if err := WithTx(func(tx *sql.Tx) error {
		if _, err := tx.Exec(`INSERT INTO guests (first_name) VALUES ('RolledBack')`); err != nil {
			return err
		}
		return errors.New("boom")
	}); err == nil {
		t.Fatal("expected the transaction error to propagate")
	}
	guests, _ = GetAllGuests()
	if len(guests) != 1 {
		t.Fatalf("got %d guests after rollback, want 1", len(guests))
	}

	// a panic rolls back and re-panics
	func() {
		defer func() {
			if recover() == nil {
				t.Error("WithTx did not re-panic")
			}
		}()
		_ = WithTx(func(tx *sql.Tx) error {
			if _, err := tx.Exec(`INSERT INTO guests (first_name) VALUES ('Panicked')`); err != nil {
				return err
			}
			panic("kaboom")
		})
	}()
	guests, _ = GetAllGuests()
	if len(guests) != 1 {
		t.Fatalf("got %d guests after panic rollback, want 1", len(guests))
	}
}

func TestQueryAndInsertAll(t *testing.T) {
	newTestDB(t)

	// a failing scan function surfaces its error
	_, err := queryAll(`SELECT id FROM guests`, func(rows *sql.Rows) (int, error) {
		return 0, errors.New("scan blew up")
	})
	if err == nil {
		// no rows means the scan callback never runs; seed a row and retry
		if err := CreateGuest("Scan", "Me", "adult"); err != nil {
			t.Fatal(err)
		}
		_, err = queryAll(`SELECT id FROM guests`, func(rows *sql.Rows) (int, error) {
			return 0, errors.New("scan blew up")
		})
		if err == nil {
			t.Fatal("expected the scan error to propagate from queryAll")
		}
	}

	// a broken query surfaces its error
	if _, err := queryAll(`SELECT nope FROM missing_table`, scanAccommodation); err == nil {
		t.Fatal("expected a query error from queryAll")
	}

	// insertAll surfaces write errors (dangling media FK)
	err = insertAll(DB, "places", placeColumns, []models.Place{{Label: "x", MediaID: 9999}},
		func(place models.Place, index int) []any {
			return []any{
				models.PlaceKindStory, place.Label, place.Name, place.Address, place.Date,
				place.Lat, place.Lng, nullableID(place.MediaID), index,
			}
		})
	if err == nil {
		t.Fatal("expected the FK error to propagate from insertAll")
	}
}

// Error branches: with the handle closed every entry point must surface the
// driver error instead of panicking or returning zero values.
func TestClosedDBErrors(t *testing.T) {
	newTestDB(t)
	if err := DB.Close(); err != nil {
		t.Fatal(err)
	}

	if _, err := GetSettings(); err == nil {
		t.Error("GetSettings on closed DB: want error")
	}
	if err := CreateGuest("A", "B", "adult"); err == nil {
		t.Error("CreateGuest on closed DB: want error")
	}
	if _, err := GetAllGuests(); err == nil {
		t.Error("GetAllGuests on closed DB: want error")
	}
	if _, err := GetGuest(1); err == nil {
		t.Error("GetGuest on closed DB: want error")
	}
	if err := UpdateGuest(1, "A", "B", "adult"); err == nil {
		t.Error("UpdateGuest on closed DB: want error")
	}
	if err := DeleteGuest(1); err == nil {
		t.Error("DeleteGuest on closed DB: want error")
	}
	if err := CycleConfirmed(1, "ceremony"); err == nil {
		t.Error("CycleConfirmed on closed DB: want error")
	}
	if _, _, _, _, _, _, err := CountConfirmed(); err == nil {
		t.Error("CountConfirmed on closed DB: want error")
	}
	if _, _, _, _, err := CountConfirmedByType(); err == nil {
		t.Error("CountConfirmedByType on closed DB: want error")
	}
	if _, _, err := GetGuestsPaginated(1, 10, ""); err == nil {
		t.Error("GetGuestsPaginated on closed DB: want error")
	}
	if _, _, err := GetGuestsPaginated(1, 10, "x"); err == nil {
		t.Error("GetGuestsPaginated search on closed DB: want error")
	}
	if _, err := GuestNameGroupsByCounter("invited"); err == nil {
		t.Error("GuestNameGroupsByCounter on closed DB: want error")
	}
	if _, err := CreateInvitation([]int{1}, "x"); err == nil {
		t.Error("CreateInvitation on closed DB: want error")
	}
	if err := UpdateInvitationLabel(1, "x"); err == nil {
		t.Error("UpdateInvitationLabel on closed DB: want error")
	}
	if _, err := GetInvitation(1); err == nil {
		t.Error("GetInvitation on closed DB: want error")
	}
	if _, err := GetAllInvitations(); err == nil {
		t.Error("GetAllInvitations on closed DB: want error")
	}
	if err := DeleteInvitation(1); err == nil {
		t.Error("DeleteInvitation on closed DB: want error")
	}
	if err := MarkInvitationViewed(1); err == nil {
		t.Error("MarkInvitationViewed on closed DB: want error")
	}
	if err := ResetInvitationViewed(1); err == nil {
		t.Error("ResetInvitationViewed on closed DB: want error")
	}
	if err := SetGuestRSVP(1, nil, nil); err == nil {
		t.Error("SetGuestRSVP on closed DB: want error")
	}
	if _, err := GetInvitationByCode("x"); err == nil {
		t.Error("GetInvitationByCode on closed DB: want error")
	}
	if err := CreatePoll("q", "d"); err == nil {
		t.Error("CreatePoll on closed DB: want error")
	}
	if err := UpdatePoll(1, "q", "d"); err == nil {
		t.Error("UpdatePoll on closed DB: want error")
	}
	if _, err := GetPoll(1); err == nil {
		t.Error("GetPoll on closed DB: want error")
	}
	if err := DeletePoll(1); err == nil {
		t.Error("DeletePoll on closed DB: want error")
	}
	if _, err := GetAllPolls(); err == nil {
		t.Error("GetAllPolls on closed DB: want error")
	}
	if _, err := GetAllPollsWithCounts(); err == nil {
		t.Error("GetAllPollsWithCounts on closed DB: want error")
	}
	if err := SavePollAnswers(1, map[int]models.PollAnswer{1: {PollID: 1, Answer: true}}); err == nil {
		t.Error("SavePollAnswers on closed DB: want error")
	}
	if _, err := GetPollAnswersForGuests([]int{1}); err == nil {
		t.Error("GetPollAnswersForGuests on closed DB: want error")
	}
	if err := CreateRegistryItem("x", 1, 0); err == nil {
		t.Error("CreateRegistryItem on closed DB: want error")
	}
	if _, err := GetAllRegistryItems(); err == nil {
		t.Error("GetAllRegistryItems on closed DB: want error")
	}
	if _, err := GetRegistryItem(1); err == nil {
		t.Error("GetRegistryItem on closed DB: want error")
	}
	if err := UpdateRegistryItem(1, "x", 1, 0); err == nil {
		t.Error("UpdateRegistryItem on closed DB: want error")
	}
	if err := DeleteRegistryItem(1); err == nil {
		t.Error("DeleteRegistryItem on closed DB: want error")
	}
	if err := MoveRegistryItem(1, "down"); err == nil {
		t.Error("MoveRegistryItem on closed DB: want error")
	}
	if err := CreateGift(1, "d", nil, nil); err == nil {
		t.Error("CreateGift on closed DB: want error")
	}
	if _, err := GetAllGifts(); err == nil {
		t.Error("GetAllGifts on closed DB: want error")
	}
	if _, err := GetGift(1); err == nil {
		t.Error("GetGift on closed DB: want error")
	}
	if err := UpdateGift(1, 1, "d", nil, false); err == nil {
		t.Error("UpdateGift on closed DB: want error")
	}
	if err := DeleteGift(1); err == nil {
		t.Error("DeleteGift on closed DB: want error")
	}
	if _, err := GetClaimedAmountsByItem(); err == nil {
		t.Error("GetClaimedAmountsByItem on closed DB: want error")
	}
	if _, err := InsertMedia(DB, "image/png", []byte("x")); err == nil {
		t.Error("InsertMedia on closed DB: want error")
	}
	if _, err := GetMedia(1); err == nil {
		t.Error("GetMedia on closed DB: want error")
	}
	if _, _, err := GetMediaMeta(1); err == nil {
		t.Error("GetMediaMeta on closed DB: want error")
	}
	if err := DeleteMedia(DB, 1); err == nil {
		t.Error("DeleteMedia on closed DB: want error")
	}
	if err := CreateSoundtrackEvent("t", "a", "u", "i"); err == nil {
		t.Error("CreateSoundtrackEvent on closed DB: want error")
	}
	if _, err := GetAllSoundtrackEvents(); err == nil {
		t.Error("GetAllSoundtrackEvents on closed DB: want error")
	}
	if err := DeleteSoundtrackEvent(1); err == nil {
		t.Error("DeleteSoundtrackEvent on closed DB: want error")
	}
	if _, err := GetPlaces(models.PlaceKindStory); err == nil {
		t.Error("GetPlaces on closed DB: want error")
	}
	if _, err := GetAccommodations(); err == nil {
		t.Error("GetAccommodations on closed DB: want error")
	}
	if _, err := GetImpersonations(); err == nil {
		t.Error("GetImpersonations on closed DB: want error")
	}
	if _, err := CountImpersonations(); err == nil {
		t.Error("CountImpersonations on closed DB: want error")
	}
	if _, err := GetParkingSpots(); err == nil {
		t.Error("GetParkingSpots on closed DB: want error")
	}
	if _, err := GetHeroBackgrounds(); err == nil {
		t.Error("GetHeroBackgrounds on closed DB: want error")
	}
	if _, err := GetHomepageLabels("en"); err == nil {
		t.Error("GetHomepageLabels on closed DB: want error")
	}
	if _, err := GetAllHomepageLabels(); err == nil {
		t.Error("GetAllHomepageLabels on closed DB: want error")
	}
	if err := ReplacePlaces(DB, models.PlaceKindStory, nil); err == nil {
		t.Error("ReplacePlaces on closed DB: want error")
	}
	if err := ReplaceAccommodations(DB, nil); err == nil {
		t.Error("ReplaceAccommodations on closed DB: want error")
	}
	if err := ReplaceImpersonations(DB, nil); err == nil {
		t.Error("ReplaceImpersonations on closed DB: want error")
	}
	if err := ReplaceParkingSpots(DB, nil); err == nil {
		t.Error("ReplaceParkingSpots on closed DB: want error")
	}
	if err := ReplaceHeroBackgrounds(DB, nil); err == nil {
		t.Error("ReplaceHeroBackgrounds on closed DB: want error")
	}
	if err := ReplaceHomepageLabels(DB, map[string]map[string]string{}); err == nil {
		t.Error("ReplaceHomepageLabels on closed DB: want error")
	}
	if err := UpdateSettings(DB, models.Settings{}); err == nil {
		t.Error("UpdateSettings on closed DB: want error")
	}
	if err := WithTx(func(tx *sql.Tx) error { return nil }); err == nil {
		t.Error("WithTx on closed DB: want error")
	}
}

// A dropped table fails only the functions that touch it, which is what the
// handlers' selective 500 branches rely on.
func TestDroppedTableErrors(t *testing.T) {
	newTestDB(t)
	if _, err := DB.Exec(`DROP TABLE polls`); err != nil {
		t.Fatal(err)
	}
	if _, err := GetAllPolls(); err == nil {
		t.Error("GetAllPolls with dropped table: want error")
	}
	if err := DeletePoll(1); err == nil {
		t.Error("DeletePoll with dropped table: want error")
	}
	// unrelated functions keep working
	if err := CreateGuest("Still", "Works", "adult"); err != nil {
		t.Errorf("CreateGuest after dropping polls: %v", err)
	}
}

func TestSeedSettingsAfterDrop(t *testing.T) {
	newTestDB(t)
	if _, err := DB.Exec(`DROP TABLE settings`); err != nil {
		t.Fatal(err)
	}
	// seedSettings logs and returns instead of panicking
	seedSettings()
	if _, err := GetSettings(); err == nil {
		t.Error("GetSettings with dropped table: want error")
	}
}

func TestInitPanicsOnUnwritablePath(t *testing.T) {
	newTestDB(t)
	defer func() {
		if recover() == nil {
			t.Error("Init with a DSN in a missing directory did not panic")
		}
		// restore a working handle for any later test
		Init(filepath.Join(t.TempDir(), "restored.db"))
	}()
	Init(filepath.Join(t.TempDir(), "missing-dir", "test.db"))
}
