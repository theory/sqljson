package types

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimestampTZ(t *testing.T) {
	t.Parallel()
	tz := time.FixedZone("", -9+secondsPerHour)
	ctx := ContextWithTZ(context.Background(), tz)

	for _, tc := range timestampTestCases(t) {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)
			r := require.New(t)

			// Don't test Time and TimeTZ
			switch tc.ctor(time.Time{}, &time.Location{}).(type) {
			case *Time, *TimeTZ:
				return
			}

			ts := NewTimestampTZ(ctx, tc.time)
			a.Equal(&TimestampTZ{Time: tc.time, tz: tz}, ts)
			a.Equal(tc.time, ts.GoTime())
			a.Equal(tc.time.Format(timestampTZOutputFormat), ts.String())

			// Check JSON
			json, err := ts.MarshalJSON()
			r.NoError(err)
			a.JSONEq(fmt.Sprintf("%q", ts.Format(timestampTZOutputFormat)), string(json))
			ts2 := new(TimestampTZ)
			r.NoError(ts2.UnmarshalJSON(json))
			a.Equal(ts.Time, ts2.In(ts.Location()))

			// Test Conversion methods.
			a.Equal(NewDate(ts.In(tz)), ts.ToDate(ctx))
			a.Equal(NewTime(ts.In(tz)), ts.ToTime(ctx))
			a.Equal(NewTimeTZ(ts.In(tz)), ts.ToTimeTZ(ctx))
			a.Equal(NewTimestamp(ts.In(tz)), ts.ToTimestamp(ctx))
		})
	}
}

func TestTimestampTZInvalidJSON(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		test   string
		value  string
		format string
	}{
		{"dumb", `"i am not a timestamp"`, timestampTZHourFormat},
		{"pos_secs", `"i am not a timestamp+01:01:01"`, timestampTZSecondFormat},
		{"neg_secs", `"i am not a timestamp-01:01:01"`, timestampTZSecondFormat},
		{"pos_mins", `"i am not a timestamp+01:01"`, timestampTZMinuteFormat},
		{"neg_mins", `"i am not a timestamp-01:01"`, timestampTZMinuteFormat},
		{"pos_hours", `"i am not a timestamp+01"`, timestampTZHourFormat},
		{"neg_hours", `"i am not a timestamp-01"`, timestampTZHourFormat},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			r := require.New(t)

			ts := new(TimestampTZ)
			err := ts.UnmarshalJSON([]byte(tc.value))
			r.Error(err)
			r.EqualError(err, fmt.Sprintf(
				"type: Cannot parse %v as %q",
				tc.value, tc.format,
			))
			r.ErrorIs(err, ErrSQLType)
		})
	}
}

func TestTimestampTZCompare(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	now := time.Now()
	ts := &TimestampTZ{Time: now}
	a.Equal(-1, ts.Compare(now.Add(1*time.Hour)))
	a.Equal(1, ts.Compare(now.Add(-2*time.Hour)))
	a.Equal(0, ts.Compare(now))
	a.Equal(0, ts.Compare(now.Add(0)))
}
