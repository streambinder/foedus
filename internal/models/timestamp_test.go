package models

import (
	"testing"
	"time"
)

// Scan is the only thing standing between an INTEGER column and a time.Time
// field, and a wrong arm here fails at runtime rather than at compile time.
func TestTimestampScan(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		src     any
		want    time.Time
		wantErr bool
	}{
		{"epoch seconds", int64(1776697048), time.Date(2026, 4, 20, 14, 57, 28, 0, time.UTC), false},
		{"null", nil, time.Time{}, false},
		{"zero epoch", int64(0), time.Unix(0, 0).UTC(), false},
		// legacy DATETIME columns are gone, so text is a schema mismatch now
		{"text rejected", "2026-04-20 14:57:28", time.Time{}, true},
		{"time.Time rejected", time.Now(), time.Time{}, true},
		{"float rejected", 1.5, time.Time{}, true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			var timestamp Timestamp
			err := timestamp.Scan(testCase.src)
			if (err != nil) != testCase.wantErr {
				t.Fatalf("Scan(%#v) error = %v, wantErr %v", testCase.src, err, testCase.wantErr)
			}
			if !testCase.wantErr && !timestamp.Equal(testCase.want) {
				t.Errorf("Scan(%#v) = %v, want %v", testCase.src, timestamp.Time, testCase.want)
			}
		})
	}
}

func TestTimestampValue(t *testing.T) {
	value, err := Timestamp{time.Date(2026, 4, 20, 14, 57, 28, 0, time.UTC)}.Value()
	if err != nil || value != int64(1776697048) {
		t.Errorf("Value() = %v, %v; want 1776697048, nil", value, err)
	}
	// the zero time must round-trip as NULL, not as year 1
	if value, err = (Timestamp{}).Value(); err != nil || value != nil {
		t.Errorf("zero Value() = %v, %v; want nil, nil", value, err)
	}
}
