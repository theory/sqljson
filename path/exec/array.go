package exec

import (
	"context"
	"fmt"

	"github.com/theory/sqljson/path/ast"
)

// execSubscript executes node, which must be an a ast.BinarySubscript
// operator, against value and returns the subscript indexes.
func (exec *Executor) execSubscript(
	ctx context.Context,
	node ast.Node,
	value any,
	arraySize int,
) (int, int, error) {
	subscript, ok := node.(*ast.BinaryNode)
	if !ok || subscript.Operator() != ast.BinarySubscript {
		return 0, 0, fmt.Errorf(
			"%w: jsonpath array subscript is not a single numeric value",
			ErrExecution,
		)
	}

	indexFrom, err := exec.getArrayIndex(ctx, subscript.Left(), value)
	if err != nil {
		return 0, 0, err
	}

	indexTo := indexFrom
	if right := subscript.Right(); right != nil {
		indexTo, err = exec.getArrayIndex(ctx, right, value)
		if err != nil {
			return 0, 0, err
		}
	}

	if !exec.ignoreStructuralErrors && (indexFrom < 0 || indexFrom > indexTo || indexTo >= arraySize) {
		return 0, 0, fmt.Errorf(
			"%w: jsonpath array subscript is out of bounds",
			ErrVerbose,
		)
	}

	if indexFrom < 0 {
		indexFrom = 0
	}

	if indexTo >= arraySize {
		indexTo = arraySize - 1
	}

	return indexFrom, indexTo, nil
}

// execArrayIndex executes node against value and passes the values selected
// to the next node. value must be an array ([]any) unless exec.autoWrap
// returns true, in which case it is considered the sole value in an array.
func (exec *Executor) execArrayIndex(
	ctx context.Context,
	node *ast.ArrayIndexNode,
	value any,
	found *valueList,
) (resultStatus, error) {
	res := statusNotFound
	var resErr error

	if array, ok := value.([]any); ok || exec.autoWrap() {
		if !ok {
			array = []any{value} // auto wrap
		}

		size := len(array)
		next := node.Next()
		innermostArraySize := exec.innermostArraySize
		defer func() { exec.innermostArraySize = innermostArraySize }()
		exec.innermostArraySize = size // for LAST evaluation

		for _, subscript := range node.Subscripts() {
			indexFrom, indexTo, err := exec.execSubscript(ctx, subscript, value, size)
			if err != nil {
				return exec.returnError(err)
			}

			for index := indexFrom; index <= indexTo; index++ {
				v := array[index]
				if v == nil {
					continue
				}

				if next == nil && found == nil {
					return statusOK, nil
				}

				res, resErr = exec.executeNextItem(ctx, node, next, v, found)
				if res.failed() || (res == statusOK && found == nil) {
					break
				}
			}
		}

		return res, resErr
	}

	// In strict mode we accept only arrays.
	return exec.returnVerboseError(fmt.Errorf(
		"%w: jsonpath array accessor can only be applied to an array",
		ErrVerbose,
	))
}

// executeItemUnwrapTargetArray unwraps the current array item and executes
// node for each of its elements.
func (exec *Executor) executeItemUnwrapTargetArray(
	ctx context.Context,
	node ast.Node,
	value any,
	found *valueList,
) (resultStatus, error) {
	array, ok := value.([]any)
	if !ok {
		return statusFailed, fmt.Errorf(
			"%w: invalid json array value type: %T",
			ErrInvalid, value,
		)
	}

	return exec.executeAnyItem(ctx, node, array, found, 1, 1, 1, false, false)
}

// getArrayIndex executes an array subscript expression and converts the
// resulting numeric item to the integer type with truncation.
func (exec *Executor) getArrayIndex(
	ctx context.Context,
	node ast.Node,
	value any,
) (int, error) {
	found := newList()
	res, err := exec.executeItem(ctx, node, value, found)
	if res == statusFailed {
		return 0, err
	}

	if len(found.list) != 1 {
		return 0, fmt.Errorf(
			"%w: jsonpath array subscript is not a single numeric value",
			ErrVerbose,
		)
	}

	return getJSONInt32(found.list[0], "array subscript")
}
