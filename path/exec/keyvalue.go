package exec

import (
	"context"
	"fmt"
	"reflect"
	"slices"

	"github.com/theory/sqljson/path/ast"
	"golang.org/x/exp/maps" // Switch to maps when go 1.22 dropped
)

// kvBaseObject represents the "base object" and its "id" for .keyvalue()
// evaluation.
type kvBaseObject struct {
	addr uintptr
	id   int
}

// addrOf returns the pointer address of obj when obj is a valid JSON
// container: one of map[string]any, []any, or Vars. Otherwise it returns 0.
// Used for .keyvalue() ID generation.
func addrOf(obj any) uintptr {
	switch obj := obj.(type) {
	case []any, map[string]any, Vars:
		return reflect.ValueOf(obj).Pointer()
	default:
		return 0
	}
}

// OffsetOf returns the offset of obj from bo. This is the difference between
// their pointer addresses.
func (bo kvBaseObject) OffsetOf(obj any) int64 {
	addr := addrOf(obj)
	if addr > bo.addr {
		return int64(addr - bo.addr)
	}
	return int64(bo.addr - addr)
}

// setTempBaseObject sets obj as exec.baseObject and returns a function that
// will reset it to the previous value.
func (exec *Executor) setTempBaseObject(obj any, id int) func() {
	bo := exec.baseObject
	exec.baseObject.addr = addrOf(obj)
	exec.baseObject.id = id
	return func() { exec.baseObject = bo }
}

// executeKeyValueMethod implements the .keyvalue() method.
//
// .keyvalue() method returns a sequence of object's key-value pairs in the
// following format: '{ "key": key, "value": value, "id": id }'.
//
// "id" field is an object identifier which is constructed from the two parts:
// base object id and its binary offset from the base object:
// id = exec.baseObject.id * 10000000000 + exec.baseObject.OffsetOf(object).
//
// 10000000000 (10^10) -- is the first round decimal number greater than 2^32
// (maximal offset in jsonb). The decimal multiplier is used here to improve
// the readability of identifiers.
//
// exec.baseObject is usually the root object of the path (context item '$')
// or path variable '$var' (literals can't produce objects for now). Objects
// generated by keyvalue() itself, they become base object for the subsequent
// .keyvalue().
//
//   - ID of '$' is 0.
//   - ID of '$var' is 10000000000.
//   - IDs for objects generated by .keyvalue() are assigned using global counter
//     exec.lastGeneratedObjectId: 20000000000, 30000000000, 40000000000, etc.
func (exec *Executor) executeKeyValueMethod(
	ctx context.Context,
	node ast.Node,
	value any,
	found *valueList,
	unwrap bool,
) (resultStatus, error) {
	var obj map[string]any
	switch val := value.(type) {
	case []any:
		if unwrap {
			return exec.executeItemUnwrapTargetArray(ctx, node, value, found)
		}
		return exec.returnVerboseError(fmt.Errorf(
			`%w: jsonpath item method .keyvalue() can only be applied to an object`,
			ErrVerbose,
		))
	case map[string]any:
		obj = val
	default:
		return exec.returnVerboseError(fmt.Errorf(
			`%w: jsonpath item method .keyvalue() can only be applied to an object`,
			ErrVerbose,
		))
	}

	if len(obj) == 0 {
		// no key-value pairs
		return statusNotFound, nil
	}

	next := node.Next()
	if next == nil && found == nil {
		return statusOK, nil
	}

	id := exec.baseObject.OffsetOf(obj)
	const tenTen = 10000000000 // 10^10
	id += int64(exec.baseObject.id) * tenTen

	// Process the keys in a deterministic order for consistent ID assignment.
	keys := maps.Keys(obj)
	slices.Sort(keys)

	var res resultStatus
	for _, k := range keys {
		obj := map[string]any{"key": k, "value": obj[k], "id": id}
		exec.lastGeneratedObjectID++
		defer exec.setTempBaseObject(obj, exec.lastGeneratedObjectID)()

		var err error
		res, err = exec.executeNextItem(ctx, node, next, obj, found)
		if res == statusFailed {
			return res, err
		}

		if res == statusOK && found == nil {
			break
		}
	}
	return res, nil
}
