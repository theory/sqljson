package types

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimeTZ(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	r := require.New(t)

	for _, tc := range timestampTestCases(t) {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Remove the date from all the test cases.
			exp := time.Date(
				0, 1, 1,
				tc.exp.Hour(), tc.exp.Minute(), tc.exp.Second(),
				tc.exp.Nanosecond(), tc.exp.Location(),
			)

			ts, err := ParseTimeTZ(tc.value)
			if !tc.ok {
				a.Nil(ts)
				r.EqualError(err, fmt.Sprintf(
					`type: format is not recognized: %q`, tc.value,
				))
				r.ErrorIs(err, ErrSQLType)
				return
			}

			r.NoError(err)
			a.Equal(&TimeTZ{Time: exp}, ts)
			a.Equal(exp.Format(timeTZSecondFormat), ts.String())

			// Check JSON
			json, err := ts.MarshalJSON()
			r.NoError(err)
			a.Equal(fmt.Sprintf("%q", ts.String()), string(json))
			ts2 := new(TimeTZ)
			r.NoError(ts2.UnmarshalJSON(json))
			a.Equal(ts, ts2)
		})
	}
}

func TestTimeTZInvalidJSON(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		value  string
		format string
	}{
		{"dumb", `"i am not a timestamp"`, timeTZHourFormat},
		{"pos_secs", `"i am not a timestamp+01:01:01"`, timeTZSecondFormat},
		{"neg_secs", `"i am not a timestamp-01:01:01"`, timeTZSecondFormat},
		{"pos_mins", `"i am not a timestamp+01:01"`, timeTZMinuteFormat},
		{"neg_mins", `"i am not a timestamp-01:01"`, timeTZMinuteFormat},
		{"pos_hours", `"i am not a timestamp+01"`, timeTZHourFormat},
		{"neg_hours", `"i am not a timestamp-01"`, timeTZHourFormat},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ts := new(TimeTZ)
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

func TestTimeTZCompare(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	now := time.Now().Local()
	ts := &TimeTZ{Time: now}
	a.Equal(-1, ts.Compare(&TimeTZ{Time: now.Add(1 * time.Hour)}))
	a.Equal(1, ts.Compare(&TimeTZ{Time: now.Add(-2 * time.Hour)}))
	a.Equal(0, ts.Compare(&TimeTZ{Time: now}))
	a.Equal(0, ts.Compare(&TimeTZ{Time: now.Add(0)}))
	a.Equal(1, ts.Compare(nil))
}
