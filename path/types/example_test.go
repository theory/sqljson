//nolint:godot
package types_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/theory/sqljson/path/types"
)

// Postgres:
//
//	david=# select jsonb_path_query_tz('"2023-08-15 12:34:56+05"', '$.timestamp_tz()');
//	     jsonb_path_query_tz
//	-----------------------------
//	 "2023-08-15T12:34:56+05:00"
//	(1 row)
//
//	david=# select jsonb_path_query_tz('"2023-08-15 12:34:56+05"', '$.timestamp_tz().string()');
//	   jsonb_path_query_tz
//	--------------------------
//	 "2023-08-15 07:34:56+00"
//	(1 row)
//
// [types.TimestampTZ]:
func Example_uTC() {
	offsetPlus5 := time.FixedZone("", 5*3600)
	ctx := types.ContextWithTZ(context.Background(), time.UTC)

	timestamp := types.NewTimestampTZ(
		ctx,
		time.Date(2023, 8, 15, 12, 34, 56, 0, offsetPlus5),
	)

	fmt.Printf("%v\n", timestamp)
	// Output: 2023-08-15T12:34:56+05:00
}

// Postgres:
//
//	david=# set time zone 'America/New_York';
//	SET
//	david=# select jsonb_path_query_tz('"2023-08-15 12:34:56+05"', '$.timestamp_tz()');
//	     jsonb_path_query_tz
//	-----------------------------
//	 "2023-08-15T12:34:56+05:00"
//	(1 row)
//
//	david=# select jsonb_path_query_tz('"2023-08-15 12:34:56+05"', '$.timestamp_tz().string()');
//	   jsonb_path_query_tz
//	--------------------------
//	 "2023-08-15 03:34:56-04"
//	(1 row)
//
// [types.TimestampTZ]:
func Example_nYC() {
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

	fmt.Printf("%v\n", timestamp)
	// Output: 2023-08-15T12:34:56+05:00
}

// Postgres:
//
//	david=# set time zone 'America/New_York';
//	SET
//	david=# select jsonb_path_query_tz('"2023-08-15"', '$.date()');
//	 jsonb_path_query_tz
//	---------------------
//	 "2023-08-15"
//	(1 row)
//
//	david=# select jsonb_path_query_tz('"2023-08-15"', '$.timestamp()');
//	  jsonb_path_query_tz
//	-----------------------
//	 "2023-08-15T00:00:00"
//	(1 row)
//
//	david=# select jsonb_path_query_tz('"2023-08-15"', '$.timestamp_tz()');
//	     jsonb_path_query_tz
//	-----------------------------
//	 "2023-08-15T04:00:00+00:00"
//	(1 row)
//
// [types.Date]:
func ExampleDate() {
	date := types.NewDate(time.Date(2023, 8, 15, 12, 34, 56, 0, time.UTC))
	fmt.Printf("%v\n", date)

	tz, err := time.LoadLocation("America/New_York")
	if err != nil {
		log.Fatal(err)
	}
	ctx := types.ContextWithTZ(context.Background(), tz)

	fmt.Printf("%v\n", date.ToTimestamp(ctx))
	// Difference in cast value formatting thread:
	// https://www.postgresql.org/message-id/flat/7DE080CE-6D8C-4794-9BD1-7D9699172FAB%40justatheory.com
	fmt.Printf("%v\n", date.ToTimestampTZ(ctx))
	// Output: 2023-08-15
	// 2023-08-15T00:00:00
	// 2023-08-15T00:00:00-04:00
}

// Postgres:
//
//	david=# set time zone 'America/Phoenix';
//	SET
//	david=# select jsonb_path_query_tz('"12:34:56"', '$.time()');
//	 jsonb_path_query_tz
//	---------------------
//	 "12:34:56"
//	(1 row)
//
//	david=# select jsonb_path_query_tz('"12:34:56"', '$.time_tz()');
//	 jsonb_path_query_tz
//	---------------------
//	 "12:34:56-07:00"
//	(1 row)
//
// [types.Time]:
func ExampleTime() {
	aTime := types.NewTime(time.Date(2023, 8, 15, 12, 34, 56, 0, time.UTC))
	fmt.Printf("%v\n", aTime)

	tz, err := time.LoadLocation("America/Phoenix")
	if err != nil {
		log.Fatal(err)
	}
	ctx := types.ContextWithTZ(context.Background(), tz)
	fmt.Printf("%v\n", aTime.ToTimeTZ(ctx))
	// Output: 12:34:56
	// 12:34:56-07:00
}

// Postgres:
//
//	david=# set time zone 'UTC';
//	SET
//	david=# select jsonb_path_query_tz('"12:34:56-04:00"', '$.time_tz()');
//	 jsonb_path_query_tz
//	---------------------
//	 "12:34:56-04:00"
//	(1 row)
//
//	david=# select jsonb_path_query_tz('"12:34:56-04:00"', '$.time()');
//	 jsonb_path_query_tz
//	---------------------
//	 "12:34:56"
//	(1 row)
//
//	david=# set time zone 'America/New_York';
//	SET
//	david=# select jsonb_path_query_tz('"12:34:56-04:00"', '$.time()');
//	 jsonb_path_query_tz
//	---------------------
//	 "12:34:56"
//	(1 row)
//
// [types.TimeTZ]:
func ExampleTimeTZ() {
	tz, err := time.LoadLocation("America/New_York")
	if err != nil {
		log.Fatal(err)
	}

	timeTZ := types.NewTimeTZ(time.Date(2023, 8, 15, 12, 34, 56, 0, tz))
	fmt.Printf("%v\n", timeTZ)

	ctx := types.ContextWithTZ(context.Background(), time.UTC)
	fmt.Printf("%v\n", timeTZ.ToTime(ctx))

	//nolint:gosmopolitan
	ctx = types.ContextWithTZ(context.Background(), time.Local)
	fmt.Printf("%v\n", timeTZ.ToTime(ctx))
	// Output: 12:34:56-04:00
	// 12:34:56
	// 12:34:56
}

// Postgres:
//
//	david=# set time zone 'America/Phoenix';
//	SET
//	david=# select jsonb_path_query_tz('"2023-08-15 12:34:56"', '$.timestamp()');
//	  jsonb_path_query_tz
//	-----------------------
//	 "2023-08-15T12:34:56"
//	(1 row)
//
//	david=# select jsonb_path_query_tz('"2023-08-15 12:34:56"', '$.date()');
//	 jsonb_path_query_tz
//	---------------------
//	 "2023-08-15"
//	(1 row)
//
//	david=# select jsonb_path_query_tz('"2023-08-15 12:34:56"', '$.time()');
//	 jsonb_path_query_tz
//	---------------------
//	 "12:34:56"
//	(1 row)
//
//	david=# select jsonb_path_query_tz('"2023-08-15 12:34:56"', '$.timestamp_tz()');
//	     jsonb_path_query_tz
//	-----------------------------
//	 "2023-08-15T19:34:56+00:00"
//	(1 row)
//
// [types.Timestamp]:
func ExampleTimestamp() {
	ts := types.NewTimestamp(time.Date(2023, 8, 15, 12, 34, 56, 0, time.UTC))
	fmt.Printf("%v\n", ts)

	tz, err := time.LoadLocation("America/Phoenix")
	if err != nil {
		log.Fatal(err)
	}
	ctx := types.ContextWithTZ(context.Background(), tz)
	fmt.Printf("%v\n", ts.ToDate(ctx))
	fmt.Printf("%v\n", ts.ToTime(ctx))
	// Difference in cast value formatting thread:
	// https://www.postgresql.org/message-id/flat/7DE080CE-6D8C-4794-9BD1-7D9699172FAB%40justatheory.com
	fmt.Printf("%v\n", ts.ToTimestampTZ(ctx))
	// Output: 2023-08-15T12:34:56
	// 2023-08-15
	// 12:34:56
	// 2023-08-15T12:34:56-07:00
}

// Postgres:
//
//	david=# set time zone 'UTC';
//	SET
//	david=# select jsonb_path_query_tz('"2023-08-15 12:34:56-04"', '$.timestamp_tz()');
//	     jsonb_path_query_tz
//	-----------------------------
//	 "2023-08-15T12:34:56-04:00"
//	(1 row)
//
//	david=# select jsonb_path_query_tz('"2023-08-15 12:34:56-04"', '$.timestamp()');
//	  jsonb_path_query_tz
//	-----------------------
//	 "2023-08-15T16:34:56"
//	(1 row)
//
//	david=# select jsonb_path_query_tz('"2023-08-15 12:34:56-04"', '$.date()');
//	 jsonb_path_query_tz
//	---------------------
//	 "2023-08-15"
//	(1 row)
//
//	david=# select jsonb_path_query_tz('"2023-08-15 12:34:56-04"', '$.time()');
//	 jsonb_path_query_tz
//	---------------------
//	 "16:34:56"
//	(1 row)
//
//	david=# set time zone 'America/Los_Angeles';
//	david=# select jsonb_path_query_tz('"2023-08-15 12:34:56-04"', '$.timestamp()');
//	  jsonb_path_query_tz
//	-----------------------
//	 "2023-08-15T09:34:56"
//	(1 row)
//
//	david=# select jsonb_path_query_tz('"2023-08-15 12:34:56-04"', '$.date()');
//	 jsonb_path_query_tz
//	---------------------
//	 "2023-08-15"
//	(1 row)
//
//	david=# select jsonb_path_query_tz('"2023-08-15 12:34:56-04"', '$.time()');
//	 jsonb_path_query_tz
//	---------------------
//	 "09:34:56"
//	(1 row)
//
// [types.TimestampTZ]:
func ExampleTimestampTZ() {
	tz, err := time.LoadLocation("America/New_York")
	if err != nil {
		log.Fatal(err)
	}

	ctx := types.ContextWithTZ(context.Background(), time.UTC)
	tsTZ := types.NewTimestampTZ(ctx, time.Date(2023, 8, 15, 12, 34, 56, 0, tz))
	fmt.Printf("%v\n", tsTZ)
	fmt.Printf("%v\n", tsTZ.ToTimestamp(ctx))
	fmt.Printf("%v\n", tsTZ.ToDate(ctx))
	fmt.Printf("%v\n", tsTZ.ToTime(ctx))

	tz, err = time.LoadLocation("America/Los_Angeles")
	if err != nil {
		log.Fatal(err)
	}
	ctx = types.ContextWithTZ(context.Background(), tz)
	fmt.Printf("%v\n", tsTZ.ToTimestamp(ctx))
	fmt.Printf("%v\n", tsTZ.ToDate(ctx))
	fmt.Printf("%v\n", tsTZ.ToTime(ctx))
	// Output: 2023-08-15T12:34:56-04:00
	// 2023-08-15T16:34:56
	// 2023-08-15
	// 16:34:56
	// 2023-08-15T09:34:56
	// 2023-08-15
	// 09:34:56
}
