package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"time"

	//nolint
	"syscall/js"

	"github.com/theory/sqljson/path"
	"github.com/theory/sqljson/path/exec"
	"github.com/theory/sqljson/path/types"
)

const (
	optQuery int = 1 << iota
	optExistsOrMatch
	optFirst
	optSilent
	optTZCompare
	optLocalTZ
	optIndent
)

func query(_ js.Value, args []js.Value) any {
	query := args[0].String()
	target := args[1].String()
	vars := args[2].String()
	opts := args[3].Int()

	return execute(query, target, vars, opts)
}

func main() {
	c := make(chan struct{}, 0)

	js.Global().Set("query", js.FuncOf(query))
	js.Global().Set("optQuery", js.ValueOf(optQuery))
	js.Global().Set("optExistsOrMatch", js.ValueOf(optExistsOrMatch))
	js.Global().Set("optFirst", js.ValueOf(optFirst))
	js.Global().Set("optSilent", js.ValueOf(optSilent))
	js.Global().Set("optTZCompare", js.ValueOf(optTZCompare))
	js.Global().Set("optLocalTZ", js.ValueOf(optLocalTZ))
	js.Global().Set("optIndent", js.ValueOf(optIndent))

	<-c
}

func execute(query, target, vars string, opts int) string {
	// Parse the JSON.
	var value any
	if err := json.Unmarshal([]byte(target), &value); err != nil {
		return fmt.Sprintf("Error parsing JSON: %v", err)
	}

	// Parse the SQL jsonpath query.
	p, err := path.Parse(query)
	if err != nil {
		return fmt.Sprintf("Error parsing %v", err)
	}

	// Use local time zone if requested.
	ctx := context.Background()
	if opts&optLocalTZ == optLocalTZ {
		ctx = types.ContextWithTZ(ctx, time.Local)
	}

	// Assemble the options.
	options, msg := assembleOptions(opts, vars)
	if msg != "" {
		return msg
	}

	// Execute the query against the JSON.
	var res any
	switch {
	case opts&optQuery == optQuery:
		res, err = p.Query(ctx, value, options...)
	case opts&optExistsOrMatch == optExistsOrMatch:
		res, err = p.ExistsOrMatch(ctx, value, options...)
	case opts&optFirst == optFirst:
		res, err = p.First(ctx, value, options...)
	}

	// Error handling.
	if err != nil {
		if errors.Is(err, exec.NULL) {
			return "null"
		}
		return fmt.Sprintf("Error %v", err)
	}

	// Serialize the result
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if opts&optIndent == optIndent {
		enc.SetIndent("", "  ")
	}
	if err := enc.Encode(res); err != nil {
		return fmt.Sprintf("Error parsing results: %v", err)
	}

	return html.EscapeString(buf.String())
}

func assembleOptions(opts int, vars string) ([]exec.Option, string) {
	options := []exec.Option{}
	if opts&optSilent == optSilent {
		options = append(options, exec.WithSilent())
	}

	if opts&optTZCompare == optTZCompare {
		options = append(options, exec.WithTZ())
	}

	if vars != "" {
		var varsMap map[string]any
		if err := json.Unmarshal([]byte(vars), &varsMap); err != nil {
			return nil, fmt.Sprintf("Error parsing variables: %v", err)
		}

		options = append(options, exec.WithVars(varsMap))
	}

	return options, ""
}
