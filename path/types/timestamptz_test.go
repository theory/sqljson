package types

import (
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

	for _, tc := range timestampTestCases(t) {
		if !tc.ok {
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ts := NewTimestampTZ(tc.exp)
			a.Equal(&TimestampTZ{Time: tc.exp}, ts)
			a.Equal(tc.exp.Format(timestampTZSecondFormat), ts.String())

			// Check JSON
			json, err := ts.MarshalJSON()
			r.NoError(err)
			a.Equal(fmt.Sprintf("%q", ts.String()), string(json))
			ts2 := new(TimestampTZ)
			r.NoError(ts2.UnmarshalJSON(json))
			a.Equal(ts, ts2)
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
