package types

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTime(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	r := require.New(t)
	loc := time.FixedZone("", 10*secondsPerHour+secondsPerHour/2)
	ctx := ContextWithTZ(context.Background(), loc)

	for _, tc := range timestampTestCases(t) {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Only test Time and TimeTZ
			switch tc.ctor(time.Time{}, &time.Location{}).(type) {
			case *Timestamp, *TimestampTZ, *Date:
				return
			}

			// Remove the time zone and date from all the test cases.
			exp := time.Date(
				0, 1, 1,
				tc.time.Hour(), tc.time.Minute(), tc.time.Second(),
				tc.time.Nanosecond(), offsetZero,
			)

			ts := NewTime(tc.time)
			a.Equal(&Time{Time: exp}, ts)
			a.Equal(exp, ts.GoTime())
			a.Equal(exp.Format(timeFormat), ts.String())

			// Check JSON
			json, err := ts.MarshalJSON()
			r.NoError(err)
			a.JSONEq(fmt.Sprintf("%q", ts.String()), string(json))
			ts2 := new(Time)
			r.NoError(ts2.UnmarshalJSON(json))
			a.Equal(ts, ts2)

			// Test ToTimeTZ.
			a.Equal(
				NewTimeTZ(time.Date(
					0, 1, 1,
					ts.Hour(), ts.Minute(), ts.Second(), ts.Nanosecond(),
					loc,
				)),
				ts.ToTimeTZ(ctx),
			)
		})
	}
}

func TestTimeInvalidJSON(t *testing.T) {
	t.Parallel()
	ts := new(Time)
	err := ts.UnmarshalJSON([]byte(`"i am not a time"`))
	require.Error(t, err)
	require.EqualError(t, err, fmt.Sprintf(
		"type: Cannot parse %q as %q",
		"i am not a time", timeFormat,
	))
	require.ErrorIs(t, err, ErrSQLType)
}

func TestTimeCompare(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	now := time.Now().UTC()
	ts := &Time{Time: now}
	a.Equal(-1, ts.Compare(now.Add(1*time.Hour)))
	a.Equal(1, ts.Compare(now.Add(-2*time.Hour)))
	a.Equal(0, ts.Compare(now))
	a.Equal(0, ts.Compare(now.Add(0)))
}
