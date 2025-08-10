package types

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

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

			// Convert to dates.
			exp := tc.time
			tc.time = time.Date(
				exp.Year(), exp.Month(), exp.Day(),
				0, 0, 0, 0, offsetZero,
			)
			date := NewDate(tc.time)
			a.Equal(&Date{Time: tc.time}, date)
			a.Equal(tc.time, date.GoTime())
			a.Equal(tc.time.Format(dateFormat), date.String())

			// Check JSON
			json, err := date.MarshalJSON()
			r.NoError(err)
			a.JSONEq(fmt.Sprintf("%q", date.String()), string(json))
			ts2 := new(Date)
			r.NoError(ts2.UnmarshalJSON(json))
			a.Equal(date, ts2)

			// Test Conversion functions.
			loc := time.FixedZone("", -3*secondsPerHour)
			ctx := ContextWithTZ(ctx, loc)
			a.Equal(NewTimestamp(date.Time), date.ToTimestamp(ctx))
			a.Equal(
				NewTimestampTZ(
					ctx,
					time.Date(
						date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, loc,
					),
				),
				date.ToTimestampTZ(ctx),
			)
		})
	}
}

func TestDateInvalidJSON(t *testing.T) {
	t.Parallel()
	ts := new(Date)
	err := ts.UnmarshalJSON([]byte(`"i am not a date"`))
	require.Error(t, err)
	require.EqualError(t, err, fmt.Sprintf(
		"type: Cannot parse %q as %q",
		"i am not a date", dateFormat,
	))
	require.ErrorIs(t, err, ErrSQLType)
}

func TestDateCompare(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	apr29 := time.Date(2024, 4, 29, 0, 0, 0, 0, time.UTC)
	date := &Date{Time: apr29}
	a.Equal(-1, date.Compare(time.Date(2024, 4, 30, 0, 0, 0, 0, time.UTC)))
	a.Equal(1, date.Compare(time.Date(2024, 4, 28, 0, 0, 0, 0, time.UTC)))
	a.Equal(0, date.Compare(apr29))
	a.Equal(0, date.Compare(time.Date(2024, 4, 29, 0, 0, 0, 0, time.UTC)))
}
