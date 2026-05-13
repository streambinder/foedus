package database

import (
	"database/sql"
	"log/slog"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

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
	SeedSettings()
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

func migrate() {
	start := time.Now()
	statements := []string{
		`CREATE TABLE IF NOT EXISTS settings (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS guests (
			id                  INTEGER PRIMARY KEY AUTOINCREMENT,
			first_name          TEXT NOT NULL,
			last_name           TEXT NOT NULL DEFAULT '',
			type                TEXT NOT NULL DEFAULT 'adult' CHECK (type IN ('adult','child','infant','vendor')),
			confirmed_ceremony  INTEGER,
			confirmed_reception INTEGER,
			invitation_id       INTEGER REFERENCES invitations(id),
			invitation_guest_order INTEGER,
			created_at          DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at          DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS gifts (
			id               INTEGER PRIMARY KEY AUTOINCREMENT,
			amount           INTEGER NOT NULL,
			donor            TEXT NOT NULL DEFAULT '',
			registry_item_id INTEGER REFERENCES registry_items(id),
			confirmed        INTEGER NOT NULL DEFAULT 0,
			created_at       DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS media (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			mime       TEXT NOT NULL,
			bytes      BLOB NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS registry_items (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			name       TEXT NOT NULL,
			price      INTEGER NOT NULL,
			media_id   INTEGER REFERENCES media(id),
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS invitations (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			code       TEXT NOT NULL UNIQUE,
			label      TEXT NOT NULL DEFAULT '',
			viewed_at  DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS polls (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			question    TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS poll_answers (
			id       INTEGER PRIMARY KEY AUTOINCREMENT,
			poll_id  INTEGER NOT NULL REFERENCES polls(id),
			guest_id INTEGER NOT NULL REFERENCES guests(id),
			answer   INTEGER NOT NULL DEFAULT 0,
			notes    TEXT NOT NULL DEFAULT '',
			UNIQUE(poll_id, guest_id)
		)`,
		`CREATE TABLE IF NOT EXISTS soundtrack_events (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			title      TEXT NOT NULL DEFAULT '',
			artist     TEXT NOT NULL DEFAULT '',
			url        TEXT NOT NULL DEFAULT '',
			invite_id  TEXT NOT NULL DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
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
