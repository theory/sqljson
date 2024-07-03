package types

import (
	"context"
	"math"
	"time"
)

// ParseTime parses src into [time.Time] by iterating through a list of valid
// date, time, and timestamp formats according to SQL/JSON standard: date,
// time_tz, time, timestamp_tz, and timestamp. Returns false if the string
// cannot be parsed by any of the formats.
//
// We also support ISO 8601 format (with "T") for timestamps, because
// PostgreSQL to_json() and to_jsonb() functions use this format.
func ParseTime(ctx context.Context, src string, precision int) (DateTime, bool) {
	// Date first.
	value, err := time.Parse("2006-01-02", src)
	if err == nil {
		return NewDate(value), true
	}

	// Time with TZ
	for _, format := range []string{
		"15:04:05Z07",
		"15:04:05Z07:00",
	} {
		value, err := time.Parse(format, src)
		if err == nil {
			return NewTimeTZ(adjustPrecision(offsetOnlyTimeFor(value), precision)), true
		}
	}

	// Time without TZ
	value, err = time.Parse("15:04:05", src)
	if err == nil {
		return NewTime(adjustPrecision(value, precision)), true
	}

	// Timestamp with tz, with and without "T"
	for _, format := range []string{
		"2006-01-02T15:04:05Z07",
		"2006-01-02 15:04:05Z07",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05Z07:00",
	} {
		value, err := time.Parse(format, src)
		if err == nil {
			return NewTimestampTZ(ctx, adjustPrecision(value, precision)), true
		}
	}

	// Timestamp without tz, with and without "T"
	for _, format := range []string{
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	} {
		value, err := time.Parse(format, src)
		if err == nil {
			return NewTimestamp(adjustPrecision(value, precision)), true
		}
	}

	// Not found.
	return nil, false
}

func adjustPrecision(value time.Time, precision int) time.Time {
	if precision > -1 {
		value = value.Round(time.Second / time.Duration(math.Pow10(precision)))
	}
	return value
}

// // https://www.postgresql.org/docs/devel/functions-formatting.html
// // https://pkg.go.dev/time#pkg-constants
// var formatMap = map[string]string{
// 	"HH":    "03",      // hour of day (01–12)
// 	"HH12":  "03",      // hour of day (01–12)
// 	"HH24":  "15",      // hour of day (00–23)
// 	"MI":    "04",      // minute (00–59)
// 	"SS":    "05",      // second (00–59)
// 	"MS":    ".000",    // millisecond (000–999)
// 	"US":    ".000000", // microsecond (000000–999999)
// 	"FF1":   ".0",      // tenth of second (0–9)
// 	"FF2":   ".00",     // hundredth of second (00–99)
// 	"FF3":   ".000",    // millisecond (000–999)
// 	"FF4":   ".0000",   // tenth of a millisecond (0000–9999)
// 	"FF5":   ".00000",  // hundredth of a millisecond (00000–99999)
// 	"FF6":   ".000000", // microsecond (000000–999999)
// 	"SSSS":  "",        // seconds past midnight (0–86399)
// 	"SSSSS": "",        // seconds past midnight (0–86399)
// 	"AM":    "PM",      // meridiem indicator (without periods)
// 	"PM":    "PM",
// 	"am":    "pm",
// 	"pm":    "pm",
// 	"A.M.":  "", // meridiem indicator (with periods)
// 	"P.M.":  "",
// 	"a.m.":  "",
// 	"p.m.":  "",
// 	"Y,YYY": "",     // year (4 or more digits) with comma
// 	"YYYY":  "2006", // year (4 or more digits)
// 	"YYY":   "",     // last 3 digits of year
// 	"YY":    "06",   // last 2 digits of year
// 	"Y":     "",     // last digit of year
// 	"IYYY":  "",     // ISO 8601 week-numbering year (4 or more digits)
// 	"IYY":   "",     // last 3 digits of ISO 8601 week-numbering year
// 	"IY":    "",     // last 2 digits of ISO 8601 week-numbering year
// 	"I":     "",     // last digit of ISO 8601 week-numbering year
// 	"BC":    "BC",   // era indicator (without periods)
// 	"AD":    "",
// 	"bc":    "",
// 	"ad":    "",
// 	"B.C.":  "", // era indicator (with periods)
// 	"A.D.":  "",
// 	"b.c.":  "",
// 	"a.d.":  "",

// 	"MONTH": "",        // full upper case month name (blank-padded to 9 chars)
// 	"Month": "January", // full capitalized month name (blank-padded to 9 chars)
// 	"month": "",        // full lower case month name (blank-padded to 9 chars)
// 	"MON":   "",        // abbreviated upper case month name (3 chars in English, localized lengths vary)
// 	"Mon":   "Jan",     // abbreviated capitalized month name (3 chars in English, localized lengths vary)
// 	"mon":   "",        // abbreviated lower case month name (3 chars in English, localized lengths vary)
// 	"MM":    "01",      // month number (01–12)
// 	"DAY":   "",        // full upper case day name (blank-padded to 9 chars)
// 	"Day":   "Monday",  // full capitalized day name (blank-padded to 9 chars)
// 	"day":   "",        // full lower case day name (blank-padded to 9 chars)
// 	"DY":    "",        // abbreviated upper case day name (3 chars in English, localized lengths vary)
// 	"Dy":    "Mon",     // abbreviated capitalized day name (3 chars in English, localized lengths vary)
// 	"dy":    "",        // abbreviated lower case day name (3 chars in English, localized lengths vary)
// 	"DDD":   "",        // day of year (001–366)
// 	"IDDD":  "",        // day of ISO 8601 week-numbering year
// (001–371; day 1 of the year is Monday of the first ISO week)
// 	"DD":    "02",      // day of month (01–31)
// 	"D":     "",        // day of the week, Sunday (1) to Saturday (7)
// 	"ID":    "",        // ISO 8601 day of the week, Monday (1) to Sunday (7)
// 	"W":     "",        // week of month (1–5) (the first week starts on the first day of the month)
// 	"WW":    "",        // week number of year (1–53) (the first week starts on the first day of the year)
// 	"IW":    "",        // week number of ISO 8601 week-numbering year
// (01–53; the first Thursday of the year is in week 1)
// 	"CC":    "",        // century (2 digits) (the twenty-first century starts on 2001-01-01)
// 	"J":     "",        // Julian Date (integer days since November 24, 4714 BC at local midnight; see Section B.7)
// 	"Q":     "",        // quarter
// 	"RM":    "",        // month in upper case Roman numerals (I–XII; I=January)
// 	"rm":    "",        // month in lower case Roman numerals (i–xii; i=January)
// 	"TZ":    "MST",     // upper case time-zone abbreviation
// 	"tz":    "",        // lower case time-zone abbreviation
// 	"TZH":   "-07",     // time-zone hours
// 	"TZM":   "",        // time-zone minutes
// 	"OF":    "-07:00",  // time-zone offset from UTC (HH or HH:MM)
// }
