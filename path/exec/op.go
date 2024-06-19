package exec

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/theory/sqljson/path/ast"
	"golang.org/x/exp/maps"
)

// execBinaryNode executes node's binary operation against value.
func (exec *Executor) execBinaryNode(
	ctx context.Context,
	node *ast.BinaryNode,
	value any,
	found *valueList,
	unwrap bool,
) (resultStatus, error) {
	switch node.Operator() {
	case ast.BinaryAnd, ast.BinaryOr, ast.BinaryEqual, ast.BinaryNotEqual,
		ast.BinaryLess, ast.BinaryLessOrEqual, ast.BinaryGreater,
		ast.BinaryGreaterOrEqual, ast.BinaryStartsWith:
		// Binary boolean types.
		res, err := exec.executeBoolItem(ctx, node, value, true)
		return exec.appendBoolResult(ctx, node, found, res, err)
	case ast.BinaryAdd, ast.BinarySub, ast.BinaryMul, ast.BinaryDiv, ast.BinaryMod:
		return exec.execBinaryMathExpr(ctx, node, value, found)
	case ast.BinaryDecimal:
		return exec.executeNumberMethod(ctx, node, value, found, unwrap, node.Operator())
	case ast.BinarySubscript:
		// This should not happen because the Parser disallows it.
		return statusFailed, fmt.Errorf(
			"%w: evaluating jsonpath subscript expression outside of array subscript",
			ErrExecution,
		)
	}

	return statusNotFound, nil
}

// execBinaryNode executes node's unary operation against value.
func (exec *Executor) execUnaryNode(
	ctx context.Context,
	node *ast.UnaryNode,
	value any,
	found *valueList,
	unwrap bool,
) (resultStatus, error) {
	switch node.Operator() {
	case ast.UnaryNot, ast.UnaryIsUnknown, ast.UnaryExists:
		// Binary boolean types.
		res, err := exec.executeBoolItem(ctx, node, value, true)
		return exec.appendBoolResult(ctx, node, found, res, err)
	case ast.UnaryFilter:
		if unwrap {
			if _, ok := value.([]any); ok {
				return exec.executeItemUnwrapTargetArray(ctx, node, value, found)
			}
		}

		st, err := exec.executeNestedBoolItem(ctx, node.Operand(), value)
		if st != predTrue {
			return statusNotFound, err
		}
		return exec.executeNextItem(ctx, node, nil, value, found)
	case ast.UnaryPlus:
		return exec.execUnaryMathExpr(ctx, node, value, intSelf, floatSelf, found)
	case ast.UnaryMinus:
		return exec.execUnaryMathExpr(ctx, node, value, intUMinus, floatUMinus, found)
	case ast.UnaryDateTime, ast.UnaryDate, ast.UnaryTime, ast.UnaryTimeTZ,
		ast.UnaryTimestamp, ast.UnaryTimestampTZ:
		if unwrap {
			if array, ok := value.([]any); ok {
				return exec.executeAnyItem(ctx, node, array, found, 1, 1, 1, false, false)
			}
		}
		return exec.executeDateTimeMethod(ctx, node, value, found)
	}

	return statusNotFound, nil
}

// execRegexNode executes regex against value.
func (exec *Executor) execRegexNode(
	ctx context.Context,
	regex *ast.RegexNode,
	value any,
	found *valueList,
) (resultStatus, error) {
	// Binary boolean type.
	res, err := exec.executeBoolItem(ctx, regex, value, true)
	return exec.appendBoolResult(ctx, regex, found, res, err)
}

func (exec *Executor) tempSetIgnoreStructuralErrors(val bool) func() {
	savedIgnoreStructuralErrors := exec.ignoreStructuralErrors
	exec.ignoreStructuralErrors = val
	return func() { exec.ignoreStructuralErrors = savedIgnoreStructuralErrors }
}

// execAnyNode handles the execution of node. value must be either
// map[string]any or []any. If found is not nil then resultStatus should be
// ignored.
func (exec *Executor) execAnyNode(
	ctx context.Context,
	node *ast.AnyNode,
	value any,
	found *valueList,
) (resultStatus, error) {
	next := node.Next()
	// first try without any intermediate steps
	if node.First() == 0 {
		defer exec.tempSetIgnoreStructuralErrors(true)()
		res, err := exec.executeNextItem(ctx, node, next, value, found)
		if err != nil || (res == statusOK && found == nil) {
			return res, err
		}
	}

	switch value := value.(type) {
	case map[string]any:
		return exec.executeAnyItem(
			ctx, next, maps.Values(value), found, 1,
			node.First(), node.Last(), true, exec.autoUnwrap(),
		)
	case []any:
		return exec.executeAnyItem(
			ctx, next, value, found, 1,
			node.First(), node.Last(), true, exec.autoUnwrap(),
		)
	}

	return statusNotFound, nil
}

// collection converts v into a slice of values if it's either a map or a
// slice. Otherwise it returns nil.
func collection(v any) []any {
	switch v := v.(type) {
	case map[string]any:
		return maps.Values(v) // Just work with the values
	case []any:
		return v
	}
	return nil
}

// executeAnyItem is the implementation of several jsonpath nodes:
//
//   - ast.AnyNode (.** accessor)
//   - ast.ConstAnyKey (.* accessor)
//   - ast.ConstAnyArray ([*] accessor)
//
// The value parameter must be a slice of values; the caller must properly
// extract the values from a map. If found is not nil then resultStatus should
// be ignored.
func (exec *Executor) executeAnyItem(
	ctx context.Context,
	node ast.Node,
	value []any,
	found *valueList,
	level, first, last uint32,
	ignoreStructuralErrors, unwrapNext bool,
) (resultStatus, error) {
	res := statusNotFound
	var err error
	if level > last {
		return res, nil
	}

	// When found is not nil, executeAnyItem can return statusNotFound even
	// when items were found. This seems to be because it returns the last
	// result in the list it iterates over or from a recursive call. This
	// isn't super important for the top-level query functions, which pay
	// attention to either the contents of found or the result, and not both.
	// But to be internally consistent, look at the size of the found values
	// and return statusOK below if it has grown, regardless of what the last
	// result was.
	size := 0
	if found != nil {
		size = len(found.list)
	}

	// Recursively iterate over jsonb objects/arrays
	for _, v := range value {
		col := collection(v)

		if level >= first || (first == math.MaxUint32 && last == math.MaxUint32 && col == nil) {
			// check expression
			switch {
			case node != nil:
				if ignoreStructuralErrors {
					defer exec.tempSetIgnoreStructuralErrors(true)()
				}
				res, err = exec.executeItemOptUnwrapTarget(ctx, node, v, found, unwrapNext)
				if res.failed() || (res == statusOK && found == nil) {
					return res, err
				}
			case found != nil:
				found.append(v)
				res = statusOK
			default:
				return statusOK, nil
			}
		}

		if level < last {
			res, err = exec.executeAnyItem(
				ctx, node, col, found, level+1, first, last, ignoreStructuralErrors, unwrapNext,
			)
			if res.failed() || (res == statusOK && found == nil) {
				return res, err
			}
		}
	}

	// Always return OK if items were found.
	if found != nil && res != statusFailed && err == nil && len(found.list) > size {
		res = statusOK
	}

	return res, err
}

// executeLikeRegex is the LIKE_REGEX predicate callback.
// Implements predicateCallback.
func (exec *Executor) executeLikeRegex(node ast.Node, value, _ any) (predOutcome, error) {
	rn, ok := node.(*ast.RegexNode)
	if !ok {
		panic(fmt.Sprintf(
			"Node %T passed to executeLikeRegex is not an ast.RegexNode",
			node,
		))
	}

	str, ok := value.(string)
	if !ok {
		return predUnknown, nil
	}

	if rn.Regexp().MatchString(str) {
		return predTrue, nil
	}
	return predFalse, nil
}

// executeStartsWith is the STARTS_WITH predicate callback. It returns
// predTrue when whole string starts with initial and predFalse if it does
// not. Returns predUnknown if either whole or initial is not a string.
// Implements predicateCallback.
func executeStartsWith(_ ast.Node, whole, initial any) (predOutcome, error) {
	if str, ok := whole.(string); ok {
		if prefix, ok := initial.(string); ok {
			if strings.HasPrefix(str, prefix) {
				return predTrue, nil
			}
			return predFalse, nil
		}
	}
	return predUnknown, nil
}
