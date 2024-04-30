package types

import (
	"fmt"
	"time"
)

// TimeTZ represents the PostgreSQL TimeTZ type.
type TimeTZ struct {
	// Time is the underlying time.Time value.
	time.Time
}

// ParseTimeTZ parses src into a TimeTZ without time zone. Returns an
// error if the format of src cannot be determined and parsed.
func ParseTimeTZ(src string) (*TimeTZ, error) {
	ts, ok := parseTime(src)
	if !ok {
		return nil, fmt.Errorf(
			`%w: format is not recognized: "%v"`,
			ErrSQLType, src,
		)
	}

	// Convert result type to TimeTZ wit time zone.
	ts = time.Date(
		0, 1, 1,
		ts.Hour(), ts.Minute(), ts.Second(), ts.Nanosecond(),
		ts.Location(),
	)
	return &TimeTZ{ts}, nil
}

const (
	// timeTZSecondFormat represents the canonical string format for
	// TimeTZ values, and supports parsing 00:00:00 zones.
	timeTZSecondFormat = "15:04:05.999999999Z07:00:00"
	// timeTZMinuteFormat supports parsing 00:00 zones.
	timeTZMinuteFormat = "15:04:05.999999999Z07:00"
	// timeTZHourFormat supports parsing 00 zones.
	timeTZHourFormat = "15:04:05.999999999Z07"
)

// String returns the string representation of ts using the format
// "15:04:05.999999999Z07:00:00".
func (ts *TimeTZ) String() string {
	return ts.Time.Format(timeTZSecondFormat)
}

// Compare compares the time instant ts with u. If ts is before u, it returns
// -1; if ts is after u, it returns +1; if they're the same, it returns 0.
func (ts *TimeTZ) Compare(u *TimeTZ) int {
	if u == nil {
		return ts.Time.Compare(time.Time{})
	}
	return ts.Time.Compare(u.Time)
}

// MarshalJSON implements the json.Marshaler interface. The time is a quoted
// string using the "15:04:05.999999999Z07:00:00" format.
func (ts TimeTZ) MarshalJSON() ([]byte, error) {
	const timeJSONSize = len(timeTZSecondFormat) + len(`""`)
	b := make([]byte, 0, timeJSONSize)
	b = append(b, '"')
	b = ts.Time.AppendFormat(b, timeTZSecondFormat)
	b = append(b, '"')
	return b, nil
}

// UnmarshalJSON implements the json.Unmarshaler interface. The time must be a
// quoted string in one of the following formats:
//   - 15:04:05.999999999Z07:00:00
//   - 15:04:05.999999999Z07:00
//   - 15:04:05.999999999Z07
func (ts *TimeTZ) UnmarshalJSON(data []byte) error {
	str := data[1 : len(data)-1] // Unquote

	// Figure out which TZ format we need.
	var format string
	const (
		secPlace = 9
		minPlace = 6
	)
	size := len(str)
	switch {
	case str[size-secPlace] == '-' || str[size-secPlace] == '+':
		format = timeTZSecondFormat
	case str[size-minPlace] == '-' || str[size-minPlace] == '+':
		format = timeTZMinuteFormat
	default:
		format = timeTZHourFormat
	}

	tim, err := time.Parse(format, string(str))
	if err != nil {
		return fmt.Errorf("%w: Cannot parse %s as %q", ErrSQLType, data, format)
	}
	*ts = TimeTZ{Time: tim}
	return nil
}
