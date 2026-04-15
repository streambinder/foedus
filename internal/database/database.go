package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func Init(dsn string) {
	var err error
	DB, err = sql.Open("sqlite", dsn)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}

	// single connection prevents "database is locked" on concurrent writes
	DB.SetMaxOpenConns(1)

	migrate()
	SeedSettings()
}

func migrate() {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS settings (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS guests (
			id                  INTEGER PRIMARY KEY AUTOINCREMENT,
			first_name          TEXT NOT NULL,
			last_name           TEXT NOT NULL DEFAULT '',
			confirmed_ceremony  INTEGER,
			confirmed_reception INTEGER,
			invitation_id       INTEGER REFERENCES invitations(id),
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
		`CREATE TABLE IF NOT EXISTS registry_items (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			name       TEXT NOT NULL,
			price      INTEGER NOT NULL,
			image      TEXT NOT NULL DEFAULT '',
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS invitations (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			code       TEXT NOT NULL UNIQUE,
			viewed_at  DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS polls (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			question   TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS poll_answers (
			id       INTEGER PRIMARY KEY AUTOINCREMENT,
			poll_id  INTEGER NOT NULL REFERENCES polls(id),
			guest_id INTEGER NOT NULL REFERENCES guests(id),
			answer   INTEGER NOT NULL DEFAULT 0,
			notes    TEXT NOT NULL DEFAULT '',
			UNIQUE(poll_id, guest_id)
		)`,
	}
	for _, s := range statements {
		if _, err := DB.Exec(s); err != nil {
			log.Fatalf("migration failed: %v", err)
		}
	}

	var version int
	if err := DB.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		log.Fatalf("failed to read schema version: %v", err)
	}
	if version < 1 {
		if _, err := DB.Exec(`UPDATE gifts SET amount = CAST(amount / 100 AS INTEGER)`); err != nil {
			log.Fatalf("failed to migrate gift amounts to integer euros: %v", err)
		}
		version = 1
	}
	if version < 2 {
		hasNotes, err := tableHasColumn("poll_answers", "notes")
		if err != nil {
			log.Fatalf("failed to inspect poll_answers schema: %v", err)
		}
		if !hasNotes {
			if _, err := DB.Exec(`ALTER TABLE poll_answers ADD COLUMN notes TEXT NOT NULL DEFAULT ''`); err != nil {
				log.Fatalf("failed to add notes column to poll_answers: %v", err)
			}
		}
		version = 2
	}
	if version < 3 {
		hasConfirmed, err := tableHasColumn("gifts", "confirmed")
		if err != nil {
			log.Fatalf("failed to inspect gifts schema: %v", err)
		}
		if !hasConfirmed {
			if _, err := DB.Exec(`ALTER TABLE gifts ADD COLUMN confirmed INTEGER NOT NULL DEFAULT 0`); err != nil {
				log.Fatalf("failed to add confirmed column to gifts: %v", err)
			}
		}
		version = 3
	}
	if version < 4 {
		hasSortOrder, err := tableHasColumn("registry_items", "sort_order")
		if err != nil {
			log.Fatalf("failed to inspect registry_items schema: %v", err)
		}
		if !hasSortOrder {
			if _, err := DB.Exec(`ALTER TABLE registry_items ADD COLUMN sort_order INTEGER NOT NULL DEFAULT 0`); err != nil {
				log.Fatalf("failed to add sort_order column to registry_items: %v", err)
			}
		}

		rows, err := DB.Query(`SELECT id FROM registry_items ORDER BY created_at DESC, id DESC`)
		if err != nil {
			log.Fatalf("failed to load registry_items for sort_order backfill: %v", err)
		}
		var ids []int
		for rows.Next() {
			var id int
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				log.Fatalf("failed to scan registry_items id for sort_order backfill: %v", err)
			}
			ids = append(ids, id)
		}
		if err := rows.Close(); err != nil {
			log.Fatalf("failed to close registry_items rows for sort_order backfill: %v", err)
		}
		for idx, id := range ids {
			if _, err := DB.Exec(`UPDATE registry_items SET sort_order = ? WHERE id = ?`, idx+1, id); err != nil {
				log.Fatalf("failed to backfill registry_items sort_order: %v", err)
			}
		}
		version = 4
	}
	if _, err := DB.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, version)); err != nil {
		log.Fatalf("failed to persist schema version: %v", err)
	}
}

func tableHasColumn(tableName, columnName string) (bool, error) {
	rows, err := DB.Query(fmt.Sprintf(`PRAGMA table_info(%s)`, tableName))
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, columnType string
		var notNull, pk int
		var defaultValue any
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			return false, err
		}
		if name == columnName {
			return true, nil
		}
	}
	return false, rows.Err()
}
