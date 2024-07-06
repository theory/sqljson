Go SQL/JSON
===========

[![License](https://img.shields.io/badge/License-PostgreSQL-blue.svg)](https://opensource.org/license/postgresql "⚖️ License")
[![GoDoc](https://godoc.org/github.com/theory/sqljson?status.svg)](https://pkg.go.dev/github.com/theory/sqljson "📄 Documentation")
[![Go Report Card](https://goreportcard.com/badge/github.com/theory/sqljson)](https://goreportcard.com/report/github.com/theory/sqljson "🗃️ Report Card")
[![Build Status](https://github.com/theory/sqljson/actions/workflows/ci.yml/badge.svg)](https://github.com/theory/sqljson/actions/workflows/ci.yml "🛠️ Build Status")
[![Code Coverage](https://codecov.io/gh/theory/sqljson/graph/badge.svg?token=DIFED324ZY)](https://codecov.io/gh/theory/sqljson "📊 Code Coverage")

The SQL/JSON package provides PostgreSQL-compatible SQL-standard SQL/JSON
functionality in Go. For now that means [jsonpath](path/). An example:

``` go
func main() {
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
```

See the [path README](./path/README.md) for a complete description of the
SQL/JSON path language, and the [Go doc] for usage and examples.

Or take the [🛝 Playground] for a spin ([direct link for above example]).
Implemented as a single-page stateless JavaScript and [Go WebAssembly] app.

## Copyright

Copyright © 1996-2024 The PostgreSQL Global Development Group

Copyright © 2024 David E. Wheeler

  [Go doc]: https://pkg.go.dev/github.com/theory/sqljson/path
  [🛝 Playground]: https://theory.github.io/sqljson/playground
  [direct link for above example]: https://theory.github.io/sqljson/playground/?p=%2524.track.segments%255B*%255D%2520%253F%2520%28%2540.HR%2520%253E%2520130%29.%2522start%2520time%2522&j=%257B%250A%2520%2520%2522track%2522%253A%2520%257B%250A%2520%2520%2520%2520%2522segments%2522%253A%2520%255B%250A%2520%2520%2520%2520%2520%2520%257B%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522location%2522%253A%2520%2520%2520%255B%252047.763%252C%252013.4034%2520%255D%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522start%2520time%2522%253A%2520%25222018-10-14%252010%253A05%253A14%2522%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522HR%2522%253A%252073%250A%2520%2520%2520%2520%2520%2520%257D%252C%250A%2520%2520%2520%2520%2520%2520%257B%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522location%2522%253A%2520%2520%2520%255B%252047.706%252C%252013.2635%2520%255D%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522start%2520time%2522%253A%2520%25222018-10-14%252010%253A39%253A21%2522%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522HR%2522%253A%2520135%250A%2520%2520%2520%2520%2520%2520%257D%250A%2520%2520%2520%2520%255D%250A%2520%2520%257D%250A%257D&a=&o=1&v=v0.1.0
  [Go WebAssembly]: https://go.dev/wiki/WebAssembly
