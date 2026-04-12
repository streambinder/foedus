package models

import "time"

type Setting struct {
	Key   string
	Value string
}

type Place struct {
	Label   string  `json:"label"`
	Name    string  `json:"name"`
	Address string  `json:"address"`
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
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

type WeddingSettings struct {
	Spouse1Name              string
	Spouse2Name              string
	CeremonyAddress          string
	CeremonyLocation         string
	CeremonyDatetime         string
	ReceptionAddress         string
	ReceptionLocation        string
	CeremonyImage            string
	ReceptionImage           string
	BankAccountIBAN          string
	BankAccountHolder        string
	SpotifyPlaylists         []string
	Places                   []Place
	AccommodationSuggestions []AccommodationSuggestion
	Impersonations           []Impersonation
	HomepageLabels           map[string]map[string]string
	SharePreviewImage        string
}

func (s WeddingSettings) IsConfigured() bool {
	return s.Spouse1Name != "" && s.Spouse2Name != ""
}

type Guest struct {
	ID                 int
	FirstName          string
	LastName           string
	ConfirmedCeremony  *bool
	ConfirmedReception *bool
	InvitationID       *int
	PollAnswers        []PollAnswer
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type Poll struct {
	ID         int
	Question   string
	TotalCount int      // computed at query time
	YesVoters  []string // guest names who answered yes
	CreatedAt  time.Time
}

type PollAnswer struct {
	PollID int
	Answer bool
	Notes  string
}

type Invitation struct {
	ID        int
	Code      string
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
	Price     int    // whole currency units (e.g. euros), no cents
	Image     string // base64 data URI
	CreatedAt time.Time
}
