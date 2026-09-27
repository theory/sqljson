Go SQL/JSON
===========

[![License](https://img.shields.io/badge/License-PostgreSQL-blue.svg)](https://opensource.org/license/postgresql "⚖️ License")
[![GoDoc](https://godoc.org/github.com/theory/sqljson?status.svg)](https://pkg.go.dev/github.com/theory/sqljson "📄 Documentation")
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
Implemented as a single-page stateless JavaScript and [TinyGo]-compiled [Wasm]
app.

## Strict No LLM/No AI Policy

This software is 100% made by humans for humans and welcomes human
contributions. That means:

*   No LLM-generated content, whether it be code or prose.
*   No paraphrasing LLM-generated content.
*   No LLMs for editing, including fixing spelling or grammatical errors.
*   No LLMs for translation. English is encouraged, but not required. You are
    welcome to post in your native language and rely on others to have their
    own translation tools of choice to interpret your words.
*   No LLMs for brainstorming and then sharing the results of that
    brainstorming, even if you create the prose. If you use a chatbot to give
    you advice on a comment on the issue tracker, that comment is unwelcome.
*   No LLMs for finding bugs.
*   No talking about use of chatbot/LLM services.

Borrowed from the [Zig Code of Conduct].

## Copyright

Copyright © 1996-2025 The PostgreSQL Global Development Group

Copyright © 2024-2026 David E. Wheeler

  [Go doc]: https://pkg.go.dev/github.com/theory/sqljson/path
  [🛝 Playground]: https://theory.github.io/sqljson/playground
  [direct link for above example]: https://theory.github.io/sqljson/playground/?p=%2524.track.segments%255B*%255D%2520%253F%2520%28%2540.HR%2520%253E%2520130%29.%2522start%2520time%2522&j=%257B%250A%2520%2520%2522track%2522%253A%2520%257B%250A%2520%2520%2520%2520%2522segments%2522%253A%2520%255B%250A%2520%2520%2520%2520%2520%2520%257B%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522location%2522%253A%2520%2520%2520%255B%252047.763%252C%252013.4034%2520%255D%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522start%2520time%2522%253A%2520%25222018-10-14%252010%253A05%253A14%2522%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522HR%2522%253A%252073%250A%2520%2520%2520%2520%2520%2520%257D%252C%250A%2520%2520%2520%2520%2520%2520%257B%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522location%2522%253A%2520%2520%2520%255B%252047.706%252C%252013.2635%2520%255D%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522start%2520time%2522%253A%2520%25222018-10-14%252010%253A39%253A21%2522%252C%250A%2520%2520%2520%2520%2520%2520%2520%2520%2522HR%2522%253A%2520135%250A%2520%2520%2520%2520%2520%2520%257D%250A%2520%2520%2520%2520%255D%250A%2520%2520%257D%250A%257D&a=&o=1&v=v0.1.0
  [TinyGo]: https://tinygo.org
  [Wasm]: https://webassembly.org "WebAssembly"
  [Zig Code of Conduct]: https://ziglang.org/code-of-conduct/
