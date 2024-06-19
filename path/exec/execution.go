package exec

import (
	"context"
	"fmt"

	"github.com/theory/sqljson/path/ast"
)

// query is the main entry point for all path executions. It executes node
// against value and appends results to vals if vals is not nil. Returns
// statusOK if values are found, statusNotFound if none are found, and
// statusFailed on error. When statusFailed is returned, an error will also be
// returned, except when query.verbose is false and the error is ErrVerbose.
func (exec *Executor) query(ctx context.Context, vals *valueList, node ast.Node, value any) (resultStatus, error) {
	if exec.strictAbsenceOfErrors() && vals == nil {
		// In strict mode we must get a complete list of values to check that
		// there are no errors at all.
		vals := newList()
		res, err := exec.executeItem(ctx, node, value, vals)
		if res.failed() {
			return res, err
		}

		if vals.isEmpty() {
			return statusNotFound, nil
		}
		return statusOK, nil
	}

	return exec.executeItem(ctx, node, value, vals)
}

// executeItem executes jsonpath with automatic unwrapping of current item in
// lax mode.
func (exec *Executor) executeItem(
	ctx context.Context,
	node ast.Node,
	value any,
	found *valueList,
) (resultStatus, error) {
	return exec.executeItemOptUnwrapTarget(ctx, node, value, found, exec.autoUnwrap())
}

// executeItemOptUnwrapResult is the same as executeItem(), but when unwrap is
// true, it automatically unwraps each array item from the resulting sequence
// in lax mode. The found parameter must not be nil.
func (exec *Executor) executeItemOptUnwrapResult(
	ctx context.Context,
	node ast.Node,
	value any,
	unwrap bool,
	found *valueList,
) (resultStatus, error) {
	if unwrap && exec.autoUnwrap() {
		seq := newList()
		res, err := exec.executeItem(ctx, node, value, seq)
		if res.failed() {
			return res, err
		}

		for _, item := range seq.list {
			switch item := item.(type) {
			case []any:
				_, _ = exec.executeItemUnwrapTargetArray(ctx, nil, item, found)
			default:
				found.append(item)
			}
		}
		return statusOK, nil
	}
	return exec.executeItem(ctx, node, value, found)
}

// executeItemOptUnwrapResultSilent is the same as executeItemOptUnwrapResult,
// but with error suppression.
func (exec *Executor) executeItemOptUnwrapResultSilent(
	ctx context.Context,
	node ast.Node,
	value any,
	unwrap bool,
	found *valueList,
) (resultStatus, error) {
	verbose := exec.verbose
	exec.verbose = false
	defer func(e *Executor, te bool) { e.verbose = te }(exec, verbose)
	return exec.executeItemOptUnwrapResult(ctx, node, value, unwrap, found)
}

// executeItemOptUnwrapTarget is the main executor function: walks on jsonpath
// structure, finds relevant parts of value and evaluates expressions over
// them. When unwrap is true, the current SQL/JSON item is unwrapped if it is
// an array. Before execution it checks ctx and base with statusNotFound if it
// is done.
func (exec *Executor) executeItemOptUnwrapTarget(
	ctx context.Context,
	node ast.Node,
	value any,
	found *valueList,
	unwrap bool,
) (resultStatus, error) {
	// Check for interrupts.
	select {
	case <-ctx.Done():
		return statusFailed, fmt.Errorf("%w: %w", ErrExecution, ctx.Err())
	default:
	}

	switch node := node.(type) {
	case *ast.ConstNode:
		return exec.execConstNode(ctx, node, value, found, unwrap)
	case *ast.StringNode:
		return exec.execLiteral(ctx, node, node.Text(), found)
	case *ast.IntegerNode:
		return exec.execLiteral(ctx, node, node.Int(), found)
	case *ast.NumericNode:
		return exec.execLiteral(ctx, node, node.Float(), found)
	case *ast.VariableNode:
		return exec.execVariable(ctx, node, found)
	case *ast.KeyNode:
		return exec.execKeyNode(ctx, node, value, found, unwrap)
	case *ast.BinaryNode:
		return exec.execBinaryNode(ctx, node, value, found, unwrap)
	case *ast.UnaryNode:
		return exec.execUnaryNode(ctx, node, value, found, unwrap)
	case *ast.RegexNode:
		return exec.execRegexNode(ctx, node, value, found)
	case *ast.MethodNode:
		return exec.execMethodNode(ctx, node, value, found, unwrap)
	case *ast.AnyNode:
		return exec.execAnyNode(ctx, node, value, found)
	case *ast.ArrayIndexNode:
		return exec.execArrayIndex(ctx, node, value, found)
	}

	return statusFailed, fmt.Errorf("%w: Unknown node type %T", ErrInvalid, node)
}

// executeNextItem executes the next jsonpath item if it exists. Otherwise, if
// found is not nil it appends value to found.
func (exec *Executor) executeNextItem(
	ctx context.Context,
	cur, next ast.Node,
	value any,
	found *valueList,
) (resultStatus, error) {
	var hasNext bool
	switch {
	case cur == nil:
		hasNext = next != nil
	case next != nil:
		hasNext = cur.Next() != nil
	default:
		next = cur.Next()
		hasNext = next != nil
	}

	if hasNext {
		return exec.executeItem(ctx, next, value, found)
	}

	if found != nil {
		found.append(value)
	}

	return statusOK, nil
}
