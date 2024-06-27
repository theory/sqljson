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
	a := assert.New(t)
	r := require.New(t)
	tz := time.FixedZone("", -9+secondsPerHour)
	ctx := ContextWithTZ(context.Background(), tz)

	for _, tc := range timestampTestCases(t) {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Don't test Time and TimeTZ
			switch tc.ctor(time.Time{}, &time.Location{}).(type) {
			case *Time, *TimeTZ:
				return
			}

			ts := NewTimestampTZ(ctx, tc.time)
			a.Equal(&TimestampTZ{Time: tc.time, tz: tz}, ts)
			a.Equal(tc.time, ts.GoTime())
			loc := tc.time.In(tz)
			a.Equal(loc.Format(timestampTZOutputFormat), ts.String())
			if _, off := loc.Zone(); off%secondsPerHour != 0 {
				a.Equal(loc.Format(timestampTZOutputFormat), ts.ToString(ctx))
			} else {
				a.Equal(loc.Format(timestampTZOffHourOutputFormat), ts.ToString(ctx))
			}

			// Check JSON
			json, err := ts.MarshalJSON()
			r.NoError(err)
			a.Equal(fmt.Sprintf("%q", ts.Time.Format(timestampTZOutputFormat)), string(json))
			ts2 := new(TimestampTZ)
			r.NoError(ts2.UnmarshalJSON(json))
			a.Equal(ts.Time, ts2.In(ts.Location()))

			// Test Conversion methods.
			a.Equal(NewDate(ts.Time.In(tz)), ts.ToDate(ctx))
			a.Equal(NewTime(ts.Time.In(tz)), ts.ToTime(ctx))
			a.Equal(NewTimeTZ(ts.Time.In(tz)), ts.ToTimeTZ(ctx))
			a.Equal(NewTimestamp(ts.Time.In(tz)), ts.ToTimestamp(ctx))
		})
	}
}

func TestTimestampTZToString(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tc := range timestampTestCases(t) {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Don't test Time and TimeTZ
			switch tc.ctor(time.Time{}, &time.Location{}).(type) {
			case *Time, *TimeTZ:
				return
			}

			for _, zc := range []struct {
				name     string
				offset   int
				withMins bool
			}{
				{"utc", 0, false},
				{"plus_4", 4 * secondsPerHour, false},
				{"minus_3", -3 * secondsPerHour, false},
				{"five_30", 5*secondsPerHour + 30*60, true},
				{"neg_2_15", -2*secondsPerHour + 15*60, true},
			} {
				t.Run(zc.name, func(t *testing.T) {
					t.Parallel()
					tz := time.FixedZone("", zc.offset)
					ctx := ContextWithTZ(context.Background(), tz)
					ts := NewTimestampTZ(ctx, tc.time)
					loc := tc.time.In(tz)
					if zc.withMins {
						a.Equal(loc.Format(timestampTZOutputFormat), ts.ToString(ctx))
					} else {
						a.Equal(loc.Format(timestampTZOffHourOutputFormat), ts.ToString(ctx))
					}
				})
			}
		})
	}
}

func TestTimestampTZInvalidJSON(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
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
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ts := new(TimestampTZ)
			err := ts.UnmarshalJSON([]byte(tc.value))
			require.Error(t, err)
			require.EqualError(t, err, fmt.Sprintf(
				"type: Cannot parse %v as %q",
				tc.value, tc.format,
			))
			require.ErrorIs(t, err, ErrSQLType)
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
