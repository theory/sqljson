package types

import (
	"fmt"
	"time"
)

// TimestampTZ represents the PostgreSQL timestamp with time zone type.
type TimestampTZ struct {
	// Time is the underlying time.Time value.
	time.Time
}

// ParseTimestampTZ parses src into a timestamp wit time zone. Returns an
// error if the format of src cannot be determined and parsed.
func ParseTimestampTZ(src string) (*TimestampTZ, error) {
	ts, ok := parseTime(src)
	if !ok {
		return nil, fmt.Errorf(
			`%w: format is not recognized: "%v"`,
			ErrSQLType, src,
		)
	}
	return &TimestampTZ{ts}, nil
}

const (
	// timestampTZSecondFormat represents the canonical string format for
	// TimestampTZ values, and supports parsing 00:00:00 zones.
	timestampTZSecondFormat = "2006-01-02T15:04:05.999999999Z07:00:00"
	// timestampTZMinuteFormat supports parsing 00:00 zones.
	timestampTZMinuteFormat = "2006-01-02T15:04:05.999999999Z07:00"
	// timestampTZHourFormat supports parsing 00 zones.
	timestampTZHourFormat = "2006-01-02T15:04:05.999999999Z07"
)

// String returns the string representation of ts using the format
// "2006-01-02T15:04:05.999999999Z07:00:00".
func (ts TimestampTZ) String() string {
	return ts.Time.Format(timestampTZSecondFormat)
}

// Compare compares the time instant ts with u. If ts is before u, it returns
// -1; if ts is after u, it returns +1; if they're the same, it returns 0.
func (ts TimestampTZ) Compare(u *Timestamp) int {
	if u == nil {
		return ts.Time.Compare(time.Time{})
	}
	return ts.Time.Compare(u.Time)
}

// MarshalJSON implements the json.Marshaler interface. The time is a quoted
// string using the "2006-01-02T15:04:05.999999999Z07:00:00" format.
func (ts TimestampTZ) MarshalJSON() ([]byte, error) {
	const timestampJSONSize = len(timestampTZSecondFormat) + len(`""`)
	b := make([]byte, 0, timestampJSONSize)
	b = append(b, '"')
	b = ts.Time.AppendFormat(b, timestampTZSecondFormat)
	b = append(b, '"')
	return b, nil
}

// UnmarshalJSON implements the json.Unmarshaler interface. The time must be a
// quoted string in one of the following formats:
//   - 2006-01-02T15:04:05.999999999Z07:00:00
//   - 2006-01-02T15:04:05.999999999Z07:00
//   - 2006-01-02T15:04:05.999999999Z07
func (ts *TimestampTZ) UnmarshalJSON(data []byte) error {
	str := data[1 : len(data)-1] // Unquote

	// Figure out which TZ format we need.
	var format string
	const (
		secPlace = 9
		minPlace = 6
	)
	size := len(str)
	switch {
	case size >= 9 && (str[size-secPlace] == '-' || str[size-secPlace] == '+'):
		format = timestampTZSecondFormat
	case size >= 6 && (str[size-minPlace] == '-' || str[size-minPlace] == '+'):
		format = timestampTZMinuteFormat
	default:
		format = timestampTZHourFormat
	}

	tim, err := time.Parse(format, string(str))
	if err != nil {
		return fmt.Errorf("%w: Cannot parse %s as %q", ErrSQLType, data, format)
	}
	*ts = TimestampTZ{Time: tim}
	return nil
}
