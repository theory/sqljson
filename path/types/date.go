package types

import (
	"fmt"
	"time"
)

// Date represents the PostgreSQL date type.
type Date struct {
	time.Time
}

// NewDate coerces src into a Date.
func NewDate(src time.Time) *Date {
	// Convert result type to a date
	return &Date{
		time.Date(src.Year(), src.Month(), src.Day(), 0, 0, 0, 0, time.UTC),
	}
}

// GoTime returns the underlying time.Time object.
func (d *Date) GoTime() time.Time { return d.Time }

// dateFormat represents the canonical string format for Date values.
const dateFormat = "2006-01-02"

// String returns the string representation of d.
func (d *Date) String() string {
	return d.Time.Format(dateFormat)
}

// Compare compares the time instant d with u. If d is before u, it returns
// -1; if d is after u, it returns +1; if they're the same, it returns 0.
func (d *Date) Compare(u time.Time) int {
	return d.Time.Compare(u)
}

// MarshalJSON implements the json.Marshaler interface. The time is a quoted
// string in the RFC 3339 format with sub-second precision.
func (d Date) MarshalJSON() ([]byte, error) {
	const dateJSONSize = len(dateFormat) + len(`""`)
	b := make([]byte, 0, dateJSONSize)
	b = append(b, '"')
	b = d.Time.AppendFormat(b, dateFormat)
	b = append(b, '"')
	return b, nil
}

// UnmarshalJSON implements the json.Unmarshaler interface. The time must be a
// quoted string in the RFC 3339 format.
func (d *Date) UnmarshalJSON(data []byte) error {
	tim, err := time.Parse(dateFormat, string(data[1:len(data)-1]))
	if err != nil {
		return fmt.Errorf("%w: Cannot parse %s as %q", ErrSQLType, data, dateFormat)
	}
	*d = Date{Time: tim}
	return nil
}
