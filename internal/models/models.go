package models

import "time"

type Setting struct {
	Key   string
	Value string
}

type WeddingSettings struct {
	Spouse1Name   string
	Spouse2Name   string
	CeremonyDate  string
	ChurchName    string
	ChurchAddress string
	PartyVenue    string
	PartyAddress  string
}

type Guest struct {
	ID           int
	Name         string
	Email        string
	PlusOne      bool
	DietaryNotes string
	Notes        string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
