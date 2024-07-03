//nolint:godot
package path_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/theory/sqljson/path"
	"github.com/theory/sqljson/path/exec"
	"github.com/theory/sqljson/path/parser"
	"github.com/theory/sqljson/path/types"
)

// SQL-standard path expressions hew to the SQL standard, which allows
// Boolean predicates only in ?() filter expressions, and can return
// any number of results.
//
// PostgreSQL jsonb_path_query():
//
//	=> SELECT '{
//	  "track": {
//	    "segments": [
//	      {
//	        "location":   [ 47.763, 13.4034 ],
//	        "start time": "2018-10-14 10:05:14",
//	        "HR": 73
//	      },
//	      {
//	        "location":   [ 47.706, 13.2635 ],
//	        "start time": "2018-10-14 10:39:21",
//	        "HR": 135
//	      }
//	    ]
//	  }
//	}' AS json \gset
//
//	=> SELECT jsonb_path_query(:'json', '$.track.segments[*] ? (@.HR > 130)."start time"');
//	   jsonb_path_query
//	-----------------------
//	 "2018-10-14 10:39:21"
//	(1 row)
//
// [Path.Query]:
func Example_sQLStandardPath() {
	src := []byte(`{
	  "track": {
	    "segments": [
	      {
	        "location":   [ 47.763, 13.4034 ],
	        "start time": "2018-10-14 10:05:14",
	        "HR": 73
	      },
	      {
	        "location":   [ 47.706, 13.2635 ],
	        "start time": "2018-10-14 10:39:21",
	        "HR": 135
	      }
	    ]
	  }
	}`)

	// Parse the JSON.
	var value any
	if err := json.Unmarshal(src, &value); err != nil {
		log.Fatal(err)
	}

	// Parse the SQL-standard jsonpath query.
	p, err := path.Parse(`$.track.segments[*] ? (@.HR > 130)."start time"`)
	if err != nil {
		log.Fatal(err)
	}

	// Execute the query against the JSON.
	items, err := p.Query(context.Background(), value)
	if err != nil {
		log.Fatal(err)
	}

	// Print the results.
	fmt.Printf("%v\n", items)
	// Output: [2018-10-14 10:39:21]
}

// Boolean predicate check expressions are a PostgreSQL extension that allow
// path expression to be a Boolean predicate, which can return only true,
// false, and null.
//
// PostgreSQL jsonb_path_query():
//
//	=> SELECT '{
//	  "track": {
//	    "segments": [
//	      {
//	        "location":   [ 47.763, 13.4034 ],
//	        "start time": "2018-10-14 10:05:14",
//	        "HR": 73
//	      },
//	      {
//	        "location":   [ 47.706, 13.2635 ],
//	        "start time": "2018-10-14 10:39:21",
//	        "HR": 135
//	      }
//	    ]
//	  }
//	}' AS json \gset
//
//	=> SELECT jsonb_path_query(:'json', '$.track.segments[*].HR > 130');
//	 jsonb_path_query
//	------------------
//	 true
//	(1 row)
//
// [Path.Query]:
func Example_predicateCheckPath() {
	src := []byte(`{
		"track": {
		  "segments": [
			{
			  "location":   [ 47.763, 13.4034 ],
			  "start time": "2018-10-14 10:05:14",
			  "HR": 73
			},
			{
			  "location":   [ 47.706, 13.2635 ],
			  "start time": "2018-10-14 10:39:21",
			  "HR": 135
			}
		  ]
		}
	  }`)

	// Parse the JSON.
	var value any
	if err := json.Unmarshal(src, &value); err != nil {
		log.Fatal(err)
	}

	// Parse the Postgres predicate check jsonpath query.
	p, err := path.Parse(`$.track.segments[*].HR > 130`)
	if err != nil {
		log.Fatal(err)
	}

	// Execute the query against the JSON.
	matched, err := p.Match(context.Background(), value)
	if err != nil {
		log.Fatal(err)
	}

	// Print the results.
	fmt.Printf("%v\n", matched)
	// Output: true
}

func ExampleParse() {
	p, err := path.Parse("$.x [*] ? ( @ > 2 )")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%v\n", p)
	// Output: $."x"[*]?(@ > 2)
}

func ExampleMustParse() {
	p := path.MustParse("$.x [*] ? ( @ > 2 )")
	fmt.Printf("%v\n", p)
	// Output: $."x"[*]?(@ > 2)
}

func ExampleNew() {
	ast, err := parser.Parse("$.x [*] ? ( @ > 2 )")
	if err != nil {
		log.Fatal(err)
	}
	p := path.New(ast)
	fmt.Printf("%v\n", p)
	// Output: $."x"[*]?(@ > 2)
}

func ExamplePath_PgIndexOperator() {
	p := path.MustParse("$.x[*] ?(@ > 2)")
	fmt.Printf("SQL Standard:    %v\n", p.PgIndexOperator())
	p = path.MustParse("$.x[*] > 2")
	fmt.Printf("Predicate Check: %v\n", p.PgIndexOperator())
	// Output: SQL Standard:    @?
	// Predicate Check: @@
}

// [exec.WithVars] provides named values to be substituted into the
// path expression. PostgreSQL jsonb_path_query() example:
//
//	=> SELECT jsonb_path_query('{"a":[1,2,3,4,5]}', '$.a[*] ? (@ >= $min && @ <= $max)', '{"min":2, "max":4}');
//	jsonb_path_query
//	------------------
//	 2
//	 3
//	 4
//	(3 rows)
//
// [Path.Query] using [exec.WithVars]:
func Example_withVars() {
	p := path.MustParse("$.a[*] ? (@ >= $min && @ <= $max)")
	var value any
	if err := json.Unmarshal([]byte(`{"a":[1,2,3,4,5]}`), &value); err != nil {
		log.Fatal(err)
	}

	res, err := p.Query(
		context.Background(),
		value,
		exec.WithVars(exec.Vars{"min": float64(2), "max": float64(4)}),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%v\n", res)
	// Output: [2 3 4]
}

// [exec.WithTZ] allows comparisons of date and time values that require
// timezone-aware conversions. By default such conversions are made relative
// to UTC, but can be made relative to another (user-preferred) time zone by
// using [types.ContextWithTZ] to add it to the context passed to the query
// method.
//
// This is the equivalent to using the *_tz() PostgreSQL functions. For
// example, this call to jsonb_path_query_tz() converts "2015-08-02", which
// has no offset, to a timestamptz in UTC, to compare to the two values. It
// selects only "2015-08-02 23:00:00-05" because, once it converts to PDT, its
// value is "2015-08-02 21:00:00-07", while "2015-08-02 01:00:00-05" resolves
// to "2015-08-01 23:00:00-07", which is less than 2015-08-02:
//
//	=> SET time zone 'PST8PDT';
//	SET
//	=> SELECT jsonb_path_query_tz(
//	    '["2015-08-02 01:00:00-05", "2015-08-02 23:00:00-05"]',
//	    '$[*] ? (@.datetime() >= "2015-08-02".date())'
//	);
//	   jsonb_path_query_tz
//	--------------------------
//	 "2015-08-02 23:00:00-05"
//
// Here's the equivalent using [types.ContextWithTZ] to set the time zone
// context in which [Path.Query] operates, and where [exec.WithTZ] allows
// conversion between timestamps with and without time zones:
func Example_withTZ() {
	// Configure time zone to use when casting.
	loc, err := time.LoadLocation("PST8PDT")
	if err != nil {
		log.Fatal(err)
	}

	// Query in the context of that time zone.
	p := path.MustParse(`$[*] ? (@.datetime() >= "2015-08-02".date())`)
	res, err := p.Query(
		types.ContextWithTZ(context.Background(), loc),
		[]any{"2015-08-01 02:00:00-05", "2015-08-02 23:00:00-05"},
		exec.WithTZ(),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%v\n", res)
	// Output: [2015-08-02 23:00:00-05]
}

// [exec.WithSilent] suppresses [exec.ErrVerbose] errors, including missing
// object field or array element, unexpected JSON item type, and datetime
// and numeric errors. This behavior might be helpful when searching JSON
// entities of varying structure.
//
// For example, this PostgreSQL jsonb_path_query() call raises an error
// because index 1 it out of bounds of the array, ["hi"], which has only one
// value, and so raises an error:
//
//	=> SELECT jsonb_path_query(target => '["hi"]', path => 'strict $[1]');
//	ERROR:  jsonpath array subscript is out of bounds
//
// Passing the silent parameter suppresses the error:
//
//	=> SELECT jsonb_path_query(target => '["hi"]', path => 'strict $[1]', silent => true);
//	 jsonb_path_query
//	------------------
//	(0 rows)
//
// Here's the equivalent call to [Path.Query] without and then with the
// [exec.WithSilent] option:
func Example_withSilent() {
	// Execute query with array index out of bounds.
	p := path.MustParse("strict $[1]")
	ctx := context.Background()
	res, err := p.Query(ctx, []any{"hi"})
	fmt.Printf("%v: %v\n", res, err)

	// WithSilent suppresses the error.
	res, err = p.Query(ctx, []any{"hi"}, exec.WithSilent())
	fmt.Printf("%v: %v\n", res, err)
	// Output: []: exec: jsonpath array subscript is out of bounds
	// []: <nil>
}

func ExamplePath_Exists_nULL() {
	p := path.MustParse("strict $[1]")
	ctx := context.Background()
	res, err := p.Exists(ctx, []any{"hi"}, exec.WithSilent())
	if err != nil {
		if errors.Is(err, exec.NULL) {
			// The outcome was actually unknown.
			fmt.Println("result was null")
		} else {
			// Some other error.
			log.Fatal(err)
		}
	} else {
		// Result is known.
		fmt.Printf("%v\n", res)
	}
	// Output: result was null
}
