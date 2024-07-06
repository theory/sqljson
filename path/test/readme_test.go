package path_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/theory/sqljson/path"
	"github.com/theory/sqljson/path/types"
)

func decode(src []byte) any {
	var value any
	if err := json.Unmarshal(src, &value); err != nil {
		log.Fatal(err)
	}
	return value
}

func val(src string) any {
	var value any
	if err := json.Unmarshal([]byte(src), &value); err != nil {
		log.Fatal(err)
	}
	return value
}

func pp(val any) {
	js, err := json.Marshal(val)
	if err != nil {
		log.Fatal(err)
	}
	//nolint:forbidigo
	fmt.Println(string(js))
}

func ppi(val any) {
	js, err := json.MarshalIndent(val, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	//nolint:forbidigo
	fmt.Println(string(js))
}

func src() []byte {
	return []byte(`{
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
}

func Example_unmarshal() {
	var value any
	if err := json.Unmarshal(src(), &value); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%T\n", value)
	// Output: map[string]interface {}
}

func Example_segments() {
	value := decode(src())
	ppi(path.MustQuery("$.track.segments", value))
	// Output: [
	//   [
	//     {
	//       "HR": 73,
	//       "location": [
	//         47.763,
	//         13.4034
	//       ],
	//       "start time": "2018-10-14 10:05:14"
	//     },
	//     {
	//       "HR": 135,
	//       "location": [
	//         47.706,
	//         13.2635
	//       ],
	//       "start time": "2018-10-14 10:39:21"
	//     }
	//   ]
	// ]
}

func Example_anyArray() {
	value := decode(src())
	pp(path.MustQuery("$.track.segments[*].location", value))
	// Output: [[47.763,13.4034],[47.706,13.2635]]
}

func Example_indexZero() {
	value := decode(src())
	pp(path.MustQuery("$.track.segments[0].location", value))
	// Output: [[47.763,13.4034]]
}

func Example_seg_size() {
	value := decode(src())
	pp(path.MustQuery("$.track.segments.size()", value))
	// Output: [2]
}

func Example_gt_130() {
	value := decode(src())
	pp(path.MustQuery("$.track.segments[*].HR ? (@ > 130)", value))
	// Output: [135]
}

func Example_gt_130_time() {
	value := decode(src())
	pp(path.MustQuery(
		`$.track.segments[*] ? (@.HR > 130)."start time"`,
		value,
	))
	// Output: ["2018-10-14 10:39:21"]
}

func Example_coords() {
	value := decode(src())
	pp(path.MustQuery(
		`$.track.segments[*] ? (@.location[1] < 13.4) ? (@.HR > 130)."start time"`,
		value,
	))
	// Output: ["2018-10-14 10:39:21"]
}

func Example_loc_high() {
	value := decode(src())
	pp(path.MustQuery(
		`$.track.segments[*] ? (@.location[1] < 13.4).HR ? (@ > 130)`,
		value,
	))
	// Output: [135]
}

func Example_track_high() {
	value := decode(src())
	pp(path.MustQuery(
		`$.track ? (exists(@.segments[*] ? (@.HR > 130))).segments.size()`,
		value,
	))
	// Output: [2]
}

func Example_pred_std() {
	value := decode(src())
	pp(path.MustQuery("$.track.segments ?(@[*].HR > 130)", value))
	// Output: [{"HR":135,"location":[47.706,13.2635],"start time":"2018-10-14 10:39:21"}]
}

func Example_pred() {
	value := decode(src())
	pp(path.MustQuery("$.track.segments[*].HR > 130", value))
	// Output: [true]
}

func Example_lax() {
	value := decode(src())
	pp(path.MustQuery("lax $.track.segments.location", value))
	// Output: [[47.763,13.4034],[47.706,13.2635]]
}

func expectError() {
	if e := recover(); e != nil {
		//nolint:forbidigo
		fmt.Printf("panic: %v\n", e)
	}
}

func Example_strict_panic() {
	value := decode(src())
	defer expectError()
	pp(path.MustQuery("strict $.track.segments.location", value))
	// Output: panic: exec: jsonpath member accessor can only be applied to an object
}

func Example_unwrap() {
	value := decode(src())
	pp(path.MustQuery("strict $.track.segments[*].location", value))
	// Output: [[47.763,13.4034],[47.706,13.2635]]
}

func Example_any_lax() {
	value := decode(src())
	pp(path.MustQuery("lax $.**.HR", value))
	// Output: [73,135,73,135]
}

func Example_any_strict() {
	value := decode(src())
	pp(path.MustQuery("strict $.**.HR", value))
	// Output: [73,135]
}

func Example_lax_unexpected() {
	value := decode(src())
	pp(path.MustQuery("lax $.track.segments[*].location", value))
	// Output: [[47.763,13.4034],[47.706,13.2635]]
}

func Example_lax_filter() {
	value := decode(src())
	pp(path.MustQuery(
		"lax $.track.segments[*].location ?(@[*] > 15)",
		value,
	))
	// Output: [47.763,47.706]
}

func Example_strict_filter() {
	value := decode(src())
	pp(path.MustQuery(
		"strict $.track.segments[*].location ?(@[*] > 15)",
		value,
	))
	// Output: [[47.763,13.4034],[47.706,13.2635]]
}

func Example_add() {
	pp(path.MustQuery("$[0] + 3", val("2"))) // → [5]
	// Output: [5]
}

func Example_plus() {
	pp(path.MustQuery("+ $.x", val(`{"x": [2,3,4]}`))) // → [2, 3, 4]
	// Output: [2,3,4]
}

func Example_sub() {
	pp(path.MustQuery("7 - $[0]", val("[2]"))) // → [5]
	// Output: [5]
}

func Example_neg() {
	pp(path.MustQuery("- $.x", val(`{"x": [2,3,4]}`))) // → [-2,-3,-4]
	// Output: [-2,-3,-4]
}

func Example_mul() {
	pp(path.MustQuery("2 * $[0]", val("4"))) // → [8]
	// Output: [8]
}

func Example_div() {
	pp(path.MustQuery("$[0] / 2", val("[8.5]"))) // → [4.25]
	// Output: [4.25]
}

func Example_mod() {
	pp(path.MustQuery("$[0] % 10", val("[32]"))) // → [2]
	// Output: [2]
}

func Example_type() {
	pp(path.MustQuery("$[*].type()", val(`[1, "2", {}]`))) // → ["number","string","object"]
	// Output: ["number","string","object"]
}

func Example_size() {
	pp(path.MustQuery("$.m.size()", val(`{"m": [11, 15]}`))) // → [2]
	// Output: [2]
}

func Example_boolean() {
	pp(path.MustQuery("$[*].boolean()", val(`[1, "yes", false]`))) // → [true,true,false]
	// Output: [true,true,false]
}

func Example_string() {
	pp(path.MustQuery("$[*].string()", val(`[1.23, "xyz", false]`))) // → ["1.23","xyz","false"]
	pp(path.MustQuery("$.datetime().string()", "2023-08-15"))        // → ["2023-08-15"]
	// Output: ["1.23","xyz","false"]
	// ["2023-08-15"]
}

func Example_double() {
	pp(path.MustQuery("$.len.double() * 2", val(`{"len": "1.9"}`))) // → [3.8]
	// Output: [3.8]
}

func Example_ceiling() {
	pp(path.MustQuery("$.h.ceiling()", val(`{"h": 1.3}`))) // → [2]
	// Output: [2]
}

func Example_floor() {
	pp(path.MustQuery("$.h.floor()", val(`{"h": 1.7}`))) // → [1]
	// Output: [1]
}

func Example_abs() {
	pp(path.MustQuery("$.z.abs()", val(`{"z": -0.3}`))) // → [0.3]
	// Output: [0.3]
}

func Example_bigint() {
	pp(path.MustQuery("$.len.bigint()", val(`{"len": "9876543219"}`))) // → [9876543219]
	// Output: [9876543219]
}

func Example_decimal() {
	pp(path.MustQuery("$.decimal(6, 2)", val("1234.5678"))) // → [1234.57]
	// Output: [1234.57]
}

func Example_integer() {
	pp(path.MustQuery("$.len.integer()", val(`{"len": "12345"}`))) // → [12345]
	// Output: [12345]
}

func Example_number() {
	pp(path.MustQuery("$.len.number()", val(`{"len": "123.45"}`))) // → [123.45]
	// Output: [123.45]
}

func Example_datetime() {
	pp(path.MustQuery(
		`$[*] ? (@.datetime() < "2015-08-02".datetime())`,
		val(`["2015-08-01", "2015-08-12"]`),
	)) // → "2015-8-01"
	// Output: ["2015-08-01"]
}

func Example_datetime_format() {
	defer expectError()
	pp(path.MustQuery(
		`$[*].datetime("HH24:MI")`, val(`["12:30", "18:40"]`),
	)) // → ["12:30:00", "18:40:00"]
	// Output: panic: exec: .datetime(template) is not yet supported
}

func Example_date() {
	pp(path.MustQuery("$.date()", "2023-08-15")) // → ["2023-08-15"]
	// Output: ["2023-08-15"]
}

func Example_time() {
	pp(path.MustQuery("$.time()", "12:34:56")) // → ["12:34:56"]
	// Output: ["12:34:56"]
}

func Example_time_precision() {
	pp(path.MustQuery("$.time(2)", "12:34:56.789")) // → ["12:34:56.79"]
	// Output: ["12:34:56.79"]
}

func Example_time_tz() {
	pp(path.MustQuery("$.time_tz()", "12:34:56+05:30")) // → ["12:34:56+05:30"]
	// Output: ["12:34:56+05:30"]
}

func Example_time_tz_precision() {
	pp(path.MustQuery("$.time_tz(2)", "12:34:56.789+05:30")) // → ["12:34:56.79+05:30"]
	// Output: ["12:34:56.79+05:30"]
}

func Example_timestamp() {
	pp(path.MustQuery("$.timestamp()", "2023-08-15 12:34:56")) // → "2023-08-15T12:34:56"
	// Output: ["2023-08-15T12:34:56"]
}

func Example_timestamp_precision() {
	arg := "2023-08-15 12:34:56.789"
	pp(path.MustQuery("$.timestamp(2)", arg)) // → ["2023-08-15T12:34:56.79"]
	// Output: ["2023-08-15T12:34:56.79"]
}

func Example_timestamp_tz() {
	arg := "2023-08-15 12:34:56+05:30"
	pp(path.MustQuery("$.timestamp_tz()", arg)) // → ["2023-08-15T12:34:56+05:30"]
	// Output: ["2023-08-15T12:34:56+05:30"]
}

func Example_timestamp_tz_precision() {
	arg := "2023-08-15 12:34:56.789+05:30"
	pp(path.MustQuery("$.timestamp_tz(2)", arg)) // → ["2023-08-15T12:34:56.79+05:30"]
	// Output: ["2023-08-15T12:34:56.79+05:30"]
}

func Example_keyvalue() {
	pp(path.MustQuery("$.keyvalue()", val(`{"x": "20", "y": 32}`)))
	// → [{"id":0,"key":"x","value":"20"},{"id":0,"key":"y","value":32}]

	// Output: [{"id":0,"key":"x","value":"20"},{"id":0,"key":"y","value":32}]
}

func Example_eq() {
	pp(path.MustQuery("$[*] ? (@ == 1)", val(`[1, "a", 1, 3]`)))   // → [1,1]
	pp(path.MustQuery(`$[*] ? (@ == "a")`, val(`[1, "a", 1, 3]`))) // → ["a"]
	// Output: [1,1]
	// ["a"]
}

func Example_ne() {
	pp(path.MustQuery("$[*] ? (@ != 1)", val(`[1, 2, 1, 3]`)))      // → [2,3]
	pp(path.MustQuery(`$[*] ? (@ <> "b")`, val(`["a", "b", "c"]`))) // → ["a","c"]
	// Output: [2,3]
	// ["a","c"]
}

func Example_lt() {
	pp(path.MustQuery("$[*] ? (@ < 2)", val(`[1, 2, 3]`))) // → [1]
	// Output: [1]
}

func Example_le() {
	pp(path.MustQuery(`$[*] ? (@ <= "b")`, val(`["a", "b", "c"]`))) // → ["a","b"]
	// Output: ["a","b"]
}

func Example_gt() {
	pp(path.MustQuery("$[*] ? (@ > 2)", val(`[1, 2, 3]`))) // → [3]
	// Output: [3]
}

func Example_ge() {
	pp(path.MustQuery("$[*] ? (@ >= 2)", val(`[1, 2, 3]`))) // → [2,3]
	// Output: [2,3]
}

func Example_true() {
	arg := val(`[
	  {"name": "John", "parent": false},
	  {"name": "Chris", "parent": true}
	]`)
	pp(path.MustQuery("$[*] ? (@.parent == true)", arg)) // → [{"name":"Chris","parent":true}]
	// Output: [{"name":"Chris","parent":true}]
}

func Example_false() {
	arg := val(`[
	  {"name": "John", "parent": false},
	  {"name": "Chris", "parent": true}
	]`)
	pp(path.MustQuery("$[*] ? (@.parent == false)", arg)) // → [{"name":"John","parent":false}]
	// Output: [{"name":"John","parent":false}]
}

func Example_null() {
	arg := val(`[
	  {"name": "Mary", "job": null},
	  {"name": "Michael", "job": "driver"}
	]`)
	pp(path.MustQuery("$[*] ? (@.job == null) .name", arg)) // → ["Mary"]
	// Output: ["Mary"]
}

func Example_and() {
	pp(path.MustQuery("$[*] ? (@ > 1 && @ < 5)", val(`[1, 3, 7]`))) // → [3]
	// Output: [3]
}

func Example_or() {
	pp(path.MustQuery("$[*] ? (@ < 1 || @ > 5)", val(`[1, 3, 7]`))) // → [7]
	// Output: [7]
}

func Example_not() {
	pp(path.MustQuery("$[*] ? (!(@ < 5))", val(`[1, 3, 7]`))) // → [7]
	// Output: [7]
}

func Example_is_unknown() {
	pp(path.MustQuery("$[*] ? ((@ > 0) is unknown)", val(`[-1, 2, 7, "foo"]`))) // → ["foo"]
	// Output: ["foo"]
}

func Example_like_regex() {
	arg := val(`["abc", "abd", "aBdC", "abdacb", "babc"]`)
	pp(path.MustQuery(`$[*] ? (@ like_regex "^ab.*c")`, arg))          // → ["abc","abdacb"]
	pp(path.MustQuery(`$[*] ? (@ like_regex "^ab.*c" flag "i")`, arg)) // → ["abc","aBdC","abdacb"]
	// Output: ["abc","abdacb"]
	// ["abc","aBdC","abdacb"]
}

func Example_starts_with() {
	arg := val(`["John Smith", "Mary Stone", "Bob Johnson"]`)
	pp(path.MustQuery(`$[*] ? (@ starts with "John")`, arg)) // → ["John Smith"]
	// Output: ["John Smith"]
}

func Example_exists() {
	arg := val(`{"x": [1, 2], "y": [2, 4]}`)
	pp(path.MustQuery("strict $.* ? (exists (@ ? (@[*] > 2)))", arg))              // → [[2,4]]
	pp(path.MustQuery("strict $ ? (exists (@.name)) .name", val(`{"value": 42}`))) // → []
	// Output: [[2,4]]
	// []
}

func Example_regexp_string() {
	p := path.MustParse("$.* ?(@ like_regex \"^\\\\d+$\")")
	pp(p.MustQuery(context.Background(), val(`{"x": "42", "y": "no"}`))) // → ["42"]
	// Output: ["42"]
}

func Example_regexp_literal() {
	p := path.MustParse(`$.* ?(@ like_regex "^\\d+$")`)
	pp(p.MustQuery(context.Background(), val(`{"x": "42", "y": "no"}`))) // → ["42"]
	// Output: ["42"]
}

func Example_custom_time_zone() {
	p := path.MustParse("$.timestamp_tz().string()")
	arg := "2023-08-15 12:34:56+05:30"
	pp(p.MustQuery(context.Background(), arg)) // → ["2023-08-15T07:04:56+00"]

	// Add a time zone to the context.
	tz, err := time.LoadLocation("America/New_York")
	if err != nil {
		log.Fatal(err)
	}
	ctx := types.ContextWithTZ(context.Background(), tz)

	// The output will now be in the custom time zone.
	pp(p.MustQuery(ctx, arg)) // → ["2023-08-15T03:04:56-04"]
	// Output: ["2023-08-15T07:04:56+00"]
	// ["2023-08-15T03:04:56-04"]
}
