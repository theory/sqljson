package types

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:dupl
func TestDate(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	r := require.New(t)

	for _, tc := range timestampTestCases(t) {
		// Convert to dates.
		exp := tc.exp
		tc.exp = time.Date(
			exp.Year(), exp.Month(), exp.Day(),
			0, 0, 0, 0, time.UTC,
		)
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ts, err := ParseDate(tc.value)
			if !tc.ok {
				a.Nil(ts)
				r.EqualError(err, fmt.Sprintf(
					`type: format is not recognized: %q`, tc.value,
				))
				r.ErrorIs(err, ErrSQLType)
				return
			}

			r.NoError(err)
			a.Equal(&Date{Time: tc.exp}, ts)
			a.Equal(tc.exp.Format(dateFormat), ts.String())

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
	a.Equal(-1, date.Compare(&Date{Time: time.Date(2024, 4, 30, 0, 0, 0, 0, time.UTC)}))
	a.Equal(1, date.Compare(&Date{Time: time.Date(2024, 4, 28, 0, 0, 0, 0, time.UTC)}))
	a.Equal(0, date.Compare(&Date{Time: apr29}))
	a.Equal(0, date.Compare(&Date{Time: time.Date(2024, 4, 29, 0, 0, 0, 0, time.UTC)}))
	a.Equal(1, date.Compare(nil))
}
