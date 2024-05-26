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
	time  time.Time
	ctor  func(t time.Time) DateTime
}

func newDate(t time.Time) DateTime        { return &Date{t} }
func newTime(t time.Time) DateTime        { return &Time{t} }
func newTimeTZ(t time.Time) DateTime      { return &TimeTZ{t} }
func newTimestamp(t time.Time) DateTime   { return &Timestamp{t} }
func newTimestampTZ(t time.Time) DateTime { return &TimestampTZ{t} }

func timestampTestCases(t *testing.T) []TSTestCase {
	t.Helper()
	return []TSTestCase{
		// Date
		{
			name:  "date",
			value: "2024-04-29",
			time:  time.Date(2024, 4, 29, 0, 0, 0, 0, time.UTC),
			ctor:  newDate,
		},
		// time with time zone
		{
			name:  "time_tz_hms",
			value: "14:15:31+01:22:30",
			time:  time.Date(0, 1, 1, 14, 15, 31, 0, pos(1, 22, 30)),
			ctor:  newTimeTZ,
		},
		{
			name:  "time_tz_sub_hms",
			value: "14:15:31.785996+01:22:30",
			time:  time.Date(0, 1, 1, 14, 15, 31, 785996000, pos(1, 22, 30)),
			ctor:  newTimeTZ,
		},
		{
			name:  "time_tz_sub_neg_hms",
			value: "14:15:31.785996-01:22:30",
			time:  time.Date(0, 1, 1, 14, 15, 31, 785996000, neg(1, 22, 30)),
			ctor:  newTimeTZ,
		},
		{
			name:  "time_tz_neg_hms",
			value: "14:15:31-01:22:30",
			time:  time.Date(0, 1, 1, 14, 15, 31, 0, neg(1, 22, 30)),
			ctor:  newTimeTZ,
		},
		{
			name:  "time_tz_sub_hm",
			value: "14:15:31.785996+03:14",
			time:  time.Date(0, 1, 1, 14, 15, 31, 785996000, pos(3, 14, 0)),
			ctor:  newTimeTZ,
		},
		{
			name:  "time_tz_hm",
			value: "14:15:31+03:14",
			time:  time.Date(0, 1, 1, 14, 15, 31, 0, pos(3, 14, 0)),
			ctor:  newTimeTZ,
		},
		{
			name:  "time_tz_sub_neg_hm",
			value: "14:15:31.785996-03:14",
			time:  time.Date(0, 1, 1, 14, 15, 31, 785996000, neg(3, 14, 0)),
			ctor:  newTimeTZ,
		},
		{
			name:  "time_tz_neg_hm",
			value: "14:15:31-03:14",
			time:  time.Date(0, 1, 1, 14, 15, 31, 0, neg(3, 14, 0)),
			ctor:  newTimeTZ,
		},
		{
			name:  "time_tz_sub_h",
			value: "14:15:31.785996+01",
			time:  time.Date(0, 1, 1, 14, 15, 31, 785996000, pos(1, 0, 0)),
			ctor:  newTimeTZ,
		},
		{
			name:  "time_tz_h",
			value: "14:15:31+01",
			time:  time.Date(0, 1, 1, 14, 15, 31, 0, pos(1, 0, 0)),
			ctor:  newTimeTZ,
		},
		{
			name:  "time_tz_sub_neg_h",
			value: "14:15:31.785996-11",
			time:  time.Date(0, 1, 1, 14, 15, 31, 785996000, neg(11, 0, 0)),
			ctor:  newTimeTZ,
		},
		{
			name:  "time_tz_neg_h",
			value: "14:15:31-11",
			time:  time.Date(0, 1, 1, 14, 15, 31, 0, neg(11, 0, 0)),
			ctor:  newTimeTZ,
		},
		{
			name:  "time_tz_sub_z",
			value: "14:15:31.785996Z",
			time:  time.Date(0, 1, 1, 14, 15, 31, 785996000, time.UTC),
			ctor:  newTimeTZ,
		},
		{
			name:  "time_tz_z",
			value: "14:15:31Z",
			time:  time.Date(0, 1, 1, 14, 15, 31, 0, time.UTC),
			ctor:  newTimeTZ,
		},
		// time without time zone
		{
			name:  "time_sub",
			value: "14:15:31.785996",
			time:  time.Date(0, 1, 1, 14, 15, 31, 785996000, time.UTC),
			ctor:  newTime,
		},
		{
			name:  "time_no_sub",
			value: "14:15:31",
			time:  time.Date(0, 1, 1, 14, 15, 31, 0, time.UTC),
			ctor:  newTime,
		},
		// timestamp "T" with time zone
		{
			name:  "timestamp_t_tz_sub_hms",
			value: "2024-04-29T15:11:38.06318+02:30:04",
			time:  time.Date(2024, 4, 29, 15, 11, 38, 63180000, pos(2, 30, 4)),
			ctor:  newTimestampTZ,
		},
		{
			name:  "timestamp_t_tz_hms",
			value: "2024-04-29T15:11:38+02:30:04",
			time:  time.Date(2024, 4, 29, 15, 11, 38, 0, pos(2, 30, 4)),
			ctor:  newTimestampTZ,
		},
		{
			name:  "timestamp_t_tz_sub_neg_hms",
			value: "2024-04-29T15:11:38.06318-02:30:04",
			time:  time.Date(2024, 4, 29, 15, 11, 38, 63180000, neg(2, 30, 4)),
			ctor:  newTimestampTZ,
		},
		{
			name:  "timestamp_t_tz_neg_hms",
			value: "2024-04-29T15:11:38-02:30:04",
			time:  time.Date(2024, 4, 29, 15, 11, 38, 0, neg(2, 30, 4)),
			ctor:  newTimestampTZ,
		},
		{
			name:  "timestamp_t_tz_sub_z",
			value: "2024-04-29T15:11:38.06318Z",
			time:  time.Date(2024, 4, 29, 15, 11, 38, 63180000, time.UTC),
			ctor:  newTimestampTZ,
		},
		{
			name:  "timestamp_t_tz_z",
			value: "2024-04-29T15:11:38Z",
			time:  time.Date(2024, 4, 29, 15, 11, 38, 0, time.UTC),
			ctor:  newTimestampTZ,
		},
		// timestamp "T" without time zone
		{
			name:  "timestamp_t_sub_hms",
			value: "2024-04-29T15:11:38.06318",
			time:  time.Date(2024, 4, 29, 15, 11, 38, 63180000, time.UTC),
			ctor:  newTimestamp,
		},
		{
			name:  "timestamp_t_hms",
			value: "2024-04-29T15:11:38",
			time:  time.Date(2024, 4, 29, 15, 11, 38, 0, time.UTC),
			ctor:  newTimestamp,
		},

		// timestamp " " with time zone
		{
			name:  "timestamp_tz_sub_hms",
			value: "2024-04-29 15:11:38.06318+02:30:04",
			time:  time.Date(2024, 4, 29, 15, 11, 38, 63180000, pos(2, 30, 4)),
			ctor:  newTimestampTZ,
		},
		{
			name:  "timestamp_tz_hms",
			value: "2024-04-29 15:11:38+02:30:04",
			time:  time.Date(2024, 4, 29, 15, 11, 38, 0, pos(2, 30, 4)),
			ctor:  newTimestampTZ,
		},
		{
			name:  "timestamp_tz_sub_neg_hms",
			value: "2024-04-29 15:11:38.06318-02:30:04",
			time:  time.Date(2024, 4, 29, 15, 11, 38, 63180000, neg(2, 30, 4)),
			ctor:  newTimestampTZ,
		},
		{
			name:  "timestamp_tz_neg_hms",
			value: "2024-04-29 15:11:38-02:30:04",
			time:  time.Date(2024, 4, 29, 15, 11, 38, 0, neg(2, 30, 4)),
			ctor:  newTimestampTZ,
		},
		{
			name:  "timestamp_tz_sub_z",
			value: "2024-04-29 15:11:38.06318Z",
			time:  time.Date(2024, 4, 29, 15, 11, 38, 63180000, time.UTC),
			ctor:  newTimestampTZ,
		},
		{
			name:  "timestamp_tz_z",
			value: "2024-04-29 15:11:38Z",
			time:  time.Date(2024, 4, 29, 15, 11, 38, 0, time.UTC),
			ctor:  newTimestampTZ,
		},
		// timestamp " " without time zone
		{
			name:  "timestamp_sub_hms",
			value: "2024-04-29 15:11:38.06318",
			time:  time.Date(2024, 4, 29, 15, 11, 38, 63180000, time.UTC),
			ctor:  newTimestamp,
		},
		{
			name:  "timestamp_hms",
			value: "2024-04-29 15:11:38",
			time:  time.Date(2024, 4, 29, 15, 11, 38, 0, time.UTC),
			ctor:  newTimestamp,
		},
	}
}

func TestParseTime(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tc := range timestampTestCases(t) {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tim, ok := ParseTime(tc.value, -1)
			a.True(ok)
			a.Equal(tc.ctor(tc.time), tim)
		})
	}
}

func TestParseFail(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tc := range []struct {
		name  string
		value string
	}{
		{"bogus", "bogus"},
		{"bad_date", "2024-02-30"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tim, ok := ParseTime(tc.value, -1)
			a.False(ok)
			a.Nil(tim)
		})
	}
}

func TestParseTimePrecision(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tc := range []struct {
		name  string
		value string
		one   int
		two   int
		six   int
	}{
		{
			name:  "time_nine_places",
			value: "14:15:31.78599685301",
			one:   800000000,
			two:   790000000,
			six:   785997000,
		},
		{
			name:  "time_six_places",
			value: "14:15:31.785996",
			one:   800000000,
			two:   790000000,
			six:   785996000,
		},
		{
			name:  "time_three_places",
			value: "14:15:31.785",
			one:   800000000,
			two:   790000000,
			six:   785000000,
		},
		{
			name:  "time_two_places",
			value: "14:15:31.78",
			one:   800000000,
			two:   780000000,
			six:   780000000,
		},
		{
			name:  "time_one_place",
			value: "14:15:31.7",
			one:   700000000,
			two:   700000000,
			six:   700000000,
		},
		{
			name:  "ts_nine_places",
			value: "2020-03-11T11:22:42.465029739+01",
			one:   500000000,
			two:   470000000,
			six:   465030000,
		},
		{
			name:  "ts_six_places",
			value: "2020-03-11T11:22:42.465029+01",
			one:   500000000,
			two:   470000000,
			six:   465029000,
		},
		{
			name:  "ts_three_places",
			value: "2020-03-11T11:22:42.465+01",
			one:   500000000,
			two:   470000000,
			six:   465000000,
		},
		{
			name:  "ts_two_places",
			value: "2020-03-11T11:22:42.46+01",
			one:   500000000,
			two:   460000000,
			six:   460000000,
		},
		{
			name:  "ts_one_place",
			value: "2020-03-11T11:22:42.4+01",
			one:   400000000,
			two:   400000000,
			six:   400000000,
		},
		{
			name:  "ts_no_places",
			value: "2020-03-11T11:22:42+01",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cmpNano(a, tc.value, 0, 0)
			cmpNano(a, tc.value, 1, tc.one)
			cmpNano(a, tc.value, 2, tc.two)
			cmpNano(a, tc.value, 6, tc.six)
		})
	}
}

func cmpNano(a *assert.Assertions, value string, precision, exp int) {
	dt, ok := ParseTime(value, precision)
	a.True(ok)
	a.Implements((*DateTime)(nil), dt)
	a.Equal(exp, dt.GoTime().Nanosecond())
}
