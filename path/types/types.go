/*
Package types provides PostgresSQL-compatible data types for SQL/JSON Path
execution.

It makes every effort to duplicate the behavior of PostgreSQL JSONB dates and
times in particular in order to compatibly execute date and time comparisons
in SQL/JSON Path expressions.

# DateTime Types

Package maps the Postgres date and time types to these [DateTime]-implementing
types:

  - date: [Date]
  - time: [Time]
  - timetz: [TimeTZ]
  - timestamp: [Timestamp]
  - timestamptz: [TimestampTZ]

Each provides a constructor that takes a [time.Time] object, which defines the
underlying representation. Each also provides casting functions between the
types, but only for supported casts.

# Time Zones

Like the PostgreSQL timetz and timestamptz types, [TimeTZ] and [TimestampTZ]
do not store time zone information, but an offset from UTC. Even when passed a
[time.Time] object with a detailed location, the constructors will strip it
out and retain only the offset for the [time.Time] value.

By default, the types package operates on and displays dates and times in the
context of UTC. This affects conversion between time zone and non-time zone
data types, in particular. To change the time zone in which such operations
execute,

When required to operate on dates and times in the context of a time zone, the
types package defaults to UTC. For example, a TimestampTZ stringifies into
UTC:

	offsetPlus5 := time.FixedZone("", 5*3600)
	timestamp := types.NewTimestampTZ(
		context.Background(),
		time.Date(2023, 8, 15, 12, 34, 56, 0, offsetPlus5),
	)
	fmt.Printf("%v\n", timestamp) // → 2023-08-15T07:34:56+00:00

To operate in a the context of a different time zone, use [ContextWithTZ] to
add it to the context passed to any constructor or method that takes a context:

	tz, err := time.LoadLocation("America/New_York")
	if err != nil {
		log.Fatal(err)
	}
	ctx := types.ContextWithTZ(context.Background(), tz)

	offsetPlus5 := time.FixedZone("", 5*3600)
	timestamp := types.NewTimestampTZ(
		ctx,
		time.Date(2023, 8, 15, 12, 34, 56, 0, offsetPlus5),
	)

	fmt.Printf("%v\n", timestamp)        // → 2023-08-15T07:34:56+00:00
	fmt.Println(timestamp.ToString(ctx)) // → 2023-08-15T03:34:56-04

This time zone affects casts, as well, between offset-aware types ([TimeTZ],
[TimestampTZ]) and offset-unaware types ([Date], [Time], [Timestamp]). For any
execution, be sure to pass the same context to all operations.
*/
package types

import (
	"context"
	"errors"
	"time"
)

// ErrSQLType wraps errors returned by the types package.
var ErrSQLType = errors.New("type")

// secondsPerHour contains the number of seconds in an hour (excluding leap
// seconds).
const secondsPerHour = 60 * 60

// DateTime defines the interface for all date and time data types.
type DateTime interface {
	// GoTime returns the underlying time.Time object.
	GoTime() time.Time

	// ToString returns the output appropriate for the jsonpath string()
	// method, where appropriate in the time zone in ctx.
	ToString(ctx context.Context) string
}
