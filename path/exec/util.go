package exec

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/theory/sqljson/path/ast"
)

// castJSONNumber casts a num to a an int64 (preferably) or to a float64,
// passing the result through intCallback or floatCallback, respectively.
// Returns false if num cannot be parsed into an int64 or float64.
func castJSONNumber(num json.Number, intCallback intCallback, floatCallback floatCallback) (any, bool) {
	if integer, err := num.Int64(); err == nil {
		return intCallback(integer), true
	} else if float, err := num.Float64(); err == nil {
		return floatCallback(float), true
	}

	return nil, false
}

// getNodeInt32 extracts an int32 from node and returns it. The meth and field
// params are used in error messages.
func getNodeInt32(meth any, node ast.Node, field string) (int, error) {
	var num int64
	switch node := node.(type) {
	case *ast.IntegerNode:
		num = node.Int()
	case *ast.NumericNode:
		num = int64(node.Float())
	default:
		return 0, fmt.Errorf(
			"%w: invalid jsonpath item type for %v %v",
			ErrExecution, meth, field,
		)
	}

	if num > math.MaxInt32 || num < math.MinInt32 {
		return 0, fmt.Errorf(
			"%w: %v of jsonpath item method %v is out of integer range",
			ErrVerbose, field, meth,
		)
	}

	return int(num), nil
}

// getJSONInt32 casts val to int32 and returns it. The op param is used in
// error messages.
func getJSONInt32(op string, val any) (int, error) {
	var num int64
	switch val := val.(type) {
	case int64:
		num = val
	case float64:
		num = int64(val)
	case json.Number:
		if integer, err := val.Int64(); err == nil {
			num = integer
		} else if float, err := val.Float64(); err == nil {
			num = int64(float)
		} else {
			// Should not happen.
			return 0, fmt.Errorf(
				"%w: jsonpath %v is not a single numeric value",
				ErrVerbose, op,
			)
		}
	default:
		return 0, fmt.Errorf(
			"%w: jsonpath %v is not a single numeric value",
			ErrVerbose, op,
		)
	}

	if num > math.MaxInt32 || num < math.MinInt32 {
		return 0, fmt.Errorf(
			"%w: jsonpath %v is out of integer range",
			ErrVerbose, op,
		)
	}

	return int(num), nil
}
