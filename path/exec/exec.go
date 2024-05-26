// Package exec provides the routines for SQL/JSON path execution.
package exec

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/theory/sqljson/path/ast"
	"github.com/theory/sqljson/path/types"
	"golang.org/x/exp/maps"
)

// Things to improve or document as different:
//   - .datetime(template)
//   - Replace statusFailed with silence-able error
//   - Some time_tz comparisons still not quite right
//   - Allow single-digit tz offsets, e.g., `+1` instead of `+01`
//   - Allow space between seconds and offset
//   - Years > 9999
//   - .keyvalue() offsets for arrays?
//   - Less accuracy than Postgres NUMERICs: Switch to
//     github.com/shopspring/decimal?
//   - Go regexp package varies from Postgres regex
//   - Implement interfaces to be compatible with the SQL-standard
//     json_exists(), json_query(), and json_value() functions added in Postgres 17.
//
// vars represents JSON path variables and their values.
type vars map[string]any

var (
	// ErrExecution errors are returned in strict mode.
	ErrExecution = errors.New("exec")
	// ErrInvalid errors are always returned.
	ErrInvalid = fmt.Errorf("%w", ErrExecution)
	// ErrNull errors are returned when Postgres would return NULL from Match
	// and Exists.
	ErrNull = fmt.Errorf("%w: NULL", ErrExecution)
)

// predOutcome represents the result of jsonpath predicate evaluation.
type predOutcome uint8

const (
	predFalse predOutcome = iota
	predTrue
	predUnknown
)

// resultStatus represents the result of jsonpath expression evaluation.
type resultStatus uint8

const (
	statusOK resultStatus = iota
	statusNotFound
	statusFailed
)

func (s resultStatus) failed() bool {
	return s == statusFailed
}

// List of jsonb values with shortcut for single-value list.
type valueList struct {
	list []any
}

func newList() *valueList {
	return &valueList{list: make([]any, 0, 1)}
}

func (vl *valueList) isEmpty() bool {
	return len(vl.list) == 0
}

func (vl *valueList) append(val any) {
	vl.list = append(vl.list, val)
}

type kvBaseObject struct {
	addr uintptr
	id   int
}

func addrOf(obj any) uintptr {
	switch obj := obj.(type) {
	case []any, map[string]any, vars:
		return reflect.ValueOf(obj).Pointer()
	default:
		return 0
	}
}

func (bo kvBaseObject) OffsetOf(obj any) int64 {
	addr := addrOf(obj)
	if addr > bo.addr {
		return int64(addr - bo.addr)
	}
	return int64(bo.addr - addr)
}

// Executor represents the context for jsonpath execution.
type Executor struct {
	vars                  vars         // variables to substitute into jsonpath
	root                  any          // for $ evaluation
	current               any          // for @ evaluation
	baseObject            kvBaseObject // "base object" for .keyvalue() evaluation
	lastGeneratedObjectID int          // "id" counter for .keyvalue() evaluation
	innermostArraySize    int          // for LAST array index evaluation
	path                  *ast.AST

	// with "true" structural errors such as absence of required json item or
	// unexpected json item type are ignored
	ignoreStructuralErrors bool

	// with "false" all suppressible errors are suppressed
	throwErrors bool
	useTZ       bool
}

// Option specifies an execution option.
type Option func(*Executor)

// WithVars passes specifies variables to use during execution.
func WithVars(vars vars) Option { return func(e *Executor) { e.vars = vars } }

// WithTZ allows casting between TZ and non-TZ time and timestamp types.
func WithTZ() Option { return func(e *Executor) { e.useTZ = true } }

// WithSilent suppresses the following errors: missing object field or array
// element, unexpected JSON item type, datetime and numeric errors. This
// behavior emulates the behavior of the PostgreSQL @? and @@ operators, and
// might be helpful when searching JSON document collections of varying
// structure.
func WithSilent() Option { return func(e *Executor) { e.throwErrors = false } }

func newExec(path *ast.AST, opt ...Option) *Executor {
	e := &Executor{
		path:                   path,
		innermostArraySize:     -1,
		ignoreStructuralErrors: path.IsLax(),
		lastGeneratedObjectID:  1, // Reserved for IDs from vars
		throwErrors:            true,
	}

	for _, o := range opt {
		o(e)
	}
	return e
}

// Query returns all JSON items returned by the JSON path for the specified
// JSON value. For SQL-standard JSON path expressions it returns the JSON
// values selected from target. For predicate check expressions it returns the
// result of the predicate check: true, false, or null (false + ErrNull). The
// optional [WithVars] and [WithSilent] Options act the same as for [Exists].
func Query(ctx context.Context, path *ast.AST, value any, opt ...Option) ([]any, error) {
	exec := newExec(path, opt...)
	vals, err := exec.execute(ctx, value)
	if err != nil {
		return nil, err
	}
	return vals.list, nil
}

// First returns the first JSON item returned by the JSON path for the
// specified JSON value, or nil if there are no results. The parameters are
// the same as for [Query].
func First(ctx context.Context, path *ast.AST, value any, opt ...Option) (any, error) {
	exec := newExec(path, opt...)
	vals, err := exec.execute(ctx, value)
	if err != nil {
		return nil, err
	}
	if vals.isEmpty() {
		//nolint:nilnil
		return nil, nil
	}
	return vals.list[0], nil
}

// Exists checks whether the JSON path returns any item for the specified JSON
// value. (This is useful only with SQL-standard JSON path expressions, not
// predicate check expressions, since those always return a value.) If the
// [WithVars] Option is specified its fields provide named values to be
// substituted into the jsonpath expression. If the [WithSilent] Option is
// specified, the function suppresses some errors. If the [WithTZ] Option is
// specified, it allows comparisons of date/time values that require
// timezone-aware conversions. The example below requires interpretation of
// the date-only value 2015-08-02 as a timestamp with time zone, so the result
// depends on the current TimeZone setting:
//
//	Exists(
//		[]any{"2015-08-01 12:00:00-05"},
//		`$[*] ? (@.datetime() < "2015-08-02".datetime())`,
//		WithTZ(),
//	) → true
func Exists(ctx context.Context, path *ast.AST, value any, opt ...Option) (bool, error) {
	exec := newExec(path, opt...)
	// if exec.throwErrors && exec.path.IsPredicate() {
	// 	return statusFailed, fmt.Errorf(
	// 		"%w: Exists expects a SQL standard path expression",
	// 		ErrExecution,
	// 	)
	// }

	res, err := exec.exists(ctx, value)
	if err != nil {
		return false, err
	}
	if res.failed() {
		return false, ErrNull
	}
	return res == statusOK, nil
}

// Match returns the result of a JSON path predicate check for the specified
// JSON value. (This is useful only with predicate check expressions, not
// SQL-standard JSON path expressions, since it will either fail or return
// NULL if the path result is not a single boolean value.) The optional
// [WithVars] and [WithSilent] Options act the same as for [Exists].
func Match(ctx context.Context, path *ast.AST, value any, opt ...Option) (bool, error) {
	exec := newExec(path, opt...)
	// if exec.throwErrors && !exec.path.IsPredicate() {
	// 	return false, fmt.Errorf(
	// 		"%w: Match expects a predicate path expression",
	// 		ErrExecution,
	// 	)
	// }

	vals, err := exec.execute(ctx, value)
	if err != nil {
		return false, err
	}

	if len(vals.list) == 1 {
		switch val := vals.list[0].(type) {
		case nil:
			return false, ErrNull
		case bool:
			return val, nil
		}
	}

	if exec.throwErrors {
		return false, fmt.Errorf(
			"%w: single boolean result is expected",
			ErrExecution,
		)
	}

	return false, ErrNull
}

func (exec *Executor) strictAbsenceOfErrors() bool { return exec.path.IsStrict() }
func (exec *Executor) autoUnwrap() bool            { return exec.path.IsLax() }
func (exec *Executor) autoWrap() bool              { return exec.path.IsLax() }

func (exec *Executor) execute(ctx context.Context, jVal any) (*valueList, error) {
	exec.root = jVal
	exec.current = jVal
	vals := newList()
	_, err := exec.exec(ctx, vals, exec.path.Root(), jVal)
	return vals, err
}

// exists returns true if the path passed to New() returns at least one item
// for json. This function provides the equivalent of the Postgres @? operator
// when exec.throwErrors is false.
func (exec *Executor) exists(ctx context.Context, json any) (resultStatus, error) {
	exec.root = json
	exec.current = json
	res, err := exec.exec(ctx, nil, exec.path.Root(), json)
	if err != nil {
		return res, err
	}
	return res, nil
}

func (exec *Executor) exec(ctx context.Context, vals *valueList, node ast.Node, jVal any) (resultStatus, error) {
	if exec.strictAbsenceOfErrors() && vals == nil {
		// In strict mode we must get a complete list of values to check that
		// there are no errors at all.
		vals := newList()
		res, err := exec.executeItem(ctx, node, jVal, vals)
		if res.failed() {
			return res, err
		}

		if vals.isEmpty() {
			return statusNotFound, nil
		}
		return statusOK, nil
	}

	return exec.executeItem(ctx, node, jVal, vals)
}

// Execute jsonpath with automatic unwrapping of current item in lax mode.
func (exec *Executor) executeItem(
	ctx context.Context,
	node ast.Node,
	jVal any,
	found *valueList,
) (resultStatus, error) {
	return exec.executeItemOptUnwrapTarget(ctx, node, jVal, found, exec.autoUnwrap())
}

// Same as executeItem(), but when "unwrap == true" automatically unwraps
// each array item from the resulting sequence in lax mode.
func (exec *Executor) executeItemOptUnwrapResult(
	ctx context.Context,
	node ast.Node,
	jVal any,
	unwrap bool,
	found *valueList,
) (resultStatus, error) {
	if unwrap && exec.autoUnwrap() {
		seq := newList()
		res, err := exec.executeItem(ctx, node, jVal, seq)
		if res.failed() {
			return res, err
		}

		for _, item := range seq.list {
			switch item.(type) {
			case []any:
				_, _ = exec.executeItemUnwrapTargetArray(ctx, nil, item, found)
			default:
				found.append(item)
			}
		}
		return statusOK, nil
	}
	return exec.executeItem(ctx, node, jVal, found)
}

// executeItemOptUnwrapResultNoThrow is the same as
// executeItemOptUnwrapResult, but with error suppression.
func (exec *Executor) executeItemOptUnwrapResultNoThrow(
	ctx context.Context,
	node ast.Node,
	jVal any,
	unwrap bool,
	found *valueList,
) (resultStatus, error) {
	throwErrors := exec.throwErrors
	exec.throwErrors = false
	defer func(e *Executor, te bool) { e.throwErrors = te }(exec, throwErrors)
	return exec.executeItemOptUnwrapResult(ctx, node, jVal, unwrap, found)
}

func executeIntegerMath(lhs, rhs int64, op ast.BinaryOperator) (int64, error) {
	//nolint:exhaustive
	switch op {
	case ast.BinaryAdd:
		return lhs + rhs, nil
	case ast.BinarySub:
		return lhs - rhs, nil
	case ast.BinaryMul:
		return lhs * rhs, nil
	case ast.BinaryDiv:
		if rhs == 0 {
			return 0, fmt.Errorf("%w: division by zero", ErrExecution)
		}
		return lhs / rhs, nil
	case ast.BinaryMod:
		if rhs == 0 {
			return 0, fmt.Errorf("%w: division by zero", ErrExecution)
		}
		return lhs % rhs, nil
	default:
		return 0, fmt.Errorf("%w: %v is not a binary math operator", ErrExecution, op)
	}
}

func executeFloatMath(lhs, rhs float64, op ast.BinaryOperator) (float64, error) {
	//nolint:exhaustive
	switch op {
	case ast.BinaryAdd:
		return lhs + rhs, nil
	case ast.BinarySub:
		return lhs - rhs, nil
	case ast.BinaryMul:
		return lhs * rhs, nil
	case ast.BinaryDiv:
		if rhs == 0 {
			return 0, fmt.Errorf("%w: division by zero", ErrExecution)
		}
		return lhs / rhs, nil
	case ast.BinaryMod:
		if rhs == 0 {
			return 0, fmt.Errorf("%w: division by zero", ErrExecution)
		}
		return math.Mod(lhs, rhs), nil
	default:
		return 0, fmt.Errorf("%w: %v is not a binary math operator", ErrExecution, op)
	}
}

func mathOperandErr(op ast.BinaryOperator, pos string) error {
	return fmt.Errorf(
		"%w: %v operand of jsonpath operator %v is not a single numeric value",
		ErrExecution, pos, op,
	)
}

// execUnaryMathExpr executes a unary arithmetic expression for each numeric
// item in its operand's sequence. Array operand is automatically unwrapped
// in lax mode.
//
//nolint:funlen,gocognit
func (exec *Executor) execUnaryMathExpr(
	ctx context.Context,
	node *ast.UnaryNode,
	jVal any,
	intCallback intCallback,
	floatCallback floatCallback,
	found *valueList,
) (resultStatus, error) {
	seq := newList()
	res, err := exec.executeItemOptUnwrapResult(ctx, node.Operand(), jVal, true, seq)
	if res == statusFailed {
		return res, err
	}

	res = statusNotFound
	next := node.Next()
	var val any

	for _, v := range seq.list {
		val = v
		switch v := v.(type) {
		case int64:
			if found == nil && next == nil {
				return statusOK, nil
			}
			val = intCallback(v)
		case float64:
			if found == nil && next == nil {
				return statusOK, nil
			}
			val = floatCallback(v)
		case json.Number:
			if found == nil && next == nil {
				return statusOK, nil
			}
			if integer, err := v.Int64(); err == nil {
				val = intCallback(integer)
			} else if float, err := v.Float64(); err == nil {
				val = floatCallback(float)
			} else {
				return exec.returnError(fmt.Errorf(
					"%w: operand of unary jsonpath operator %v is not a numeric value",
					ErrExecution, node.Operator(),
				))
			}
		default:
			if found != nil || next != nil {
				return exec.returnError(fmt.Errorf(
					"%w: operand of unary jsonpath operator %v is not a numeric value",
					ErrExecution, node.Operator(),
				))
			}
		}

		nextRes, err := exec.executeNextItem(ctx, node, next, val, found)
		if nextRes.failed() {
			return nextRes, err
		}
		if nextRes == statusOK {
			if found == nil {
				return statusOK, nil
			}
			res = nextRes
		}
	}

	return res, nil
}

func (exec *Executor) execBinaryMathExpr(
	ctx context.Context,
	node *ast.BinaryNode,
	jVal any,
	op ast.BinaryOperator,
	found *valueList,
) (resultStatus, error) {
	// Get the left node.
	// XXX: The standard says only operands of multiplicative expressions are
	// unwrapped. We extend it to other binary arithmetic expressions too.
	lSeq := newList()
	res, err := exec.executeItemOptUnwrapResult(ctx, node.Left(), jVal, true, lSeq)
	if res == statusFailed {
		return res, err
	}

	if len(lSeq.list) != 1 {
		return exec.returnError(mathOperandErr(op, "left"))
	}

	rSeq := newList()
	res, err = exec.executeItemOptUnwrapResult(ctx, node.Right(), jVal, true, rSeq)
	if res == statusFailed {
		return res, err
	}

	if len(rSeq.list) != 1 {
		return exec.returnError(mathOperandErr(op, "right"))
	}

	val, err := execMathOp(lSeq.list[0], rSeq.list[0], op)
	if err != nil {
		return exec.returnError(err)
	}

	next := node.Next()
	if next == nil && found == nil {
		return statusOK, nil
	}

	return exec.executeNextItem(ctx, node, next, val, found)
}

func execMathOp(left, right any, op ast.BinaryOperator) (any, error) {
	switch left := left.(type) {
	case int64:
		switch right := right.(type) {
		case int64:
			return executeIntegerMath(left, right, op)
		case float64:
			return executeFloatMath(float64(left), right, op)
		case json.Number:
			if right, err := right.Int64(); err == nil {
				return executeIntegerMath(left, right, op)
			}
			if right, err := right.Float64(); err == nil {
				return executeFloatMath(float64(left), right, op)
			} else {
				return nil, mathOperandErr(op, "right")
			}
		default:
			return nil, mathOperandErr(op, "right")
		}
	case float64:
		switch right := right.(type) {
		case float64:
			return executeFloatMath(left, right, op)
		case int64:
			return executeFloatMath(left, float64(right), op)
		case json.Number:
			if right, err := right.Float64(); err == nil {
				return executeFloatMath(left, right, op)
			} else {
				return nil, mathOperandErr(op, "right")
			}
		default:
			return nil, mathOperandErr(op, "right")
		}
	case json.Number:
		if left, err := left.Int64(); err == nil {
			return execMathOp(left, right, op)
		}
		if left, err := left.Float64(); err == nil {
			return execMathOp(left, right, op)
		}
	}

	return nil, mathOperandErr(op, "left")
}

func (exec *Executor) setTempBaseObject(obj any, id int) func() {
	bo := exec.baseObject
	exec.baseObject.addr = addrOf(obj)
	exec.baseObject.id = id
	return func() { exec.baseObject = bo }
}

func (exec *Executor) returnError(err error) (resultStatus, error) {
	if exec.throwErrors {
		return statusFailed, err
	}
	return statusFailed, nil
}

// executeItemOptUnwrapTarget is the main executor function: walks on jsonpath structure, finds
// relevant parts of jsonb and evaluates expressions over them.
// When 'unwrap' is true current SQL/JSON item is unwrapped if it is an array.
//
//nolint:funlen,gocognit,gocyclo,maintidx
func (exec *Executor) executeItemOptUnwrapTarget(
	ctx context.Context,
	node ast.Node,
	jVal any,
	found *valueList,
	unwrap bool,
) (resultStatus, error) {
	// Check for interrupts.
	select {
	case <-ctx.Done():
		return statusNotFound, nil
	default:
	}

	var elem ast.Node
	res := statusNotFound
	var err error
	// defer func() { exec.baseObject.counter += 1 }()

	switch node := node.(type) {
	case *ast.ConstNode:
		switch node.Const() {
		case ast.ConstNull, ast.ConstTrue, ast.ConstFalse:
			elem = node.Next()
			if elem == nil && found == nil {
				return statusOK, nil
			}

			var v any
			if node.Const() == ast.ConstNull {
				v = nil
			} else {
				v = node.Const() == ast.ConstTrue
			}

			return exec.executeNextItem(ctx, node, elem, v, found)
		case ast.ConstRoot:
			defer exec.setTempBaseObject(exec.root, 0)()
			return exec.executeNextItem(ctx, node, nil, exec.root, found)
		case ast.ConstCurrent:
			return exec.executeNextItem(ctx, node, nil, exec.current, found)
		case ast.ConstAnyKey:
			switch jVal := jVal.(type) {
			case map[string]any:
				return exec.executeAnyItem(
					ctx, node.Next(), maps.Values(jVal), found,
					1, 1, 1, false, exec.autoUnwrap(),
				)
			// case []any:
			// 	return exec.executeItemUnwrapTargetArray(ctx, node, jVal, found)
			default:
				if !exec.ignoreStructuralErrors {
					return exec.returnError(fmt.Errorf(
						"%w: jsonpath wildcard member accessor can only be applied to an object",
						ErrExecution,
					))
				}
			}
		case ast.ConstAnyArray:
			if jVal, ok := jVal.([]any); ok {
				return exec.executeAnyItem(ctx, node.Next(), jVal, found, 1, 1, 1, false, exec.autoUnwrap())
			}

			if exec.autoWrap() {
				return exec.executeNextItem(ctx, node, nil, jVal, found)
			}

			if !exec.ignoreStructuralErrors {
				return exec.returnError(fmt.Errorf(
					"%w: jsonpath wildcard array accessor can only be applied to an array",
					ErrExecution,
				))
			}

		case ast.ConstLast:
			if exec.innermostArraySize < 0 {
				return statusFailed, fmt.Errorf(
					"%w: evaluating jsonpath LAST outside of array subscript",
					ErrInvalid,
				)
			}

			next := node.Next()
			if next == nil && found == nil {
				return statusOK, nil
			}

			last := int64(exec.innermostArraySize - 1)
			return exec.executeNextItem(ctx, node, next, last, found)
		}
	case *ast.StringNode:
		elem = node.Next()
		if elem == nil && found == nil {
			return statusOK, nil
		}
		return exec.executeNextItem(ctx, node, elem, node.Text(), found)
	case *ast.IntegerNode:
		elem = node.Next()
		if elem == nil && found == nil {
			return statusOK, nil
		}
		return exec.executeNextItem(ctx, node, elem, node.Int(), found)
	case *ast.NumericNode:
		elem = node.Next()
		if elem == nil && found == nil {
			return statusOK, nil
		}
		return exec.executeNextItem(ctx, node, elem, node.Float(), found)
	case *ast.VariableNode:
		elem = node.Next()
		// getJsonPathVariable
		if val, ok := exec.vars[node.Text()]; ok {
			// keyvalue ID 1 reserved for variables.
			defer exec.setTempBaseObject(exec.vars, 1)()
			return exec.executeNextItem(ctx, node, elem, val, found)
		} else {
			// Return error for missing variable.
			return statusFailed, fmt.Errorf(
				"%w: could not find jsonpath variable %q",
				ErrInvalid, node.Text(),
			)
		}
	case *ast.KeyNode:
		key := node.Text()
		switch jVal := jVal.(type) {
		case map[string]any:
			val, ok := jVal[key]
			if ok {
				return exec.executeNextItem(ctx, node, nil, val, found)
			}

			if !exec.ignoreStructuralErrors {
				if !exec.throwErrors {
					return statusFailed, nil
				}

				return statusFailed, fmt.Errorf(
					`%w: JSON object does not contain key "%s"`,
					ErrExecution, key,
				)
			}
		case []any:
			if unwrap {
				return exec.executeAnyItem(ctx, node, jVal, found, 1, 1, 1, false, false)
			}
		}
		if !exec.ignoreStructuralErrors {
			return exec.returnError(fmt.Errorf(
				"%w: jsonpath member accessor can only be applied to an object",
				ErrExecution,
			))
		}

	case *ast.BinaryNode:
		switch node.Operator() {
		case ast.BinaryAnd, ast.BinaryOr, ast.BinaryEqual, ast.BinaryNotEqual,
			ast.BinaryLess, ast.BinaryLessOrEqual, ast.BinaryGreater,
			ast.BinaryGreaterOrEqual, ast.BinaryStartsWith:
			// Binary boolean types.
			res, err := exec.executeBoolItem(ctx, node, jVal, true)
			return exec.appendBoolResult(ctx, node, found, res, err)
		case ast.BinaryAdd, ast.BinarySub, ast.BinaryMul, ast.BinaryDiv, ast.BinaryMod:
			return exec.execBinaryMathExpr(ctx, node, jVal, node.Operator(), found)
		case ast.BinaryDecimal:
			return exec.executeNumberMethod(ctx, node, jVal, found, unwrap)
		case ast.BinarySubscript:
			// This should not happen.
			return statusFailed, fmt.Errorf(
				"%w: evaluating jsonpath subscript expression outside of array subscript",
				ErrInvalid,
			)
		}
	case *ast.UnaryNode:
		switch node.Operator() {
		case ast.UnaryNot, ast.UnaryIsUnknown, ast.UnaryExists:
			// Binary boolean types.
			res, err := exec.executeBoolItem(ctx, node, jVal, true)
			return exec.appendBoolResult(ctx, node, found, res, err)
		case ast.UnaryFilter:
			if unwrap {
				if _, ok := jVal.([]any); ok {
					return exec.executeItemUnwrapTargetArray(ctx, node, jVal, found)
				}
			}

			st, err := exec.executeNestedBoolItem(ctx, node.Operand(), jVal)
			if st != predTrue {
				return statusNotFound, err
			}
			return exec.executeNextItem(ctx, node, nil, jVal, found)
		case ast.UnaryPlus:
			return exec.execUnaryMathExpr(ctx, node, jVal, intSelf, floatSelf, found)
		case ast.UnaryMinus:
			return exec.execUnaryMathExpr(ctx, node, jVal, intUMinus, floatUMinus, found)
		case ast.UnaryDateTime, ast.UnaryDate, ast.UnaryTime, ast.UnaryTimeTZ,
			ast.UnaryTimestamp, ast.UnaryTimestampTZ:
			if unwrap {
				if array, ok := jVal.([]any); ok {
					return exec.executeAnyItem(ctx, node, array, found, 1, 1, 1, false, false)
				}
			}
			return exec.executeDateTimeMethod(ctx, node, jVal, found)
		}
	case *ast.RegexNode:
		// Binary boolean type.
		res, err := exec.executeBoolItem(ctx, node, jVal, true)
		return exec.appendBoolResult(ctx, node, found, res, err)

	case *ast.MethodNode:
		switch name := node.Name(); name {
		case ast.MethodNumber:
			return exec.executeNumberMethod(ctx, node, jVal, found, unwrap)
		case ast.MethodAbs:
			return exec.executeNumericItemMethod(
				ctx, node, jVal, unwrap,
				intAbs, math.Abs, found,
			)
		case ast.MethodFloor:
			return exec.executeNumericItemMethod(
				ctx, node, jVal, unwrap,
				intSelf, math.Floor, found,
			)
		case ast.MethodCeiling:
			return exec.executeNumericItemMethod(
				ctx, node, jVal, unwrap,
				intSelf, math.Ceil, found,
			)

		case ast.MethodType:
			var typeName string
			switch jVal.(type) {
			case map[string]any:
				typeName = "object"
			case []any:
				typeName = "array"
			case string:
				typeName = "string"
			case int64, float64, json.Number:
				typeName = "number"
			case bool:
				typeName = "boolean"
			case *types.Date:
				typeName = "date"
			case *types.Time:
				typeName = "time without time zone"
			case *types.TimeTZ:
				typeName = "time with time zone"
			case *types.Timestamp:
				typeName = "timestamp without time zone"
			case *types.TimestampTZ:
				typeName = "timestamp with time zone"
			case nil:
				typeName = "null"
			}

			return exec.executeNextItem(ctx, node, nil, typeName, found)
		case ast.MethodSize:
			size := 1
			switch jVal := jVal.(type) {
			case []any:
				size = len(jVal)
			default:
				if !exec.autoWrap() {
					if !exec.ignoreStructuralErrors {
						return exec.returnError(fmt.Errorf(
							"%w: jsonpath item method %v can only be applied to an array",
							ErrExecution, name,
						))
					}
				}
			}
			return exec.executeNextItem(ctx, node, nil, int64(size), found)
		case ast.MethodDouble:
			var double float64
			switch val := jVal.(type) {
			case []any:
				if unwrap {
					return exec.executeItemUnwrapTargetArray(ctx, node, jVal, found)
				}
				return exec.returnError(fmt.Errorf(
					"%w: jsonpath item method %v can only be applied to a string or numeric value",
					ErrExecution, name,
				))
			case int64:
				double = float64(val)
			case float64:
				double = val
			case json.Number:
				var err error
				double, err = val.Float64()
				if err != nil {
					return statusFailed, fmt.Errorf(
						`%w: argument "%v" of jsonpath item method %v is invalid for type double precision`,
						ErrExecution, val, name,
					)
				}
			case string:
				var err error
				double, err = strconv.ParseFloat(val, 64)
				if err != nil {
					return statusFailed, fmt.Errorf(
						`%w: argument "%v" of jsonpath item method %v is invalid for type double precision`,
						ErrExecution, val, name,
					)
				}
			default:
				return exec.returnError(fmt.Errorf(
					"%w: jsonpath item method %v can only be applied to a string or numeric value",
					ErrExecution, name,
				))
			}

			if math.IsInf(double, 0) || math.IsNaN(double) {
				return exec.returnError(fmt.Errorf(
					"%w: NaN or Infinity is not allowed for jsonpath item method %v",
					ErrExecution, name,
				))
			}

			return exec.executeNextItem(ctx, node, nil, double, found)

		case ast.MethodInteger:
			var integer int64
			switch val := jVal.(type) {
			case []any:
				if unwrap {
					return exec.executeItemUnwrapTargetArray(ctx, node, jVal, found)
				}
				return exec.returnError(fmt.Errorf(
					"%w: jsonpath item method %v can only be applied to a string or numeric value",
					ErrExecution, name,
				))
			case int64:
				integer = val
			case float64:
				integer = int64(math.Round(val))
			case json.Number:
				var err error
				integer, err = val.Int64()
				if err != nil || integer > math.MaxInt32 || integer < math.MinInt32 {
					return exec.returnError(fmt.Errorf(
						`%w: argument "%v" of jsonpath item method %v is invalid for type integer`,
						ErrExecution, jVal, name,
					))
				}
			case string:
				var err error
				integer, err = strconv.ParseInt(val, 10, 32)
				if err != nil {
					return exec.returnError(fmt.Errorf(
						`%w: argument "%v" of jsonpath item method %v is invalid for type integer`,
						ErrExecution, jVal, name,
					))
				}
			default:
				return exec.returnError(fmt.Errorf(
					"%w: jsonpath item method %v can only be applied to a string or numeric value",
					ErrExecution, name,
				))
			}

			if integer > math.MaxInt32 || integer < math.MinInt32 {
				return exec.returnError(fmt.Errorf(
					`%w: argument "%v" of jsonpath item method %v is invalid for type integer`,
					ErrExecution, jVal, name,
				))
			}

			return exec.executeNextItem(ctx, node, nil, integer, found)

		case ast.MethodBigInt:
			var bigInt int64
			switch val := jVal.(type) {
			case []any:
				if unwrap {
					return exec.executeItemUnwrapTargetArray(ctx, node, jVal, found)
				}
				return exec.returnError(fmt.Errorf(
					"%w: jsonpath item method %v can only be applied to a string or numeric value",
					ErrExecution, name,
				))
			case int64:
				bigInt = val
			case float64:
				if val > math.MaxInt64 || val < math.MinInt64 {
					return exec.returnError(fmt.Errorf(
						`%w: argument "%v" of jsonpath item method %v is invalid for type bigint`,
						ErrExecution, val, name,
					))
				}
				bigInt = int64(math.Round(val))
			case json.Number:
				var err error
				bigInt, err = val.Int64()
				if err != nil {
					return exec.returnError(fmt.Errorf(
						`%w: argument "%v" of jsonpath item method %v is invalid for type bigint`,
						ErrExecution, val, name,
					))
				}
			case string:
				var err error
				bigInt, err = strconv.ParseInt(val, 10, 64)
				if err != nil {
					return exec.returnError(fmt.Errorf(
						`%w: argument "%v" of jsonpath item method %v is invalid for type bigint`,
						ErrExecution, val, name,
					))
				}
			default:
				return exec.returnError(fmt.Errorf(
					"%w: jsonpath item method %v can only be applied to a string or numeric value",
					ErrExecution, name,
				))
			}

			return exec.executeNextItem(ctx, node, nil, bigInt, found)

		case ast.MethodString:
			var str string
			switch val := jVal.(type) {
			case string:
				str = val
			case fmt.Stringer:
				// Covers json.Number and date/time types (ISO-8601 only, no date style)
				str = val.String()
			case int64:
				str = strconv.FormatInt(val, 10)
			case float64:
				str = strconv.FormatFloat(val, 'f', -1, 64)
			case bool:
				if val {
					str = "true"
				} else {
					str = "false"
				}
			default:
				return exec.returnError(fmt.Errorf(
					`%w: jsonpath item method %v can only be applied to a bool, string, numeric, or datetime value`,
					ErrExecution, name,
				))
			}
			return exec.executeNextItem(ctx, node, nil, str, found)

		case ast.MethodBoolean:
			var boolean bool
			switch val := jVal.(type) {
			case []any:
				if unwrap {
					return exec.executeItemUnwrapTargetArray(ctx, node, jVal, found)
				}
				return exec.returnError(fmt.Errorf(
					"%w: jsonpath item method %v can only be applied to a bool, string, or numeric value",
					ErrExecution, name,
				))
			case bool:
				boolean = val
			case int64:
				boolean = val != 0
			case float64:
				if val != math.Trunc(val) {
					return exec.returnError(fmt.Errorf(
						`%w: argument "%v" of jsonpath item method %v is invalid for type boolean`,
						ErrExecution, val, name,
					))
				}
				boolean = val != 0

			case json.Number:
				num, err := val.Int64()
				if err != nil {
					return exec.returnError(fmt.Errorf(
						`%w: argument "%v" of jsonpath item method %v is invalid for type boolean`,
						ErrExecution, val, name,
					))
				}
				boolean = num != 0
			case string:
				size := len(val)
				if size == 0 {
					return exec.returnError(fmt.Errorf(
						`%w: argument "%v" of jsonpath item method %v is invalid for type boolean`,
						ErrExecution, val, name,
					))
				}

				var matched bool
				switch val[0] {
				case 't', 'T':
					if size == 1 || strings.EqualFold(val, "true") {
						matched = true
						boolean = true
					}
				case 'f', 'F':
					if size == 1 || strings.EqualFold(val, "false") {
						matched = true
					}
				case 'y', 'Y':
					if size == 1 || strings.EqualFold(val, "yes") {
						matched = true
						boolean = true
					}
				case 'n', 'N':
					if size == 1 || strings.EqualFold(val, "no") {
						matched = true
					}
				case 'o', 'O':
					if strings.EqualFold(val, "on") {
						matched = true
						boolean = true
					} else if strings.EqualFold(val, "off") {
						matched = true
					}
				case '1':
					if size == 1 {
						matched = true
						boolean = true
					}
				case '0':
					if size == 1 {
						matched = true
					}
				}

				if !matched {
					return exec.returnError(fmt.Errorf(
						`%w: argument "%v" of jsonpath item method %v is invalid for type boolean`,
						ErrExecution, val, name,
					))
				}

			default:
				return exec.returnError(fmt.Errorf(
					"%w: jsonpath item method %v can only be applied to a bool, string, or numeric value",
					ErrExecution, name,
				))
			}
			return exec.executeNextItem(ctx, node, nil, boolean, found)

		case ast.MethodKeyValue:
			return exec.executeKeyValueMethod(ctx, node, jVal, found, unwrap)
		}

	case *ast.AnyNode:
		next := node.Next()
		// first try without any intermediate steps
		if node.First() == 0 {
			savedIgnoreStructuralErrors := false
			exec.ignoreStructuralErrors = savedIgnoreStructuralErrors
			res, err = exec.executeNextItem(ctx, node, next, jVal, found)
			exec.ignoreStructuralErrors = savedIgnoreStructuralErrors
			if res == statusOK && found == nil {
				return res, err
			}
		}

		switch jVal := jVal.(type) {
		case map[string]any:
			return exec.executeAnyItem(
				ctx, next, maps.Values(jVal), found, 1,
				node.First(), node.Last(), true, exec.autoUnwrap(),
			)
		case []any:
			return exec.executeAnyItem(
				ctx, next, jVal, found, 1,
				node.First(), node.Last(), true, exec.autoUnwrap(),
			)
		}

	case *ast.ArrayIndexNode:
		res := statusNotFound
		var resErr error

		//nolint:nestif
		if array, ok := jVal.([]any); ok || exec.autoWrap() {
			innermostArraySize := exec.innermostArraySize
			if !ok {
				// auto wrap
				array = []any{jVal}
			}
			size := len(array)
			next := node.Next()
			exec.innermostArraySize = size // for LAST evaluation

			for _, subscript := range node.Subscripts() {
				subscript, ok := subscript.(*ast.BinaryNode)
				if !ok || subscript.Operator() != ast.BinarySubscript {
					return statusFailed, fmt.Errorf(
						"%w: jsonpath array subscript is not a single numeric value",
						ErrExecution,
					)
				}
				indexFrom, err := exec.getArrayIndex(ctx, subscript.Left(), jVal)
				if err != nil {
					return exec.returnError(err)
				}

				indexTo := indexFrom
				if right := subscript.Right(); right != nil {
					indexTo, err = exec.getArrayIndex(ctx, right, jVal)
					if err != nil {
						return exec.returnError(err)
					}
				}

				if !exec.ignoreStructuralErrors && (indexFrom < 0 || indexFrom > indexTo || indexTo >= size) {
					return exec.returnError(fmt.Errorf(
						"%w: jsonpath array subscript is out of bounds",
						ErrExecution,
					))
				}

				if indexFrom < 0 {
					indexFrom = 0
				}

				if indexTo >= size {
					indexTo = size - 1
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
					if res.failed() {
						break
					}

					if res == statusOK && found == nil {
						break
					}
				}
				if res.failed() {
					break
				}
				if res == statusOK && found == nil {
					break
				}
			}
			exec.innermostArraySize = innermostArraySize
			return res, resErr
		} else if !exec.ignoreStructuralErrors {
			return exec.returnError(fmt.Errorf(
				"%w: jsonpath array accessor can only be applied to an array",
				ErrExecution,
			))
		}
	}

	return res, err
}

// executeNumberMethod implements the numeric() and decimal() methods. It
// varies somewhat from Postgres because Postgres uses its arbitrary precision
// numeric type, which can be huge and precise, while we use only float64 and
// int64 values. If we ever switched to the github.com/shopspring/decimal
// package we could make it more precise, especially when JSON numbers are
// parsed using json.Number.
//
//nolint:funlen,gocognit
func (exec *Executor) executeNumberMethod(
	ctx context.Context,
	node ast.Node,
	// precision, scale int,
	jVal any,
	found *valueList,
	unwrap bool,
) (resultStatus, error) {
	var (
		num float64
		err error
	)

	switch val := jVal.(type) {
	case []any:
		if unwrap {
			return exec.executeItemUnwrapTargetArray(ctx, node, val, found)
		}
		return exec.returnError(fmt.Errorf(
			`%w: jsonpath item method %v can only be applied to a string or numeric value`,
			ErrInvalid, node,
		))
	case float64:
		num = val
	case int64:
		num = float64(val)
	case json.Number:
		num, err = val.Float64()
	case string:
		// cast string as number
		num, err = strconv.ParseFloat(val, 64)
	default:
		return exec.returnError(fmt.Errorf(
			`%w: jsonpath item method %v can only be applied to a string or numeric value`,
			ErrInvalid, node,
		))
	}

	if err != nil {
		return exec.returnError(fmt.Errorf(
			`%w: argument "%v" of jsonpath item method %v is invalid for type numeric`,
			ErrInvalid, jVal, node,
		))
	}

	if math.IsInf(num, 0) || math.IsNaN(num) {
		return exec.returnError(fmt.Errorf(
			"%w: NaN or Infinity is not allowed for jsonpath item method %v",
			ErrInvalid, node,
		))
	}

	// For the .decimal() method, we must have the precision and optional
	// scale. Convert them to int32, format the number as string and then
	// parse back into a float.
	//nolint:nestif
	if node, ok := node.(*ast.BinaryNode); ok && node.Operator() == ast.BinaryDecimal && node.Left() != nil {
		op := node.Operator()
		precision, err := getNodeInt32(op, node.Left(), "precision")
		if err != nil {
			if errors.Is(err, ErrInvalid) {
				return statusFailed, err
			}
			return exec.returnError(err)
		}

		// Verify the precision
		// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/backend/utils/adt/numeric.c#L1326-L1330
		const numericMaxPrecision = 1000
		if precision < 1 || precision > numericMaxPrecision {
			return statusFailed, fmt.Errorf(
				"%w: NUMERIC precision %d must be between 1 and %d",
				ErrInvalid, precision, numericMaxPrecision,
			)
		}

		scale := 0
		if right := node.Right(); right != nil {
			var err error
			scale, err = getNodeInt32(op, right, "scale")
			if err != nil {
				if errors.Is(err, ErrInvalid) {
					return statusFailed, err
				}
				return exec.returnError(err)
			}

			// Verify the scale.
			// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/backend/utils/adt/numeric.c#L1331-L1335
			const numericMinScale = -1000
			const numericMaxScale = 1000
			if scale < numericMinScale || scale > numericMaxScale {
				return statusFailed, fmt.Errorf(
					"%w: NUMERIC scale %d must be between %d and %d",
					ErrInvalid, scale, numericMinScale, numericMaxScale,
				)
			}
		}

		// Round to the scale.
		ratio := math.Pow10(scale)
		rounded := math.Round(num*ratio) / ratio

		// Count the digits before the decimal point.
		numStr := strconv.FormatFloat(rounded, 'f', -1, 64)
		count := 0
		for _, ch := range numStr {
			if ch == '.' {
				break
			}
			if '1' <= ch && ch <= '9' {
				count++
			}
		}

		// Make sure it's got no more than precision digits.
		if count > 0 && count > precision-scale {
			return exec.returnError(fmt.Errorf(
				`%w: argument "%v" of jsonpath item method %v is invalid for type numeric`,
				ErrInvalid, jVal, op,
			))
		}
		num = rounded
	}

	return exec.executeNextItem(ctx, node, nil, num, found)
}

func getNodeInt32(meth any, node ast.Node, field string) (int, error) {
	var num int64
	switch node := node.(type) {
	case *ast.IntegerNode:
		num = node.Int()
	case *ast.NumericNode:
		num = int64(node.Float())
	default:
		return 0, fmt.Errorf(
			"%w: invalid jsonpath item type for %v %v",
			ErrInvalid, meth, field,
		)
	}

	if num > math.MaxInt32 || num < math.MinInt32 {
		return 0, fmt.Errorf(
			"%w: %v of jsonpath item method %v is out of integer range",
			ErrExecution, field, meth,
		)
	}

	return int(num), nil
}

func getJSONInt32(op string, val any) (int, error) {
	var num int64
	switch val := val.(type) {
	case int64:
		num = val
	case float64:
		num = int64(val)
	case json.Number:
		if integer, err := val.Int64(); err == nil {
			num = integer
		} else if float, err := val.Float64(); err == nil {
			num = int64(float)
		} else {
			// Should not happen.
			return 0, fmt.Errorf(
				"%w: jsonpath %v is not a single numeric value",
				ErrExecution, op,
			)
		}
	default:
		return 0, fmt.Errorf(
			"%w: jsonpath %v is not a single numeric value",
			ErrExecution, op,
		)
	}

	if num > math.MaxInt32 || num < math.MinInt32 {
		return 0, fmt.Errorf(
			"%w: jsonpath %v is out of integer range",
			ErrExecution, op,
		)
	}

	return int(num), nil
}

// executeItemUnwrapTargetArray unwraps the current array item and executes
// node for each of its elements.
func (exec *Executor) executeItemUnwrapTargetArray(
	ctx context.Context,
	node ast.Node,
	jVal any,
	found *valueList,
) (resultStatus, error) {
	array, ok := jVal.([]any)
	if !ok {
		return statusFailed, fmt.Errorf(
			"%w: invalid json array value type: %T",
			ErrExecution, jVal,
		)
	}

	return exec.executeAnyItem(ctx, node, array, found, 1, 1, 1, false, false)
}

//nolint:funlen,gocognit
func (exec *Executor) executeBoolItem(
	ctx context.Context,
	node ast.Node,
	jVal any,
	canHaveNext bool,
) (predOutcome, error) {
	if !canHaveNext && node.Next() != nil {
		return predUnknown, fmt.Errorf("%w: boolean jsonpath item cannot have next item", ErrInvalid)
	}

	//nolint:exhaustive
	switch node := node.(type) {
	case *ast.BinaryNode:
		switch node.Operator() {
		case ast.BinaryAnd:
			res, err := exec.executeBoolItem(ctx, node.Left(), jVal, false)
			if res == predFalse {
				return res, err
			}

			// SQL/JSON says that we should check second arg in case of error
			res2, err2 := exec.executeBoolItem(ctx, node.Right(), jVal, false)
			if res2 == predTrue {
				return res, err2
			}
			return res2, err
		case ast.BinaryOr:
			res, err := exec.executeBoolItem(ctx, node.Left(), jVal, false)
			if res == predTrue {
				return res, err
			}
			res2, err2 := exec.executeBoolItem(ctx, node.Right(), jVal, false)
			if res2 == predFalse {
				return res, err
			}
			return res2, err2
		case ast.BinaryEqual, ast.BinaryNotEqual, ast.BinaryLess,
			ast.BinaryGreater, ast.BinaryLessOrEqual, ast.BinaryGreaterOrEqual:
			return exec.executePredicate(ctx, node, node.Left(), node.Right(), jVal, true, exec.compareItems)
		case ast.BinaryStartsWith:
			return exec.executePredicate(ctx, node, node.Left(), node.Right(), jVal, false, exec.executeStartsWith)
		}
	case *ast.UnaryNode:
		//nolint:exhaustive
		switch node.Operator() {
		case ast.UnaryNot:
			res, err := exec.executeBoolItem(ctx, node.Operand(), jVal, false)
			switch res {
			case predUnknown:
				return res, err
			case predTrue:
				return predFalse, nil
			case predFalse:
				return predTrue, nil
			}
		case ast.UnaryIsUnknown:
			res, _ := exec.executeBoolItem(ctx, node.Operand(), jVal, false)
			if res == predUnknown {
				return predTrue, nil
			}
			return predFalse, nil
		case ast.UnaryExists:
			if exec.strictAbsenceOfErrors() {
				// In strict mode we must get a complete list of values to
				// check that there are no errors at all.
				vals := newList()
				res, err := exec.executeItemOptUnwrapResultNoThrow(ctx, node.Operand(), jVal, false, vals)
				if res == statusFailed {
					return predUnknown, err
				}
				if vals.isEmpty() {
					return predFalse, nil
				}
				return predTrue, nil
			}

			res, err := exec.executeItemOptUnwrapResultNoThrow(ctx, node.Operand(), jVal, false, nil)
			if res == statusFailed {
				return predUnknown, err
			}
			if res == statusOK {
				return predTrue, nil
			}
			return predFalse, nil
		}
	case *ast.RegexNode:
		return exec.executePredicate(ctx, node, node.Operand(), nil, jVal, false, exec.executeLikeRegex)
	}
	return predUnknown, fmt.Errorf(
		"%w: invalid boolean jsonpath item type: %T",
		ErrInvalid, node,
	)
}

/*
 * Convert boolean execution status 'res' to a boolean JSON item and execute
 * next jsonpath.
 */
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
	var jVal any

	if res == predUnknown {
		jVal = nil
	} else {
		jVal = res == predTrue
	}

	return exec.executeNextItem(ctx, node, next, jVal, found)
}

/*
 * Execute next jsonpath item if exists.  Otherwise put "v" to the "found"
 * list if provided.
 */
func (exec *Executor) executeNextItem(
	ctx context.Context,
	cur, next ast.Node,
	jVal any,
	found *valueList,
	// copy bool,
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
		return exec.executeItem(ctx, next, jVal, found)
	}

	if found != nil {
		found.append(jVal)
	}

	return statusOK, nil
}

// executeAnyItem is the implementation of several jsonpath nodes:
//   - jpiAny (.** accessor)
//   - jpiAnyKey (.* accessor)
//   - jpiAnyArray ([*] accessor)
func (exec *Executor) executeAnyItem(
	ctx context.Context,
	node ast.Node,
	jVal []any,
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

	for _, v := range jVal {
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

type predicateCallback func(node ast.Node, left, right any) (predOutcome, error)

// executePredicate executes a unary or binary predicate.
//
// Predicates have existence semantics, because their operands are item
// sequences. Pairs of items from the left and right operand's sequences are
// checked. TRUE returned only if any pair satisfying the condition is found.
// In strict mode, even if the desired pair has already been found, all pairs
// still need to be examined to check the absence of errors. If any error
// occurs, UNKNOWN (analogous to SQL NULL) is returned.
func (exec *Executor) executePredicate(
	ctx context.Context,
	pred, left, right ast.Node,
	jVal any,
	unwrapRightArg bool,
	callback predicateCallback,
) (predOutcome, error) {
	hasErr := false
	found := false

	// Left argument is always auto-unwrapped.
	lSeq := newList()
	res, err := exec.executeItemOptUnwrapResultNoThrow(ctx, left, jVal, true, lSeq)
	if res == statusFailed {
		return predUnknown, err
	}

	rSeq := newList()
	if right != nil {
		// Right argument is conditionally auto-unwrapped.
		res, err := exec.executeItemOptUnwrapResultNoThrow(ctx, right, jVal, unwrapRightArg, rSeq)
		if res == statusFailed {
			return predUnknown, err
		}
	} else {
		// Right arg is nil.
		rSeq.append(nil)
	}

	for _, lVal := range lSeq.list {
		// Loop over right arg sequence.
		for _, rVal := range rSeq.list {
			res, err := callback(pred, lVal, rVal)
			if err != nil {
				return predUnknown, err
			}
			//nolint:exhaustive
			switch res {
			case predUnknown:
				if exec.strictAbsenceOfErrors() {
					return predUnknown, nil
				}
				hasErr = true
			case predTrue:
				if !exec.strictAbsenceOfErrors() {
					return predTrue, nil
				}
				found = true
			}
		}
	}

	if found { // possible only in strict mode
		return predTrue, nil
	}

	if hasErr { //  possible only in lax mode
		return predUnknown, nil
	}

	return predFalse, nil
}

// Compare two SQL/JSON items using comparison operation 'op'.
//
//nolint:funlen
func (exec *Executor) compareItems(node ast.Node, left, right any) (predOutcome, error) {
	var cmp int
	var res bool
	bin, ok := node.(*ast.BinaryNode)
	if !ok {
		panic(fmt.Sprintf("Invalid node type %T passed to compareItems", node))
	}
	op := bin.Operator()

	if (left == nil && right != nil) || (right == nil && left != nil) {
		// Equality and order comparison of nulls to non-nulls returns
		// always false, but inequality comparison returns true.
		if op == ast.BinaryNotEqual {
			return predTrue, nil
		}
		return predFalse, nil
	}

	switch left := left.(type) {
	case nil:
		cmp = 0
	case bool:
		right, ok := right.(bool)
		if !ok {
			return predUnknown, nil
		}
		switch {
		case left == right:
			cmp = 0
		case left:
			cmp = 1
		default:
			cmp = -1
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
			if cmp == 0 {
				return predTrue, nil
			}
			return predFalse, nil
		}
	case *types.Date, *types.Time, *types.TimeTZ, *types.Timestamp, *types.TimestampTZ:
		var err error
		cmp, err = compareDatetime(left, right, exec.useTZ)
		if cmp < -1 || err != nil {
			return predUnknown, err
		}
	case map[string]any, []any:
		// non-scalars are not comparable
		return predUnknown, nil
	default:
		return predUnknown, fmt.Errorf("%w: invalid json value type %T", ErrInvalid, left)
	}

	//nolint:exhaustive
	switch op {
	case ast.BinaryEqual:
		res = cmp == 0
	case ast.BinaryNotEqual:
		res = cmp != 0
	case ast.BinaryLess:
		res = cmp < 0
	case ast.BinaryGreater:
		res = cmp > 0
	case ast.BinaryLessOrEqual:
		res = cmp <= 0
	case ast.BinaryGreaterOrEqual:
		res = cmp >= 0
	default:
		return predUnknown, fmt.Errorf("%w: unrecognized jsonpath operation %v", ErrInvalid, op)
	}

	if res {
		return predTrue, nil
	}
	return predFalse, nil
}

func compareNumbers[T int | int64 | float64](left, right T) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}

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
		default:
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

// Return error when timezone required for casting from type1 to type2.
func tzRequiredCast(type1, type2 string) error {
	return fmt.Errorf(
		"%w: cannot convert value from %v to %v without time zone usage. HINT: Use WithTZ() option for time zone support",
		ErrExecution, type1, type2,
	)
}

func incomparableDateTime(val any) (int, error) {
	return 0, fmt.Errorf(
		"%w: unrecognized SQL/JSON datetime type %T",
		ErrInvalid, val,
	)
}

// compareDatetime performs a Cross-type comparison of two datetime SQL/JSON
// items. Returns -1 if items are incomparable. Returns an error if a cast
// timezone and it is not used.
//
//nolint:funlen
func compareDatetime(val1, val2 any, useTZ bool) (int, error) {
	switch val1 := val1.(type) {
	case *types.Date:
		switch val2 := val2.(type) {
		case *types.Date:
			return val1.Compare(val2.Time), nil
		case *types.Timestamp:
			return val1.Compare(val2.Time), nil
		case *types.TimestampTZ:
			if !useTZ {
				return 0, tzRequiredCast("date", "timestamptz")
			}
			return val1.Compare(val2.Time.UTC()), nil
		case *types.Time, *types.TimeTZ:
			// Incomparable types
			return -2, nil
		default:
			return incomparableDateTime(val2)
		}
	case *types.Time:
		switch val2 := val2.(type) {
		case *types.Time:
			return val1.Compare(val2.Time), nil
		case *types.TimeTZ:
			if !useTZ {
				return 0, tzRequiredCast("time", "timetz")
			}
			return types.NewTimeTZ(val1.Time).Compare(val2.Time), nil

		case *types.Date, *types.Timestamp, *types.TimestampTZ:
			// Incomparable types
			return -2, nil
		default:
			return incomparableDateTime(val2)
		}
	case *types.TimeTZ:
		switch val2 := val2.(type) {
		case *types.Time:
			if !useTZ {
				return 0, tzRequiredCast("time", "timetz")
			}
			return val1.Compare(val2.Time), nil
		case *types.TimeTZ:
			return val1.Compare(val2.Time), nil
		case *types.Date, *types.Timestamp, *types.TimestampTZ:
			// Incomparable types
			return -2, nil
		default:
			return incomparableDateTime(val2)
		}
	case *types.Timestamp:
		switch val2 := val2.(type) {
		case *types.Date:
			return val1.Compare(val2.Time), nil
		case *types.Timestamp:
			return val1.Compare(val2.Time), nil
		case *types.TimestampTZ:
			if !useTZ {
				return 0, tzRequiredCast("timestamp", "timestamptz")
			}
			return val1.Compare(val2.Time.UTC()), nil
		case *types.Time, *types.TimeTZ:
			// Incomparable types
			return -2, nil
		default:
			return incomparableDateTime(val2)
		}
	case *types.TimestampTZ:
		switch val2 := val2.(type) {
		case *types.Date:
			if !useTZ {
				return 0, tzRequiredCast("date", "timestamptz")
			}
			return val1.Compare(val2.Time.UTC()), nil
		case *types.Timestamp:
			if !useTZ {
				return 0, tzRequiredCast("timestamp", "timestamptz")
			}
			return val1.Compare(val2.Time.UTC()), nil
		case *types.TimestampTZ:
			return val1.Compare(val2.Time), nil
		case *types.Time, *types.TimeTZ:
			// Incomparable types
			return -2, nil
		default:
			return incomparableDateTime(val2)
		}
	default:
		return incomparableDateTime(val1)
	}
}

// executeLikeRegex is the LIKE_REGEX predicate callback.
func (exec *Executor) executeLikeRegex(node ast.Node, jVal, _ any) (predOutcome, error) {
	rn, ok := node.(*ast.RegexNode)
	if !ok {
		panic(fmt.Sprintf(
			"Node %T passed to executeLikeRegex is not an ast.RegexNode",
			node,
		))
	}

	str, ok := jVal.(string)
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

type intCallback func(int64) int64

type floatCallback func(float64) float64

func intAbs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

func intSelf(x int64) int64       { return x }
func floatSelf(x float64) float64 { return x }

func intUMinus(x int64) int64       { return -x }
func floatUMinus(x float64) float64 { return -x }

// executeNumericItemMethod executes numeric item methods (.abs(), .floor(),
// .ceil()) using the specified intCallback or floatCallback.
func (exec *Executor) executeNumericItemMethod(
	ctx context.Context,
	node ast.Node,
	jVal any,
	unwrap bool,
	intCallback intCallback,
	floatCallback floatCallback,
	found *valueList,
) (resultStatus, error) {
	var num any

	switch val := jVal.(type) {
	case []any:
		if unwrap {
			return exec.executeItemUnwrapTargetArray(ctx, node, jVal, found)
		}
	case int64:
		num = intCallback(val)
	case float64:
		num = floatCallback(val)
	case json.Number:
		if integer, err := val.Int64(); err == nil {
			num = intCallback(integer)
		} else if float, err := val.Float64(); err == nil {
			num = floatCallback(float)
		} else {
			return exec.returnError(fmt.Errorf(
				"%w: jsonpath item method %v can only be applied to a numeric value",
				ErrExecution, node,
			))
		}
	default:
		return exec.returnError(fmt.Errorf(
			"%w: jsonpath item method %v can only be applied to a numeric value",
			ErrExecution, node,
		))
	}

	return exec.executeNextItem(ctx, node, node.Next(), num, found)
}

// getArrayIndex executes an array subscript expression and converts the
// resulting numeric item to the integer type with truncation.
func (exec *Executor) getArrayIndex(
	ctx context.Context,
	node ast.Node,
	jVal any,
) (int, error) {
	found := newList()
	res, err := exec.executeItem(ctx, node, jVal, found)
	if res == statusFailed {
		return 0, err
	}

	if len(found.list) != 1 {
		return 0, fmt.Errorf(
			"%w: jsonpath array subscript is not a single numeric value",
			ErrExecution,
		)
	}

	return getJSONInt32("array subscript", found.list[0])
}

// executeNestedBoolItem executes a nested (filters etc.) boolean expression
// pushing current SQL/JSON item onto the stack.
func (exec *Executor) executeNestedBoolItem(
	ctx context.Context,
	node ast.Node,
	jVal any,
) (predOutcome, error) {
	prev := exec.current
	defer func(e *Executor, c any) { e.current = c }(exec, prev)
	exec.current = jVal
	return exec.executeBoolItem(ctx, node, jVal, false)
}

// executeDateTimeMethod implements .datetime() and related methods.
//
// Converts a string into a date/time value. The actual type is determined at
// run time.
// If an argument is provided, this argument is used as a template string.
// Otherwise, the first fitting ISO format is selected.
//
// .date(), .time(), .time_tz(), .timestamp(), .timestamp_tz() methods don't
// have a format, so ISO format is used. However, except for .date(), they all
// take an optional time precision.
//
//nolint:funlen,gocognit,gocyclo,maintidx
func (exec *Executor) executeDateTimeMethod(
	ctx context.Context,
	node *ast.UnaryNode,
	jVal any,
	found *valueList,
) (resultStatus, error) {
	op := node.Operator()

	datetime, ok := jVal.(string)
	if !ok {
		return exec.returnError(fmt.Errorf(
			"%w: jsonpath item method %v() can only be applied to a string",
			ErrExecution, op,
		))
	}

	arg := node.Operand()
	var timeVal any

	// .datetime(template) has an argument, the rest of the methods don't have
	// an argument.  So we handle that separately.
	if op == ast.UnaryDateTime && arg != nil {
		// XXX: Requires a format parser, so defer for now.
		return statusFailed, fmt.Errorf(
			"%w: .datetime(template) is not yet supported",
			ErrInvalid,
		)
		// var str *ast.StringNode
		// str, ok = arg.(*ast.StringNode)
		// if !ok {
		// 	return statusFailed, fmt.Errorf(
		// 		"%w: invalid jsonpath item type for .datetime() argument",
		// 		ErrInvalid,
		// 	)
		// }
		// value, ok = types.ParseDateTime(str.Text(), datetime)
		// timeVal = types.NewTimestampTZ(value)
	} // else {
	// Check for optional precision for methods other than .datetime() and
	// .date()
	precision := -1
	//nolint:nestif
	if op != ast.UnaryDateTime && op != ast.UnaryDate && arg != nil {
		var err error
		precision, err = getNodeInt32(op.String()+"()", arg, "time precision")
		if err != nil {
			if errors.Is(err, ErrInvalid) {
				return statusFailed, err
			}
			return exec.returnError(err)
		}
		const maxTimestampPrecision = 6
		if precision < 0 {
			return exec.returnError(fmt.Errorf(
				"%w: time precision of jsonpath item method %v() is invalid",
				ErrExecution, op,
			))
		}
		if precision > maxTimestampPrecision {
			// pg: issues a warning
			precision = maxTimestampPrecision
		}
	}

	// Parse the value.
	timeVal, ok = types.ParseTime(datetime, precision)
	if !ok {
		return exec.returnError(fmt.Errorf(
			`%w: %v format is not recognized: "%v"`,
			ErrExecution, op.String()[1:], datetime,
		))
	}
	// }

	// The parsing above processes the entire input string and returns the
	// best fitted datetime type. So, if this call is for a specific datatype,
	// then we do the conversion here. Return an error for incompatible types.
	//nolint:exhaustive
	switch op {
	case ast.UnaryDateTime:
		// Nothing to do for DATETIME
	case ast.UnaryDate:
		// Convert result type to date
		switch tv := timeVal.(type) {
		case *types.Date:
			// Nothing to do for DATE
		case *types.Time, *types.TimeTZ:
			// Incompatible.
			return exec.returnError(notRecognized(op, datetime))
		case *types.Timestamp:
			timeVal = types.NewDate(tv.Time)
		case *types.TimestampTZ:
			if !exec.useTZ {
				return statusFailed, tzRequiredCast("timestamptz", "date")
			}
			timeVal = types.NewDate(tv.Time.UTC())
		default:
			return statusFailed, fmt.Errorf("%w: type %T not supported", ErrInvalid, tv)
		}
	case ast.UnaryTime:
		switch tv := timeVal.(type) {
		case *types.Date:
			return exec.returnError(notRecognized(op, datetime))
		case *types.Time:
			// Nothing to do for time
		case *types.TimeTZ:
			if !exec.useTZ {
				return statusFailed, tzRequiredCast("timetz", "time")
			}
			timeVal = types.NewTime(tv.Time)
		case *types.Timestamp:
			timeVal = types.NewTime(tv.Time)
		case *types.TimestampTZ:
			if !exec.useTZ {
				return statusFailed, tzRequiredCast("timestamptz", "time")
			}
			timeVal = types.NewTime(tv.Time.UTC())
		default:
			return statusFailed, fmt.Errorf("%w: type %T not supported", ErrInvalid, tv)
		}
	case ast.UnaryTimeTZ:
		switch tv := timeVal.(type) {
		case *types.Date, *types.Timestamp:
			return exec.returnError(notRecognized(op, datetime))
		case *types.Time:
			if !exec.useTZ {
				return statusFailed, tzRequiredCast("time", "timetz")
			}
			timeVal = types.NewTimeTZ(tv.Time.UTC())
		case *types.TimeTZ:
			// Nothing to do for TIMETZ
		case *types.TimestampTZ:
			// Retain the offset.
			timeVal = types.NewTimeTZ(tv.Time)
		default:
			return statusFailed, fmt.Errorf("%w: type %T not supported", ErrInvalid, tv)
		}
	case ast.UnaryTimestamp:
		switch tv := timeVal.(type) {
		case *types.Date:
			timeVal = types.NewTimestamp(tv.Time)
		case *types.Time, *types.TimeTZ:
			return exec.returnError(notRecognized(op, datetime))
		case *types.Timestamp:
			// Nothing to do for TIMESTAMP
		case *types.TimestampTZ:
			if !exec.useTZ {
				return statusFailed, tzRequiredCast("timestamptz", "timestamp")
			}
			timeVal = types.NewTimestamp(tv.Time.UTC())
		default:
			return statusFailed, fmt.Errorf("%w: type %T not supported", ErrInvalid, tv)
		}
	case ast.UnaryTimestampTZ:
		switch tv := timeVal.(type) {
		case *types.Date:
			if !exec.useTZ {
				return statusFailed, tzRequiredCast("date", "timestamptz")
			}
			timeVal = types.NewTimestampTZ(tv.Time)
		case *types.Time, *types.TimeTZ:
			return exec.returnError(notRecognized(op, datetime))
		case *types.Timestamp:
			if !exec.useTZ {
				return statusFailed, tzRequiredCast("timestamp", "timestamptz")
			}
			timeVal = types.NewTimestampTZ(tv.Time.UTC())
		case *types.TimestampTZ:
			// Nothing to do for TIMESTAMPTZ
		default:
			return statusFailed, fmt.Errorf("%w: type %T not supported", ErrInvalid, tv)
		}
	default:
		return statusFailed, fmt.Errorf("%w: unrecognized jsonpath item type: %T", ErrInvalid, op)
	}

	next := node.Next()
	if next == nil && found == nil {
		return statusOK, nil
	}

	return exec.executeNextItem(ctx, node, next, timeVal, found)
}

func notRecognized(op ast.UnaryOperator, datetime string) error {
	return fmt.Errorf(
		`%w: %v format is not recognized: "%v"`,
		ErrExecution, op.String()[1:], datetime,
	)
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
	jVal any,
	found *valueList,
	unwrap bool,
) (resultStatus, error) {
	var obj map[string]any
	switch val := jVal.(type) {
	case []any:
		if unwrap {
			return exec.executeItemUnwrapTargetArray(ctx, node, jVal, found)
		}
		return exec.returnError(fmt.Errorf(
			`%w: jsonpath item method .keyvalue() can only be applied to an object`,
			ErrExecution,
		))
	case map[string]any:
		obj = val
	default:
		return exec.returnError(fmt.Errorf(
			`%w: jsonpath item method .keyvalue() can only be applied to an object`,
			ErrExecution,
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
