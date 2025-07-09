Go SQL/JSON Path
================

The path package ports the SQL/JSON Path data type from PostgreSQL to Go. It
supports both SQL-standard path expressions and PostgreSQL-specific predicate
check expressions.

> 💡 Use the [🛝 Playground] links below to run the examples in this document,
> and to experiment with jsonpath execution. The Go SQL/JSON Path Playground
> is a single-page stateless JavaScript and [TinyGo]-compiled [Wasm] app that
> offers permalink generation to share examples, like [this one].

## The SQL/JSON Path Language

> This section was ported from the [PostgreSQL docs].

<!-- See ./tests/example_test.go for runnable versions of all the examples. -->
<!-- https://github.com/postgres/postgres/blob/7bd752c/doc/src/sgml/func.sgml#L17709-L19028 -->

SQL/JSON Path is a query language for JSON values. A path expression applied
to a JSON value produces a JSON result.

SQL/JSON path expressions specify item(s) to be retrieved from a JSON value,
similarly to XPath expressions used for access to XML content. In Go, path
expressions are implemented in the path package and can use any elements
described [below](#syntax).

### Syntax

The path package implements support for the SQL/JSON path language in Go to
efficiently query JSON data. It provides an abstract syntax tree  of the
parsed SQL/JSON path expression that specifies the items to be retrieved by
the path engine from the JSON data for further processing with the SQL/JSON
query functions.

The semantics of SQL/JSON path predicates and operators generally follow SQL.
At the same time, to provide a natural way of working with JSON data, SQL/JSON
path syntax uses some JavaScript conventions:

*   Dot (`.`) is used for member access.

*   Square brackets (`[]`) are used for array access.

*   SQL/JSON arrays are 0-relative, like Go slices, but unlike regular SQL
    arrays, which start from 1.

Numeric literals in SQL/JSON path expressions follow JavaScript rules, which
are different from Go, SQL, and JSON in some minor details. For example,
SQL/JSON path allows `.1` and `1.`, which are invalid in JSON. Non-decimal
integer literals and underscore separators are supported, for example,
`1_000_000`, `0x1EEE_FFFF`, `0o273`, `0b100101`. In SQL/JSON path (and in
JavaScript, but not in SQL or Go), there must not be an underscore separator
directly after the radix prefix.

An SQL/JSON path expression is typically written as a Go string literal, so it
must be enclosed in back quotes or double quotes --- and with the latter any
double quotes within the value must be escaped (see [string literals]).

Some forms of path expressions require string literals within them. These
embedded string literals follow JavaScript/ECMAScript conventions: they must
be surrounded by double quotes, and backslash escapes may be used within them
to represent otherwise-hard-to-type characters. In particular, the way to
write a double quote within a double-quoted string literal is `\"`, and to
write a backslash itself, you must write `\\`. Other special backslash
sequences include those recognized in JSON strings: `\b`, `\f`, `\n`, `\r`,
`\t`, `\v` for various ASCII control characters, and `\uNNNN` for a Unicode
character identified by its 4-hex-digit code point. The backslash syntax also
includes two cases not allowed by JSON: `\xNN` for a character code written
with only two hex digits, and `\u{N...}` for a character code written with 1
to 6 hex digits.

A path expression consists of a sequence of path elements, which can be any of
the following:

*   Path literals of JSON primitive types: Unicode text, numeric, `true`,
    `false`, or `null`
*   Path variables listed in the [Path Variables table](#path-variables)
*   Accessor operators listed in the [Path Accessors table](#path-accessors)
*   JSON path operators and methods listed[SQL/JSON Path Operators And
    Methods](#sql-json-path-operators-and-methods)
*   Parentheses, which can be used to provide filter expressions or define the
    order of path evaluation

For details on using JSON path expressions with SQL/JSON query functions, see
[Operation](#operation).

#### Path Variables

| Variable   | Description
| ---------- | ------------------------------------------------------------------------------------------------- |
| `$`        | A variable representing the JSON value being queried (the context item).                          |
| `$varname` | A named variable. Its value can be set by the `exec.WithVars` option of Path processing functions |
| `@`        | A variable representing the result of path evaluation in filter expressions.                      |

#### Path Accessors

| Accessor Operator     | Description
| --------------------- | ------------------------------------------------------------------------------------------------- |
| `.key`, `."$varname"` | Member accessor that returns an object member with the specified key. If the key name matches some named variable starting with `$` or does not meet the JavaScript rules for an identifier, it must be enclosed in double quotes to make it a string literal.
| `.*`                  | Wildcard member accessor that returns the values of all members located at the top level of the current object.
| `.**`                 | Recursive wildcard member accessor that processes all levels of the JSON hierarchy of the current object and returns all the member values, regardless of their nesting level. This is a PostgreSQL extension of the SQL/JSON standard.
| `.**{level}`, `.**{start_level to end_level}` | Like `.**`, but selects only the specified levels of the JSON hierarchy. Nesting levels are specified as integers. Level zero corresponds to the current object. To access the lowest nesting level, you can use the `last` keyword. This is a PostgreSQL extension of the SQL/JSON standard.
| `[subscript, ...]`                            | Array element accessor. `subscript` can be given in two forms: `index` or `start_index` to `end_index`. The first form returns a single array element by its index. The second form returns an array slice by the range of indexes, including the elements that correspond to the provided `start_index` and `end_index`.<br/><br/>The specified index can be an integer, as well as an expression returning a single numeric value, which is automatically cast to integer. Index zero corresponds to the first array element. You can also use the `last` keyword to denote the last array element, which is useful for handling arrays of unknown length.
| `[*]`                 | Wildcard array element accessor that returns all array elements.

### Operation

Path query functions pass the provided path expression to the path engine for
evaluation. If the expression matches the queried JSON data, the corresponding
set of JSON items, is returned as an `[]any` slice. If there is no match, the
result will be an empty slice, `NULL`, `false`, or an error, depending on the
function. Path expressions are written in the SQL/JSON path language and can
include arithmetic expressions and functions.

A path expression consists of a sequence of elements allowed by the SQL/JSON
path language. The path expression is normally evaluated from left to right,
but you can use parentheses to change the order of operations. If the
evaluation is successful, a sequence of JSON items is produced, and the
evaluation result is returned to the Path query function that completes the
specified computation.

To refer to the JSON value being queried (the context item), use the `$`
variable in the path expression. The first element of a path must always be
`$`. It can be followed by one or more accessor operators, which go down the
JSON structure level by level to retrieve sub-items of the context item. Each
accessor operator acts on the result(s) of the previous evaluation step,
producing zero, one, or more output items from each input item.

For example, suppose you have some JSON data from a GPS tracker that you would
like to parse, such as:

``` go
var src = []byte(`{
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
```

The path package expects JSON to be decoded into a Go value, one of `string`,
`float64`, [`json.Number`], `map[string]any`, or `[]any` — which are the
values produced by unmarshaling data into an `any` value. For the above JSON,
unmarshal it like so:

``` go
var value any
if err := json.Unmarshal(src, &value); err != nil {
    log.Fatal(err)
}
fmt.Printf("%T\n", value)
```

The output shows the parsed data type:

``` go
map[string]interface {}
```

Note that examples below encode results as JSON for legibility using a
function like this:

``` go
func pp(val any) {
    js, err := json.Marshal(val)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(string(js))
}
```

To retrieve the available track segments, you need to use the `.key` accessor
operator to descend through surrounding JSON objects, for example:

``` go
pp(path.MustQuery("$.track.segments", value))
```

And the output (indented for legibility; [playground][play01]):

``` json
[
  [
    {
      "HR": 73,
      "location": [
        47.763,
        13.4034
      ],
      "start time": "2018-10-14 10:05:14"
    },
    {
      "HR": 135,
      "location": [
        47.706,
        13.2635
      ],
      "start time": "2018-10-14 10:39:21"
    }
  ]
]
```

To retrieve the contents of an array, you typically use the `[*]` operator. The
following example will return the location coordinates for all the available
track segments ([playground][play02]):

``` go
pp(path.MustQuery("$.track.segments[*].location", value))
```

``` json
[[47.763,13.4034],[47.706,13.2635]]
```

Here we started with the whole JSON input value (`$`), then the `.track`
accessor selected the JSON object associated with the `"track"` object key,
then the `.segments` accessor selected the JSON array associated with the
`"segments"` key within that object, then the `[*]` accessor selected each
element of that array (producing a series of items), then the `.location`
accessor selected the JSON array associated with the `"location"` key within
each of those objects. In this example, each of those objects had a
`"location"` key; but if any of them did not, the `.location` accessor would
have simply produced no output for that input item.

To return the coordinates of the first segment only, you can specify the
corresponding subscript in the `[]` accessor operator. Recall that JSON array
indexes are 0-relative ([playground][play03]):

```go
pp(path.MustQuery("$.track.segments[0].location", value))
```

``` json
[[47.763,13.4034]]
```

The result of each path evaluation step can be processed by one or more of the
json path operators and methods listed [below](#sqljson-path-operators-and-methods).
Each method name must be preceded by a dot. For example, you can get the size
of an array ([playground][play04]):

```go
pp(path.MustQuery("$.track.segments.size()", value))
```

``` json
[2]
```

More examples of using jsonpath operators and methods within path expressions
appear [below](#sqljson-path-operators-and-methods).

A path can also contain *filter* expressions that work similarly to the `WHERE`
clause in SQL. A filter expression begins with a question mark and provides a
condition in parentheses:

```
? (condition)
```

Filter expressions must be written just after the path evaluation step to
which they should apply. The result of that step is filtered to include only
those items that satisfy the provided condition. SQL/JSON defines three-valued
logic, so the condition can produce `true`, `false`, or `unknown`. The unknown
value plays the same role as SQL `NULL` and Go `nil` and can be tested for
with the `is unknown` predicate. Further path evaluation steps use only those
items for which the filter expression returned `true`.

The functions and operators that can be used in filter expressions are listed
[below](#filter-expression-elements). Within a filter expression, the `@`
variable denotes the value being considered (i.e., one result of the preceding
path step). You can write accessor operators after `@` to retrieve component
items.

For example, suppose you would like to retrieve all heart rate values higher
than 130. You can achieve this as follows ([playground][play05]):

```go
pp(path.MustQuery("$.track.segments[*].HR ? (@ > 130)", value))
```

``` json
[135]
```

To get the start times of segments with such values, you have to filter out
irrelevant segments before selecting the start times, so the filter expression
is applied to the previous step, and the path used in the condition is
different ([playground][play06]):

```go
pp(path.MustQuery(
    `$.track.segments[*] ? (@.HR > 130)."start time"`,
    value,
))
```

``` json
["2018-10-14 10:39:21"]
```

You can use several filter expressions in sequence, if required. The following
example selects start times of all segments that contain locations with
relevant coordinates and high heart rate values ([playground][play07]):

```go
pp(path.MustQuery(
    `$.track.segments[*] ? (@.location[1] < 13.4) ? (@.HR > 130)."start time"`,
    value,
))
```

```json
["2018-10-14 10:39:21"]
```

Using filter expressions at different nesting levels is also allowed. The
following example first filters all segments by location, and then returns
high heart rate values for these segments, if available ([playground][play08]):

```go
pp(path.MustQuery(
    `$.track.segments[*] ? (@.location[1] < 13.4).HR ? (@ > 130)`,
    value,
))
```

```json
[135]
```

You can also nest filter expressions within each other. This example returns
the size of the track if it contains any segments with high heart rate values,
or an empty sequence otherwise ([playground][play09]):

```go
pp(path.MustQuery(
    `$.track ? (exists(@.segments[*] ? (@.HR > 130))).segments.size()`,
    value,
))
```

```go
[2]
```

### Deviations From The SQL Standard

PostgreSQL's implementation of the SQL/JSON path language, and therefore also
this Go implementation, has the following deviations from the SQL/JSON
standard.

#### Boolean Predicate Check Expressions

As an extension to the SQL standard, a PostgreSQL path expression can be a
Boolean predicate, whereas the SQL standard allows predicates only within
filters. While SQL-standard path expressions return the relevant element(s) of
the queried JSON value, predicate check expressions return the single
three-valued JSON result of the predicate: `true`, `false`, or `nil`. For
example, we could write this SQL-standard filter expression
([playground][play10]):

```go
pp(path.MustQuery("$.track.segments ?(@[*].HR > 130)", value))
```

The result:

```json
[{"HR":135,"location":[47.706,13.2635],"start time":"2018-10-14 10:39:21"}]
```

The similar predicate check expression simply returns `true`, indicating that a
match exists ([playground][play11]):

```go
pp(path.MustQuery("$.track.segments[*].HR > 130", value))
```

```go
[true]
```

**Note:** PostgreSQL predicate check expressions require the `@@` operator,
while SQL-standard path expressions require the `@?` operator. Use the
`PgIndexOperator` method to pass the appropriate operator to PostgreSQL.

#### Regular Expression Interpretation

There are minor differences in the interpretation of regular expression
patterns used in `like_regex` filters, as described
[below](#sqljson-regular-expressions).

### Strict And Lax Modes

When you query JSON data, the path expression may not match the actual JSON
data structure. An attempt to access a non-existent member of an object or
element of an array is defined as a structural error. SQL/JSON path
expressions have two modes of handling structural errors:

*   lax (default) — the path engine implicitly adapts the queried data to
    the specified path. Any structural errors that cannot be fixed as
    described below are suppressed, producing no match.

*   strict — if a structural error occurs, an error is raised.

Lax mode facilitates matching of a JSON document and path expression when the
JSON data does not conform to the expected schema. If an operand does not
match the requirements of a particular operation, it can be automatically
wrapped as an SQL/JSON array, or unwrapped by converting its elements into an
SQL/JSON sequence before performing the operation. Also, comparison operators
and most methods automatically unwrap their operands in lax mode, so you can
compare SQL/JSON arrays out-of-the-box. An array of size 1 is considered equal
to its sole element. Automatic unwrapping is not performed when:

*   The path expression contains `type()` or `size()` methods that return the
    type and the number of elements in the array, respectively.

*   The queried JSON data contain nested arrays. In this case, only the
    outermost array is unwrapped, while all the inner arrays remain unchanged.
    Thus, implicit unwrapping can only go one level down within each path
    evaluation step.

For example, when querying the GPS data listed above, you can abstract from
the fact that it stores an array of segments when using lax mode
([playground][play12]):

```go
pp(path.MustQuery("lax $.track.segments.location", value))
```

``` json
[[47.763,13.4034],[47.706,13.2635]]
```

In strict mode, the specified path must exactly match the structure of the
queried JSON document, so using this path expression will cause an error
([playground][play13]):

```go
pp(path.MustQuery("strict $.track.segments.location", value))
```

``` text
panic: exec: jsonpath member accessor can only be applied to an object
```

To get the same result as in lax mode, you have to explicitly unwrap the
segments array ([playground][play14]):

```go
pp(path.MustQuery("strict $.track.segments[*].location", value))
```

``` json
[[47.763,13.4034],[47.706,13.2635]]
```

The unwrapping behavior of lax mode can lead to surprising results. For
instance, the following query using the `.**` accessor selects every `HR` value
twice ([playground][play15]):

```go
pp(path.MustQuery("lax $.**.HR", value))
```

``` go
[73,135,73,135]
```

This happens because the `.**` accessor selects both the segments array and
each of its elements, while the `.HR` accessor automatically unwraps arrays
when using lax mode. To avoid surprising results, we recommend using the `.**`
accessor only in strict mode. The following query selects each `HR` value just
once ([playground][play16]):

```go
pp(path.MustQuery("strict $.**.HR", value))
```

``` json
[73,135]
```

The unwrapping of arrays can also lead to unexpected results. Consider this
example, which selects all the location arrays ([playground][play17]):

```go
pp(path.MustQuery("lax $.track.segments[*].location", value))
```

``` json
[[47.763,13.4034],[47.706,13.2635]]
```

As expected it returns the full arrays. But applying a filter expression
causes the arrays to be unwrapped to evaluate each item, returning only the
items that match the expression ([playground][play18]):

```go
pp(path.MustQuery(
    "lax $.track.segments[*].location ?(@[*] > 15)",
    value,
))
```

``` json
[47.763,47.706]
```

This despite the fact that the full arrays are selected by the path
expression. Use strict mode to restore selecting the arrays
([playground][play19]):

```go
pp(path.MustQuery(
    "strict $.track.segments[*].location ?(@[*] > 15)",
    value,
))
```

``` json
[[47.763,13.4034],[47.706,13.2635]]
```

### SQL/JSON Path Operators And Methods

The list of operators and methods available in JSON path expressions. Note
that while the unary operators and methods can be applied to multiple values
resulting from a preceding path step, the binary operators (addition etc.) can
only be applied to single values. In lax mode, methods applied to an array
will be executed for each value in the array. The exceptions are `.type()` and
`.size()`, which apply to the array itself.

**Note:** The examples below use this utility function to marshall JSON
arguments:

``` go
func val(src string) any {
    var value any
    if err := json.Unmarshal([]byte(src), &value); err != nil {
        log.Fatal(err)
    }
    return value
}
```

#### `number + number → number`

Addition ([playground][play20]):

``` go
pp(path.MustQuery("$[0] + 3", val("2"))) // → [5]
```

#### `+ number → number`

Unary plus (no operation); unlike addition, this can iterate over multiple
values ([playground][play21]):

``` go
pp(path.MustQuery("+ $.x", val(`{"x": [2,3,4]}`))) // → [2, 3, 4]
```

#### `number - number → number`

Subtraction ([playground][play22]):

``` go
pp(path.MustQuery("7 - $[0]", val("[2]"))) // → [5]
```

#### `- number → number`

Negation; unlike subtraction, this can iterate over multiple values
([playground][play23]):

``` go
pp(path.MustQuery("- $.x", val(`{"x": [2,3,4]}`))) // → [-2,-3,-4]
```

#### `number * number → number`

Multiplication ([playground][play24]):

``` go
pp(path.MustQuery("2 * $[0]", val("4"))) // → [8]
```
#### `number / number → number`

Division ([playground][play25]):

``` go
pp(path.MustQuery("$[0] / 2", val("[8.5]"))) // → [4.25]
```

#### `number % number → number`

Modulo (remainder) ([playground][play26]):

``` go
pp(path.MustQuery("$[0] % 10", val("[32]"))) // → [2]
```

#### `value . type() → string`

Type of the JSON item ([playground][play27]):

``` go
pp(path.MustQuery("$[*].type()", val(`[1, "2", {}]`))) // → ["number","string","object"]
```

#### `value . size() → number`

Size of the JSON item (number of array elements, or 1 if not an array;
[playground][play28]):

``` go
pp(path.MustQuery("$.m.size()", val(`{"m": [11, 15]}`))) // → [2]
```

#### `value . boolean() → boolean`

Boolean value converted from a JSON boolean, number, or string
([playground][play29]):

``` go
pp(path.MustQuery("$[*].boolean()", val(`[1, "yes", false]`))) // → [true,true,false]
```

#### `value . string() → string`

String value converted from a JSON boolean, number, string, or datetime
([playground][play30], [playground][play31]):

``` go
pp(path.MustQuery("$[*].string()", val(`[1.23, "xyz", false]`)))    // → ["1.23","xyz","false"]
pp(path.MustQuery("$.timestamp().string()", "2023-08-15 12:34:56")) // → ["2023-08-15T12:34:56"]
```

#### `value . double() → number`

Approximate floating-point number converted from a JSON number or string
([playground][play32]):

``` go
pp(path.MustQuery(" ", val(`{"len": "1.9"}`))) // → [3.8]
```

#### `number . ceiling() → number`

Nearest integer greater than or equal to the given number
([playground][play33]):

``` go
pp(path.MustQuery("$.h.ceiling()", val(`{"h": 1.3}`))) // → [2]
```

#### `number . floor() → number`

Nearest integer less than or equal to the given number ([playground][play34]):

``` go
pp(path.MustQuery("$.h.floor()", val(`{"h": 1.7}`))) // → [1]
```

#### `number . abs() → number`

Absolute value of the given number ([playground][play35]):

``` go
pp(path.MustQuery("$.z.abs()", val(`{"z": -0.3}`))) // → [0.3]
```

#### `value . bigint() → bigint`

Big integer value converted from a JSON number or string
([playground][play36]):

``` go
pp(path.MustQuery("$.len.bigint()", val(`{"len": "9876543219"}`))) // → [9876543219]
```

#### `value . decimal( [ precision [ , scale ] ] ) → decimal`

Rounded decimal value converted from a JSON number or string. Precision and
scale must be integer values ([playground][play37]):

``` go
pp(path.MustQuery("$.decimal(6, 2)", val("1234.5678"))) // → [1234.57]
```

#### `value . integer() → integer`

Integer value converted from a JSON number or string ([playground][play38]):

``` go
pp(path.MustQuery("$.len.integer()", val(`{"len": "12345"}`))) // → [12345]
```

#### `value . number() → numeric`

Numeric value converted from a JSON number or string ([playground][play39]):

``` go
pp(path.MustQuery("$.len.number()", val(`{"len": "123.45"}`))) // → [123.45]
```

#### `string . datetime() → types.DateTime`

Date/time value converted from a string ([playground][play40]):

``` go
pp(path.MustQuery(
    `$[*] ? (@.datetime() < "2015-08-02".datetime())`,
    val(`["2015-08-01", "2015-08-12"]`),
)) // → ["2015-8-01"]
```

#### `string . datetime(template) → types.DateTime`

Date/time value converted from a string using the specified to_timestamp
template.

**NOTE:** Currently unimplemented, raises an error ([playground][play41]):

``` go
pp(path.MustQuery(
    `$[*].datetime("HH24:MI")`, val(`["12:30", "18:40"]`),
)) // → panic: exec: .datetime(template) is not yet supported
```

#### `string . date() → types.Date`

Date value converted from a string ([playground][play42]):

``` go
pp(path.MustQuery("$.date()", "2023-08-15")) // → ["2023-08-15"]
```

#### `string . time() → types.Time`

Time without time zone value converted from a string ([playground][play43]):

``` go
pp(path.MustQuery("$.time()", "12:34:56")) // → ["12:34:56"]
```

#### `string . time(precision) → types.Time`

Time without time zone value converted from a string, with fractional seconds
adjusted to the given precision ([playground][play44]):

``` go
pp(path.MustQuery("$.time(2)", "12:34:56.789")) // → ["12:34:56.79"]
```

#### `string . time_tz() → types.TimeTZ`

Time with time zone value converted from a string ([playground][play45]):

``` go
pp(path.MustQuery("$.time_tz()", "12:34:56+05:30")) // → ["12:34:56+05:30"]
```

#### `string . time_tz(precision) → types.TimeTZ`

Time with time zone value converted from a string, with fractional seconds
adjusted to the given precision ([playground][play46]):

``` go
pp(path.MustQuery("$.time_tz(2)", "12:34:56.789+05:30")) // → ["12:34:56.79+05:30"]
```

#### `string . timestamp() → types.Timestamp`

Timestamp without time zone value converted from a string ([playground][play47]):

``` go
pp(path.MustQuery("$.timestamp()", "2023-08-15 12:34:56")) // → ["2023-08-15T12:34:56"]
```

#### `string . timestamp(precision) → types.Timestamp`

Timestamp without time zone value converted from a string, with fractional
seconds adjusted to the given precision ([playground][play48]):

``` go
arg := "2023-08-15 12:34:56.789"
pp(path.MustQuery("$.timestamp(2)", arg)) // → ["2023-08-15T12:34:56.79"]
```

#### `string . timestamp_tz() → types.TimestampTZ`

Timestamp with time zone value converted from a string ([playground][play49]):

``` go
arg := "2023-08-15 12:34:56+05:30"
pp(path.MustQuery("$.timestamp_tz()", arg)) // → ["2023-08-15T12:34:56+05:30"]
```

#### `string . timestamp_tz(precision) → types.TimestampTZ`

Timestamp with time zone value converted from a string, with fractional
seconds adjusted to the given precision ([playground][play50]):

``` go
arg := "2023-08-15 12:34:56.789+05:30"
pp(path.MustQuery("$.timestamp_tz(2)", arg)) // → ["2023-08-15T12:34:56.79+05:30"]
```

#### `object . keyvalue() → []map[string]any`

The object's key-value pairs, represented as an array of objects containing
three fields: "key", "value", and "id"; "id" is a unique identifier of the
object the key-value pair belongs to ([playground][play51]):

``` go
pp(path.MustQuery("$.keyvalue()", val(`{"x": "20", "y": 32}`)))
// → [{"id":0,"key":"x","value":"20"},{"id":0,"key":"y","value":32}]
```

### Filter Expression Elements

The filter expression elements available in JSON path.

#### `value == value → boolean`

Equality comparison (this, and the other comparison operators, work on all
JSON scalar values; [playground][play52], [playground][play53]):

``` go
pp(path.MustQuery("$[*] ? (@ == 1)", val(`[1, "a", 1, 3]`)))   // → [1,1]
pp(path.MustQuery(`$[*] ? (@ == "a")`, val(`[1, "a", 1, 3]`))) // → ["a"]
```

#### `value != value → boolean`

#### `value <> value → boolean`

Non-equality comparison ([playground][play54], [playground][play55]):

``` go
pp(path.MustQuery("$[*] ? (@ != 1)", val(`[1, 2, 1, 3]`)))      // → [2,3]
pp(path.MustQuery(`$[*] ? (@ <> "b")`, val(`["a", "b", "c"]`))) // → ["a","c"]
```

#### `value < value → boolean`

Less-than comparison ([playground][play56]):

``` go
pp(path.MustQuery("$[*] ? (@ < 2)", val(`[1, 2, 3]`))) // → [1]
```

#### `value <= value → boolean`

Less-than-or-equal-to comparison ([playground][play57]):

``` go
pp(path.MustQuery(`$[*] ? (@ <= "b")`, val(`["a", "b", "c"]`))) // → ["a","b"]
```

#### `value > value → boolean`

Greater-than comparison ([playground][play58]):

``` go
pp(path.MustQuery("$[*] ? (@ > 2)", val(`[1, 2, 3]`))) // → [3]
```

#### `value >= value → boolean`

Greater-than-or-equal-to comparison ([playground][play59]):

``` go
pp(path.MustQuery("$[*] ? (@ >= 2)", val(`[1, 2, 3]`))) // → [2,3]
```

#### `true → boolean`

JSON constant true ([playground][play60]):

``` go
arg := val(`[
  {"name": "John", "parent": false},
  {"name": "Chris", "parent": true}
]`)
pp(path.MustQuery("$[*] ? (@.parent == true)", arg)) // → [{"name":"Chris","parent":true}]
```

#### `false → boolean`

JSON constant false ([playground][play61]):

``` go
arg := val(`[
  {"name": "John", "parent": false},
  {"name": "Chris", "parent": true}
]`)
pp(path.MustQuery("$[*] ? (@.parent == false)", arg)) // → [{"name":"John","parent":false}]
```

#### `null → value`

JSON constant null (note that, unlike in SQL, comparison to null works
normally; [playground][play62]):

``` go
arg := val(`[
  {"name": "Mary", "job": null},
  {"name": "Michael", "job": "driver"}
]`)
pp(path.MustQuery("$[*] ? (@.job == null) .name", arg)) // → ["Mary"]
```

#### `boolean && boolean → boolean`

Boolean `AND` ([playground][play63]):

``` go
pp(path.MustQuery("$[*] ? (@ > 1 && @ < 5)", val(`[1, 3, 7]`))) // → [3]
```

#### `boolean || boolean → boolean`

Boolean `OR` ([playground][play64]):

``` go
pp(path.MustQuery("$[*] ? (@ < 1 || @ > 5)", val(`[1, 3, 7]`))) // → [7]
```

#### `! boolean → boolean`

Boolean `NOT` ([playground][play65]):

``` go
pp(path.MustQuery("$[*] ? (!(@ < 5))", val(`[1, 3, 7]`))) // → [7]
```

#### `boolean is unknown → boolean`

Tests whether a Boolean condition is unknown ([playground][play66]):

``` go
pp(path.MustQuery("$[*] ? ((@ > 0) is unknown)", val(`[-1, 2, 7, "foo"]`))) // → ["foo"]
```

#### `string like_regex string [ flag string ] → boolean`

Tests whether the first operand matches the regular expression given by the
second operand, optionally with modifications described by a string of flag
characters (see [SQL/JSON Regular Expressions](#sqljson-regular-expressions);
[playground][play67], [playground][play68]):

``` go
arg := val(`["abc", "abd", "aBdC", "abdacb", "babc"]`)
pp(path.MustQuery(`$[*] ? (@ like_regex "^ab.*c")`, arg))          // → ["abc","abdacb"]
pp(path.MustQuery(`$[*] ? (@ like_regex "^ab.*c" flag "i")`, arg)) // → ["abc","aBdC","abdacb"]
```

#### `string starts with string → boolean`

Tests whether the second operand is an initial substring of the first operand
([playground][play69]):

``` go
arg := val(`["John Smith", "Mary Stone", "Bob Johnson"]`)
pp(path.MustQuery(`$[*] ? (@ starts with "John")`, arg)) // → ["John Smith"]
```

#### `exists ( path_expression ) → boolean`

Tests whether a path expression matches at least one SQL/JSON item. Returns
unknown if the path expression would result in an error; the second example
uses this to avoid a no-such-key error in strict mode
([playground][play70], [playground][play71]):

``` go
arg := val(`{"x": [1, 2], "y": [2, 4]}`)
pp(path.MustQuery("strict $.* ? (exists (@ ? (@[*] > 2)))", arg))              // → [[2,4]]
pp(path.MustQuery("strict $ ? (exists (@.name)) .name", val(`{"value": 42}`))) // → []
```

### SQL/JSON Regular Expressions

SQL/JSON path expressions allow matching text to a regular expression with the
`like_regex` filter. For example, the following SQL/JSON path query would
case-insensitively match all strings in an array that start with an English
vowel:

```jsonpath
$[*] ? (@ like_regex "^[aeiou]" flag "i")
```

The optional `flag` string may include one or more of the characters `i` for
case-insensitive match, `m` to allow `^` and `$` to match at newlines, `s` to
allow `.` to match a newline, and `q` to quote the whole pattern (reducing the
behavior to a simple substring match).

The SQL/JSON standard borrows its definition for regular expressions from the
`LIKE_REGEX` operator, which in turn uses the XQuery standard. The path
package follows the example of PostgreSQL, using the [regexp] package to
implement `like_regex`. This leads to various minor discrepancies from
standard SQL/JSON behavior, which are cataloged in [Differences From SQL
Standard And XQuery]. Note, however, that the flag-letter incompatibilities
described there do not apply to SQL/JSON, as it translates the XQuery flag
letters to match what the [regexp] package expects.

There are also variations between PostgreSQL regular expression syntax and go
regular expression syntax, cataloged [below](#compatibility).

Keep in mind that the pattern argument of `like_regex` is a JSON path string
literal, written according to the rules given [above](#syntax). This means in
particular that any backslashes in the regular expression must be doubled in
double-quoted strings. For example, to match string values of the root
document that contain only digits:

``` go
p := path.MustParse("$.* ?(@ like_regex \"^\\\\d+$\")")
pp(p.MustQuery(context.Background(), val(`{"x": "42", "y": "no"}`))) // → ["42"]
```

This doubling upon doubling is required to escape backslashes once for go
parsing and a second time for JSON path string parsing.

We therefore recommend using raw [string literals] (backtick strings) to
compose path expressions with double quotes or backslashes, both of which are
common in `like_regex` expressions. Raw strings require double backslashes in
regular expressions only once, for the path string parsing
([playground][play72]):

``` go
p := path.MustParse(`$.* ?(@ like_regex "^\\d+$")`)
pp(p.MustQuery(context.Background(), val(`{"x": "42", "y": "no"}`))) // → ["42"]
```

## Compatibility

As a direct port from the Postgres source, the path package strives to
maintain the highest level of compatibility. Still, there remain some
unavoidable differences and to-dos. These include:

*   Numbers. The Postgres [JSONB] type implements numbers as [arbitrary
    precision numbers]. This contrasts with Go JSON parsing, which by default
    parses numbers into `float64` values. Decimal numbers outside the range of
    `float64` are not supported and will trigger an error. For numbers within
    `float64` range, warnings about the precision of [floating point math]
    apply.

    For [json.Number]s, however, the path package first attempts to treat them
    as `int64` values and falls back on `float64` only if all the values in an
    expression cannot be parsed as integers. This increases precision for
    integer-only expressions. We therefore recommend parsing JSON with
    [json.Decoder.UseNumber].

    This incompatibility may be addressed in the future, perhaps by using
    [decimal] for all numeric operations.

*   `datetime(template)`. The `datetime()` method has been implemented, but
    `datetime(template)` has not. Use of the template parameter will raise an
    error. This issue will likely be addressed in a future release.

*   Date and time parsing. The path package relies uses the [time] packages's
    [layouts] to parse values in the datetime methods (`datetime()`,
    `timestamp()`, `timestamp_tz()`, etc.). These layouts are stricter about
    the formats they'll parse than [Postgres date/time formatting].

    As a result, some values parsed by the Postgres datetime methods will not
    be parsed by this package. Examples include values with extra spaces
    between the time and time zone, and missing leading zeros on the day and
    month.

    This issue will likely be addressed when the `datetime(template)` method
    is implemented, as it will require adopting the full Postgres date/time
    formatting language.

*   Time zones. Postgres operates on time and time values in the context of
    the time zone defined by the [TimeZone GUC] or the server's system time
    zone. The path package does not rely on such global configuration. It
    instead uses the time zone configured in the context passed by the path
    queries ([playground][play73]), and defaults to UTC if it's not set or
    included in the value ([playground][play74]):

    ```go
    p := path.MustParse("$.timestamp_tz()")
    arg := "2023-08-15 12:34:56"
    pp(p.MustQuery(context.Background(), arg, exec.WithTZ())) // → ["2023-08-15T12:34:56+00:00"]

    // Add a time zone to the context.
    tz, err := time.LoadLocation("America/New_York")
    if err != nil {
    	log.Fatal(err)
    }
    ctx := types.ContextWithTZ(context.Background(), tz)

    // The output will now be in the custom time zone.
    pp(p.MustQuery(ctx, arg, exec.WithTZ())) // → ["2023-08-15T12:34:56-04:00"]
    ```

*   Regular expressions. Whereas the Postgres implementation of the `like_regex`
    expression relies on its [POSIX regular expression engine], the Go version
    relies on the [regexp] package. We have attempted to configure things for
    full compatibility with the Postgres implementation (including the same
    diversions from XQuery regular expressions), but some variation is likely.

    Notably, a number of escapes and character classes vary:

    | Escape       | PostgresSQL                           | Go                                    |
    | ------------ | ------------------------------------- | ------------------------------------- |
    | `\a`         | alert (bell) character                | alert (bell) character                |
    | `\A`         | at beginning of text                  | at beginning of text                  |
    | `\b`         | backspace                             | at ASCII word boundary                |
    | `\B`         | synonym for backslash (`\`)           | not at ASCII word boundary            |
    | `\cX`        | low-order 5 bits comparison           | N/A                                   |
    | `\d`         | digit                                 | digit                                 |
    | `\D`         | non-digit                             | non-digit                             |
    | `\e`         | `ESC` or octal `033`                  | N/A                                   |
    | `\f`         | form feed                             | form feed                             |
    | `\m`         | beginning of a word                   | N/A                                   |
    | `\M`         | end of a word                         | N/A                                   |
    | `\n`         | newline                               | newline                               |
    | `\Q...\E`    | N/A                                   | literal `...`                         |
    | `\r`         | carriage return                       | carriage return                       |
    | `\s`         | whitespace character                  | whitespace character                  |
    | `\S`         | non-whitespace character              | non-whitespace character              |
    | `\t`         | horizontal tab                        | horizontal tab                        |
    | `\uwxyz`     | character with hex value `0xwxyz`     | N/A (see `\x{}`)                      |
    | `\Ustuvwxyz` | character with hex value `0xstuvwxyz` | N/A (see `\x{}`)                      |
    | `\v`         | vertical tab                          | vertical tab                          |
    | `\w`         | word character                        | word character                        |
    | `\W`         | non-word character                    | non-word character                    |
    | `\xhhh`      | character with hex value `0xhhh`      | character with hex value `0xhhh`      |
    | `\xy`        | character with octal value `0xy`      | N/A                                   |
    | `\x{10FFFF}` | N/A (see `\U`)                        | hex character code                    |
    | `\y`         | beginning or end of a word            | N/A (see `\b`)                        |
    | `\Y`         | not the beginning or end of a word    | N/A (see `\B`)                        |
    | `\z`         | N/A (see `\Z`)                        | end of text                           |
    | `\Z`         | end of text                           | N/A (see `\z`)                        |
    | `\0`         | the null byte                         | N/A                                   |
    | `\*`         | literal punctuation character `*`     | literal `*` punctuation character `*` |

*   Identifiers. Postgres jsonpath parsing is quite liberal in what it allows
    in unquoted identifiers. The allowed characters are defined by the
    [ECMAScript standard] are stricter, and this package hews closer to the
    standard.

    The upshot is that expressions allowed by Postgres, such as `x.🎉`, are
    better written as `x."🎉"` for compatibility with the standard and to work
    with both this package and Postgres.

*   `keyvalue()` IDs. Postgres creates IDs for the output of the `keyvalue()`
    method by comparing memory addresses between JSONB values. This works well
    for JSONB because it has a highly-structured, well-ordered layout. The
    path package follows this pattern.

    However, The addresses of nested `map[string]any` and `[]any` values in Go
    are less stable. Ids will therefore sometimes vary between executions —
    especially for slices. However, the IDs determined for a single object or
    array should be stable through repeated query executions and calls to
    `keyvalue()`.

## Copyright

Copyright © 1996-2025 The PostgreSQL Global Development Group

Copyright © 2024-2025 David E. Wheeler

  [🛝 Playground]: https://theory.github.io/sqljson "Go SQL/JSON Path Playground"
  [TinyGo]: https://tinygo.org
  [Wasm]: https://webassembly.org "WebAssembly"
  [this one]: https://theory.github.io/sqljson/?p=%2524.track.segments%255B*%255D.location&j=%257B%250A%2520%2520%2522track%2522%253A%2520%257B%250A%2520%2520%2520%2520%2522segments%2522%253A%2520%255B%250A%2520%2520%2520%2520%2520%2520%257B%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522location%2522%253A%2520%2520%2520%255B%252047.763%252C%252013.4034%2520%255D%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522start%2520time%2522%253A%2520%25222018-10-14%252010%253A05%253A14%2522%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522HR%2522%253A%252073%250A%2520%2520%2520%2520%2520%2520%257D%252C%250A%2520%2520%2520%2520%2520%2520%257B%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522location%2522%253A%2520%2520%2520%255B%252047.706%252C%252013.2635%2520%255D%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522start%2520time%2522%253A%2520%25222018-10-14%252010%253A39%253A21%2522%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522HR%2522%253A%2520135%250A%2520%2520%2520%2520%2520%2520%257D%250A%2520%2520%2520%2520%255D%250A%2520%2520%257D%250A%257D&a=&o=1
  [PostgreSQL docs]: https://www.postgresql.org/docs/devel/functions-json.html#FUNCTIONS-SQLJSON-PATH
    "PostgreSQL Documentation: “The SQL/JSON Path Language”"
  [`json.Number`]: https://pkg.go.dev/encoding/json#Number
  [string literals]: https://go.dev/ref/spec#String_literals
    "Go Language Spec: String literals"
  [regexp]: https://pkg.go.dev/regexp "Go Standard Library: regexp"
  [Differences From SQL Standard And XQuery]: https://www.postgresql.org/docs/devel/functions-matching.html#POSIX-VS-XQUERY
    "PostgreSQL Documentation: “Differences From SQL Standard And XQuery”"
  [JSONB]: https://www.postgresql.org/docs/current/datatype-json.html
  [arbitrary precision numbers]: https://www.postgresql.org/docs/current/datatype-numeric.html#DATATYPE-NUMERIC-DECIMAL
  [floating point math]: https://en.wikipedia.org/wiki/Floating-point_arithmetic
  [json.Number]: https://pkg.go.dev/encoding/json#Number
  [json.Decoder.UseNumber]: https://pkg.go.dev/encoding/json#Decoder.UseNumber
  [decimal]: https://pkg.go.dev/github.com/shopspring/decimal
  [time]: https://pkg.go.dev/time
  [layouts]: https://pkg.go.dev/time#pkg-constants
  [Postgres date/time formatting]: https://www.postgresql.org/docs/current/functions-formatting.html
  [ECMAScript standard]: https://262.ecma-international.org/#sec-identifier-names
  [POSIX regular expression engine]: https://www.postgresql.org/docs/devel/functions-matching.html#FUNCTIONS-POSIX-REGEXP
  [regexp]: https://pkg.go.dev/regexp
  [backspace character]: https://en.wikipedia.org/wiki/Backspace
  [TimeZone GUC]: https://www.postgresql.org/docs/current/runtime-config-client.html#GUC-TIMEZONE
  [types.ContextWithTZ]: https://pkg.go.dev/github.com/theory/sqljson/path/types#ContextWithTZ
  [output format]: https://www.postgresql.org/docs/current/datatype-datetime.html#DATATYPE-DATETIME-OUTPUT

  <!-- Playground Links -->
  [play01]: https://theory.github.io/sqljson/?p=%2524.track.segments&j=%257B%250A%2520%2520%2522track%2522%253A%2520%257B%250A%2520%2520%2520%2520%2522segments%2522%253A%2520%255B%250A%2520%2520%2520%2520%2520%2520%257B%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522location%2522%253A%2520%2520%2520%255B%252047.763%252C%252013.4034%2520%255D%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522start%2520time%2522%253A%2520%25222018-10-14%252010%253A05%253A14%2522%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522HR%2522%253A%252073%250A%2520%2520%2520%2520%2520%2520%257D%252C%250A%2520%2520%2520%2520%2520%2520%257B%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522location%2522%253A%2520%2520%2520%255B%252047.706%252C%252013.2635%2520%255D%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522start%2520time%2522%253A%2520%25222018-10-14%252010%253A39%253A21%2522%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522HR%2522%253A%2520135%250A%2520%2520%2520%2520%2520%2520%257D%250A%2520%2520%2520%2520%255D%250A%2520%2520%257D%250A%257D&a=&o=33
  [play02]: https://theory.github.io/sqljson/?p=%2524.track.segments%255B*%255D.location&j=%257B%250A%2520%2520%2522track%2522%253A%2520%257B%250A%2520%2520%2520%2520%2522segments%2522%253A%2520%255B%250A%2520%2520%2520%2520%2520%2520%257B%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522location%2522%253A%2520%2520%2520%255B%252047.763%252C%252013.4034%2520%255D%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522start%2520time%2522%253A%2520%25222018-10-14%252010%253A05%253A14%2522%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522HR%2522%253A%252073%250A%2520%2520%2520%2520%2520%2520%257D%252C%250A%2520%2520%2520%2520%2520%2520%257B%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522location%2522%253A%2520%2520%2520%255B%252047.706%252C%252013.2635%2520%255D%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522start%2520time%2522%253A%2520%25222018-10-14%252010%253A39%253A21%2522%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522HR%2522%253A%2520135%250A%2520%2520%2520%2520%2520%2520%257D%250A%2520%2520%2520%2520%255D%250A%2520%2520%257D%250A%257D&a=&o=1
  [play03]: https://theory.github.io/sqljson/?p=%2524.track.segments%255B0%255D.location&j=%257B%250A%2520%2520%2522track%2522%253A%2520%257B%250A%2520%2520%2520%2520%2522segments%2522%253A%2520%255B%250A%2520%2520%2520%2520%2520%2520%257B%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522location%2522%253A%2520%2520%2520%255B%252047.763%252C%252013.4034%2520%255D%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522start%2520time%2522%253A%2520%25222018-10-14%252010%253A05%253A14%2522%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522HR%2522%253A%252073%250A%2520%2520%2520%2520%2520%2520%257D%252C%250A%2520%2520%2520%2520%2520%2520%257B%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522location%2522%253A%2520%2520%2520%255B%252047.706%252C%252013.2635%2520%255D%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522start%2520time%2522%253A%2520%25222018-10-14%252010%253A39%253A21%2522%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522HR%2522%253A%2520135%250A%2520%2520%2520%2520%2520%2520%257D%250A%2520%2520%2520%2520%255D%250A%2520%2520%257D%250A%257D&a=&o=1
  [play04]: https://theory.github.io/sqljson/?p=%2524.track.segments.size%28%29&j=%257B%250A%2520%2520%2522track%2522%253A%2520%257B%250A%2520%2520%2520%2520%2522segments%2522%253A%2520%255B%250A%2520%2520%2520%2520%2520%2520%257B%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522location%2522%253A%2520%2520%2520%255B%252047.763%252C%252013.4034%2520%255D%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522start%2520time%2522%253A%2520%25222018-10-14%252010%253A05%253A14%2522%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522HR%2522%253A%252073%250A%2520%2520%2520%2520%2520%2520%257D%252C%250A%2520%2520%2520%2520%2520%2520%257B%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522location%2522%253A%2520%2520%2520%255B%252047.706%252C%252013.2635%2520%255D%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522start%2520time%2522%253A%2520%25222018-10-14%252010%253A39%253A21%2522%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522HR%2522%253A%2520135%250A%2520%2520%2520%2520%2520%2520%257D%250A%2520%2520%2520%2520%255D%250A%2520%2520%257D%250A%257D&a=&o=1
  [play05]: https://theory.github.io/sqljson/?p=%2524.track.segments%255B*%255D.HR%2520%253F%2520%28%2540%2520%253E%2520130%29&j=%257B%250A%2520%2520%2522track%2522%253A%2520%257B%250A%2520%2520%2520%2520%2522segments%2522%253A%2520%255B%250A%2520%2520%2520%2520%2520%2520%257B%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522location%2522%253A%2520%2520%2520%255B%252047.763%252C%252013.4034%2520%255D%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522start%2520time%2522%253A%2520%25222018-10-14%252010%253A05%253A14%2522%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522HR%2522%253A%252073%250A%2520%2520%2520%2520%2520%2520%257D%252C%250A%2520%2520%2520%2520%2520%2520%257B%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522location%2522%253A%2520%2520%2520%255B%252047.706%252C%252013.2635%2520%255D%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522start%2520time%2522%253A%2520%25222018-10-14%252010%253A39%253A21%2522%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522HR%2522%253A%2520135%250A%2520%2520%2520%2520%2520%2520%257D%250A%2520%2520%2520%2520%255D%250A%2520%2520%257D%250A%257D&a=&o=1
  [play06]: https://theory.github.io/sqljson/?p=%2524.track.segments%255B*%255D%2520%253F%2520%28%2540.HR%2520%253E%2520130%29.%2522start%2520time%2522&j=%257B%250A%2520%2520%2522track%2522%253A%2520%257B%250A%2520%2520%2520%2520%2522segments%2522%253A%2520%255B%250A%2520%2520%2520%2520%2520%2520%257B%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522location%2522%253A%2520%2520%2520%255B%252047.763%252C%252013.4034%2520%255D%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522start%2520time%2522%253A%2520%25222018-10-14%252010%253A05%253A14%2522%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522HR%2522%253A%252073%250A%2520%2520%2520%2520%2520%2520%257D%252C%250A%2520%2520%2520%2520%2520%2520%257B%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522location%2522%253A%2520%2520%2520%255B%252047.706%252C%252013.2635%2520%255D%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522start%2520time%2522%253A%2520%25222018-10-14%252010%253A39%253A21%2522%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522HR%2522%253A%2520135%250A%2520%2520%2520%2520%2520%2520%257D%250A%2520%2520%2520%2520%255D%250A%2520%2520%257D%250A%257D&a=&o=1
  [play07]: https://theory.github.io/sqljson/?p=%2524.track.segments%255B*%255D%2520%253F%2520%28%2540.location%255B1%255D%2520%253C%252013.4%29%2520%253F%2520%28%2540.HR%2520%253E%2520130%29.%2522start%2520time%2522&j=%257B%250A%2520%2520%2522track%2522%253A%2520%257B%250A%2520%2520%2520%2520%2522segments%2522%253A%2520%255B%250A%2520%2520%2520%2520%2520%2520%257B%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522location%2522%253A%2520%2520%2520%255B%252047.763%252C%252013.4034%2520%255D%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522start%2520time%2522%253A%2520%25222018-10-14%252010%253A05%253A14%2522%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522HR%2522%253A%252073%250A%2520%2520%2520%2520%2520%2520%257D%252C%250A%2520%2520%2520%2520%2520%2520%257B%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522location%2522%253A%2520%2520%2520%255B%252047.706%252C%252013.2635%2520%255D%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522start%2520time%2522%253A%2520%25222018-10-14%252010%253A39%253A21%2522%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522HR%2522%253A%2520135%250A%2520%2520%2520%2520%2520%2520%257D%250A%2520%2520%2520%2520%255D%250A%2520%2520%257D%250A%257D&a=&o=1
  [play08]: https://theory.github.io/sqljson/?p=%2524.track.segments%255B*%255D%2520%253F%2520%28%2540.location%255B1%255D%2520%253C%252013.4%29.HR%2520%253F%2520%28%2540%2520%253E%2520130%29&j=%257B%250A%2520%2520%2522track%2522%253A%2520%257B%250A%2520%2520%2520%2520%2522segments%2522%253A%2520%255B%250A%2520%2520%2520%2520%2520%2520%257B%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522location%2522%253A%2520%2520%2520%255B%252047.763%252C%252013.4034%2520%255D%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522start%2520time%2522%253A%2520%25222018-10-14%252010%253A05%253A14%2522%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522HR%2522%253A%252073%250A%2520%2520%2520%2520%2520%2520%257D%252C%250A%2520%2520%2520%2520%2520%2520%257B%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522location%2522%253A%2520%2520%2520%255B%252047.706%252C%252013.2635%2520%255D%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522start%2520time%2522%253A%2520%25222018-10-14%252010%253A39%253A21%2522%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522HR%2522%253A%2520135%250A%2520%2520%2520%2520%2520%2520%257D%250A%2520%2520%2520%2520%255D%250A%2520%2520%257D%250A%257D&a=&o=1
  [play09]: https://theory.github.io/sqljson/?p=%2524.track%2520%253F%2520%28exists%28%2540.segments%255B*%255D%2520%253F%2520%28%2540.HR%2520%253E%2520130%29%29%29.segments.size%28%29&j=%257B%250A%2520%2520%2522track%2522%253A%2520%257B%250A%2520%2520%2520%2520%2522segments%2522%253A%2520%255B%250A%2520%2520%2520%2520%2520%2520%257B%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522location%2522%253A%2520%2520%2520%255B%252047.763%252C%252013.4034%2520%255D%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522start%2520time%2522%253A%2520%25222018-10-14%252010%253A05%253A14%2522%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522HR%2522%253A%252073%250A%2520%2520%2520%2520%2520%2520%257D%252C%250A%2520%2520%2520%2520%2520%2520%257B%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522location%2522%253A%2520%2520%2520%255B%252047.706%252C%252013.2635%2520%255D%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522start%2520time%2522%253A%2520%25222018-10-14%252010%253A39%253A21%2522%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522HR%2522%253A%2520135%250A%2520%2520%2520%2520%2520%2520%257D%250A%2520%2520%2520%2520%255D%250A%2520%2520%257D%250A%257D&a=&o=1
  [play10]: https://theory.github.io/sqljson/?p=%2524.track.segments%2520%253F%28%2540%255B*%255D.HR%2520%253E%2520130%29&j=%257B%250A%2520%2520%2522track%2522%253A%2520%257B%250A%2520%2520%2520%2520%2522segments%2522%253A%2520%255B%250A%2520%2520%2520%2520%2520%2520%257B%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522location%2522%253A%2520%2520%2520%255B%252047.763%252C%252013.4034%2520%255D%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522start%2520time%2522%253A%2520%25222018-10-14%252010%253A05%253A14%2522%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522HR%2522%253A%252073%250A%2520%2520%2520%2520%2520%2520%257D%252C%250A%2520%2520%2520%2520%2520%2520%257B%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522location%2522%253A%2520%2520%2520%255B%252047.706%252C%252013.2635%2520%255D%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522start%2520time%2522%253A%2520%25222018-10-14%252010%253A39%253A21%2522%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522HR%2522%253A%2520135%250A%2520%2520%2520%2520%2520%2520%257D%250A%2520%2520%2520%2520%255D%250A%2520%2520%257D%250A%257D&a=&o=33
  [play11]: https://theory.github.io/sqljson/?p=%2524.track.segments%255B*%255D.HR%2520%253E%2520130&j=%257B%250A%2520%2520%2522track%2522%253A%2520%257B%250A%2520%2520%2520%2520%2522segments%2522%253A%2520%255B%250A%2520%2520%2520%2520%2520%2520%257B%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522location%2522%253A%2520%2520%2520%255B%252047.763%252C%252013.4034%2520%255D%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522start%2520time%2522%253A%2520%25222018-10-14%252010%253A05%253A14%2522%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522HR%2522%253A%252073%250A%2520%2520%2520%2520%2520%2520%257D%252C%250A%2520%2520%2520%2520%2520%2520%257B%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522location%2522%253A%2520%2520%2520%255B%252047.706%252C%252013.2635%2520%255D%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522start%2520time%2522%253A%2520%25222018-10-14%252010%253A39%253A21%2522%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522HR%2522%253A%2520135%250A%2520%2520%2520%2520%2520%2520%257D%250A%2520%2520%2520%2520%255D%250A%2520%2520%257D%250A%257D&a=&o=1
  [play12]: https://theory.github.io/sqljson/?p=lax%2520%2524.track.segments.location&j=%257B%250A%2520%2520%2522track%2522%253A%2520%257B%250A%2520%2520%2520%2520%2522segments%2522%253A%2520%255B%250A%2520%2520%2520%2520%2520%2520%257B%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522location%2522%253A%2520%2520%2520%255B%252047.763%252C%252013.4034%2520%255D%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522start%2520time%2522%253A%2520%25222018-10-14%252010%253A05%253A14%2522%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522HR%2522%253A%252073%250A%2520%2520%2520%2520%2520%2520%257D%252C%250A%2520%2520%2520%2520%2520%2520%257B%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522location%2522%253A%2520%2520%2520%255B%252047.706%252C%252013.2635%2520%255D%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522start%2520time%2522%253A%2520%25222018-10-14%252010%253A39%253A21%2522%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522HR%2522%253A%2520135%250A%2520%2520%2520%2520%2520%2520%257D%250A%2520%2520%2520%2520%255D%250A%2520%2520%257D%250A%257D&a=&o=1
  [play13]: https://theory.github.io/sqljson/?p=strict%2520%2524.track.segments.location&j=%257B%250A%2520%2520%2522track%2522%253A%2520%257B%250A%2520%2520%2520%2520%2522segments%2522%253A%2520%255B%250A%2520%2520%2520%2520%2520%2520%257B%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522location%2522%253A%2520%2520%2520%255B%252047.763%252C%252013.4034%2520%255D%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522start%2520time%2522%253A%2520%25222018-10-14%252010%253A05%253A14%2522%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522HR%2522%253A%252073%250A%2520%2520%2520%2520%2520%2520%257D%252C%250A%2520%2520%2520%2520%2520%2520%257B%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522location%2522%253A%2520%2520%2520%255B%252047.706%252C%252013.2635%2520%255D%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522start%2520time%2522%253A%2520%25222018-10-14%252010%253A39%253A21%2522%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522HR%2522%253A%2520135%250A%2520%2520%2520%2520%2520%2520%257D%250A%2520%2520%2520%2520%255D%250A%2520%2520%257D%250A%257D&a=&o=1
  [play14]: https://theory.github.io/sqljson/?p=strict%2520%2524.track.segments%255B*%255D.location&j=%257B%250A%2520%2520%2522track%2522%253A%2520%257B%250A%2520%2520%2520%2520%2522segments%2522%253A%2520%255B%250A%2520%2520%2520%2520%2520%2520%257B%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522location%2522%253A%2520%2520%2520%255B%252047.763%252C%252013.4034%2520%255D%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522start%2520time%2522%253A%2520%25222018-10-14%252010%253A05%253A14%2522%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522HR%2522%253A%252073%250A%2520%2520%2520%2520%2520%2520%257D%252C%250A%2520%2520%2520%2520%2520%2520%257B%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522location%2522%253A%2520%2520%2520%255B%252047.706%252C%252013.2635%2520%255D%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522start%2520time%2522%253A%2520%25222018-10-14%252010%253A39%253A21%2522%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522HR%2522%253A%2520135%250A%2520%2520%2520%2520%2520%2520%257D%250A%2520%2520%2520%2520%255D%250A%2520%2520%257D%250A%257D&a=&o=1
  [play15]: https://theory.github.io/sqljson/?p=lax%2520%2524.**.HR&j=%257B%250A%2520%2520%2522track%2522%253A%2520%257B%250A%2520%2520%2520%2520%2522segments%2522%253A%2520%255B%250A%2520%2520%2520%2520%2520%2520%257B%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522location%2522%253A%2520%2520%2520%255B%252047.763%252C%252013.4034%2520%255D%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522start%2520time%2522%253A%2520%25222018-10-14%252010%253A05%253A14%2522%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522HR%2522%253A%252073%250A%2520%2520%2520%2520%2520%2520%257D%252C%250A%2520%2520%2520%2520%2520%2520%257B%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522location%2522%253A%2520%2520%2520%255B%252047.706%252C%252013.2635%2520%255D%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522start%2520time%2522%253A%2520%25222018-10-14%252010%253A39%253A21%2522%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522HR%2522%253A%2520135%250A%2520%2520%2520%2520%2520%2520%257D%250A%2520%2520%2520%2520%255D%250A%2520%2520%257D%250A%257D&a=&o=1
  [play16]: https://theory.github.io/sqljson/?p=strict%2520%2524.**.HR&j=%257B%250A%2520%2520%2522track%2522%253A%2520%257B%250A%2520%2520%2520%2520%2522segments%2522%253A%2520%255B%250A%2520%2520%2520%2520%2520%2520%257B%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522location%2522%253A%2520%2520%2520%255B%252047.763%252C%252013.4034%2520%255D%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522start%2520time%2522%253A%2520%25222018-10-14%252010%253A05%253A14%2522%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522HR%2522%253A%252073%250A%2520%2520%2520%2520%2520%2520%257D%252C%250A%2520%2520%2520%2520%2520%2520%257B%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522location%2522%253A%2520%2520%2520%255B%252047.706%252C%252013.2635%2520%255D%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522start%2520time%2522%253A%2520%25222018-10-14%252010%253A39%253A21%2522%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522HR%2522%253A%2520135%250A%2520%2520%2520%2520%2520%2520%257D%250A%2520%2520%2520%2520%255D%250A%2520%2520%257D%250A%257D&a=&o=1
  [play17]: https://theory.github.io/sqljson/?p=lax%2520%2524.track.segments%255B*%255D.location&j=%257B%250A%2520%2520%2522track%2522%253A%2520%257B%250A%2520%2520%2520%2520%2522segments%2522%253A%2520%255B%250A%2520%2520%2520%2520%2520%2520%257B%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522location%2522%253A%2520%2520%2520%255B%252047.763%252C%252013.4034%2520%255D%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522start%2520time%2522%253A%2520%25222018-10-14%252010%253A05%253A14%2522%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522HR%2522%253A%252073%250A%2520%2520%2520%2520%2520%2520%257D%252C%250A%2520%2520%2520%2520%2520%2520%257B%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522location%2522%253A%2520%2520%2520%255B%252047.706%252C%252013.2635%2520%255D%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522start%2520time%2522%253A%2520%25222018-10-14%252010%253A39%253A21%2522%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522HR%2522%253A%2520135%250A%2520%2520%2520%2520%2520%2520%257D%250A%2520%2520%2520%2520%255D%250A%2520%2520%257D%250A%257D&a=&o=1
  [play18]: https://theory.github.io/sqljson/?p=lax%2520%2524.track.segments%255B*%255D.location%2520%253F%28%2540%255B*%255D%2520%253E%252015%29&j=%257B%250A%2520%2520%2522track%2522%253A%2520%257B%250A%2520%2520%2520%2520%2522segments%2522%253A%2520%255B%250A%2520%2520%2520%2520%2520%2520%257B%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522location%2522%253A%2520%2520%2520%255B%252047.763%252C%252013.4034%2520%255D%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522start%2520time%2522%253A%2520%25222018-10-14%252010%253A05%253A14%2522%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522HR%2522%253A%252073%250A%2520%2520%2520%2520%2520%2520%257D%252C%250A%2520%2520%2520%2520%2520%2520%257B%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522location%2522%253A%2520%2520%2520%255B%252047.706%252C%252013.2635%2520%255D%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522start%2520time%2522%253A%2520%25222018-10-14%252010%253A39%253A21%2522%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522HR%2522%253A%2520135%250A%2520%2520%2520%2520%2520%2520%257D%250A%2520%2520%2520%2520%255D%250A%2520%2520%257D%250A%257D&a=&o=1
  [play19]: https://theory.github.io/sqljson/?p=strict%2520%2524.track.segments%255B*%255D.location%2520%253F%28%2540%255B*%255D%2520%253E%252015%29&j=%257B%250A%2520%2520%2522track%2522%253A%2520%257B%250A%2520%2520%2520%2520%2522segments%2522%253A%2520%255B%250A%2520%2520%2520%2520%2520%2520%257B%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522location%2522%253A%2520%2520%2520%255B%252047.763%252C%252013.4034%2520%255D%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522start%2520time%2522%253A%2520%25222018-10-14%252010%253A05%253A14%2522%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522HR%2522%253A%252073%250A%2520%2520%2520%2520%2520%2520%257D%252C%250A%2520%2520%2520%2520%2520%2520%257B%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522location%2522%253A%2520%2520%2520%255B%252047.706%252C%252013.2635%2520%255D%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522start%2520time%2522%253A%2520%25222018-10-14%252010%253A39%253A21%2522%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522HR%2522%253A%2520135%250A%2520%2520%2520%2520%2520%2520%257D%250A%2520%2520%2520%2520%255D%250A%2520%2520%257D%250A%257D&a=&o=1
  [play20]: https://theory.github.io/sqljson/?p=%2524%255B0%255D%2520%252B%25203&j=2&a=&o=1
  [play21]: https://theory.github.io/sqljson/?p=%252B%2520%2524.x&j=%257B%2522x%2522%253A%2520%255B2%252C3%252C4%255D%257D&a=&o=1
  [play22]: https://theory.github.io/sqljson/?p=7%2520-%2520%2524%255B0%255D&j=%255B2%255D&a=&o=1
  [play23]: https://theory.github.io/sqljson/?p=-%2520%2524.x&j=%257B%2522x%2522%253A%2520%255B2%252C3%252C4%255D%257D&a=&o=1
  [play24]: https://theory.github.io/sqljson/?p=2%2520*%2520%2524%255B0%255D&j=4&a=&o=1
  [play25]: https://theory.github.io/sqljson/?p=%2524%255B0%255D%2520%252F%25202&j=%255B8.5%255D&a=&o=1
  [play26]: https://theory.github.io/sqljson/?p=%2524%255B0%255D%2520%2525%252010&j=%255B32%255D&a=&o=1
  [play27]: https://theory.github.io/sqljson/?p=%2524%255B*%255D.type%28%29&j=%255B1%252C%2520%25222%2522%252C%2520%257B%257D%255D&a=&o=1
  [play28]: https://theory.github.io/sqljson/?p=%2524.m.size%28%29&j=%257B%2522m%2522%253A%2520%255B11%252C%252015%255D%257D&a=&o=1
  [play29]: https://theory.github.io/sqljson/?p=%2524%255B*%255D.boolean%28%29&j=%255B1%252C%2520%2522yes%2522%252C%2520false%255D&a=&o=1
  [play30]: https://theory.github.io/sqljson/?p=%2524%255B*%255D.string%28%29&j=%255B1.23%252C%2520%2522xyz%2522%252C%2520false%255D&a=&o=1
  [play31]: https://theory.github.io/sqljson/?p=%2524.timestamp%28%29.string%28%29&j=%25222023-08-15%252012%253A34%253A56%2522
  [play32]: https://theory.github.io/sqljson/?p=%2524.len.double%28%29%2520*%25202&j=%257B%2522len%2522%253A%2520%25221.9%2522%257D&a=&o=1
  [play33]: https://theory.github.io/sqljson/?p=%2524.h.ceiling%28%29&j=%257B%2522h%2522%253A%25201.3%257D&a=&o=1
  [play34]: https://theory.github.io/sqljson/?p=%2524.h.floor%28%29&j=%257B%2522h%2522%253A%25201.7%257D&a=&o=1
  [play35]: https://theory.github.io/sqljson/?p=%2524.z.abs%28%29&j=%257B%2522z%2522%253A%2520-0.3%257D&a=&o=1
  [play36]: https://theory.github.io/sqljson/?p=%2524.len.bigint%28%29&j=%257B%2522len%2522%253A%2520%25229876543219%2522%257D&a=&o=1
  [play37]: https://theory.github.io/sqljson/?p=%2524.decimal%286%252C%25202%29&j=%25221234.5678%2522&a=&o=1
  [play38]: https://theory.github.io/sqljson/?p=%2524.len.integer%28%29&j=%257B%2522len%2522%253A%2520%252212345%2522%257D&a=&o=1
  [play39]: https://theory.github.io/sqljson/?p=%2524.len.number%28%29&j=%257B%2522len%2522%253A%2520%2522123.45%2522%257D&a=&o=1
  [play40]: https://theory.github.io/sqljson/?p=%2524%255B*%255D%2520%253F%2520%28%2540.datetime%28%29%2520%253C%2520%25222015-08-02%2522.datetime%28%29%29&j=%255B%25222015-08-01%2522%252C%2520%25222015-08-12%2522%255D&a=&o=1
  [play41]: https://theory.github.io/sqljson/?p=%2524%255B*%255D.datetime%28%2522HH24%253AMI%2522%29&j=%255B%252212%253A30%2522%252C%2520%252218%253A40%2522%255D&a=&o=1
  [play42]: https://theory.github.io/sqljson/?p=%2524.date%28%29&j=2023-08-15&a=&o=1
  [play43]: https://theory.github.io/sqljson/?p=%2524.time%28%29&j=12%253A34%253A56&a=&o=1
  [play44]: https://theory.github.io/sqljson/?p=%2524.time%282%29&j=%252212%253A34%253A56.789%2522&a=&o=1
  [play45]: https://theory.github.io/sqljson/?p=%2524.time_tz%28%29&j=%252212%253A34%253A56%252B05%253A30%2522&a=%257B%257D&o=1
  [play46]: https://theory.github.io/sqljson/?p=%2524.time_tz%282%29&j=%252212%253A34%253A56.789%252B05%253A30%2522&a=&o=1
  [play47]: https://theory.github.io/sqljson/?p=%2524.timestamp%28%29&j=%25222023-08-15%252012%253A34%253A56%2522&a=&o=1
  [play48]: https://theory.github.io/sqljson/?p=%2524.timestamp%282%29&j=%25222023-08-15%252012%253A34%253A56.789%2522&a=&o=1
  [play49]: https://theory.github.io/sqljson/?p=%2524.timestamp_tz%28%29&j=%25222023-08-15%252012%253A34%253A56%252B05%253A30%2522&a=&o=1
  [play50]: https://theory.github.io/sqljson/?p=%2524.timestamp_tz%282%29&j=%25222023-08-15%252012%253A34%253A56.789%252B05%253A30%2522&a=&o=1
  [play51]: https://theory.github.io/sqljson/?p=%2524.keyvalue%28%29&j=%257B%2522x%2522%253A%2520%252220%2522%252C%2520%2522y%2522%253A%252032%257D&a=&o=1
  [play52]: https://theory.github.io/sqljson/?p=%2524%255B*%255D%2520%253F%2520%28%2540%2520%253D%253D%25201%29&j=%255B1%252C%2520%2522a%2522%252C%25201%252C%25203%255D&a=&o=1
  [play53]: https://theory.github.io/sqljson/?p=%2524%255B*%255D%2520%253F%2520%28%2540%2520%253D%253D%2520%2522a%2522%29&j=%255B1%252C%2520%2522a%2522%252C%25201%252C%25203%255D&a=&o=1
  [play54]: https://theory.github.io/sqljson/?p=%2524%255B*%255D%2520%253F%2520%28%2540%2520%21%253D%25201%29&j=%255B1%252C%25202%252C%25201%252C%25203%255D&a=&o=1
  [play55]: https://theory.github.io/sqljson/?p=%2524%255B*%255D%2520%253F%2520%28%2540%2520%253C%253E%2520%2522b%2522%29&j=%255B%2522a%2522%252C%2520%2522b%2522%252C%2520%2522c%2522%255D&a=&o=1
  [play56]: https://theory.github.io/sqljson/?p=%2524%255B*%255D%2520%253F%2520%28%2540%2520%253C%25202%29&j=%255B1%252C%25202%252C%25203%255D&a=&o=1
  [play57]: https://theory.github.io/sqljson/?p=%2524%255B*%255D%2520%253F%2520%28%2540%2520%253C%253D%2520%2522b%2522%29&j=%255B%2522a%2522%252C%2520%2522b%2522%252C%2520%2522c%2522%255D&a=&o=1
  [play58]: https://theory.github.io/sqljson/?p=%2524%255B*%255D%2520%253F%2520%28%2540%2520%253E%25202%29&j=%255B1%252C%25202%252C%25203%255D&a=&o=1
  [play59]: https://theory.github.io/sqljson/?p=%2524%255B*%255D%2520%253F%2520%28%2540%2520%253E%253D%25202%29&j=%255B1%252C%25202%252C%25203%255D&a=&o=1
  [play60]: https://theory.github.io/sqljson/?p=%2524%255B*%255D%2520%253F%2520%28%2540.parent%2520%253D%253D%2520true%29&j=%255B%250A%2520%2520%257B%2522name%2522%253A%2520%2522John%2522%252C%2520%2522parent%2522%253A%2520false%257D%252C%250A%2520%2520%257B%2522name%2522%253A%2520%2522Chris%2522%252C%2520%2522parent%2522%253A%2520true%257D%250A%255D&a=&o=1
  [play61]: https://theory.github.io/sqljson/?p=%2524%255B*%255D%2520%253F%2520%28%2540.parent%2520%253D%253D%2520false%29&j=%255B%250A%2520%2520%257B%2522name%2522%253A%2520%2522John%2522%252C%2520%2522parent%2522%253A%2520false%257D%252C%250A%2520%2520%257B%2522name%2522%253A%2520%2522Chris%2522%252C%2520%2522parent%2522%253A%2520true%257D%250A%255D&a=&o=1
  [play62]: https://theory.github.io/sqljson/?p=%2524%255B*%255D%2520%253F%2520%28%2540.job%2520%253D%253D%2520null%29%2520.name&j=%255B%250A%2520%2520%257B%2522name%2522%253A%2520%2522Mary%2522%252C%2520%2522job%2522%253A%2520null%257D%252C%250A%2520%2520%257B%2522name%2522%253A%2520%2522Michael%2522%252C%2520%2522job%2522%253A%2520%2522driver%2522%257D%250A%255D&a=&o=1
  [play63]: https://theory.github.io/sqljson/?p=%2524%255B*%255D%2520%253F%2520%28%2540%2520%253E%25201%2520%2526%2526%2520%2540%2520%253C%25205%29&j=%255B1%252C%25203%252C%25207%255D&a=&o=1
  [play64]: https://theory.github.io/sqljson/?p=%2524%255B*%255D%2520%253F%2520%28%2540%2520%253C%25201%2520%257C%257C%2520%2540%2520%253E%25205%29&j=%255B1%252C%25203%252C%25207%255D&a=&o=1
  [play65]: https://theory.github.io/sqljson/?p=%2524%255B*%255D%2520%253F%2520%28%21%28%2540%2520%253C%25205%29%29&j=%255B1%252C%25203%252C%25207%255D&a=&o=1
  [play66]: https://theory.github.io/sqljson/?p=%2524%255B*%255D%2520%253F%2520%28%28%2540%2520%253E%25200%29%2520is%2520unknown%29&j=%255B-1%252C%25202%252C%25207%252C%2520%2522foo%2522%255D&a=&o=1
  [play66]: https://theory.github.io/sqljson/?p=%2524%255B*%255D%2520%253F%2520%28%2540%2520like_regex%2520%2522%255Eab.*c%2522%29&j=%255B%2522abc%2522%252C%2520%2522abd%2522%252C%2520%2522aBdC%2522%252C%2520%2522abdacb%2522%252C%2520%2522babc%2522%255D&a=&o=1
  [play67]: https://theory.github.io/sqljson/?p=%2524%255B*%255D%2520%253F%2520%28%2540%2520like_regex%2520%2522%255Eab.*c%2522%29&j=%255B%2522abc%2522%252C%2520%2522abd%2522%252C%2520%2522aBdC%2522%252C%2520%2522abdacb%2522%252C%2520%2522babc%2522%255D&a=&o=1
  [play68]: https://theory.github.io/sqljson/?p=%2524%255B*%255D%2520%253F%2520%28%2540%2520like_regex%2520%2522%255Eab.*c%2522%2520flag%2520%2522i%2522%29&j=%255B%2522abc%2522%252C%2520%2522abd%2522%252C%2520%2522aBdC%2522%252C%2520%2522abdacb%2522%252C%2520%2522babc%2522%255D&a=&o=1
  [play69]: https://theory.github.io/sqljson/?p=%2524%255B*%255D%2520%253F%2520%28%2540%2520starts%2520with%2520%2522John%2522%29&j=%255B%2522John%2520Smith%2522%252C%2520%2522Mary%2520Stone%2522%252C%2520%2522Bob%2520Johnson%2522%255D&a=&o=1
  [play70]: https://theory.github.io/sqljson/?p=strict%2520%2524.*%2520%253F%2520%28exists%2520%28%2540%2520%253F%2520%28%2540%255B*%255D%2520%253E%25202%29%29%29&j=%257B%2522x%2522%253A%2520%255B1%252C%25202%255D%252C%2520%2522y%2522%253A%2520%255B2%252C%25204%255D%257D&a=&o=1
  [play71]: https://theory.github.io/sqljson/?p=strict%2520%2524%2520%253F%2520%28exists%2520%28%2540.name%29%29%2520.name&j=%257B%2522x%2522%253A%2520%255B1%252C%25202%255D%252C%2520%2522y%2522%253A%2520%255B2%252C%25204%255D%257D&a=%257B%2522value%2522%253A%252042%257D&o=1
  [play72]: https://theory.github.io/sqljson/?p=%2524.*%2520%253F%28%2540%2520like_regex%2520%2522%255E%255C%255Cd%252B%2524%2522%29&j=%257B%2522x%2522%253A%2520%252242%2522%252C%2520%2522y%2522%253A%2520%2522no%2522%257D&a=&o=1
  [play73]: https://theory.github.io/sqljson/?p=%2524.timestamp_tz%28%29&j=%25222023-08-15%252012%253A34%253A56%2522&o=49&a=%257B%257D
  [play74]: https://theory.github.io/sqljson/?p=%2524.timestamp_tz%28%29&j=%25222023-08-15%252012%253A34%253A56%2522&o=17&a=%257B%257D
