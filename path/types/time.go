package types

import (
	"context"
	"fmt"
	"time"
)

// Time represents the PostgreSQL time without time zone type.
type Time struct {
	// Time is the underlying time.Time value.
	time.Time
}

// NewTime coerces src into a Time.
func NewTime(src time.Time) *Time {
	// Convert result type to Time without time zone (use offset 0)
	return &Time{time.Date(
		0, 1, 1,
		src.Hour(), src.Minute(), src.Second(), src.Nanosecond(),
		offsetZero,
	)}
}

// GoTime returns the underlying time.Time object.
func (t *Time) GoTime() time.Time { return t.Time }

// timeFormat represents the canonical string format for Time
// values.
const timeFormat = "15:04:05.999999999"

// String returns the string representation of ts using the format
// "15:04:05.999999999".
func (t *Time) String() string {
	return t.Time.Format(timeFormat)
}

// ToTimeTZ converts t to *TimeTZ in the time zone in ctx. It works relative
// the current date.
func (t *Time) ToTimeTZ(ctx context.Context) *TimeTZ {
	now := time.Now()
	return NewTimeTZ(time.Date(
		now.Year(), now.Month(), now.Day(),
		t.Hour(), t.Minute(), t.Second(), t.Nanosecond(),
		TZFromContext(ctx),
	))
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
	*t = *NewTime(tim)
	return nil
}
