package types

import (
	"context"
	"fmt"
	"time"
)

// TimeTZ represents the PostgreSQL time with time zone type.
type TimeTZ struct {
	// Time is the underlying time.Time value.
	time.Time
}

// NewTimeTZ coerces src into a TimeTZ.
func NewTimeTZ(src time.Time) *TimeTZ {
	// Preserve the offset.
	return &TimeTZ{time.Date(
		0, 1, 1,
		src.Hour(), src.Minute(), src.Second(), src.Nanosecond(),
		offsetLocationFor(src),
	)}
}

// GoTime returns the underlying time.Time object.
func (t *TimeTZ) GoTime() time.Time { return t.Time }

const (
	// timeTZSecondFormat represents the canonical string format for
	// TimeTZ values, and supports parsing 00:00:00 zones.
	timeTZSecondFormat = "15:04:05.999999999Z07:00:00"
	// timeTZMinuteFormat supports parsing 00:00 zones.
	timeTZMinuteFormat = "15:04:05.999999999Z07:00"
	// timeTZHourFormat supports parsing 00 zones.
	timeTZHourFormat = "15:04:05.999999999Z07"
	// timeTZOutputFormat outputs 00:00 zones.
	timeTZOutputFormat = "15:04:05.999999999-07:00"
	// timeTZOffHourOutputFormat is the ToString output format when the offset
	// does not include minutes.
	timeTZOffHourOutputFormat = "15:04:05.999999999-07"
)

// String returns the string representation of ts using the format
// "15:04:05.999999999-07:00".
func (t *TimeTZ) String() string {
	return t.Time.Format(timeTZOutputFormat)
}

// ToString returns the jsonpath string() method format of ts in the local
// time zone using the format "2006-01-02T15:04:05.999999999-07" or
// "2006-01-02T15:04:05.999999999-07:00".
func (t *TimeTZ) ToString(context.Context) string {
	if _, off := t.Time.Zone(); off%secondsPerHour == 0 {
		return t.Time.Format(timeTZOffHourOutputFormat)
	}
	return t.Time.Format(timeTZOutputFormat)
}

// ToTime converts t to *Time.
func (t *TimeTZ) ToTime(context.Context) *Time {
	return NewTime(t.Time)
}

// Compare compares the time instant t with u. If d is before u, it returns
// -1; if t is after u, it returns +1; if they're the same, it returns 0. Note
// that the TZ offset contributes to this comparison; values with different
// offsets are never considered to be the same.
func (t *TimeTZ) Compare(u time.Time) int {
	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/backend/utils/adt/date.c#L2442-L2467

	// Primary sort is by true (GMT-equivalent) time.
	cmp := t.Time.UTC().Compare(u.UTC())
	if cmp != 0 {
		return cmp
	}

	// If same GMT time, sort by timezone; we only want to say that two
	// timetz's are equal if both the time and zone parts are equal.
	_, off1 := t.Time.Zone()
	_, off2 := u.Zone()
	if off1 > off2 {
		return -1
	}
	if off1 < off2 {
		return 1
	}
	return 0
}

// MarshalJSON implements the json.Marshaler interface. The time is a quoted
// string using the "15:04:05.999999999-07:00" format.
func (t TimeTZ) MarshalJSON() ([]byte, error) {
	const timeJSONSize = len(timeTZOutputFormat) + len(`""`)
	b := make([]byte, 0, timeJSONSize)
	b = append(b, '"')
	b = t.Time.AppendFormat(b, timeTZOutputFormat)
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
