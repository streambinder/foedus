package database

import (
	"database/sql"
	"fmt"

	"github.com/streambinder/foedus/internal/models"
)

func CreateRegistryItem(name string, price int, mediaID int) error {
	var nextSortOrder int
	if err := DB.QueryRow(`SELECT COALESCE(MAX(sort_order), 0) + 1 FROM registry_items`).Scan(&nextSortOrder); err != nil {
		return err
	}
	var mid sql.NullInt64
	if mediaID > 0 {
		mid = sql.NullInt64{Int64: int64(mediaID), Valid: true}
	}
	_, err := DB.Exec(
		`INSERT INTO registry_items (name, price, media_id, sort_order) VALUES (?, ?, ?, ?)`,
		name, price, mid, nextSortOrder,
	)
	return err
}

func scanRegistryItem(s interface {
	Scan(...any) error
},
) (models.RegistryItem, error) {
	var item models.RegistryItem
	var mid sql.NullInt64
	if err := s.Scan(&item.ID, &item.Name, &item.Price, &mid, &item.SortOrder, &item.CreatedAt); err != nil {
		return item, err
	}
	if mid.Valid {
		item.MediaID = int(mid.Int64)
	}
	return item, nil
}

func GetAllRegistryItems() ([]models.RegistryItem, error) {
	rows, err := DB.Query(`SELECT id, name, price, media_id, sort_order, created_at FROM registry_items ORDER BY sort_order ASC, created_at DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.RegistryItem
	for rows.Next() {
		item, err := scanRegistryItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func GetRegistryItem(id int) (models.RegistryItem, error) {
	return scanRegistryItem(DB.QueryRow(
		`SELECT id, name, price, media_id, sort_order, created_at FROM registry_items WHERE id = ?`, id,
	))
}

func UpdateRegistryItem(id int, name string, price int, mediaID int) error {
	var mid sql.NullInt64
	if mediaID > 0 {
		mid = sql.NullInt64{Int64: int64(mediaID), Valid: true}
	}
	_, err := DB.Exec(
		`UPDATE registry_items SET name = ?, price = ?, media_id = ? WHERE id = ?`,
		name, price, mid, id,
	)
	return err
}

func DeleteRegistryItem(id int) error {
	// gifts.registry_item_id has a FK to registry_items(id); with foreign_keys=ON
	// we must null out dependent gifts first or the DELETE fails. Keep gift rows
	// (they're real money received) but detach them from the deleted item.
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE gifts SET registry_item_id = NULL WHERE registry_item_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM registry_items WHERE id = ?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

func MoveRegistryItem(id int, direction string) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	item, err := scanRegistryItem(tx.QueryRow(
		`SELECT id, name, price, media_id, sort_order, created_at FROM registry_items WHERE id = ?`,
		id,
	))
	if err != nil {
		return err
	}

	orderClause := "ASC"
	comparison := ">"
	if direction == "up" {
		orderClause = "DESC"
		comparison = "<"
	} else if direction != "down" {
		return fmt.Errorf("invalid direction")
	}

	query := fmt.Sprintf(
		`SELECT id, sort_order FROM registry_items WHERE sort_order %s ? ORDER BY sort_order %s, id %s LIMIT 1`,
		comparison,
		orderClause,
		orderClause,
	)

	var swapID, swapSortOrder int
	err = tx.QueryRow(query, item.SortOrder).Scan(&swapID, &swapSortOrder)
	if err != nil {
		if err == sql.ErrNoRows {
			if err := tx.Commit(); err != nil {
				return err
			}
			committed = true
			return nil
		}
		return err
	}

	if _, err = tx.Exec(`UPDATE registry_items SET sort_order = ? WHERE id = ?`, -1, item.ID); err != nil {
		return err
	}
	if _, err = tx.Exec(`UPDATE registry_items SET sort_order = ? WHERE id = ?`, item.SortOrder, swapID); err != nil {
		return err
	}
	if _, err = tx.Exec(`UPDATE registry_items SET sort_order = ? WHERE id = ?`, swapSortOrder, item.ID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}
