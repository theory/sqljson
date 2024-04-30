package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func pos(h, m, s int) *time.Location {
	return time.FixedZone("", h*60*60+m*60+s)
}

func neg(h, m, s int) *time.Location {
	return time.FixedZone("", -(h*60*60 + m*60 + s))
}

type TSTestCase struct {
	name  string
	value string
	exp   time.Time
	ok    bool
}

func timestampTestCases(t *testing.T) []TSTestCase {
	t.Helper()
	return []TSTestCase{
		// Date
		{
			name:  "date",
			value: "2024-04-29",
			exp:   time.Date(2024, 4, 29, 0, 0, 0, 0, time.UTC),
			ok:    true,
		},
		// time with time zone
		{
			name:  "time_tz_hms",
			value: "14:15:31+01:22:30",
			exp:   time.Date(0, 1, 1, 14, 15, 31, 0, pos(1, 22, 30)),
			ok:    true,
		},
		{
			name:  "time_tz_sub_hms",
			value: "14:15:31.785996+01:22:30",
			exp:   time.Date(0, 1, 1, 14, 15, 31, 785996000, pos(1, 22, 30)),
			ok:    true,
		},
		{
			name:  "time_tz_sub_neg_hms",
			value: "14:15:31.785996-01:22:30",
			exp:   time.Date(0, 1, 1, 14, 15, 31, 785996000, neg(1, 22, 30)),
			ok:    true,
		},
		{
			name:  "time_tz_neg_hms",
			value: "14:15:31-01:22:30",
			exp:   time.Date(0, 1, 1, 14, 15, 31, 0, neg(1, 22, 30)),
			ok:    true,
		},
		{
			name:  "time_tz_sub_hm",
			value: "14:15:31.785996+03:14",
			exp:   time.Date(0, 1, 1, 14, 15, 31, 785996000, pos(3, 14, 0)),
			ok:    true,
		},
		{
			name:  "time_tz_hm",
			value: "14:15:31+03:14",
			exp:   time.Date(0, 1, 1, 14, 15, 31, 0, pos(3, 14, 0)),
			ok:    true,
		},
		{
			name:  "time_tz_sub_neg_hm",
			value: "14:15:31.785996-03:14",
			exp:   time.Date(0, 1, 1, 14, 15, 31, 785996000, neg(3, 14, 0)),
			ok:    true,
		},
		{
			name:  "time_tz_neg_hm",
			value: "14:15:31-03:14",
			exp:   time.Date(0, 1, 1, 14, 15, 31, 0, neg(3, 14, 0)),
			ok:    true,
		},
		{
			name:  "time_tz_sub_h",
			value: "14:15:31.785996+01",
			exp:   time.Date(0, 1, 1, 14, 15, 31, 785996000, pos(1, 0, 0)),
			ok:    true,
		},
		{
			name:  "time_tz_h",
			value: "14:15:31+01",
			exp:   time.Date(0, 1, 1, 14, 15, 31, 0, pos(1, 0, 0)),
			ok:    true,
		},
		{
			name:  "time_tz_sub_neg_h",
			value: "14:15:31.785996-11",
			exp:   time.Date(0, 1, 1, 14, 15, 31, 785996000, neg(11, 0, 0)),
			ok:    true,
		},
		{
			name:  "time_tz_neg_h",
			value: "14:15:31-11",
			exp:   time.Date(0, 1, 1, 14, 15, 31, 0, neg(11, 0, 0)),
			ok:    true,
		},
		{
			name:  "time_tz_sub_z",
			value: "14:15:31.785996Z",
			exp:   time.Date(0, 1, 1, 14, 15, 31, 785996000, time.UTC),
			ok:    true,
		},
		{
			name:  "time_tz_z",
			value: "14:15:31",
			exp:   time.Date(0, 1, 1, 14, 15, 31, 0, time.UTC),
			ok:    true,
		},
		// time without time zone
		{
			name:  "time_sub",
			value: "14:15:31.785996",
			exp:   time.Date(0, 1, 1, 14, 15, 31, 785996000, time.UTC),
			ok:    true,
		},
		{
			name:  "time_no_sub",
			value: "14:15:31",
			exp:   time.Date(0, 1, 1, 14, 15, 31, 0, time.UTC),
			ok:    true,
		},
		// timestamp "T" with time zone
		{
			name:  "timestamp_t_tz_sub_hms",
			value: "2024-04-29T15:11:38.06318+02:30:04",
			exp:   time.Date(2024, 4, 29, 15, 11, 38, 63180000, pos(2, 30, 4)),
			ok:    true,
		},
		{
			name:  "timestamp_t_tz_hms",
			value: "2024-04-29T15:11:38+02:30:04",
			exp:   time.Date(2024, 4, 29, 15, 11, 38, 0, pos(2, 30, 4)),
			ok:    true,
		},
		{
			name:  "timestamp_t_tz_sub_neg_hms",
			value: "2024-04-29T15:11:38.06318-02:30:04",
			exp:   time.Date(2024, 4, 29, 15, 11, 38, 63180000, neg(2, 30, 4)),
			ok:    true,
		},
		{
			name:  "timestamp_t_tz_neg_hms",
			value: "2024-04-29T15:11:38-02:30:04",
			exp:   time.Date(2024, 4, 29, 15, 11, 38, 0, neg(2, 30, 4)),
			ok:    true,
		},
		{
			name:  "timestamp_t_tz_sub_z",
			value: "2024-04-29T15:11:38.06318Z",
			exp:   time.Date(2024, 4, 29, 15, 11, 38, 63180000, time.UTC),
			ok:    true,
		},
		{
			name:  "timestamp_t_tz_z",
			value: "2024-04-29T15:11:38Z",
			exp:   time.Date(2024, 4, 29, 15, 11, 38, 0, time.UTC),
			ok:    true,
		},
		// timestamp "T" without time zone
		{
			name:  "timestamp_t_sub_hms",
			value: "2024-04-29T15:11:38.06318",
			exp:   time.Date(2024, 4, 29, 15, 11, 38, 63180000, time.UTC),
			ok:    true,
		},
		{
			name:  "timestamp_t_hms",
			value: "2024-04-29T15:11:38",
			exp:   time.Date(2024, 4, 29, 15, 11, 38, 0, time.UTC),
			ok:    true,
		},

		// timestamp " " with time zone
		{
			name:  "timestamp_tz_sub_hms",
			value: "2024-04-29 15:11:38.06318+02:30:04",
			exp:   time.Date(2024, 4, 29, 15, 11, 38, 63180000, pos(2, 30, 4)),
			ok:    true,
		},
		{
			name:  "timestamp_tz_hms",
			value: "2024-04-29 15:11:38+02:30:04",
			exp:   time.Date(2024, 4, 29, 15, 11, 38, 0, pos(2, 30, 4)),
			ok:    true,
		},
		{
			name:  "timestamp_tz_sub_neg_hms",
			value: "2024-04-29 15:11:38.06318-02:30:04",
			exp:   time.Date(2024, 4, 29, 15, 11, 38, 63180000, neg(2, 30, 4)),
			ok:    true,
		},
		{
			name:  "timestamp_tz_neg_hms",
			value: "2024-04-29 15:11:38-02:30:04",
			exp:   time.Date(2024, 4, 29, 15, 11, 38, 0, neg(2, 30, 4)),
			ok:    true,
		},
		{
			name:  "timestamp_tz_sub_z",
			value: "2024-04-29 15:11:38.06318Z",
			exp:   time.Date(2024, 4, 29, 15, 11, 38, 63180000, time.UTC),
			ok:    true,
		},
		{
			name:  "timestamp_tz_z",
			value: "2024-04-29 15:11:38Z",
			exp:   time.Date(2024, 4, 29, 15, 11, 38, 0, time.UTC),
			ok:    true,
		},
		// timestamp " " without time zone
		{
			name:  "timestamp_sub_hms",
			value: "2024-04-29 15:11:38.06318",
			exp:   time.Date(2024, 4, 29, 15, 11, 38, 63180000, time.UTC),
			ok:    true,
		},
		{
			name:  "timestamp_hms",
			value: "2024-04-29 15:11:38",
			exp:   time.Date(2024, 4, 29, 15, 11, 38, 0, time.UTC),
			ok:    true,
		},
		{
			name:  "invalid",
			value: "not a timestamp",
			ok:    false,
		},
	}
}

func TestParseTime(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tc := range timestampTestCases(t) {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tim, ok := parseTime(tc.value)
			a.Equal(tc.ok, ok)
			a.Equal(tc.exp, tim)
		})
	}
}
