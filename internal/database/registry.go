package database

import (
	"database/sql"
	"fmt"

	"github.com/streambinder/foedus/internal/models"
)

func CreateRegistryItem(name string, price int, image string) error {
	var nextSortOrder int
	if err := DB.QueryRow(`SELECT COALESCE(MAX(sort_order), 0) + 1 FROM registry_items`).Scan(&nextSortOrder); err != nil {
		return err
	}
	_, err := DB.Exec(
		`INSERT INTO registry_items (name, price, image, sort_order) VALUES (?, ?, ?, ?)`,
		name, price, image, nextSortOrder,
	)
	return err
}

func GetAllRegistryItems() ([]models.RegistryItem, error) {
	rows, err := DB.Query(`SELECT id, name, price, image, sort_order, created_at FROM registry_items ORDER BY sort_order ASC, created_at DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.RegistryItem
	for rows.Next() {
		var item models.RegistryItem
		if err := rows.Scan(&item.ID, &item.Name, &item.Price, &item.Image, &item.SortOrder, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func GetRegistryItem(id int) (models.RegistryItem, error) {
	var item models.RegistryItem
	err := DB.QueryRow(
		`SELECT id, name, price, image, sort_order, created_at FROM registry_items WHERE id = ?`, id,
	).Scan(&item.ID, &item.Name, &item.Price, &item.Image, &item.SortOrder, &item.CreatedAt)
	return item, err
}

func UpdateRegistryItem(id int, name string, price int, image string) error {
	_, err := DB.Exec(
		`UPDATE registry_items SET name = ?, price = ?, image = ? WHERE id = ?`,
		name, price, image, id,
	)
	return err
}

func DeleteRegistryItem(id int) error {
	_, err := DB.Exec(`DELETE FROM registry_items WHERE id = ?`, id)
	return err
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

	var item models.RegistryItem
	err = tx.QueryRow(
		`SELECT id, name, price, image, sort_order, created_at FROM registry_items WHERE id = ?`,
		id,
	).Scan(&item.ID, &item.Name, &item.Price, &item.Image, &item.SortOrder, &item.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return err
		}
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
