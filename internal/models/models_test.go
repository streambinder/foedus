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
