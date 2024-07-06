package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	optTZ
	optIndent
)

func query(_ js.Value, args []js.Value) any {
	query := args[0].String()
	target := args[1].String()
	vars := args[2].String()
	tz := args[3].String()
	opts := args[4].Int()

	return execute(query, target, vars, tz, opts)
}

func main() {
	c := make(chan struct{}, 0)

	js.Global().Set("query", js.FuncOf(query))
	js.Global().Set("optQuery", js.ValueOf(optQuery))
	js.Global().Set("optExistsOrMatch", js.ValueOf(optExistsOrMatch))
	js.Global().Set("optFirst", js.ValueOf(optFirst))
	js.Global().Set("optSilent", js.ValueOf(optSilent))
	js.Global().Set("optTZ", js.ValueOf(optTZ))
	js.Global().Set("optIndent", js.ValueOf(optIndent))

	<-c
}

func execute(query, target, vars, tz string, opts int) string {
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

	// Parse the time zone (currently returns unimplemented error).
	ctx := context.Background()
	if zone, err := time.LoadLocation(tz); zone != nil {
		ctx = types.ContextWithTZ(ctx, zone)
	} else if err != nil {
		// ERR: not implemented on js
		// log.Printf("tzdata: %v", err)
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
	var js []byte
	if opts&optIndent == optIndent {
		js, err = json.MarshalIndent(res, "", "  ")
	} else {
		js, err = json.Marshal(res)
	}

	if err != nil {
		return fmt.Sprintf("Error parsing results: %v", err)
	}

	return string(js)
}

func assembleOptions(opts int, vars string) ([]exec.Option, string) {
	options := []exec.Option{}
	if opts&optSilent == optSilent {
		options = append(options, exec.WithSilent())
	}

	if opts&optTZ == optTZ {
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
