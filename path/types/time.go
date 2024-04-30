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

// ParseTime parses src into a Time without time zone. Returns an
// error if the format of src cannot be determined and parsed.
func ParseTime(src string) (*Time, error) {
	ts, ok := parseTime(src)
	if !ok {
		return nil, fmt.Errorf(
			`%w: format is not recognized: "%v"`,
			ErrSQLType, src,
		)
	}

	// Convert result type to Time without time zone (use UTC)
	ts = time.Date(
		0, 1, 1,
		ts.Hour(), ts.Minute(), ts.Second(), ts.Nanosecond(),
		time.UTC,
	)
	return &Time{ts}, nil
}

// timeFormat represents the canonical string format for Time
// values.
const timeFormat = "15:04:05.999999999"

// String returns the string representation of ts using the format
// "15:04:05.999999999".
func (ts *Time) String() string {
	return ts.Time.Format(timeFormat)
}

// Compare compares the time instant ts with u. If ts is before u, it returns
// -1; if ts is after u, it returns +1; if they're the same, it returns 0.
func (ts *Time) Compare(u *Time) int {
	if u == nil {
		return ts.Time.Compare(time.Time{})
	}
	return ts.Time.Compare(u.Time)
}

// MarshalJSON implements the json.Marshaler interface. The time is a quoted
// string using the "15:04:05.999999999" format.
func (ts Time) MarshalJSON() ([]byte, error) {
	const timeJSONSize = len(timeFormat) + len(`""`)
	b := make([]byte, 0, timeJSONSize)
	b = append(b, '"')
	b = ts.Time.AppendFormat(b, timeFormat)
	b = append(b, '"')
	return b, nil
}

// UnmarshalJSON implements the json.Unmarshaler interface. The time must be a
// quoted string in the "15:04:05.999999999" format.
func (ts *Time) UnmarshalJSON(data []byte) error {
	tim, err := time.Parse(timeFormat, string(data[1:len(data)-1]))
	if err != nil {
		return fmt.Errorf(
			"%w: Cannot parse %s as %q",
			ErrSQLType, data, timeFormat,
		)
	}
	*ts = Time{Time: tim}
	return nil
}
