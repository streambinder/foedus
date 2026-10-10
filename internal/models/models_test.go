package models

import (
	"testing"
	"time"
)

// the read-only flip is date arithmetic against a dashboard string: a wrong
// layout or boundary here would either freeze the public site too early or
// leave contributions open forever, so the boundaries are pinned explicitly.
func TestEventPassedAt(t *testing.T) {
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	for _, testCase := range []struct {
		name     string
		datetime string
		want     bool
	}{
		{"empty datetime never passes", "", false},
		{"garbage datetime never passes", "not-a-date", false},
		{"future wedding", "2026-12-01T10:00", false},
		{"wedding today", "2026-09-22T12:00", false},
		{"wedding yesterday", "2026-09-21T12:00", false},
		{"exactly one month ago is not passed", "2026-08-22T12:00", false},
		{"one month and a day ago is passed", "2026-08-21T12:00", true},
		{"wedding two months ago", "2026-07-22T12:00", true},
		{"wedding last year", "2025-09-22T12:00", true},
		{"date-only layout past", "2026-01-15", true},
		{"date-only layout future", "2026-12-15", false},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if got := eventPassedAt(testCase.datetime, now); got != testCase.want {
				t.Fatalf("eventPassedAt(%q) = %v, want %v", testCase.datetime, got, testCase.want)
			}
		})
	}
}

func TestIsConfigured(t *testing.T) {
	for _, tc := range []struct {
		name     string
		settings Settings
		want     bool
	}{
		{"both names set", Settings{GroomName: "Davide", BrideName: "Agnese"}, true},
		{"missing groom", Settings{BrideName: "Agnese"}, false},
		{"missing bride", Settings{GroomName: "Davide"}, false},
		{"empty settings", Settings{}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.settings.IsConfigured(); got != tc.want {
				t.Errorf("IsConfigured() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestEventPassed(t *testing.T) {
	if (Settings{}).EventPassed() {
		t.Error("empty ceremony datetime must never report the event as passed")
	}
	if !(Settings{CeremonyDatetime: "2000-01-01T10:00"}).EventPassed() {
		t.Error("a ceremony in the year 2000 must report the event as passed")
	}
	if (Settings{CeremonyDatetime: "2999-01-01T10:00"}).EventPassed() {
		t.Error("a ceremony in the year 2999 must not report the event as passed")
	}
}

func TestIsHexColor(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want bool
	}{
		{"#6f7665", true},
		{"#FFFFFF", true},
		{"#123abc", true},
		{"", false},
		{"6f7665", false},
		{"#fff", false},
		{"#gggggg", false},
		{"#1234567", false},
		{"red", false},
		{"#6f7665;", false},
		{"\"#6f7665\"", false},
	} {
		if got := IsHexColor(tc.in); got != tc.want {
			t.Errorf("IsHexColor(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}
