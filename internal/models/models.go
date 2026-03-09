package models

import "time"

type Setting struct {
	Key   string
	Value string
}

type WeddingSettings struct {
	Spouse1Name       string
	Spouse2Name       string
	CeremonyAddress   string
	CeremonyLocation  string
	CeremonyDatetime  string
	ReceptionAddress  string
	ReceptionLocation string
	BankAccountIBAN   string
	BankAccountHolder string
}

type Guest struct {
	ID                 int
	FirstName          string
	LastName           string
	ConfirmedCeremony  *bool
	ConfirmedReception *bool
	InvitationID       *int
	CreatedAt          time.Time
	UpdatedAt          time.Time
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
	Amount         int    // cents
	Donor          string // who sent the gift
	RegistryItemID *int   // fk to registry_items, nil for generic gifts
	CreatedAt      time.Time
}

type RegistryItem struct {
	ID        int
	Name      string
	Price     int    // whole currency units (e.g. euros), no cents
	Image     string // base64 data URI
	CreatedAt time.Time
}
