package types

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

//nolint:unparam // keep s in case we need it in the future.
func pos(h, m, s int) *time.Location {
	return time.FixedZone("", h*60*60+m*60+s)
}

//nolint:unparam // keep s in case we need it in the future.
func neg(h, m, s int) *time.Location {
	return time.FixedZone("", -(h*60*60 + m*60 + s))
}

type TSTestCase struct {
	name  string
	value string
	time  time.Time
	ctor  func(t time.Time, tz *time.Location) DateTime
}

func newTestDate(t time.Time, _ *time.Location) DateTime         { return &Date{t} }
func newTestTime(t time.Time, _ *time.Location) DateTime         { return &Time{t} }
func newTestTimeTZ(t time.Time, _ *time.Location) DateTime       { return &TimeTZ{t} }
func newTestTimestamp(t time.Time, _ *time.Location) DateTime    { return &Timestamp{t} }
func newTestTimestampTZ(t time.Time, tz *time.Location) DateTime { return &TimestampTZ{t, tz} }

func timestampTestCases(t *testing.T) []TSTestCase {
	t.Helper()
	return []TSTestCase{
		// Date
		{
			name:  "date",
			value: "2024-04-29",
			time:  time.Date(2024, 4, 29, 0, 0, 0, 0, offsetZero),
			ctor:  newTestDate,
		},
		// time with time zone
		{
			name:  "time_tz_hm",
			value: "14:15:31+01:22",
			time:  time.Date(0, 1, 1, 14, 15, 31, 0, pos(1, 22, 0)),
			ctor:  newTestTimeTZ,
		},
		{
			name:  "time_tz_sub_hm",
			value: "14:15:31.785996+01:22",
			time:  time.Date(0, 1, 1, 14, 15, 31, 785996000, pos(1, 22, 0)),
			ctor:  newTestTimeTZ,
		},
		{
			name:  "time_tz_pos_hm",
			value: "14:15:31.785996+03:14",
			time:  time.Date(0, 1, 1, 14, 15, 31, 785996000, pos(3, 14, 0)),
			ctor:  newTestTimeTZ,
		},
		{
			name:  "time_tz_sub_neg_hm",
			value: "14:15:31.785996-03:14",
			time:  time.Date(0, 1, 1, 14, 15, 31, 785996000, neg(3, 14, 0)),
			ctor:  newTestTimeTZ,
		},
		{
			name:  "time_tz_neg_hm",
			value: "14:15:31-03:14",
			time:  time.Date(0, 1, 1, 14, 15, 31, 0, neg(3, 14, 0)),
			ctor:  newTestTimeTZ,
		},
		{
			name:  "time_tz_sub_h",
			value: "14:15:31.785996+01",
			time:  time.Date(0, 1, 1, 14, 15, 31, 785996000, pos(1, 0, 0)),
			ctor:  newTestTimeTZ,
		},
		{
			name:  "time_tz_h",
			value: "14:15:31+01",
			time:  time.Date(0, 1, 1, 14, 15, 31, 0, pos(1, 0, 0)),
			ctor:  newTestTimeTZ,
		},
		{
			name:  "time_tz_sub_neg_h",
			value: "14:15:31.785996-11",
			time:  time.Date(0, 1, 1, 14, 15, 31, 785996000, neg(11, 0, 0)),
			ctor:  newTestTimeTZ,
		},
		{
			name:  "time_tz_neg_h",
			value: "14:15:31-11",
			time:  time.Date(0, 1, 1, 14, 15, 31, 0, neg(11, 0, 0)),
			ctor:  newTestTimeTZ,
		},
		{
			name:  "time_tz_sub_z",
			value: "14:15:31.785996Z",
			time:  time.Date(0, 1, 1, 14, 15, 31, 785996000, offsetZero),
			ctor:  newTestTimeTZ,
		},
		{
			name:  "time_tz_z",
			value: "14:15:31Z",
			time:  time.Date(0, 1, 1, 14, 15, 31, 0, offsetZero),
			ctor:  newTestTimeTZ,
		},
		// time without time zone
		{
			name:  "time_sub",
			value: "14:15:31.785996",
			time:  time.Date(0, 1, 1, 14, 15, 31, 785996000, offsetZero),
			ctor:  newTestTime,
		},
		{
			name:  "time_no_sub",
			value: "14:15:31",
			time:  time.Date(0, 1, 1, 14, 15, 31, 0, offsetZero),
			ctor:  newTestTime,
		},
		// timestamp "T" with time zone
		{
			name:  "timestamp_t_tz_sub_hm",
			value: "2024-04-29T15:11:38.06318+02:30",
			time:  time.Date(2024, 4, 29, 15, 11, 38, 63180000, pos(2, 30, 0)),
			ctor:  newTestTimestampTZ,
		},
		{
			name:  "timestamp_t_tz_hm",
			value: "2024-04-29T15:11:38+02:30",
			time:  time.Date(2024, 4, 29, 15, 11, 38, 0, pos(2, 30, 0)),
			ctor:  newTestTimestampTZ,
		},
		{
			name:  "timestamp_t_tz_sub_neg_hm",
			value: "2024-04-29T15:11:38.06318-02:30",
			time:  time.Date(2024, 4, 29, 15, 11, 38, 63180000, neg(2, 30, 0)),
			ctor:  newTestTimestampTZ,
		},
		{
			name:  "timestamp_t_tz_neg_hm",
			value: "2024-04-29T15:11:38-02:30",
			time:  time.Date(2024, 4, 29, 15, 11, 38, 0, neg(2, 30, 0)),
			ctor:  newTestTimestampTZ,
		},
		{
			name:  "timestamp_t_tz_sub_z",
			value: "2024-04-29T15:11:38.06318Z",
			time:  time.Date(2024, 4, 29, 15, 11, 38, 63180000, offsetZero),
			ctor:  newTestTimestampTZ,
		},
		{
			name:  "timestamp_t_tz_z",
			value: "2024-04-29T15:11:38Z",
			time:  time.Date(2024, 4, 29, 15, 11, 38, 0, offsetZero),
			ctor:  newTestTimestampTZ,
		},
		// timestamp "T" without time zone
		{
			name:  "timestamp_t_sub_hms",
			value: "2024-04-29T15:11:38.06318",
			time:  time.Date(2024, 4, 29, 15, 11, 38, 63180000, offsetZero),
			ctor:  newTestTimestamp,
		},
		{
			name:  "timestamp_t_hms",
			value: "2024-04-29T15:11:38",
			time:  time.Date(2024, 4, 29, 15, 11, 38, 0, offsetZero),
			ctor:  newTestTimestamp,
		},

		// timestamp " " with time zone
		{
			name:  "timestamp_tz_sub_hm",
			value: "2024-04-29 15:11:38.06318+02:30",
			time:  time.Date(2024, 4, 29, 15, 11, 38, 63180000, pos(2, 30, 0)),
			ctor:  newTestTimestampTZ,
		},
		{
			name:  "timestamp_tz_hm",
			value: "2024-04-29 15:11:38+02:30",
			time:  time.Date(2024, 4, 29, 15, 11, 38, 0, pos(2, 30, 0)),
			ctor:  newTestTimestampTZ,
		},
		{
			name:  "timestamp_tz_sub_neg_hm",
			value: "2024-04-29 15:11:38.06318-02:30",
			time:  time.Date(2024, 4, 29, 15, 11, 38, 63180000, neg(2, 30, 0)),
			ctor:  newTestTimestampTZ,
		},
		{
			name:  "timestamp_tz_neg_hm",
			value: "2024-04-29 15:11:38-02:30",
			time:  time.Date(2024, 4, 29, 15, 11, 38, 0, neg(2, 30, 0)),
			ctor:  newTestTimestampTZ,
		},
		{
			name:  "timestamp_tz_sub_z",
			value: "2024-04-29 15:11:38.06318Z",
			time:  time.Date(2024, 4, 29, 15, 11, 38, 63180000, offsetZero),
			ctor:  newTestTimestampTZ,
		},
		{
			name:  "timestamp_tz_z",
			value: "2024-04-29 15:11:38Z",
			time:  time.Date(2024, 4, 29, 15, 11, 38, 0, offsetZero),
			ctor:  newTestTimestampTZ,
		},
		// timestamp " " without time zone
		{
			name:  "timestamp_sub_hms",
			value: "2024-04-29 15:11:38.06318",
			time:  time.Date(2024, 4, 29, 15, 11, 38, 63180000, offsetZero),
			ctor:  newTestTimestamp,
		},
		{
			name:  "timestamp_hms",
			value: "2024-04-29 15:11:38",
			time:  time.Date(2024, 4, 29, 15, 11, 38, 0, offsetZero),
			ctor:  newTestTimestamp,
		},
	}
}

func TestParseTime(t *testing.T) {
	t.Parallel()
	for _, tc := range timestampTestCases(t) {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			for _, zc := range zoneTestCases() {
				t.Run(zc.name, func(t *testing.T) {
					t.Parallel()
					a := assert.New(t)

					ctx := ContextWithTZ(context.Background(), zc.loc)
					tim, ok := ParseTime(ctx, tc.value, -1)
					a.True(ok)
					a.Equal(tc.ctor(tc.time, TZFromContext(ctx)), tim)
				})
			}
		})
	}
}

func TestParseFail(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	for _, tc := range []struct {
		name  string
		value string
	}{
		{"bogus", "bogus"},
		{"bad_date", "2024-02-30"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)

			tim, ok := ParseTime(ctx, tc.value, -1)
			a.False(ok)
			a.Nil(tim)
		})
	}
}

func TestParseTimePrecision(t *testing.T) {
	t.Parallel()

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
			a := assert.New(t)

			cmpNano(a, tc.value, 0, 0)
			cmpNano(a, tc.value, 1, tc.one)
			cmpNano(a, tc.value, 2, tc.two)
			cmpNano(a, tc.value, 6, tc.six)
		})
	}
}

func cmpNano(a *assert.Assertions, value string, precision, exp int) {
	dt, ok := ParseTime(context.Background(), value, precision)
	a.True(ok)
	a.Implements((*DateTime)(nil), dt)
	a.Equal(exp, dt.GoTime().Nanosecond())
}
