package models

// place kinds — same entity, two roles on the homepage: story places render on
// the timeline, honeymoon ones in their own section.
const (
	PlaceKindStory     = "story"
	PlaceKindHoneymoon = "honeymoon"
)

// Settings is the wedding's scalar configuration — one row, one struct. Every
// collection that used to live here as a JSON blob is now its own relation.
type Settings struct {
	GroomName           string
	BrideName           string
	CeremonyAddress     string
	CeremonyLocation    string
	CeremonyCity        string
	CeremonyDatetime    string
	CeremonyLat         float64
	CeremonyLng         float64
	CeremonyMediaID     int
	ReceptionAddress    string
	ReceptionLocation   string
	ReceptionCity       string
	ReceptionDatetime   string
	ReceptionLat        float64
	ReceptionLng        float64
	ReceptionMediaID    int
	BankAccountIBAN     string
	BankAccountHolder   string
	SpotifyPlaylist     string
	SharePreviewMediaID int
}

func (s Settings) IsConfigured() bool {
	return s.GroomName != "" && s.BrideName != ""
}

type Place struct {
	ID        int
	Kind      string // story | honeymoon
	Label     string
	Date      string
	Name      string
	Address   string
	Lat       float64
	Lng       float64
	MediaID   int
	SortOrder int
	CreatedAt Timestamp
}

// ParkingSpot is a bare lat/lng pin for the ceremony parking map — no label or
// photo, only somewhere to route to.
type ParkingSpot struct {
	ID        int
	Lat       float64
	Lng       float64
	SortOrder int
	CreatedAt Timestamp
}

type Accommodation struct {
	ID          int
	Name        string
	Description string
	URL         string
	SortOrder   int
	CreatedAt   Timestamp
}

// Impersonation is a chat persona: a codename plus the profile text injected
// into the LLM system prompt.
type Impersonation struct {
	ID        int
	Codename  string
	Profile   string
	SortOrder int
	CreatedAt Timestamp
}

// HeroBackground pairs a horizontal and a vertical image for the landing
// section; one pair is picked at random per homepage load.
type HeroBackground struct {
	ID             int
	DesktopMediaID int
	MobileMediaID  int
	SortOrder      int
	CreatedAt      Timestamp
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
	CreatedAt          Timestamp
	UpdatedAt          Timestamp
}

type Poll struct {
	ID          int
	Question    string
	Description string
	TotalCount  int         // computed at query time
	YesVoters   []PollVoter // guests who answered yes
	CreatedAt   Timestamp
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
	ViewedAt  *Timestamp
	CreatedAt Timestamp
	Guests    []Guest
}

type Gift struct {
	ID             int
	Amount         int    // whole currency units (e.g. euros), no cents
	Donor          string // who sent the gift
	RegistryItemID *int   // fk to registry_items, nil for generic gifts
	Confirmed      bool
	CreatedAt      Timestamp
}

type RegistryItem struct {
	ID        int
	Name      string
	Price     int // whole currency units (e.g. euros), no cents
	MediaID   int
	SortOrder int
	CreatedAt Timestamp
}

type SoundtrackEvent struct {
	ID        int
	Title     string
	Artist    string
	URL       string
	InviteID  string
	CreatedAt Timestamp
}
