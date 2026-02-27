package handlers

import (
	"encoding/csv"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/dpucci/foedus/internal/database"
	"github.com/dpucci/foedus/internal/i18n"
	"github.com/dpucci/foedus/templates"
	"github.com/gofiber/fiber/v2"
)

func setFlash(c *fiber.Ctx, msg string) {
	c.Cookie(&fiber.Cookie{
		Name:    "flash",
		Value:   url.QueryEscape(msg),
		Expires: time.Now().Add(5 * time.Second),
		Path:    "/dashboard",
	})
}

func getFlash(c *fiber.Ctx) string {
	raw := c.Cookies("flash")
	if raw == "" {
		return ""
	}
	// expire cookie immediately
	c.Cookie(&fiber.Cookie{
		Name:    "flash",
		Value:   "",
		Expires: time.Now().Add(-1 * time.Hour),
		Path:    "/dashboard",
	})
	msg, _ := url.QueryUnescape(raw)
	return msg
}

func getT(c *fiber.Ctx) i18n.T {
	if t, ok := c.Locals("t").(i18n.T); ok {
		return t
	}
	return i18n.NewT("en")
}

func getLang(c *fiber.Ctx) string {
	if lang, ok := c.Locals("lang").(string); ok {
		return lang
	}
	return "en"
}

const guestsPerPage = 20

func DashboardIndex(c *fiber.Ctx) error {
	settings, err := database.GetAllSettings()
	if err != nil {
		return c.Status(500).SendString("failed to load settings")
	}
	confirmed, totalGuests, err := database.CountConfirmed()
	if err != nil {
		return c.Status(500).SendString("failed to count guests")
	}
	gifts, err := database.GetAllGifts()
	if err != nil {
		return c.Status(500).SendString("failed to load gifts")
	}
	search := strings.TrimSpace(c.Query("q"))
	page, _ := strconv.Atoi(c.Query("page", "1"))
	if page < 1 {
		page = 1
	}
	guests, filteredTotal, err := database.GetGuestsPaginated(page, guestsPerPage, search)
	if err != nil {
		return c.Status(500).SendString("failed to load guests")
	}
	totalPages := (filteredTotal + guestsPerPage - 1) / guestsPerPage
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	csrfToken, _ := c.Locals("csrf").(string)
	return Render(c, templates.Dashboard(settings, guests, gifts, confirmed, totalGuests, page, totalPages, search, csrfToken, getFlash(c), StripeEnabled(), getT(c), getLang(c)))
}

func SaveSettings(c *fiber.Ctx) error {
	keys := []string{
		"spouse1_name", "spouse2_name", "ceremony_datetime",
		"ceremony_address", "ceremony_location",
		"reception_address", "reception_location",
	}
	for _, key := range keys {
		if err := database.UpdateSetting(key, c.FormValue(key)); err != nil {
			return c.Status(500).SendString("failed to save settings")
		}
	}
	setFlash(c, getT(c)("flash.settings_saved"))
	return c.Redirect("/dashboard")
}

func AddGuest(c *fiber.Ctx) error {
	if err := database.CreateGuest(
		c.FormValue("first_name"),
		c.FormValue("last_name"),
	); err != nil {
		return c.Status(500).SendString("failed to add guest")
	}
	setFlash(c, getT(c)("flash.guest_added"))
	return c.Redirect("/dashboard")
}

func ImportGuestsCSV(c *fiber.Ctx) error {
	fh, err := c.FormFile("csv_file")
	if err != nil {
		return c.Status(400).SendString("no file uploaded")
	}
	f, err := fh.Open()
	if err != nil {
		return c.Status(500).SendString("failed to open file")
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1 // allow variable column count
	r.TrimLeadingSpace = true

	var imported int
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		// join all columns into a single full name
		fullName := strings.TrimSpace(strings.Join(record, " "))
		if fullName == "" {
			continue
		}
		// skip header row
		lower := strings.ToLower(fullName)
		if strings.Contains(lower, "first") || strings.Contains(lower, "last") || strings.Contains(lower, "nome") || strings.Contains(lower, "cognome") {
			continue
		}
		// split by space: last token = last name, rest = first name
		parts := strings.Fields(fullName)
		var firstName, lastName string
		if len(parts) == 1 {
			firstName = parts[0]
		} else {
			lastName = parts[len(parts)-1]
			firstName = strings.Join(parts[:len(parts)-1], " ")
		}
		if err := database.CreateGuest(firstName, lastName); err != nil {
			continue
		}
		imported++
	}
	setFlash(c, strconv.Itoa(imported)+" "+getT(c)("flash.guests_imported"))
	return c.Redirect("/dashboard")
}

func ToggleConfirmed(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).SendString("invalid id")
	}
	if err := database.ToggleConfirmed(id); err != nil {
		return c.Status(500).SendString("failed to toggle confirmed")
	}
	return c.Redirect("/dashboard")
}

func EditGuestPage(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).SendString("invalid id")
	}
	guest, err := database.GetGuest(id)
	if err != nil {
		return c.Status(404).SendString("guest not found")
	}
	csrfToken, _ := c.Locals("csrf").(string)
	return Render(c, templates.EditGuest(guest, csrfToken, getT(c), getLang(c)))
}

func UpdateGuest(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).SendString("invalid id")
	}
	if err := database.UpdateGuest(
		id,
		c.FormValue("first_name"),
		c.FormValue("last_name"),
	); err != nil {
		return c.Status(500).SendString("failed to update guest")
	}
	setFlash(c, getT(c)("flash.guest_updated"))
	return c.Redirect("/dashboard")
}

func DeleteGuest(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).SendString("invalid id")
	}
	if err := database.DeleteGuest(id); err != nil {
		return c.Status(500).SendString("failed to delete guest")
	}
	setFlash(c, getT(c)("flash.guest_deleted"))
	return c.Redirect("/dashboard")
}
