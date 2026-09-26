package database

import (
	"path/filepath"
	"testing"
	"time"
)

// STRICT + INTEGER timestamps move a whole class of failure from compile time to
// run time: a column type the driver can't hand back as the Go field type, or a
// value sqlite refuses to store, both build fine and blow up on first request.
// These tests exercise migrate() and the read/write paths against a real DB.

func newTestDB(t *testing.T) {
	t.Helper()
	Init(filepath.Join(t.TempDir(), "test.db"))
	t.Cleanup(func() { DB.Close() })
}

func TestMigrateCreatesStrictTables(t *testing.T) {
	newTestDB(t)

	rows, err := DB.Query(`SELECT name, sql FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var name, schema string
		if err := rows.Scan(&name, &schema); err != nil {
			t.Fatal(err)
		}
		count++
		// sqlite normalises the suffix to ") STRICT" in the stored DDL
		if !hasSuffixFold(schema, "strict") {
			t.Errorf("table %s is not STRICT: %s", name, schema)
		}
	}
	if count == 0 {
		t.Fatal("no tables created")
	}
}

func TestStrictRejectsMistypedWrite(t *testing.T) {
	newTestDB(t)

	if _, err := DB.Exec(`INSERT INTO guests (first_name, created_at) VALUES ('x', 'not a date')`); err == nil {
		t.Fatal("expected STRICT to reject a TEXT value in the INTEGER created_at column")
	}
	// the pre-STRICT schema would have silently stored this text
	if _, err := DB.Exec(`INSERT INTO guests (first_name, created_at) VALUES ('x', CURRENT_TIMESTAMP)`); err == nil {
		t.Fatal("expected STRICT to reject CURRENT_TIMESTAMP (text) in the INTEGER created_at column")
	}
}

func TestTimestampRoundTrip(t *testing.T) {
	newTestDB(t)

	before := time.Now().UTC().Add(-time.Second)
	if err := CreateGuest("Ada", "Lovelace", "adult"); err != nil {
		t.Fatal(err)
	}
	guests, err := GetAllGuests()
	if err != nil {
		t.Fatal(err)
	}
	if len(guests) != 1 {
		t.Fatalf("got %d guests, want 1", len(guests))
	}

	// the default must arrive as a usable time.Time, not a raw int64 or a
	// failed scan — this is what breaks if the Scanner is missing
	created := guests[0].CreatedAt
	if created.IsZero() || created.Before(before) || created.After(time.Now().UTC().Add(time.Second)) {
		t.Fatalf("created_at %v is not around now", created.Time)
	}

	// UpdatedAt must move forward on write; unixepoch() is second-granular so
	// assert monotonicity against the row's own creation, not a sleep
	if err := UpdateGuest(guests[0].ID, "Ada", "Byron", "adult"); err != nil {
		t.Fatal(err)
	}
	updated, err := GetGuest(guests[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.UpdatedAt.Before(created.Time) {
		t.Errorf("updated_at %v predates created_at %v", updated.UpdatedAt.Time, created.Time)
	}
}

func TestNullableTimestampRoundTrip(t *testing.T) {
	newTestDB(t)

	if err := CreateGuest("Grace", "Hopper", "adult"); err != nil {
		t.Fatal(err)
	}
	guests, _ := GetAllGuests()
	code, err := CreateInvitation([]int{guests[0].ID}, "Grace")
	if err != nil {
		t.Fatal(err)
	}

	inv, err := GetInvitationByCode(code)
	if err != nil {
		t.Fatal(err)
	}
	if inv.ViewedAt != nil {
		t.Errorf("fresh invitation has ViewedAt %v, want nil", inv.ViewedAt.Time)
	}

	if err := MarkInvitationViewed(inv.ID); err != nil {
		t.Fatal(err)
	}
	if inv, err = GetInvitationByCode(code); err != nil {
		t.Fatal(err)
	}
	if inv.ViewedAt == nil || inv.ViewedAt.IsZero() {
		t.Fatal("ViewedAt still unset after MarkInvitationViewed")
	}

	if err := ResetInvitationViewed(inv.ID); err != nil {
		t.Fatal(err)
	}
	if inv, err = GetInvitationByCode(code); err != nil {
		t.Fatal(err)
	}
	if inv.ViewedAt != nil {
		t.Errorf("ViewedAt %v survived reset, want nil", inv.ViewedAt.Time)
	}
}

func hasSuffixFold(s, suffix string) bool {
	if len(s) < len(suffix) {
		return false
	}
	tail := s[len(s)-len(suffix):]
	for i := range tail {
		a, b := tail[i], suffix[i]
		if a >= 'A' && a <= 'Z' {
			a += 'a' - 'A'
		}
		if b >= 'A' && b <= 'Z' {
			b += 'a' - 'A'
		}
		if a != b {
			return false
		}
	}
	return true
}
