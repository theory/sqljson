// Package main performs a basic JSONPath query in order to test WASM compilation.
package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/theory/sqljson/path"
)

func main() {
	// Parse a jsonpath query.
	p, _ := path.Parse(`$.foo`)

	// Select values from unmarshaled JSON input.
	result, _ := p.Query(context.Background(), []byte(`{"foo": "bar"}`))

	// Show the result.
	//nolint:errchkjson
	items, _ := json.Marshal(result)

	//nolint:forbidigo
	fmt.Printf("%s\n", items)
}
