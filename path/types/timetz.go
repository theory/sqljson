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

// NewTimeTZ coerces src into a TimeTZ without time zone.
func NewTimeTZ(src time.Time) *TimeTZ {
	// Convert result type to TimeTZ wit time zone.
	return &TimeTZ{time.Date(
		0, 1, 1,
		src.Hour(), src.Minute(), src.Second(), src.Nanosecond(),
		src.Location(),
	)}
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
func (t *TimeTZ) String() string {
	return t.Time.Format(timeTZSecondFormat)
}

// Compare compares the time instant t with u. If d is before u, it returns
// -1; if t is after u, it returns +1; if they're the same, it returns 0.
func (t *TimeTZ) Compare(u time.Time) int {
	return t.Time.Compare(u)
}

// MarshalJSON implements the json.Marshaler interface. The time is a quoted
// string using the "15:04:05.999999999Z07:00:00" format.
func (t TimeTZ) MarshalJSON() ([]byte, error) {
	const timeJSONSize = len(timeTZSecondFormat) + len(`""`)
	b := make([]byte, 0, timeJSONSize)
	b = append(b, '"')
	b = t.Time.AppendFormat(b, timeTZSecondFormat)
	b = append(b, '"')
	return b, nil
}

// UnmarshalJSON implements the json.Unmarshaler interface. The time must be a
// quoted string in one of the following formats:
//   - 15:04:05.999999999Z07:00:00
//   - 15:04:05.999999999Z07:00
//   - 15:04:05.999999999Z07
func (t *TimeTZ) UnmarshalJSON(data []byte) error {
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
	*t = TimeTZ{Time: tim}
	return nil
}
