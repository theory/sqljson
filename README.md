Go SQL/JSON Path Playground
===========================

The source for the [Go SQL/JSON Path Playground], a stateless single-page web
site for experimenting with the [Go SQL/JSON Path] package. Compiled via [Go
WebAssembly] into a ca. 4.5 MB (1.3 MB compressed) [Wasm] file and loaded
directly into the page. All functionality implemented in JavaScript and Go,
[Go JSONPath Playground], [Goldmark Playground] and [serde_json_path Sandbox].

Usage
-----

On load, the form will be filled with sample JSON, a randomly-selected example
query, and, in some cases, option adjustments for the query. Hit the "Run
Query" button to see the values the path query selects from the JSON.

To try your own, paste the JSON to query into the "JSON" field and input the
jsonpath expression into the "Path" field, then hit the "Run Query" button to
see the the values the path query selects from the JSON.

That's it.

Read on for details and additional features.

### Docs

The two buttons in the top-right corner provide documentation and links.

*   Hit the button with the circled question mark in the top right corner to
    reveal a table summarizing the SQL/JSON Path syntax.

*   Hit the button with the circled i for information about the SQL/JSON Path
    playground.

### Mode

Choose the mode in which to execute the jsonpath query. The options are:

*   **Query**: Use [Query] to return an array of all the JSON items returned
    by the Path from the JSON.
*   **First**: Like Query, but uses [First] to return only the first item, if
    any.
*   **Exists or Match**: Use [ExistsOrMatch] to return `true` or `false`
    depending on whether the query does or does not find results or match
    values, and `null` if the result is unknown.

For the subtleties on the two behaviors of jsonpath expressions that use
`Exists` or `Match`, see [Two Types of Queries].

### Options

Select options for execution and the display of results:

*   **WithSilent**: Use [WithSilent] to suppress some errors, including missing
    object field or array element, unexpected JSON item type, and datetime and
    numeric errors.
*   **WithTZ**: Use [WithTZ] to allow comparisons of datetime values that
    require timezone-aware conversions.
*   **LocalTZ**: Use [ContextWithTZ] to parse times and timestamps in the
    context of your browser's local time zone instead of [UTC].

### Permalink

Hit this button to reload the page with a URL that contains the contents of
all the fields. Use for sharing.

Note that the Playground is stateless; no data is stored except in the
Permalink URL itself (and whatever data collection GitHub injects; see its
[privacy statement] for details).

### Path

Input the jsonpath expression to execute into this field. See the [language
docs] or the [PostgreSQL docs] for details on the jsonpath language. Example:

```jsonpath
$.a[*] ? (@ >= $min && @ <= $max)
```

### Variables

Input the variables used in the *Path* as a JSON object. For example, the
*Path* example above references two variables, `$min` and `$max`. The object
to set their values might be:

``` json
{ "min": 2, "max": 4 }
```

### JSON

Input the JSON against which to execute the *Path* expression. May be any kind
of JSON value, including objects, arrays, and scalar values. An example that
the above Path expression successfully executes against:

```json
{ "a": [1,2,3,4,5] }
```

## Syntax Summary

| Syntax Element     | Description                                                             |
| ------------------ | ----------------------------------------------------------------------- |
| `$`                | root node identifier                                                    |
| `@`                | current node identifier (valid only within filter selectors)            |
| `."name"`          | name selector: selects a named child of an object                       |
| `.name`            | shorthand for `."name"`                                                 |
| `.*`               | wildcard selector: selects all children of a node                       |
| `.**`              | recursive wildcard accessor: selects zero or more descendants of a node |
| `.**{3}`           | recursive wildcard accessor: selects up to specified level of hierarchy |
| `.**{2 to 5}`      | recursive wildcard accessor: selects from start to end level            |
| `[<subscripts>]`   | array selector with comma-delimited subscripts                          |
| `[3]`              | index selector subscript: selects an indexed child of an array          |
| `[3 to last]`      | array slice subscript: select slice from start to end index (or `last`) |
| `[*]`              | wildcard array selector: returns all array elements.                    |
| `$var_name`        | a variable referring to a value in the Vars object                      |
| `strict`           | raise error on a structural error                                       |
| `lax`              | suppress structural errors                                              |
| `?(<expr>)`        | filter selector: selects and transforms children                        |
| `.size()`          | method selector                                                         |

## Copyright and License

Copyright (c) 2024 David E. Wheeler. Distributed under the [PostgreSQL License].

Based on [Goldmark Playground] the [serde_json_path Sandbox], with icons from
[Boxicons], all distributed under the [MIT License].

  [Go SQL/JSON Path Playground]: https://theory.github.io/sqljson/playground
  [Go SQL/JSON Path]: https://pkg.go.dev/github.com/theory/sqljson/path
    "pkg.go.dev: github.com/theory/sqljson/path"
  [Wasm]: https://webassembly.org "WebAssembly"
  [Go WebAssembly]: https://go.dev/wiki/WebAssembly
  [Go JSONPath Playground]: https://theory.github.io/jsonpath/playground
  [Goldmark Playground]: https://yuin.github.io/goldmark/playground
  [serde_json_path Sandbox]: https://serdejsonpath.live
  [Query]: https://pkg.go.dev/github.com/theory/sqljson@v0.1.0/path#Path.Query
  [First]: https://pkg.go.dev/github.com/theory/sqljson@v0.1.0/path#Path.First
  [ExistsOrMatch]: https://pkg.go.dev/github.com/theory/sqljson@v0.1.0/path#Path.ExistsOrMatch
  [Two Types of Queries]: https://pkg.go.dev/github.com/theory/sqljson@v0.1.0/path#hdr-Two_Types_of_Queries
  [WithSilent]: https://pkg.go.dev/github.com/theory/sqljson@v0.1.0/path#example-package-WithSilent
  [WithTZ]: https://pkg.go.dev/github.com/theory/sqljson@v0.1.0/path#example-package-WithTZ
  [ContextWithTZ]: https://pkg.go.dev/github.com/theory/sqljson/path/types#ContextWithTZ
  [UTC]: https://en.wikipedia.org/wiki/Coordinated_Universal_Time
  [privacy statement]: https://docs.github.com/en/site-policy/privacy-policies/github-general-privacy-statement
  [language docs]: https://github.com/theory/sqljson/blob/main/path/README.md
  [PostgreSQL docs]: https://www.postgresql.org/docs/devel/functions-json.html#FUNCTIONS-SQLJSON-PATH
  [PostgreSQL License]: https://www.opensource.org/licenses/postgresql
  [MIT License]: https://opensource.org/license/mit
