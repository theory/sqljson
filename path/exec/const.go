package exec

import (
	"context"
	"fmt"

	"github.com/theory/sqljson/path/ast"
	"golang.org/x/exp/maps" // Switch to maps when go 1.22 dropped
)

// execConstNode Executes node against value.
func (exec *Executor) execConstNode(
	ctx context.Context,
	node *ast.ConstNode,
	value any,
	found *valueList,
	unwrap bool,
) (resultStatus, error) {
	switch node.Const() {
	case ast.ConstNull, ast.ConstTrue, ast.ConstFalse:
		return exec.execLiteralConst(ctx, node, found)
	case ast.ConstRoot:
		defer exec.setTempBaseObject(exec.root, 0)()
		return exec.executeNextItem(ctx, node, nil, exec.root, found)
	case ast.ConstCurrent:
		return exec.executeNextItem(ctx, node, nil, exec.current, found)
	case ast.ConstAnyKey:
		return exec.execAnyKey(ctx, node, value, found, unwrap)
	case ast.ConstAnyArray:
		return exec.execAnyArray(ctx, node, value, found)
	case ast.ConstLast:
		return exec.execLastConst(ctx, node, found)
	}

	// Should only happen if a new constant ast.Constant is not added to the
	// switch statement above.
	return statusFailed, fmt.Errorf(
		"%w: Unknown ConstNode %v", ErrInvalid, node.Const(),
	)
}

// execLiteralConst handles the execution of a null or boolean node.
func (exec *Executor) execLiteralConst(
	ctx context.Context,
	node *ast.ConstNode,
	found *valueList,
) (resultStatus, error) {
	next := node.Next()
	if next == nil && found == nil {
		return statusOK, nil
	}

	var v any
	if node.Const() == ast.ConstNull {
		v = nil
	} else {
		v = node.Const() == ast.ConstTrue
	}

	return exec.executeNextItem(ctx, node, next, v, found)
}

// execAnyKey handles execution of an ast.ConstAnyKey node. If value is an
// object, its values are passed to executeAnyItem(). If unwrap is true and
// value is an array, its values are unwrapped via
// [executeItemUnwrapTargetArray]. Otherwise it returns an error unless
// exec.ignoreStructuralErrors returns true.
func (exec *Executor) execAnyKey(
	ctx context.Context,
	node *ast.ConstNode,
	value any,
	found *valueList,
	unwrap bool,
) (resultStatus, error) {
	switch value := value.(type) {
	case map[string]any:
		return exec.executeAnyItem(
			ctx, node.Next(), maps.Values(value), found,
			1, 1, 1, false, exec.autoUnwrap(),
		)
	case []any:
		if unwrap {
			return exec.executeItemUnwrapTargetArray(ctx, node, value, found)
		}
	}

	if !exec.ignoreStructuralErrors {
		// https://github.com/postgres/postgres/blob/REL_17_2/src/backend/utils/adt/jsonpath_exec.c#L874
		return exec.returnVerboseError(fmt.Errorf(
			"%w: jsonpath wildcard member accessor can only be applied to an object",
			ErrVerbose,
		))
	}

	return statusNotFound, nil
}

// execAnyArray executes node against value. If value's type is not []any but
// exec.autoWrap() returns true, it passed it to executeNextItem to be
// unwrapped. Otherwise it returns statusFailed and an error if
// exec.ignoreStructuralErrors is false, and statusNotFound if it is true.
func (exec *Executor) execAnyArray(
	ctx context.Context,
	node *ast.ConstNode,
	value any,
	found *valueList,
) (resultStatus, error) {
	if value, ok := value.([]any); ok {
		return exec.executeAnyItem(ctx, node.Next(), value, found, 1, 1, 1, false, exec.autoUnwrap())
	}

	if exec.autoWrap() {
		return exec.executeNextItem(ctx, node, nil, value, found)
	}

	if !exec.ignoreStructuralErrors {
		// https://github.com/postgres/postgres/blob/REL_17_2/src/backend/utils/adt/jsonpath_exec.c#L851
		return exec.returnVerboseError(fmt.Errorf(
			"%w: jsonpath wildcard array accessor can only be applied to an array",
			ErrVerbose,
		))
	}

	return statusNotFound, nil
}

// execLastConst handles execution of the LAST node. Returns an error if
// execution is not currently part of an array subscript.
func (exec *Executor) execLastConst(
	ctx context.Context,
	node *ast.ConstNode,
	found *valueList,
) (resultStatus, error) {
	if exec.innermostArraySize < 0 {
		// https://github.com/postgres/postgres/blob/REL_17_2/src/backend/utils/adt/jsonpath_exec.c#L1243
		return statusFailed, fmt.Errorf(
			"%w: evaluating jsonpath LAST outside of array subscript",
			ErrExecution,
		)
	}

	next := node.Next()
	if next == nil && found == nil {
		return statusOK, nil
	}

	last := int64(exec.innermostArraySize - 1)
	return exec.executeNextItem(ctx, node, next, last, found)
}
