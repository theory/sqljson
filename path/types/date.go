package types

import (
	"fmt"
	"time"
)

// Date represents the PostgreSQL date type.
type Date struct {
	time.Time
}

// ParseDate parses src into a Date. Returns an error if the format of src
// cannot be determined and parsed.
func ParseDate(src string) (*Date, error) {
	ts, ok := parseTime(src)
	if !ok {
		return nil, fmt.Errorf(
			`%w: format is not recognized: "%v"`,
			ErrSQLType, src,
		)
	}

	// Convert result type to a date
	ts = time.Date(ts.Year(), ts.Month(), ts.Day(), 0, 0, 0, 0, time.UTC)
	return &Date{ts}, nil
}

const dateFormat = "2006-01-02"

// String returns the string representation of d.
func (d *Date) String() string {
	return d.Time.Format(dateFormat)
}

// Compare compares the time instant d with u. If d is before u, it returns
// -1; if d is after u, it returns +1; if they're the same, it returns 0.
func (d *Date) Compare(u *Date) int {
	if u == nil {
		return d.Time.Compare(time.Time{})
	}
	return d.Time.Compare(u.Time)
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
