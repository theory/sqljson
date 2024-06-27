package types

import (
	"context"
	"fmt"
	"time"
)

// TimestampTZ represents the PostgreSQL timestamp with time zone type.
type TimestampTZ struct {
	// Time is the underlying time.Time value.
	time.Time
	// tz is the time zone read from the context passed to NewTimestampTZ.
	tz *time.Location
}

// NewTimestampTZ creates a timestamp with time zone with src. The ctx param
// is used solely to determine the time zone used by [String].
func NewTimestampTZ(ctx context.Context, src time.Time) *TimestampTZ {
	return &TimestampTZ{
		tz: TZFromContext(ctx),
		Time: time.Date(
			src.Year(), src.Month(), src.Day(),
			src.Hour(), src.Minute(), src.Second(), src.Nanosecond(),
			offsetLocationFor(src),
		),
	}
}

// GoTime returns the underlying time.Time object.
func (ts *TimestampTZ) GoTime() time.Time { return ts.Time }

const (
	// timestampTZSecondFormat represents the canonical string format for
	// TimestampTZ values, and supports parsing 00:00:00 zones.
	timestampTZSecondFormat = "2006-01-02T15:04:05.999999999Z07:00:00"
	// timestampTZMinuteFormat supports parsing 00:00 zones.
	timestampTZMinuteFormat = "2006-01-02T15:04:05.999999999Z07:00"
	// timestampTZHourFormat supports parsing 00 zones.
	timestampTZHourFormat = "2006-01-02T15:04:05.999999999Z07"
	// timestampTZOutputFormat is the main output format.
	timestampTZOutputFormat = "2006-01-02T15:04:05.999999999-07:00"
	// timestampTZOffHourOutputFormat is the ToString output format when the
	// offset does not include minutes.
	timestampTZOffHourOutputFormat = "2006-01-02T15:04:05.999999999-07"
)

// String returns the string representation of ts in the time zone in the
// Context passed to NewTimestampTZ, using the format
// "2006-01-02T15:04:05.999999999-07:00".
func (ts *TimestampTZ) String() string {
	return ts.Time.In(ts.tz).Format(timestampTZOutputFormat)
}

// ToString returns the jsonpath string() method format of ts in the time zone
// in ctx and using the format "2006-01-02T15:04:05.999999999-07" or
// "2006-01-02T15:04:05.999999999-07:00".
func (ts *TimestampTZ) ToString(ctx context.Context) string {
	loc := ts.Time.In(TZFromContext(ctx))
	if _, off := loc.Zone(); off%secondsPerHour == 0 {
		return loc.Format(timestampTZOffHourOutputFormat)
	}
	return loc.Format(timestampTZOutputFormat)
}

// ToDate converts ts to *Date in the time zone in ctx.
func (ts *TimestampTZ) ToDate(ctx context.Context) *Date {
	return NewDate(ts.Time.In(TZFromContext(ctx)))
}

// ToTime converts ts to *Time in the time zone in ctx.
func (ts *TimestampTZ) ToTime(ctx context.Context) *Time {
	return NewTime(ts.Time.In(TZFromContext(ctx)))
}

// ToTimestamp converts ts to *Timestamp in the time zone in ctx.
func (ts *TimestampTZ) ToTimestamp(ctx context.Context) *Timestamp {
	return NewTimestamp(ts.Time.In(TZFromContext(ctx)))
}

// ToTimeTZ converts ts to TimeTZ in the time zone in ctx.
func (ts *TimestampTZ) ToTimeTZ(ctx context.Context) *TimeTZ {
	return NewTimeTZ(ts.Time.In(TZFromContext(ctx)))
}

// Compare compares the time instant ts with u. If ts is before u, it returns
// -1; if ts is after u, it returns +1; if they're the same, it returns 0.
func (ts *TimestampTZ) Compare(u time.Time) int {
	return ts.Time.Compare(u)
}

// MarshalJSON implements the json.Marshaler interface. The time is a quoted
// string using the "2006-01-02T15:04:05.999999999-07:00" format.
func (ts TimestampTZ) MarshalJSON() ([]byte, error) {
	const timestampJSONSize = len(timestampTZOutputFormat) + len(`""`)
	b := make([]byte, 0, timestampJSONSize)
	b = append(b, '"')
	b = ts.Time.AppendFormat(b, timestampTZOutputFormat)
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
