package database

import (
	"database/sql"
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
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			first_name TEXT NOT NULL,
			last_name  TEXT NOT NULL DEFAULT '',
			confirmed  INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS gifts (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			amount     INTEGER NOT NULL,
			currency   TEXT NOT NULL DEFAULT 'eur',
			donor      TEXT NOT NULL DEFAULT '',
			message    TEXT NOT NULL DEFAULT '',
			session_id TEXT NOT NULL UNIQUE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS registry_items (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			name       TEXT NOT NULL,
			price      INTEGER NOT NULL,
			image      TEXT NOT NULL DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}
	for _, s := range statements {
		if _, err := DB.Exec(s); err != nil {
			log.Fatalf("migration failed: %v", err)
		}
	}
}
