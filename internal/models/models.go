package models

import "time"

type Setting struct {
	Key   string
	Value string
}

type Place struct {
	Label   string  `json:"label"`
	Date    string  `json:"date"`
	MediaID int     `json:"media_id"`
	Name    string  `json:"name"`
	Address string  `json:"address"`
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
}

// Coord is a bare lat/lng point — used for ceremony parking spots, which need
// no label/photo, only a location to pin on the map and route to.
type Coord struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type Impersonation struct {
	Codename string `json:"codename"`
	Profile  string `json:"profile"`
}

type AccommodationSuggestion struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	URL         string `json:"url"`
}

type HomepageHeroBackground struct {
	DesktopMediaID int `json:"desktop_media_id"`
	MobileMediaID  int `json:"mobile_media_id"`
}

type WeddingSettings struct {
	Spouse1Name              string
	Spouse2Name              string
	CeremonyAddress          string
	CeremonyLocation         string
	CeremonyCity             string
	CeremonyDatetime         string
	CeremonyLat              float64
	CeremonyLng              float64
	ReceptionAddress         string
	ReceptionLocation        string
	ReceptionCity            string
	ReceptionDatetime        string
	ReceptionLat             float64
	ReceptionLng             float64
	CeremonyMediaID          int
	ReceptionMediaID         int
	BankAccountIBAN          string
	BankAccountHolder        string
	SpotifyPlaylist          string
	Places                   []Place
	HoneymoonLocations       []Place
	ParkingSpots             []Coord
	AccommodationSuggestions []AccommodationSuggestion
	Impersonations           []Impersonation
	HomepageLabels           map[string]map[string]string
	HomepageHeroBackgrounds  []HomepageHeroBackground
	SharePreviewMediaID      int
}

func (s WeddingSettings) IsConfigured() bool {
	return s.Spouse1Name != "" && s.Spouse2Name != ""
}

type Guest struct {
	ID                 int
	FirstName          string
	LastName           string
	Type               string // adult | child | infant | vendor — non-counted: infant, vendor
	ConfirmedCeremony  *bool
	ConfirmedReception *bool
	InvitationID       *int
	InvitationOrder    *int
	PollAnswers        []PollAnswer
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type Poll struct {
	ID          int
	Question    string
	Description string
	TotalCount  int         // computed at query time
	YesVoters   []PollVoter // guests who answered yes
	CreatedAt   time.Time
}

type PollVoter struct {
	Name  string
	Notes string
}

type PollAnswer struct {
	PollID int
	Answer bool
	Notes  string
}

type Invitation struct {
	ID        int
	Code      string
	Label     string
	ViewedAt  *time.Time
	CreatedAt time.Time
	Guests    []Guest
}

type Gift struct {
	ID             int
	Amount         int    // whole currency units (e.g. euros), no cents
	Donor          string // who sent the gift
	RegistryItemID *int   // fk to registry_items, nil for generic gifts
	Confirmed      bool
	CreatedAt      time.Time
}

type RegistryItem struct {
	ID        int
	Name      string
	Price     int // whole currency units (e.g. euros), no cents
	MediaID   int
	SortOrder int
	CreatedAt time.Time
}

type SoundtrackEvent struct {
	ID        int
	Title     string
	Artist    string
	URL       string
	InviteID  string
	CreatedAt time.Time
}
