package exec

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/theory/sqljson/path/ast"
	"github.com/theory/sqljson/path/types"
)

// compareItems compares two SQL/JSON items using comparison operation 'op'.
// Implements predicateCallback.
func (exec *Executor) compareItems(ctx context.Context, node ast.Node, left, right any) (predOutcome, error) {
	var cmp int
	bin, ok := node.(*ast.BinaryNode)
	if !ok {
		return predUnknown, fmt.Errorf(
			"%w: invalid node type %T passed to compareItems", ErrInvalid, node,
		)
	}
	op := bin.Operator()

	if (left == nil && right != nil) || (right == nil && left != nil) {
		// Equality and order comparison of nulls to non-nulls returns
		// always false, but inequality comparison returns true.
		return predFrom(op == ast.BinaryNotEqual), nil
	}

	switch left := left.(type) {
	case nil:
		cmp = 0
	case bool:
		cmp, ok = compareBool(left, right)
		if !ok {
			return predUnknown, nil
		}
	case int64, float64, json.Number:
		switch right.(type) {
		case int64, float64, json.Number:
			cmp = compareNumeric(left, right)
		default:
			return predUnknown, nil
		}
	case string:
		right, ok := right.(string)
		if !ok {
			return predUnknown, nil
		}
		cmp = strings.Compare(left, right)
		if op == ast.BinaryEqual {
			return predFrom(cmp == 0), nil
		}
	case *types.Date, *types.Time, *types.TimeTZ, *types.Timestamp, *types.TimestampTZ:
		var err error
		cmp, err = compareDatetime(ctx, left, right, exec.useTZ)
		if cmp < -1 || err != nil {
			return predUnknown, err
		}
	case map[string]any, []any:
		// non-scalars are not comparable
		return predUnknown, nil
	default:
		return predUnknown, fmt.Errorf(
			"%w: invalid json value type %T", ErrInvalid, left,
		)
	}

	return applyCompare(op, cmp)
}

// compareBool compares two boolean values and returns 0, 1, or -1. Returns
// false if right is not a bool.
func compareBool(left bool, right any) (int, bool) {
	right, ok := right.(bool)
	if !ok {
		return 0, false
	}
	switch {
	case left == right:
		return 0, true
	case left:
		return 1, true
	default:
		return -1, true
	}
}

// applyCompare applies op relative to cmp.
func applyCompare(op ast.BinaryOperator, cmp int) (predOutcome, error) {
	switch op {
	case ast.BinaryEqual:
		return predFrom(cmp == 0), nil
	case ast.BinaryNotEqual:
		return predFrom(cmp != 0), nil
	case ast.BinaryLess:
		return predFrom(cmp < 0), nil
	case ast.BinaryGreater:
		return predFrom(cmp > 0), nil
	case ast.BinaryLessOrEqual:
		return predFrom(cmp <= 0), nil
	case ast.BinaryGreaterOrEqual:
		return predFrom(cmp >= 0), nil
	default:
		// We only process binary comparison operators here.
		return predUnknown, fmt.Errorf(
			"%w: unrecognized jsonpath comparison operation %v", ErrInvalid, op,
		)
	}
}

// compareNumbers compares two numbers and returns 0, 1, or -1.
func compareNumbers[T int | int64 | float64](left, right T) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}

// compareBool compares two numeric values and returns 0, 1, or -1. The left
// and right params must be int64, float64, or json.Number values.
func compareNumeric(left, right any) int {
	switch left := left.(type) {
	case int64:
		switch right := right.(type) {
		case int64:
			return compareNumbers(left, right)
		case float64:
			return compareNumbers(float64(left), right)
		case json.Number:
			if right, err := right.Int64(); err == nil {
				return compareNumbers(left, right)
			}
			if right, err := right.Float64(); err == nil {
				return compareNumbers(float64(left), right)
			} else {
				// This should not happen.
				panic(err)
			}
		}
	case float64:
		switch right := right.(type) {
		case float64:
			return compareNumbers(left, right)
		case int64:
			return compareNumbers(left, float64(right))
		case json.Number:
			if right, err := right.Float64(); err == nil {
				return compareNumbers(left, right)
			} else {
				// This should not happen.
				panic(err)
			}
		}
	case json.Number:
		if left, err := left.Int64(); err == nil {
			return compareNumeric(left, right)
		}
		if left, err := left.Float64(); err == nil {
			return compareNumeric(left, right)
		} else {
			// This should not happen.
			panic(err)
		}
	}

	// This should not happen
	panic(fmt.Sprintf("Value not numeric: %q", left))
}
