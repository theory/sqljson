package types

import (
	"fmt"
	"time"
)

// Timestamp represents the PostgreSQL timestamp without time zone type.
type Timestamp struct {
	// Time is the underlying time.Time value.
	time.Time
}

// NewTimestamp coerces src into a Timestamp.
func NewTimestamp(src time.Time) *Timestamp {
	// Convert result type to timestamp without time zone (use UTC)
	if src.Location() != time.UTC {
		src = time.Date(
			src.Year(), src.Month(), src.Day(),
			src.Hour(), src.Minute(), src.Second(), src.Nanosecond(),
			time.UTC,
		)
	}
	return &Timestamp{src}
}

// GoTime returns the underlying time.Time object.
func (ts *Timestamp) GoTime() time.Time { return ts.Time }

// timestampFormat represents the canonical string format for Timestamp
// values.
const timestampFormat = "2006-01-02T15:04:05.999999999"

// String returns the string representation of ts using the format
// "2006-01-02T15:04:05.999999999".
func (ts *Timestamp) String() string {
	return ts.Time.Format(timestampFormat)
}

// Compare compares the time instant ts with u. If ts is before u, it returns
// -1; if ts is after u, it returns +1; if they're the same, it returns 0.
func (ts *Timestamp) Compare(u time.Time) int {
	return ts.Time.Compare(u)
}

// MarshalJSON implements the json.Marshaler interface. The time is a quoted
// string using the "2006-01-02T15:04:05.999999999" format.
func (ts Timestamp) MarshalJSON() ([]byte, error) {
	const timestampJSONSize = len(timestampFormat) + len(`""`)
	b := make([]byte, 0, timestampJSONSize)
	b = append(b, '"')
	b = ts.Time.AppendFormat(b, timestampFormat)
	b = append(b, '"')
	return b, nil
}

// UnmarshalJSON implements the json.Unmarshaler interface. The time must be a
// quoted string in the "2006-01-02T15:04:05.999999999" format.
func (ts *Timestamp) UnmarshalJSON(data []byte) error {
	tim, err := time.Parse(timestampFormat, string(data[1:len(data)-1]))
	if err != nil {
		return fmt.Errorf(
			"%w: Cannot parse %s as %q",
			ErrSQLType, data, timestampFormat,
		)
	}
	*ts = Timestamp{Time: tim}
	return nil
}
