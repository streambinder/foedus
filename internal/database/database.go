package database

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

// Querier is the subset of *sql.DB and *sql.Tx that the data helpers need, so
// the same helper can run either standalone (on DB) or inside a transaction.
type Querier interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

// queryAll runs a query and maps every row through scan, collapsing the
// identical query/defer/loop/err dance every collection getter would repeat.
func queryAll[T any](query string, scan func(*sql.Rows) (T, error), args ...any) ([]T, error) {
	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []T
	for rows.Next() {
		item, err := scan(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// insertAll writes a whole collection in slice order, handing each row's index
// to values so it can double as sort_order. The dashboard form always submits
// these lists complete, so callers wipe the table (or their slice of it) first
// and re-insert rather than diffing by id — order is the form's, not the DB's.
func insertAll[T any](q Querier, table string, columns []string, items []T, values func(T, int) []any) error {
	statement := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		table,
		strings.Join(columns, ", "),
		strings.TrimSuffix(strings.Repeat("?, ", len(columns)), ", "),
	)
	for index, item := range items {
		if _, err := q.Exec(statement, values(item, index)...); err != nil {
			return err
		}
	}
	return nil
}

// nullableID maps an absent media id to SQL NULL — 0 would violate the FK,
// which is exactly the breakage the JSON blobs used to hide.
func nullableID(id int) any {
	if id <= 0 {
		return nil
	}
	return id
}

func idOrZero(id sql.NullInt64) int {
	if !id.Valid {
		return 0
	}
	return int(id.Int64)
}

// WithTx runs fn inside a single transaction, committing on success and rolling
// back on any error or panic. Callers get all-or-nothing semantics without
// repeating the begin/commit/rollback dance.
func WithTx(fn func(tx *sql.Tx) error) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// pragmas applied via DSN so they take effect on every pooled connection.
// WAL = concurrent readers + 1 writer; busy_timeout makes writers wait on lock
// instead of failing; foreign_keys turns on the FK constraints declared in the
// schema (sqlite default is OFF); synchronous=NORMAL is the WAL-recommended
// durability/perf tradeoff (only loses last txn on power-loss, never corrupts).
const sqlitePragmas = "_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)&_pragma=synchronous(NORMAL)"

func Init(dsn string) {
	start := time.Now()
	var err error
	DB, err = sql.Open("sqlite", dsnWithPragmas(dsn))
	if err != nil {
		slog.Error("failed to open database", "dsn", dsn, "error", err.Error())
		panic(err)
	}

	// WAL allows multiple readers in parallel — bump pool so the dashboard's
	// N+1 reads don't serialize the whole site behind a single connection.
	DB.SetMaxOpenConns(8)
	DB.SetMaxIdleConns(4)
	DB.SetConnMaxIdleTime(5 * time.Minute)
	slog.Info("database connection opened", "dsn", dsn, "max_open_conns", 8)

	migrate()
	seedSettings()
	slog.Info("database initialized", "dsn", dsn, "duration_ms", time.Since(start).Milliseconds())
}

// modernc.org/sqlite parses ?_pragma=foo(bar)&_pragma=... from the DSN tail.
func dsnWithPragmas(dsn string) string {
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	return dsn + sep + sqlitePragmas
}

// Every table is STRICT: sqlite's default affinity silently coerces (a TEXT
// value lands in an INTEGER column as text), STRICT rejects the mismatch at
// write time. It permits only INT/INTEGER/REAL/TEXT/BLOB/ANY, so timestamps are
// INTEGER unix seconds rather than the old DATETIME — which was never a real
// type anyway, just NUMERIC affinity. models.Timestamp does the conversion
// Go-side. Note DEFAULT (unixepoch()) in place of CURRENT_TIMESTAMP: the latter
// writes a text literal.
func migrate() {
	start := time.Now()
	statements := []string{
		`CREATE TABLE IF NOT EXISTS guests (
			id                  INTEGER PRIMARY KEY AUTOINCREMENT,
			first_name          TEXT NOT NULL,
			last_name           TEXT NOT NULL DEFAULT '',
			type                TEXT NOT NULL DEFAULT 'adult' CHECK (type IN ('adult','child','infant','vendor')),
			confirmed_ceremony  INTEGER,
			confirmed_reception INTEGER,
			invitation_id       INTEGER REFERENCES invitations(id),
			invitation_guest_order INTEGER,
			created_at          INTEGER NOT NULL DEFAULT (unixepoch()),
			updated_at          INTEGER NOT NULL DEFAULT (unixepoch())
		) STRICT`,
		`CREATE TABLE IF NOT EXISTS gifts (
			id               INTEGER PRIMARY KEY AUTOINCREMENT,
			amount           INTEGER NOT NULL,
			donor            TEXT NOT NULL DEFAULT '',
			registry_item_id INTEGER REFERENCES registry_items(id),
			confirmed        INTEGER NOT NULL DEFAULT 0,
			created_at       INTEGER NOT NULL DEFAULT (unixepoch())
		) STRICT`,
		`CREATE TABLE IF NOT EXISTS media (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			mime       TEXT NOT NULL,
			bytes      BLOB NOT NULL,
			created_at INTEGER NOT NULL DEFAULT (unixepoch())
		) STRICT`,
		// settings holds exactly one row (CHECK id = 1) of scalar config. Anything
		// that can occur more than once is a relation of its own: the old key/value
		// table had grown JSON arrays of places, personas and image ids, which put
		// every media reference in them outside FK enforcement. Declared after
		// media so its three references resolve.
		`CREATE TABLE IF NOT EXISTS settings (
			id                     INTEGER PRIMARY KEY CHECK (id = 1),
			groom_name             TEXT NOT NULL DEFAULT '',
			bride_name             TEXT NOT NULL DEFAULT '',
			ceremony_datetime      TEXT NOT NULL DEFAULT '',
			ceremony_address       TEXT NOT NULL DEFAULT '',
			ceremony_location      TEXT NOT NULL DEFAULT '',
			ceremony_city          TEXT NOT NULL DEFAULT '',
			ceremony_lat           REAL NOT NULL DEFAULT 0,
			ceremony_lng           REAL NOT NULL DEFAULT 0,
			ceremony_media_id      INTEGER REFERENCES media(id),
			reception_datetime     TEXT NOT NULL DEFAULT '',
			reception_address      TEXT NOT NULL DEFAULT '',
			reception_location     TEXT NOT NULL DEFAULT '',
			reception_city         TEXT NOT NULL DEFAULT '',
			reception_lat          REAL NOT NULL DEFAULT 0,
			reception_lng          REAL NOT NULL DEFAULT 0,
			reception_media_id     INTEGER REFERENCES media(id),
			bank_account_iban      TEXT NOT NULL DEFAULT '',
			bank_account_holder    TEXT NOT NULL DEFAULT '',
			spotify_playlist       TEXT NOT NULL DEFAULT '',
			share_preview_media_id INTEGER REFERENCES media(id)
		) STRICT`,
		`CREATE TABLE IF NOT EXISTS registry_items (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			name       TEXT NOT NULL,
			price      INTEGER NOT NULL,
			media_id   INTEGER REFERENCES media(id),
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL DEFAULT (unixepoch())
		) STRICT`,
		`CREATE TABLE IF NOT EXISTS invitations (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			code       TEXT NOT NULL UNIQUE,
			label      TEXT NOT NULL DEFAULT '',
			viewed_at  INTEGER,
			created_at INTEGER NOT NULL DEFAULT (unixepoch())
		) STRICT`,
		`CREATE TABLE IF NOT EXISTS polls (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			question    TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			created_at  INTEGER NOT NULL DEFAULT (unixepoch())
		) STRICT`,
		`CREATE TABLE IF NOT EXISTS poll_answers (
			id       INTEGER PRIMARY KEY AUTOINCREMENT,
			poll_id  INTEGER NOT NULL REFERENCES polls(id),
			guest_id INTEGER NOT NULL REFERENCES guests(id),
			answer   INTEGER NOT NULL DEFAULT 0,
			notes    TEXT NOT NULL DEFAULT '',
			UNIQUE(poll_id, guest_id)
		) STRICT`,
		`CREATE TABLE IF NOT EXISTS soundtrack_events (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			title      TEXT NOT NULL DEFAULT '',
			artist     TEXT NOT NULL DEFAULT '',
			url        TEXT NOT NULL DEFAULT '',
			invite_id  TEXT NOT NULL DEFAULT '',
			created_at INTEGER NOT NULL DEFAULT (unixepoch())
		) STRICT`,
		// story places and honeymoon destinations are the same entity in two
		// roles, so one table with a kind discriminator; sort_order is scoped
		// per kind and mirrors the dashboard form order.
		`CREATE TABLE IF NOT EXISTS places (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			kind       TEXT NOT NULL CHECK (kind IN ('story','honeymoon')),
			label      TEXT NOT NULL DEFAULT '',
			name       TEXT NOT NULL DEFAULT '',
			address    TEXT NOT NULL DEFAULT '',
			date       TEXT NOT NULL DEFAULT '',
			lat        REAL NOT NULL DEFAULT 0,
			lng        REAL NOT NULL DEFAULT 0,
			media_id   INTEGER REFERENCES media(id),
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL DEFAULT (unixepoch())
		) STRICT`,
		`CREATE TABLE IF NOT EXISTS parking_spots (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			lat        REAL NOT NULL DEFAULT 0,
			lng        REAL NOT NULL DEFAULT 0,
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL DEFAULT (unixepoch())
		) STRICT`,
		`CREATE TABLE IF NOT EXISTS accommodations (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			name        TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			url         TEXT NOT NULL DEFAULT '',
			sort_order  INTEGER NOT NULL DEFAULT 0,
			created_at  INTEGER NOT NULL DEFAULT (unixepoch())
		) STRICT`,
		`CREATE TABLE IF NOT EXISTS impersonations (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			codename   TEXT NOT NULL DEFAULT '',
			profile    TEXT NOT NULL DEFAULT '',
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL DEFAULT (unixepoch())
		) STRICT`,
		`CREATE TABLE IF NOT EXISTS hero_backgrounds (
			id                INTEGER PRIMARY KEY AUTOINCREMENT,
			desktop_media_id  INTEGER REFERENCES media(id),
			mobile_media_id   INTEGER REFERENCES media(id),
			sort_order        INTEGER NOT NULL DEFAULT 0,
			created_at        INTEGER NOT NULL DEFAULT (unixepoch())
		) STRICT`,
		// only overridden labels are stored; a missing (lang, key) falls through
		// to the compiled-in i18n default.
		`CREATE TABLE IF NOT EXISTS homepage_labels (
			lang  TEXT NOT NULL,
			key   TEXT NOT NULL,
			value TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (lang, key)
		) STRICT`,
	}
	for index, s := range statements {
		stmtStart := time.Now()
		if _, err := DB.Exec(s); err != nil {
			slog.Error("migration statement failed", "index", index, "duration_ms", time.Since(stmtStart).Milliseconds(), "error", err.Error())
			panic(err)
		}
		slog.Debug("migration statement applied", "index", index, "duration_ms", time.Since(stmtStart).Milliseconds())
	}

	slog.Info("database migrations complete", "statements", len(statements), "duration_ms", time.Since(start).Milliseconds())
}
