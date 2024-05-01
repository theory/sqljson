package types

import (
	"fmt"
	"time"
)

// Time represents the PostgreSQL Time type.
type Time struct {
	// Time is the underlying time.Time value.
	time.Time
}

// NewTime coerces src into a Time without time zone.
func NewTime(src time.Time) *Time {
	// Convert result type to Time without time zone (use UTC)
	return &Time{time.Date(
		0, 1, 1,
		src.Hour(), src.Minute(), src.Second(), src.Nanosecond(),
		time.UTC,
	)}
}

// timeFormat represents the canonical string format for Time
// values.
const timeFormat = "15:04:05.999999999"

// String returns the string representation of ts using the format
// "15:04:05.999999999".
func (t *Time) String() string {
	return t.Time.Format(timeFormat)
}

// Compare compares the time instant t with u. If d is before u, it returns
// -1; if t is after u, it returns +1; if they're the same, it returns 0.
func (t *Time) Compare(u time.Time) int {
	return t.Time.Compare(u)
}

// MarshalJSON implements the json.Marshaler interface. The time is a quoted
// string using the "15:04:05.999999999" format.
func (t *Time) MarshalJSON() ([]byte, error) {
	const timeJSONSize = len(timeFormat) + len(`""`)
	b := make([]byte, 0, timeJSONSize)
	b = append(b, '"')
	b = t.Time.AppendFormat(b, timeFormat)
	b = append(b, '"')
	return b, nil
}

// UnmarshalJSON implements the json.Unmarshaler interface. The time must be a
// quoted string in the "15:04:05.999999999" format.
func (t *Time) UnmarshalJSON(data []byte) error {
	tim, err := time.Parse(timeFormat, string(data[1:len(data)-1]))
	if err != nil {
		return fmt.Errorf(
			"%w: Cannot parse %s as %q",
			ErrSQLType, data, timeFormat,
		)
	}
	*t = Time{Time: tim}
	return nil
}
