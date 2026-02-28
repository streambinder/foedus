package database

import "github.com/dpucci/foedus/internal/models"

func CreateRegistryItem(name string, price int, image string) error {
	_, err := DB.Exec(
		`INSERT INTO registry_items (name, price, image) VALUES (?, ?, ?)`,
		name, price, image,
	)
	return err
}

func GetAllRegistryItems() ([]models.RegistryItem, error) {
	rows, err := DB.Query(`SELECT id, name, price, image, created_at FROM registry_items ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.RegistryItem
	for rows.Next() {
		var item models.RegistryItem
		if err := rows.Scan(&item.ID, &item.Name, &item.Price, &item.Image, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func GetRegistryItem(id int) (models.RegistryItem, error) {
	var item models.RegistryItem
	err := DB.QueryRow(
		`SELECT id, name, price, image, created_at FROM registry_items WHERE id = ?`, id,
	).Scan(&item.ID, &item.Name, &item.Price, &item.Image, &item.CreatedAt)
	return item, err
}

func DeleteRegistryItem(id int) error {
	_, err := DB.Exec(`DELETE FROM registry_items WHERE id = ?`, id)
	return err
}
