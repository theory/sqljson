Go SQL/JSON Path Playground
===========================

The source for the [Go SQL/JSON Path Playground], a stateless single-page web
site for experimenting with the [Go SQL/JSON Path] package. Compiled
via [Go WebAssembly] into a ca. 5MB [Wasm] file and loaded directly into the
page. All functionality implemented in JavaScript and Go, heavily borrowed
from the [Goldmark Playground].

Usage
-----

To get started, paste the JSON to query into the JSON field and input the
jsonpath expression into the Path field, then hit the "Execute" button to see
the result of the path expression executed on the JSON.

That's it.

Read on for details and additional features.

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
*   **LocalTZ**: Use [ContextWithTZ] to convert and display times and
    timestamps in the context of your browser's local time zone instead of
    [UTC].

### Permalink

Once a query has been executed, this link will become active. Hit it to reload
the page with a URL that contains the contents of all the fields and executes
the results. Use for sharing.

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

Input the variables used in the Path as a JSON object. For example, the Path
example above references two variables, `$min` and `$max`. The object to set
their values might be:

``` json
{ "min": 2, "max": 4 }
```

### JSON

Input the JSON against which to execute the Path expression. May be any kind
of JSON value, including objects, arrays, and scalar values. An example that
the above Path expression successfully executes against:

```json
{ "a": [1,2,3,4,5] }
```

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
