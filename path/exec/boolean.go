package exec

import (
	"context"
	"fmt"

	"github.com/theory/sqljson/path/ast"
)

// executeBinaryBoolItem executes node against value and returns the result.
func (exec *Executor) executeBinaryBoolItem(
	ctx context.Context,
	node *ast.BinaryNode,
	value any,
) (predOutcome, error) {
	switch node.Operator() {
	case ast.BinaryAnd:
		res, err := exec.executeBoolItem(ctx, node.Left(), value, false)
		if res == predFalse {
			return res, err
		}

		// SQL/JSON says that we should check second arg in case of error
		res2, err2 := exec.executeBoolItem(ctx, node.Right(), value, false)
		if res2 == predTrue {
			return res, err2
		}
		return res2, err
	case ast.BinaryOr:
		res, err := exec.executeBoolItem(ctx, node.Left(), value, false)
		if res == predTrue {
			return res, err
		}
		res2, err2 := exec.executeBoolItem(ctx, node.Right(), value, false)
		if res2 == predFalse {
			return res, err
		}
		return res2, err2
	case ast.BinaryEqual, ast.BinaryNotEqual, ast.BinaryLess,
		ast.BinaryGreater, ast.BinaryLessOrEqual, ast.BinaryGreaterOrEqual:
		return exec.executePredicate(ctx, node, node.Left(), node.Right(), value, true, exec.compareItems)
	case ast.BinaryStartsWith:
		return exec.executePredicate(ctx, node, node.Left(), node.Right(), value, false, exec.executeStartsWith)
	default:
		return predFalse, fmt.Errorf(
			"%w: invalid jsonpath boolean operator %T",
			ErrInvalid, node.Operator(),
		)
	}
}

// executeUnaryBoolItem executes node, which must be a ast.UnaryNot,
// ast.UnaryIsUnknown, or ast.UnaryExists operator, against value.
func (exec *Executor) executeUnaryBoolItem(
	ctx context.Context,
	node *ast.UnaryNode,
	value any,
) (predOutcome, error) {
	switch node.Operator() {
	case ast.UnaryNot:
		res, err := exec.executeBoolItem(ctx, node.Operand(), value, false)
		switch res {
		case predUnknown:
			return res, err
		case predTrue:
			return predFalse, nil
		case predFalse:
			return predTrue, nil
		}
	case ast.UnaryIsUnknown:
		res, _ := exec.executeBoolItem(ctx, node.Operand(), value, false)
		return predFrom(res == predUnknown), nil
	case ast.UnaryExists:
		if exec.strictAbsenceOfErrors() {
			// In strict mode we must get a complete list of values to
			// check that there are no errors at all.
			vals := newList()
			res, err := exec.executeItemOptUnwrapResultSilent(ctx, node.Operand(), value, false, vals)
			if res == statusFailed {
				return predUnknown, err
			}
			if vals.isEmpty() {
				return predFalse, nil
			}
			return predTrue, nil
		}

		res, err := exec.executeItemOptUnwrapResultSilent(ctx, node.Operand(), value, false, nil)
		if res == statusFailed {
			return predUnknown, err
		}
		if res == statusOK {
			return predTrue, nil
		}
		return predFalse, nil
	default:
		// We only process boolean unary operators here.
	}

	return predFalse, fmt.Errorf(
		"%w: invalid jsonpath boolean operator %T",
		ErrInvalid, node.Operator(),
	)
}

// executeBoolItem executes node, which must be a ast.BinaryNode,
// ast.UnaryNode, or ast.RegexNode, against value.
func (exec *Executor) executeBoolItem(
	ctx context.Context,
	node ast.Node,
	value any,
	canHaveNext bool,
) (predOutcome, error) {
	if !canHaveNext && node.Next() != nil {
		return predUnknown, fmt.Errorf(
			"%w: boolean jsonpath item cannot have next item", ErrInvalid,
		)
	}

	switch node := node.(type) {
	case *ast.BinaryNode:
		return exec.executeBinaryBoolItem(ctx, node, value)
	case *ast.UnaryNode:
		return exec.executeUnaryBoolItem(ctx, node, value)
	case *ast.RegexNode:
		return exec.executePredicate(ctx, node, node.Operand(), nil, value, false, exec.executeLikeRegex)
	}

	return predUnknown, fmt.Errorf(
		"%w: invalid boolean jsonpath item type: %T",
		ErrInvalid, node,
	)
}

// appendBoolResult convert boolean execution status res to a boolean JSON
// value and executes the next jsonpath.
func (exec *Executor) appendBoolResult(
	ctx context.Context,
	node ast.Node,
	found *valueList,
	res predOutcome,
	err error,
) (resultStatus, error) {
	if err != nil {
		return statusFailed, err
	}

	next := node.Next()
	if next == nil && found == nil {
		// found singleton boolean value
		return statusOK, nil
	}
	var value any

	if res == predUnknown {
		value = nil
	} else {
		value = res == predTrue
	}

	return exec.executeNextItem(ctx, node, next, value, found)
}

// executeNestedBoolItem executes a nested (filters etc.) boolean expression
// pushing current SQL/JSON item onto the stack.
func (exec *Executor) executeNestedBoolItem(
	ctx context.Context,
	node ast.Node,
	value any,
) (predOutcome, error) {
	prev := exec.current
	defer func(e *Executor, c any) { e.current = c }(exec, prev)
	exec.current = value
	return exec.executeBoolItem(ctx, node, value, false)
}
