Go SQL/JSON Path
================

The path package ports the SQL/JSON Path data type from PostgreSQL to Go. It
supports both SQL-standard path expressions and PostgreSQL-specific predicate
check expressions. The following is ported from the [PostgreSQL docs].

**WARNING:** Path execution has not yet been implemented: the `MustQuery` and
`Query` methods currently return the JSON value passed to them, and `Exists`
always returns true. The examples below demonstrate the expected behavior, but
are subject to change once the method has been properly implemented.

The SQL/JSON Path Language
--------------------------

SQL/JSON Path is a query language for JSON values. A path expression applied
to a JSON value, a query argument, produces a JSON result.

SQL/JSON path expressions specify item(s) to be retrieved from a JSON value,
similarly to XPath expressions used for access to XML content. In Go, path
expressions are implemented in the path package and can use any elements
described [below](#syntax).

### Syntax

The path package implements support for the SQL/JSON path language in Go to
efficiently query JSON data. It provides a binary representation of the parsed
SQL/JSON path expression that specifies the items to be retrieved by the path
engine from the JSON data for further processing with the SQL/JSON query
functions.

The semantics of SQL/JSON path predicates and operators generally follow SQL.
At the same time, to provide a natural way of working with JSON data, SQL/JSON
path syntax uses some JavaScript conventions:

*   Dot (`.`) is used for member access.

*   Square brackets (`[]`) are used for array access.

*   SQL/JSON arrays are 0-relative, like Go slices, but unlike regular SQL
    arrays, which start from 1.

Numeric literals in SQL/JSON path expressions follow JavaScript rules, which
are different from Go, SQL, JSON in some minor details. For example, SQL/JSON
path allows `.1` and `1.`, which are invalid in JSON. Non-decimal integer literals
and underscore separators are supported, for example, `1_000_000`, `0x1EEE_FFFF`,
`0o273`, `0b100101`. In SQL/JSON path (and in JavaScript, but not in SQL or Go),
there must not be an underscore separator directly after the radix prefix.

An SQL/JSON path expression is typically written as a Go string literal, so it
must be enclosed in back quotes or double quotes --- and with the latter any
double quotes within the value must be escaped (see [string literals]).

Some forms of path expressions require string literals within them. These
embedded string literals follow JavaScript/ECMAScript conventions: they must
be surrounded by double quotes, and backslash escapes may be used within them
to represent otherwise-hard-to-type characters. In particular, the way to
write a double quote within an embedded string literal is `\"`, and to write a
backslash itself, you must write `\\`. Other special backslash sequences
include those recognized in JSON strings: `\b`, `\f`, `\n`, `\r`, `\t`, `\v`
for various ASCII control characters, and `\uNNNN` for a Unicode character
identified by its 4-hex-digit code point. The backslash syntax also includes
two cases not allowed by JSON: `\xNN` for a character code written with only
two hex digits, and `\u{N...}` for a character code written with 1 to 6 hex
digits.

A path expression consists of a sequence of path elements, which can be any of
the following:

*   Path literals of JSON primitive types: Unicode text, numeric, true, false, or null.

*   Path variables listed in Table 8.24.

*   Accessor operators listed in Table 8.25.

*   JSON path operators and methods listed in Section 9.16.2.3.

*   Parentheses, which can be used to provide filter expressions or define the
    order of path evaluation.

For details on using JSON path expressions with SQL/JSON query functions, see
[below](#operation).

#### Path Variables

| Variable   | Description
| ---------- | ------------------------------------------------------------------------------------------------- |
| `$`        | A variable representing the JSON value being queried (the context item).                          |
| `$varname` | A named variable. Its value can be set by the parameter vars of several JSON processing functions |
| `@`        | A variable representing the result of path evaluation in filter expressions.                      |

#### Path Accessors

| Accessor Operator | Description
| ----------------- | ------------------------------------------------------------------------------------------------- |
| `.key`, `."$varname"` | Member accessor that returns an object member with the specified key. If the key name matches some named variable starting with `$` or does not meet the JavaScript rules for an identifier, it must be enclosed in double quotes to make it a string literal.
| `.*`                  | Wildcard member accessor that returns the values of all members located at the top level of the current object.
| `.**`                 | Recursive wildcard member accessor that processes all levels of the JSON hierarchy of the current object and returns all the member values, regardless of their nesting level. This is a PostgreSQL extension of the SQL/JSON standard.
| `.**{level}`, `.**{start_level to end_level}` | Like `.**`, but selects only the specified levels of the JSON hierarchy. Nesting levels are specified as integers. Level zero corresponds to the current object. To access the lowest nesting level, you can use the `last` keyword. This is a PostgreSQL extension of the SQL/JSON standard.
| `[subscript, ...]`                            | Array element accessor. subscript can be given in two forms: `index` or `start_index` to `end_index`. The first form returns a single array element by its index. The second form returns an array slice by the range of indexes, including the elements that correspond to the provided `start_index` and `end_index`.<br/><br/>The specified index can be an integer, as well as an expression returning a single numeric value, which is automatically cast to integer. Index zero corresponds to the first array element. You can also use the `last` keyword to denote the last array element, which is useful for handling arrays of unknown length.
| `[*]`                  | Wildcard array element accessor that returns all array elements.

### Operation

JSON query functions and operators pass the provided path expression to the
path engine for evaluation. If the expression matches the queried JSON data,
the corresponding JSON item, or set of items, is returned. If there is no
match, the result will be `NULL`, `false`, or an error, depending on the
function. Path expressions are written in the SQL/JSON path language and can
include arithmetic expressions and functions.

A path expression consists of a sequence of elements allowed by the SQL/JSON
path language. The path expression is normally evaluated from left to right,
but you can use parentheses to change the order of operations. If the
evaluation is successful, a sequence of JSON items is produced, and the
evaluation result is returned to the JSON query function that completes the
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
src := `{
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
}`
```

The [path] package expects JSON to be decoded into a Go value, one of
`string`, `float64`, [json.Number], `map[string]any`, or `[]any` — which are
the values produced by unmarshaling data into an `any` value. For the above
JSON, unmarshal it like so:

``` go
var jsonValue any
if err := json.Unmarshal(src, &jsonValue); err != nil {
    log.Fatal(err)
}
fmt.Printf("%T\n", jsonValue)
```

The output shows the parsed data type:

``` go
map[string]interface {}
```

Note that examples below use `any` rather than `interface {}` for legibility.

To retrieve the available track segments, you need to use the `.key` accessor
operator to descend through surrounding JSON objects, for example:

``` go
fmt.Printf(path.MustQuery("$.track.segments", jsonValue))
```

And the output (reformatted for legibility):

``` go
[]any{
    map[string]any{
        "HR":73,
        "location":[]any{47.763, 13.4034},
        "start time":"2018-10-14 10:05:14",
    },
    map[string]any{
        "HR":135,
        "location":[]any{47.706, 13.2635},
        "start time":"2018-10-14 10:39:21",
    },
}
```

To retrieve the contents of an array, you typically use the `[*]` operator. The
following example will return the location coordinates for all the available
track segments:

``` go
fmt.Printf(path.MustQuery("$.track.segments[*].location", jsonValue))
```

``` go
[]any{[]any{47.763, 13.4034}, []any{47.706, 13.2635}}
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
indexes are 0-relative:

```go
fmt.Printf(path.MustQuery("$.track.segments[0].location", jsonValue))
```

``` go
[]any{47.763, 13.4034}
```

The result of each path evaluation step can be processed by one or more of the
json path operators and methods listed [below](#sqljson-path-operators-and-methods).
Each method name must be preceded by a dot. For example, you can get the size
of an array:

```go
fmt.Printf(path.MustQuery("$.track.segments.size()", jsonValue))
```

``` go
2
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
than 130. You can achieve this as follows:

```go
fmt.Printf(path.MustQuery("$.track.segments[*].HR ? (@ > 130)", jsonValue))
```

``` go
135
```

To get the start times of segments with such values, you have to filter out
irrelevant segments before selecting the start times, so the filter expression
is applied to the previous step, and the path used in the condition is
different:

```go
fmt.Printf(path.MustQuery(
    `$.track.segments[*] ? (@.HR > 130)."start time"`,
    jsonValue,
))
```

``` go
"2018-10-14 10:39:21"
```

You can use several filter expressions in sequence, if required. The following
example selects start times of all segments that contain locations with
relevant coordinates and high heart rate values:

```go
fmt.Printf(path.MustQuery(
    `$.track.segments[*] ? (@.location[1] < 13.4) ? (@.HR > 130)."start time"`,
    jsonValue,
))
```

```go
"2018-10-14 10:39:21"
```

Using filter expressions at different nesting levels is also allowed. The
following example first filters all segments by location, and then returns
high heart rate values for these segments, if available:

```go
fmt.Printf(path.MustQuery(
    `$.track.segments[*] ? (@.location[1] < 13.4).HR ? (@ > 130)`,
    jsonValue,
))
```

```go
135
```

You can also nest filter expressions within each other. This example returns
the size of the track if it contains any segments with high heart rate values,
or an empty sequence otherwise:

```go
fmt.Printf(path.MustQuery(
    `$.track ? (exists(@.segments[*] ? (@.HR > 130))).segments.size()`,
    jsonValue,
))
```

```go
2
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
three-valued result of the predicate: `true`, `false`, or `unknown`. For
example, we could write this SQL-standard filter expression:

```go
fmt.Printf(path.MustQuery("$.track.segments ?(@[*].HR > 130)", jsonValue))
```

The result (reformatted for legibility):

```go
map[string]any{
    "HR":135,
    "location":[]any{47.706, 13.2635},
    "start time":"2018-10-14 10:39:21",
}
```

The similar predicate check expression simply returns `true`, indicating that a
match exists:

```go
fmt.Printf(path.MustQuery("$.track.segments[*].HR > 130", jsonValue))
```

```go
true
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

*   lax (default) — the path engine implicitly adapts the queried data to the
    specified path. Any structural errors that cannot be fixed as described
    below are suppressed, producing no match.

*   `strict` — if a structural error occurs, an error is raised.

Lax mode facilitates matching of a JSON document and path expression when the
JSON data does not conform to the expected schema. If an operand does not
match the requirements of a particular operation, it can be automatically
wrapped as an SQL/JSON array, or unwrapped by converting its elements into an
SQL/JSON sequence before performing the operation. Also, comparison operators
automatically unwrap their operands in lax mode, so you can compare SQL/JSON
arrays out-of-the-box. An array of size 1 is considered equal to its sole
element. Automatic unwrapping is not performed when:

*   The path expression contains `type()` or `size()` methods that return the
    type and the number of elements in the array, respectively.

*   The queried JSON data contain nested arrays. In this case, only the
    outermost array is unwrapped, while all the inner arrays remain unchanged.
    Thus, implicit unwrapping can only go one level down within each path
    evaluation step.

For example, when querying the GPS data listed above, you can abstract from
the fact that it stores an array of segments when using lax mode:

```go
fmt.Printf(path.MustQuery("lax $.track.segments.location", jsonValue))
```

``` go
[]any{[]any{47.763, 13.4034}, []any{47.706, 13.2635}}
```

In strict mode, the specified path must exactly match the structure of the
queried JSON document, so using this path expression will cause an error:

```go
fmt.Printf(path.MustQuery("strict $.track.segments.location", jsonValue))
```

``` text
panic: path: member accessor can only be applied to an object
```

To get the same result as in lax mode, you have to explicitly unwrap the
segments array:

```go
fmt.Printf(path.MustQuery("strict $.track.segments[*].location", jsonValue))
```

``` go
[]any{[]any{47.763, 13.4034}, []any{47.706, 13.2635}}
```

The unwrapping behavior of lax mode can lead to surprising results. For
instance, the following query using the `.**` accessor selects every `HR` value
twice:


```go
fmt.Printf(path.MustQuery("lax $.**.HR", jsonValue))
```

``` go
[]any{73, 135, 73, 135}
```

This happens because the `.**` accessor selects both the segments array and
each of its elements, while the `.HR` accessor automatically unwraps arrays
when using lax mode. To avoid surprising results, we recommend using the `.**`
accessor only in strict mode. The following query selects each `HR` value just
once:

```go
fmt.Printf(path.MustQuery("strict $.**.HR", jsonValue))
```

``` go
[]any{73, 135}
```

The unwrapping of arrays can also lead to unexpected results. Consider this
example, which selects all the location arrays:

```go
fmt.Printf(path.MustQuery("lax $.track.segments[*].location", jsonValue))
```

``` go
[]any{[]any{47.763, 13.4034}, []any{47.706, 13.2635}}
```

As expected it returns the full arrays. But applying a filter expression
causes the arrays to be unwrapped to evaluate each item, returning only the
items that match the expression:

```go
fmt.Printf(path.MustQuery(
    "lax $.track.segments[*].location ?(@[*] > 15)",
    jsonValue,
))
```

``` go
[]any{47.763, 47.706}
```

This despite the fact that the full arrays are selected by the path
expression. Use strict mode to restore selecting the arrays:

```go
fmt.Printf(path.MustQuery(
    "strict $.track.segments[*].location ?(@[*] > 15)",
    jsonValue,
))
```

``` go
[]any{[]any{47.763, 13.4034}, []any{47.706, 13.2635}}
```

### SQL/JSON Path Operators And Methods

The list of operators and methods available in JSON path expressions. Note
that while the unary operators and methods can be applied to multiple values
resulting from a preceding path step, the binary operators (addition etc.) can
only be applied to single values.

#### `number + number → number`

Addition.

``` go
fmt.Println(path.MustQuery([]any{2}, "$[0] + 3")) // → 5
```

#### `+ number → number`

Unary plus (no operation); unlike addition, this can iterate over multiple
values.

``` go
arg := map[string]any{"x": []any{2,3,4}}
fmt.Println(path.MustQuery(arg, "+ $.x")) // → [2, 3, 4]
```

#### `number - number → number`

Subtraction.

``` go
fmt.Println(path.MustQuery([]any{2}, "7 - $[0]")) // → 5
```

#### `- number → number`

Negation; unlike subtraction, this can iterate over multiple values.

``` go
arg := map[string]any{"x": []any{2,3,4}}
fmt.Println(path.MustQuery(arg, "- $.x")) // → [-2, -3, -4]
```

#### `number * number → number`

Multiplication.

``` go
fmt.Println(path.MustQueryJSON([]any{3}, "2 * $[0]")) // → 8
```
#### `number / number → number`

Division.

``` go
fmt.Println(path.MustQuery([]any{8.5}, "$[0] / 2")) // → 4.25
```

#### `number % number → number`

Modulo (remainder).

``` go
fmt.Println(path.MustQuery([]any{32}, "$[0] % 10")) // → 2
```

#### `value . type() → string`

Type of the JSON item.

``` go
arg := []any{1, "2", map[string]any{}}
fmt.Println(path.MustQuery(arg, "$[*].type()")) // → []any{"number", "string", "object"}
```

#### `value . size() → number`

Size of the JSON item (number of array elements, or 1 if not an array)

``` go
arg := map[string]any{"m": []any{11, 15}}
fmt.Println(path.MustQuery(arg, "$.m.size()")) // → 2
```

#### `value . boolean() → boolean`

Boolean value converted from a JSON boolean, number, or string.

``` go
arg := []any{1, "yes", false}
fmt.Println(path.MustQuery(arg, "$[*].boolean()")) // → [true, true, false]
```

#### `value . string() → string`

String value converted from a JSON boolean, number, string, or datetime.

``` go
arg := []any{1.23, "xyz", false}
fmt.Println(path.MustQuery(arg, "$[*].string()")) // → ["1.23", "xyz", "false"]

arg = "2023-08-15"
fmt.Println(path.MustQuery(arg, "$.datetime().string()")) // → "2023-08-15"
```

#### `value . double() → number`

Approximate floating-point number converted from a JSON number or string.

``` go
arg := map[string]any{"len": "1.9"}
fmt.Println(path.MustQuery(arg, "$.len.double() * 2")) // → 3.8
```

#### `number . ceiling() → number`

Nearest integer greater than or equal to the given number.

``` go
fmt.Println(path.MustQuery(map[string]any{"h": 1.3}, "$.h.ceiling()")) // → 2
```

#### `number . floor() → number`

Nearest integer less than or equal to the given number.

``` go
fmt.Println(path.MustQuery(map[string]any{"h": 1.7}, "$.h.floor()")) // → 1
```

#### `number . abs() → number`

Absolute value of the given number.

``` go
fmt.Println(path.MustQuery(map[string]any{"z": -0.3}, "$.z.abs()")) // → 0.3
```

#### `value . bigint() → bigint`

Big integer value converted from a JSON number or string.

``` go
arg := map[string]{"len": "9876543219"}
fmt.Println(path.MustQuery(arg, "$.len.bigint()")) // → 9876543219
```

#### `value . decimal( [ precision [ , scale ] ] ) → decimal`

Rounded decimal value converted from a JSON number or string. Precision and
scale must be integer values.

``` go
fmt.Println(path.MustQuery("1234.5678", "$.decimal(6, 2)")) // → 1234.57
```

#### `value . integer() → integer`

Integer value converted from a JSON number or string.

``` go
arg := map[string]any{"len": "12345"}
fmt.Println(path.MustQuery(arg, "$.len.integer()")) // → 12345
```

#### `value . number() → numeric`

Numeric value converted from a JSON number or string.

``` go
arg := map[string]any{"len": "123.45"}
fmt.Println(path.MustQuery(arg, "$.len.number()")) // → 123.45
```

#### `string . datetime() → time.Time`

Date/time value converted from a string.

``` go
fmt.Println(path.MustQuery(
    []any{"2015-8-1", "2015-08-12"},
    `$[*] ? (@.datetime() < "2015-08-2".datetime())`,
)) // → "2015-8-1"
```

#### `string . datetime(template) → datetime_type`

Date/time value converted from a string using the specified to_timestamp
template.

``` go
fmt.Println(path.MustQuery(
    []any{"12:30", "18:40"},
    `$[*].datetime("HH24:MI")`,
)) // → ["12:30:00", "18:40:00"]
```

#### `string . date() → date`

Date value converted from a string.

``` go
fmt.Println(path.MustQuery("2023-08-15", "$.date()")) // → "2023-08-15"
```

#### `string . time() → time.Time`

Time without time zone value converted from a string.

``` go
fmt.Println(path.MustQuery("12:34:56", "$.time()")) // → "12:34:56"
```

#### `string . time(precision) → time.Time`

Time without time zone value converted from a string, with fractional seconds
adjusted to the given precision.

``` go
fmt.Println(path.MustQuery("12:34:56.789", "$.time(2)")) // → "12:34:56.79"
```

#### `string . time_tz() → time/.Time`

Time with time zone value converted from a string.

``` go
arg := "12:34:56 +05:30"
fmt.Println(path.MustQuery(arg, "$.time_tz()")) // → "12:34:56+05:30"
```

#### `string . time_tz(precision) → time.Time`

Time with time zone value converted from a string, with fractional seconds
adjusted to the given precision.

``` go
arg := "12:34:56.789 +05:30"
fmt.Println(path.MustQuery(arg, "$.time_tz(2)")) // → "12:34:56.79+05:30"
```

#### `string . timestamp() → time.Time`

Timestamp without time zone value converted from a string.

``` go
arg := "2023-08-15 12:34:56"
fmt.Println(path.MustQuery(arg, "$.timestamp()")) // → "2023-08-15T12:34:56"
```

#### `string . timestamp(precision) → time.Time`

Timestamp without time zone value converted from a string, with fractional
seconds adjusted to the given precision.

``` go
arg := "2023-08-15 12:34:56.789"
fmt.Println(path.MustQuery(arg, "$.timestamp(2)")) // → "2023-08-15T12:34:56.79"
```

#### `string . timestamp_tz() → time.Time`

Timestamp with time zone value converted from a string.

``` go
fmt.Println(path.MustQuery(
    "2023-08-15 12:34:56 +05:30",
    "$.timestamp_tz()",
)) // → "2023-08-15T12:34:56+05:30"
```

#### `string . timestamp_tz(precision) → time.Time`

Timestamp with time zone value converted from a string, with fractional
seconds adjusted to the given precision.

``` go
fmt.Println(path.MustQuery(
    "2023-08-15 12:34:56.789 +05:30",
    "$.timestamp_tz(2)",
)) // → "2023-08-15T12:34:56.79+05:30"
```

#### `object . keyvalue() → []map[string]any`

The object's key-value pairs, represented as an array of objects containing
three fields: "key", "value", and "id"; "id" is a unique identifier of the
object the key-value pair belongs to

``` go
arg := map[string]any{"x": "20", "y": 32}
fmt.Println(path.MustQuery(arg, "$.keyvalue()"))
// → [{"id": 0, "key": "x", "value": "20"}, {"id": 0, "key": "y", "value": 32}]
```

### Filter Expression Elements

The filter expression elements available in JSON path.

#### `value == value → boolean`

Equality comparison (this, and the other comparison operators, work on all
JSON scalar values).

``` go
fmt.Println(path.MustQuery([]any{1, "a", 1, 3}, "$[*] ? (@ == 1)")) // → [1, 1]
fmt.Println(path.MustQuery([]any{1, "a", 1, 3}, `$[*] ? (@ == "a"`)) // → ["a"]
```

#### `value != value → boolean`

#### `value <> value → boolean`

Non-equality comparison.

``` go
arg := []any{1, 2, 1, 3}
fmt.Println(path.MustQuery(arg, "$[*] ? (@ != 1)")) // → [2, 3]

arg = []any{"a", "b", "c"}
fmt.Println(path.MustQuery(arg, `'$[*] ? (@ <> "b")`)) // → ["a", "c"]
```

#### `value < value → boolean`

Less-than comparison.

``` go
fmt.Println(path.MustQuery([]any{1, 2, 3}, "$[*] ? (@ < 2)")) // → [1]
```

#### `value <= value → boolean`

Less-than-or-equal-to comparison.

``` go
arg := []any{"a", "b", "c"}
fmt.Println(path.MustQuery(arg, `$[*] ? (@ <= "b")`)) // → ["a", "b"]
```

#### `value > value → boolean`

Greater-than comparison

``` go
fmt.Println(path.MustQuery([]any{1, 2, 3}, "$[*] ? (@ > 2)")) // → [3]
```

#### `value >= value → boolean`

Greater-than-or-equal-to comparison.

``` go
fmt.Println(path.MustQuery([]any{1, 2, 3}, "$[*] ? (@ >= 2)")) // → [2, 3]
```

#### `true → boolean`

JSON constant true.

``` go
arg := []map[string]any{
    {"name": "John", "parent": false},
    {"name": "Chris", "parent": true},
}
fmt.Println(path.MustQuery(
    arg, "$[*] ? (@.parent == true)",
)) // → {"name": "Chris", "parent": true}
```

#### `false → boolean`

JSON constant false.

``` go
arg := []map[string]any{
    {"name": "John", "parent": false},
    {"name": "Chris", "parent": true},
}
fmt.Println(path.MustQuery(
    arg, "$[*] ? (@.parent == false)",
)) // → {"name": "John", "parent": false}
```

#### `null → value`

JSON constant null (note that, unlike in SQL, comparison to null works
normally).

``` go
arg := []map[string]any{
    {"name": "Mary", "job": null},
    {"name": "Michael", "job": "driver"},
}
fmt.Println(path.MustQuery(arg, "$[*] ? (@.job == null) .name")) // → "Mary"
```

#### `boolean && boolean → boolean`

Boolean `AND`.

``` go
fmt.Println(path.MustQuery([]any{1, 3, 7}, "$[*] ? (@ > 1 && @ < 5)")) // → 3
```

#### `boolean || boolean → boolean`

Boolean `OR`.

``` go
fmt.Println(path.MustQuery([]any{1, 3, 7}, "$[*] ? (@ < 1 || @ > 5)")) // → 7
```

#### `! boolean → boolean`

Boolean `NOT`.

``` go
fmt.Println(path.MustQuery([]any{1, 3, 7}, "$[*] ? (!(@ < 5))")) // → 7
```

#### `boolean is unknown → boolean`

Tests whether a Boolean condition is unknown.

``` go
arg := []any{-1, 2, 7, "foo"}
fmt.Println(path.MustQuery(arg, "$[*] ? ((@ > 0) is unknown)")) // → "foo"
```

#### `string like_regex string [ flag string ] → boolean`

Tests whether the first operand matches the regular expression given by the
second operand, optionally with modifications described by a string of flag
characters (see [SQL/JSON Regular Expressions](#sqljson-regular-expressions)).

``` go
arg := []any{"abc", "abd", "aBdC", "abdacb", "babc"}
fmt.Println(path.MustQuery(
    arg, `$[*] ? (@ like_regex "^ab.*c")`,
)) // → ["abc", "abdacb"]

fmt.Println(path.MustQuery(
    arg, `$[*] ? (@ like_regex "^ab.*c" flag "i"`,
)) // → ["abc", "aBdC", "abdacb"]
```

#### `string starts with string → boolean`

Tests whether the second operand is an initial substring of the first operand.

``` go
arg := []any{"John Smith", "Mary Stone", "Bob Johnson"}
fmt.Println(path.MustQuery(arg, `$[*] ? (@ starts with "John")`)) // → "John Smith"
```

#### `exists ( path_expression ) → boolean`

Tests whether a path expression matches at least one SQL/JSON item. Returns
unknown if the path expression would result in an error; the second example
uses this to avoid a no-such-key error in strict mode.

``` go
arg := map[string]any{"x": []any{1, 2}, "y": []any{2, 4}}
fmt.Println(path.MustQuery(arg, "strict $.* ? (exists (@ ? (@[*] > 2)))")) // → [2, 4]

arg = map[string]any{"value": 41}
fmt.Println(path.MustQuery(arg, "strict $ ? (exists (@.name)) .name") // → []
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
package follows the example of PostgreSQL, using the [regexp] regular
expression engine to implement `like_regex`. This leads to various minor
discrepancies from standard SQL/JSON behavior, which are cataloged in [TBD].
Note, however, that the flag-letter incompatibilities described there do not
apply to SQL/JSON, as it translates the XQuery flag letters to match what
[regxp] expects.

Keep in mind that the pattern argument of `like_regex` is a JSON path string
literal, written according to the rules given [above](#syntax). This means in
particular that any backslashes you want to use in the regular expression must
be doubled. For example, to match string values of the root document that
contain only digits:

``` jsonpath
$.* ? (@ like_regex "^\\d+$")
```

We therefor recommend using Go literal strings to compose path expressions
with double quotes or backslashes, both of which are common in `like_regex`
expressions:

``` go
p := MustCompile(`$.* ? (@ like_regex "^\\d+$")`)
```

## Copyright

Copyright © 1996-2024 The PostgreSQL Global Development Group

  [PostgreSQL docs]: https://www.postgresql.org/docs/devel/functions-json.html#FUNCTIONS-SQLJSON-PATH
    "PostgreSQL Documentation: “The SQL/JSON Path Language”"
  [string literals]: https://go.dev/ref/spec#String_literals
    "Go Language Spec: String literals"
