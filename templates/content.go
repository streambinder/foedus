package templates

import "github.com/streambinder/foedus/internal/models"

// The collections that used to ride along inside WeddingSettings are now their
// own relations, and each page loads only the ones it renders. They travel
// grouped rather than as positional parameters: Places and Honeymoon are both
// []models.Place, so as bare arguments they could be swapped with no compile
// error and a silently wrong page.

// HomeContent is what the public homepage renders.
type HomeContent struct {
	Places         []models.Place
	Honeymoon      []models.Place
	ParkingSpots   []models.ParkingSpot
	Accommodations []models.Accommodation
	HeroBackground models.HeroBackground // one pair, already picked for this render
}

// SettingsContent is what the dashboard settings form edits.
type SettingsContent struct {
	Places          []models.Place
	Honeymoon       []models.Place
	ParkingSpots    []models.ParkingSpot
	Accommodations  []models.Accommodation
	Impersonations  []models.Impersonation
	HeroBackgrounds []models.HeroBackground
	HomepageLabels  map[string]map[string]string
}
