package database

import (
	"database/sql"
)

type Media struct {
	ID    int
	Mime  string
	Bytes []byte
}

func InsertMedia(mime string, data []byte) (int, error) {
	res, err := DB.Exec(`INSERT INTO media (mime, bytes) VALUES (?, ?)`, mime, data)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func GetMedia(id int) (Media, error) {
	var m Media
	err := DB.QueryRow(`SELECT id, mime, bytes FROM media WHERE id = ?`, id).Scan(&m.ID, &m.Mime, &m.Bytes)
	return m, err
}

// GetMediaMeta returns mime + length without loading bytes — for cheap existence checks.
func GetMediaMeta(id int) (string, int, error) {
	var mime string
	var length int
	err := DB.QueryRow(`SELECT mime, length(bytes) FROM media WHERE id = ?`, id).Scan(&mime, &length)
	if err == sql.ErrNoRows {
		return "", 0, err
	}
	return mime, length, err
}

func DeleteMedia(id int) error {
	if id <= 0 {
		return nil
	}
	_, err := DB.Exec(`DELETE FROM media WHERE id = ?`, id)
	return err
}
