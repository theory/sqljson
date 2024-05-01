package types

import "time"

// ParseTime parses src into [time.Time] by iterating through a list of valid
// time and timestamp formats according to SQL/JSON standard: date, time_tz,
// time, timestamp_tz, and timestamp. Returns false if the string cannot be
// parsed by any of the formats.
//
// We also support ISO 8601 format (with "T") for timestamps, because
// PostgreSQL to_json() and to_jsonb() functions use this format, as do
// [Timestamp.MarshalJSON] and [TimestampTZ.MarshalJSON].
func ParseTime(src string) (time.Time, bool) {
	// Handle infinity and -infinity? 24:00::00 time?
	for _, format := range []string{
		// date
		"2006-01-02",
		// time with tz
		"15:04:05Z07",
		"15:04:05Z07:00",
		"15:04:05Z07:00:00",
		// time without tz
		"15:04:05",
		// timestamp with tz, with and without "T"
		"2006-01-02T15:04:05Z07",
		"2006-01-02 15:04:05Z07",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05Z07:00",
		"2006-01-02T15:04:05Z07:00:00",
		"2006-01-02 15:04:05Z07:00:00",
		// timestamp without tz, with and without "T"
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	} {
		value, err := time.Parse(format, src)
		if err == nil {
			return value, true
		}
	}
	return time.Time{}, false
}
