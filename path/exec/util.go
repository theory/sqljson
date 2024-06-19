package exec

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/theory/sqljson/path/ast"
)

// castJSONNumber casts num to a an int64 (preferably) or to a float64,
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

// getNodeInt32 extracts an int32 from node and returns it. Returns an error
// if node is not an *ast.IntegerNode or its value is out of int32 range. The
// meth and field params are used in error messages.
func getNodeInt32(node ast.Node, meth any, field string) (int, error) {
	var num int64
	switch node := node.(type) {
	case *ast.IntegerNode:
		num = node.Int()
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

// getJSONInt32 casts val to int32 and returns it. If val is a float, its
// value will be truncated, not rounded. The op param is used in error
// messages.
func getJSONInt32(val any, op string) (int, error) {
	var num int64
	switch val := val.(type) {
	case int64:
		num = val
	case float64:
		if math.IsInf(val, 0) || math.IsNaN(val) {
			return 0, fmt.Errorf(
				"%w: NaN or Infinity is not allowed for jsonpath %v",
				ErrVerbose, op,
			)
		}
		num = int64(val)
	case json.Number:
		if integer, err := val.Int64(); err == nil {
			num = integer
		} else if float, err := val.Float64(); err == nil {
			if math.IsInf(float, 0) || math.IsNaN(float) {
				return 0, fmt.Errorf(
					"%w: NaN or Infinity is not allowed for jsonpath %v",
					ErrVerbose, op,
				)
			}
			num = int64(float)
		} else {
			// json.Number should never be invalid.
			return 0, fmt.Errorf(
				"%w: jsonpath %v is not a single numeric value",
				ErrInvalid, op,
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
