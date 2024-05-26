package types

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDate(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	r := require.New(t)

	for _, tc := range timestampTestCases(t) {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Don't test Time and TimeTZ
			switch tc.ctor(time.Time{}).(type) {
			case *Time, *TimeTZ:
				return
			}

			// Convert to dates.
			exp := tc.time
			tc.time = time.Date(
				exp.Year(), exp.Month(), exp.Day(),
				0, 0, 0, 0, time.UTC,
			)
			ts := NewDate(tc.time)
			a.Equal(&Date{Time: tc.time}, ts)
			a.Equal(tc.time, ts.GoTime())
			a.Equal(tc.time.Format(dateFormat), ts.String())

			// Check JSON
			json, err := ts.MarshalJSON()
			r.NoError(err)
			a.Equal(fmt.Sprintf("%q", ts.String()), string(json))
			ts2 := new(Date)
			r.NoError(ts2.UnmarshalJSON(json))
			a.Equal(ts, ts2)
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
