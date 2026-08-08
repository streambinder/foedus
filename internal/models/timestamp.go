package models

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// Timestamp is a time.Time persisted as unix epoch seconds in an INTEGER column.
//
// STRICT tables accept only INT/INTEGER/REAL/TEXT/BLOB/ANY, so the DATETIME
// declared type is gone — and that decltype was the only thing making
// modernc.org/sqlite hand back a time.Time instead of a raw scalar. The
// conversion therefore lives here rather than in the driver.
//
// Embedding (rather than `type Timestamp time.Time`) promotes the whole
// time.Time method set, so .Format/.Before/.IsZero keep working on the field.
type Timestamp struct {
	time.Time
}

// Scan accepts int64 epoch seconds, which is all a STRICT INTEGER column can
// hold. Anything else means the column is not what this type expects, so it is
// an error rather than a best-effort conversion.
func (t *Timestamp) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		t.Time = time.Time{}
	case int64:
		t.Time = time.Unix(v, 0).UTC()
	default:
		return fmt.Errorf("models: cannot scan %T into Timestamp", src)
	}
	return nil
}

// Value writes epoch seconds, and NULL for the zero time so an unset optional
// timestamp round-trips instead of landing as year 1.
func (t Timestamp) Value() (driver.Value, error) {
	if t.IsZero() {
		return nil, nil
	}
	return t.Unix(), nil
}
