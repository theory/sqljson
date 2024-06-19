package exec

import (
	"context"
	"fmt"

	"github.com/theory/sqljson/path/ast"
)

// execLiteral handles the execution of a literal string, integer, or float
// value.
func (exec *Executor) execLiteral(
	ctx context.Context,
	node ast.Node,
	value any,
	found *valueList,
) (resultStatus, error) {
	next := node.Next()
	if next == nil && found == nil {
		return statusOK, nil
	}
	return exec.executeNextItem(ctx, node, next, value, found)
}

// execVariable handles the execution of a node, returning an error if the
// variable is not found.
func (exec *Executor) execVariable(
	ctx context.Context,
	node *ast.VariableNode,
	found *valueList,
) (resultStatus, error) {
	if val, ok := exec.vars[node.Text()]; ok {
		// keyvalue ID 1 reserved for variables.
		defer exec.setTempBaseObject(exec.vars, 1)()
		return exec.executeNextItem(ctx, node, node.Next(), val, found)
	}

	// Return error for missing variable.
	return statusFailed, fmt.Errorf(
		"%w: could not find jsonpath variable %q",
		ErrExecution, node.Text(),
	)
}

// execKeyNode executes node against value, which is expected to be of type
// map[string]any. If its type is []any and unwrap is true, it passes it to
// [executeAnyItem]. Otherwise, it returns statusFailed and an error if
// exec.ignoreStructuralErrors is false and statusNotFound and no error if
// it's true.
func (exec *Executor) execKeyNode(
	ctx context.Context,
	node *ast.KeyNode,
	value any,
	found *valueList,
	unwrap bool,
) (resultStatus, error) {
	key := node.Text()
	switch value := value.(type) {
	case map[string]any:
		val, ok := value[key]
		if ok {
			return exec.executeNextItem(ctx, node, nil, val, found)
		}

		if !exec.ignoreStructuralErrors {
			if !exec.verbose {
				return statusFailed, nil
			}

			return statusFailed, fmt.Errorf(
				`%w: JSON object does not contain key "%s"`,
				ErrVerbose, key,
			)
		}
	case []any:
		if unwrap {
			return exec.executeAnyItem(ctx, node, value, found, 1, 1, 1, false, false)
		}
	}
	if !exec.ignoreStructuralErrors {
		return exec.returnVerboseError(fmt.Errorf(
			"%w: jsonpath member accessor can only be applied to an object",
			ErrVerbose,
		))
	}

	return statusNotFound, nil
}
