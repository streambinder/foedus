package models

import "time"

type Setting struct {
	Key   string
	Value string
}

type WeddingSettings struct {
	Spouse1Name      string
	Spouse2Name      string
	CeremonyAddress  string
	CeremonyLocation string
	CeremonyDatetime string
	ReceptionAddress  string
	ReceptionLocation string
}

type Guest struct {
	ID        int
	FirstName string
	LastName  string
	Confirmed bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
