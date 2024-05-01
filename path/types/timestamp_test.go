package types

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimestamp(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	r := require.New(t)

	for _, tc := range timestampTestCases(t) {
		if !tc.ok {
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Remove the time zone from all the test cases (by making it UTC).
			exp := time.Date(
				tc.exp.Year(), tc.exp.Month(), tc.exp.Day(),
				tc.exp.Hour(), tc.exp.Minute(), tc.exp.Second(),
				tc.exp.Nanosecond(), time.UTC,
			)

			ts := NewTimestamp(tc.exp)
			a.Equal(&Timestamp{Time: exp}, ts)
			a.Equal(exp.Format(timestampFormat), ts.String())

			// Check JSON
			json, err := ts.MarshalJSON()
			r.NoError(err)
			a.Equal(fmt.Sprintf("%q", ts.String()), string(json))
			ts2 := new(Timestamp)
			r.NoError(ts2.UnmarshalJSON(json))
			a.Equal(ts, ts2)
		})
	}
}

func TestTimestampInvalidJSON(t *testing.T) {
	t.Parallel()
	ts := new(Timestamp)
	err := ts.UnmarshalJSON([]byte(`"i am not a timestamp"`))
	require.Error(t, err)
	require.EqualError(t, err, fmt.Sprintf(
		"type: Cannot parse %q as %q",
		"i am not a timestamp", timestampFormat,
	))
	require.ErrorIs(t, err, ErrSQLType)
}

func TestTimestampCompare(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	now := time.Now().UTC()
	ts := &Timestamp{Time: now}
	a.Equal(-1, ts.Compare(now.Add(1*time.Hour)))
	a.Equal(1, ts.Compare(now.Add(-2*time.Hour)))
	a.Equal(0, ts.Compare(now))
	a.Equal(0, ts.Compare(now.Add(0)))
}
