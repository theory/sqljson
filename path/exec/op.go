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
		return exec.execBinaryMathExpr(ctx, node, value, node.Operator(), found)
	case ast.BinaryDecimal:
		return exec.executeNumberMethod(ctx, node, value, found, unwrap)
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

// execAnyNode handles the execution of node. value' must be either
// map[string]any or []any.
func (exec *Executor) execAnyNode(
	ctx context.Context,
	node *ast.AnyNode,
	value any,
	found *valueList,
) (resultStatus, error) {
	next := node.Next()
	// first try without any intermediate steps
	if node.First() == 0 {
		savedIgnoreStructuralErrors := exec.ignoreStructuralErrors
		defer func() { exec.ignoreStructuralErrors = savedIgnoreStructuralErrors }()
		exec.ignoreStructuralErrors = true
		res, err := exec.executeNextItem(ctx, node, next, value, found)
		if res == statusOK && found == nil {
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

// executeAnyItem is the implementation of several jsonpath nodes:
//
//   - ast.AnyNode (.** accessor)
//   - ast.ConstAnyKey (.* accessor)
//   - ast.ConstAnyArray ([*] accessor)
//
// The value parameter must be a slice of values; the caller must properly
// extract the values from a map.
func (exec *Executor) executeAnyItem(
	ctx context.Context,
	node ast.Node,
	value []any,
	found *valueList,
	level, first, last uint32,
	ignoreStructuralErrors, unwrapNext bool,
) (resultStatus, error) {
	// Check for interrupts.
	select {
	case <-ctx.Done():
		return statusNotFound, nil
	default:
	}

	res := statusNotFound
	var err error
	if level > last {
		return res, nil
	}

	for _, v := range value {
		var col []any
		switch v := v.(type) {
		case map[string]any:
			col = maps.Values(v) // Just work with the values
		case []any:
			col = v
		}

		if level >= first || (first == math.MaxUint32 && last == math.MaxUint32 && col == nil) {
			// check expression
			switch {
			case node != nil:
				if ignoreStructuralErrors {
					savedIgnoreStructuralErrors := exec.ignoreStructuralErrors
					exec.ignoreStructuralErrors = true
					res, err = exec.executeItemOptUnwrapTarget(ctx, node, v, found, unwrapNext)
					exec.ignoreStructuralErrors = savedIgnoreStructuralErrors
				} else {
					res, err = exec.executeItemOptUnwrapTarget(ctx, node, v, found, unwrapNext)
				}

				if res.failed() || (res == statusOK && found == nil) {
					return res, err
				}
			case found != nil:
				found.append(v)
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
func (exec *Executor) executeStartsWith(_ ast.Node, whole, initial any) (predOutcome, error) {
	//nolint:gocritic // We want the single type check because .(string) would
	//convert.
	switch str := whole.(type) {
	case string:
		switch prefix := initial.(type) {
		case string:
			if strings.HasPrefix(str, prefix) {
				return predTrue, nil
			}
			return predFalse, nil
		}
	}
	return predUnknown, nil
}
