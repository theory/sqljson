package types

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func loadTZ(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

type zoneTestCase struct {
	test   string
	zone   string
	loc    *time.Location
	offset int
}

func zoneTestCases() []zoneTestCase {
	return []zoneTestCase{
		{"UTC", "UTC", loadTZ("UTC"), 0},
		{"empty", "UTC", loadTZ(""), 0},
		{"zero", "", time.FixedZone("", 0), 0},
		{"seven", "", time.FixedZone("", secondsPerHour*7), secondsPerHour * 7},
		{"neg_3", "", time.FixedZone("", secondsPerHour*-3), secondsPerHour * -3},
		{"America/New_York", "EDT", loadTZ("America/New_York"), secondsPerHour * -4},
		{"Asia/Tokyo", "JST", loadTZ("Asia/Tokyo"), secondsPerHour * 9},
		{"Africa/Nairobi", "EAT", loadTZ("Africa/Nairobi"), secondsPerHour * 3},
	}
}

func TestOffsetLocationForAndOnlyTimeFor(t *testing.T) {
	t.Parallel()

	for _, tc := range zoneTestCases() {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)

			// Create a time in the location.
			local := time.Date(2024, 6, 24, 10, 17, 32, 0, tc.loc)
			name, off := local.Zone()
			a.Equal(tc.zone, name)
			a.Equal(tc.offset, off)

			// Test offsetLocationFor
			loc := offsetLocationFor(local)
			a.Empty(loc.String())
			ts := time.Date(2024, 6, 24, 10, 17, 32, 0, loc)
			name, off = ts.Zone()
			a.Empty(name)
			a.Equal(tc.offset, off)

			// Test offsetOnlyTimeFor.
			ts = offsetOnlyTimeFor(local)
			name, off = ts.Zone()
			a.Empty(name)
			a.Equal(tc.offset, off)
		})
	}
}

func TestContextWithTZ(t *testing.T) {
	t.Parallel()

	for _, tc := range zoneTestCases() {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)

			ctx := ContextWithTZ(context.Background(), tc.loc)
			loc := TZFromContext(ctx)
			a.Equal(tc.loc, loc)
		})
	}

	t.Run("no_tz", func(t *testing.T) {
		t.Parallel()
		a := assert.New(t)

		loc := TZFromContext(context.Background())
		a.Equal(time.UTC, loc)
		loc = TZFromContext(ContextWithTZ(context.Background(), nil))
		a.Equal(time.UTC, loc)
	})
}
