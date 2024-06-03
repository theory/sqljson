package exec

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/theory/sqljson/path/ast"
	"github.com/theory/sqljson/path/types"
)

// execMethodNode dispatches the relevant method for node.
func (exec *Executor) execMethodNode(
	ctx context.Context,
	node *ast.MethodNode,
	value any,
	found *valueList,
	unwrap bool,
) (resultStatus, error) {
	switch name := node.Name(); name {
	case ast.MethodNumber:
		return exec.executeNumberMethod(ctx, node, value, found, unwrap)
	case ast.MethodAbs:
		return exec.executeNumericItemMethod(
			ctx, node, value, unwrap,
			intAbs, math.Abs, found,
		)
	case ast.MethodFloor:
		return exec.executeNumericItemMethod(
			ctx, node, value, unwrap,
			intSelf, math.Floor, found,
		)
	case ast.MethodCeiling:
		return exec.executeNumericItemMethod(
			ctx, node, value, unwrap,
			intSelf, math.Ceil, found,
		)
	case ast.MethodType:
		return exec.execMethodType(ctx, node, value, found)
	case ast.MethodSize:
		return exec.execMethodSize(ctx, node, value, found)
	case ast.MethodDouble:
		return exec.execMethodDouble(ctx, node, value, found, unwrap)
	case ast.MethodInteger:
		return exec.execMethodInteger(ctx, node, value, found, unwrap)
	case ast.MethodBigInt:
		return exec.execMethodBigInt(ctx, node, value, found, unwrap)
	case ast.MethodString:
		return exec.execMethodString(ctx, node, value, found)
	case ast.MethodBoolean:
		return exec.execMethodBoolean(ctx, node, value, found, unwrap)
	case ast.MethodKeyValue:
		return exec.executeKeyValueMethod(ctx, node, value, found, unwrap)
	}

	return statusNotFound, nil
}

// execMethodType handles the execution of .type() by determining the type of
// value and passing it to the next execution node.
func (exec *Executor) execMethodType(
	ctx context.Context,
	node *ast.MethodNode,
	value any,
	found *valueList,
) (resultStatus, error) {
	var typeName string
	switch value.(type) {
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
}

// execMethodSize handles the execution of .size() by determining the size of
// value and passing it to the next execution node. value's type should be
// []any, but it will be passed on if exec.autoWrap returns true and
// exec.ignoreStructuralErrors is true.
func (exec *Executor) execMethodSize(
	ctx context.Context,
	node *ast.MethodNode,
	value any,
	found *valueList,
) (resultStatus, error) {
	size := 1
	switch value := value.(type) {
	case []any:
		size = len(value)
	default:
		if !exec.autoWrap() && !exec.ignoreStructuralErrors {
			return exec.returnVerboseError(fmt.Errorf(
				"%w: jsonpath item method %v can only be applied to an array",
				ErrVerbose, node.Name(),
			))
		}
	}
	return exec.executeNextItem(ctx, node, nil, int64(size), found)
}

// execMethodDouble handles the execution of .double(). value must be a
// numeric value or a string that can be parsed into a float64, or an array
// ([]any) to which .double() will be applied to all of its values when unwrap
// is true.
func (exec *Executor) execMethodDouble(
	ctx context.Context,
	node *ast.MethodNode,
	value any,
	found *valueList,
	unwrap bool,
) (resultStatus, error) {
	var double float64
	name := node.Name()

	switch val := value.(type) {
	case []any:
		if unwrap {
			return exec.executeItemUnwrapTargetArray(ctx, node, value, found)
		}
		return exec.returnVerboseError(fmt.Errorf(
			"%w: jsonpath item method %v can only be applied to a string or numeric value",
			ErrVerbose, name,
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
		return exec.returnVerboseError(fmt.Errorf(
			"%w: jsonpath item method %v can only be applied to a string or numeric value",
			ErrVerbose, name,
		))
	}

	if math.IsInf(double, 0) || math.IsNaN(double) {
		return exec.returnVerboseError(fmt.Errorf(
			"%w: NaN or Infinity is not allowed for jsonpath item method %v",
			ErrVerbose, name,
		))
	}

	return exec.executeNextItem(ctx, node, nil, double, found)
}

// execMethodInteger handles the execution of .integer(). value must be a
// numeric value or a string that can be parsed into an int32, or an array
// ([]any) to which .integer() will be applied to all of its values when
// unwrap is true.
func (exec *Executor) execMethodInteger(
	ctx context.Context,
	node *ast.MethodNode,
	value any,
	found *valueList,
	unwrap bool,
) (resultStatus, error) {
	var integer int64
	name := node.Name()

	switch val := value.(type) {
	case []any:
		if unwrap {
			return exec.executeItemUnwrapTargetArray(ctx, node, value, found)
		}
		return exec.returnVerboseError(fmt.Errorf(
			"%w: jsonpath item method %v can only be applied to a string or numeric value",
			ErrVerbose, name,
		))
	case int64:
		integer = val
	case float64:
		integer = int64(math.Round(val))
	case json.Number:
		var err error
		integer, err = val.Int64()
		if err != nil || integer > math.MaxInt32 || integer < math.MinInt32 {
			return exec.returnVerboseError(fmt.Errorf(
				`%w: argument "%v" of jsonpath item method %v is invalid for type integer`,
				ErrVerbose, value, name,
			))
		}
	case string:
		var err error
		integer, err = strconv.ParseInt(val, 10, 32)
		if err != nil {
			return exec.returnVerboseError(fmt.Errorf(
				`%w: argument "%v" of jsonpath item method %v is invalid for type integer`,
				ErrVerbose, value, name,
			))
		}
	default:
		return exec.returnVerboseError(fmt.Errorf(
			"%w: jsonpath item method %v can only be applied to a string or numeric value",
			ErrVerbose, name,
		))
	}

	if integer > math.MaxInt32 || integer < math.MinInt32 {
		return exec.returnVerboseError(fmt.Errorf(
			`%w: argument "%v" of jsonpath item method %v is invalid for type integer`,
			ErrVerbose, value, name,
		))
	}

	return exec.executeNextItem(ctx, node, nil, integer, found)
}

// execMethodBigInt handles the execution of .bigint(). value must be a
// numeric value or a string that can be parsed into an int64, or an array
// ([]any) to which .bigint() will be applied to all of its values when unwrap
// is true.
func (exec *Executor) execMethodBigInt(
	ctx context.Context,
	node *ast.MethodNode,
	value any,
	found *valueList,
	unwrap bool,
) (resultStatus, error) {
	var bigInt int64
	name := node.Name()

	switch val := value.(type) {
	case []any:
		if unwrap {
			return exec.executeItemUnwrapTargetArray(ctx, node, value, found)
		}
		return exec.returnVerboseError(fmt.Errorf(
			"%w: jsonpath item method %v can only be applied to a string or numeric value",
			ErrVerbose, name,
		))
	case int64:
		bigInt = val
	case float64:
		if val > math.MaxInt64 || val < math.MinInt64 {
			return exec.returnVerboseError(fmt.Errorf(
				`%w: argument "%v" of jsonpath item method %v is invalid for type bigint`,
				ErrVerbose, val, name,
			))
		}
		bigInt = int64(math.Round(val))
	case json.Number:
		var err error
		bigInt, err = val.Int64()
		if err != nil {
			return exec.returnVerboseError(fmt.Errorf(
				`%w: argument "%v" of jsonpath item method %v is invalid for type bigint`,
				ErrVerbose, val, name,
			))
		}
	case string:
		var err error
		bigInt, err = strconv.ParseInt(val, 10, 64)
		if err != nil {
			return exec.returnVerboseError(fmt.Errorf(
				`%w: argument "%v" of jsonpath item method %v is invalid for type bigint`,
				ErrVerbose, val, name,
			))
		}
	default:
		return exec.returnVerboseError(fmt.Errorf(
			"%w: jsonpath item method %v can only be applied to a string or numeric value",
			ErrVerbose, name,
		))
	}

	return exec.executeNextItem(ctx, node, nil, bigInt, found)
}

// execMethodString handles the execution of .string(). value must be a
// string, number, boolean, or able to be cast to a string.
func (exec *Executor) execMethodString(
	ctx context.Context,
	node *ast.MethodNode,
	value any,
	found *valueList,
) (resultStatus, error) {
	var str string
	name := node.Name()

	switch val := value.(type) {
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
		return exec.returnVerboseError(fmt.Errorf(
			`%w: jsonpath item method %v can only be applied to a bool, string, numeric, or datetime value`,
			ErrVerbose, name,
		))
	}

	return exec.executeNextItem(ctx, node, nil, str, found)
}

// execMethodBoolean handles the execution of .boolean(). value must be a
// string, number, boolean, or able to be cast to a bool, int64, float64,
// [json.Number], or string — or an array ([]any) to which .boolean() will be
// applied to all of its values when unwrap is true. String values will be
// converted to bool by [execBooleanString].
func (exec *Executor) execMethodBoolean(
	ctx context.Context,
	node *ast.MethodNode,
	value any,
	found *valueList,
	unwrap bool,
) (resultStatus, error) {
	var boolean bool
	name := node.Name()

	switch val := value.(type) {
	case []any:
		if unwrap {
			return exec.executeItemUnwrapTargetArray(ctx, node, value, found)
		}
		return exec.returnVerboseError(fmt.Errorf(
			"%w: jsonpath item method %v can only be applied to a bool, string, or numeric value",
			ErrVerbose, name,
		))
	case bool:
		boolean = val
	case int64:
		boolean = val != 0
	case float64:
		if val != math.Trunc(val) {
			return exec.returnVerboseError(fmt.Errorf(
				`%w: argument "%v" of jsonpath item method %v is invalid for type boolean`,
				ErrVerbose, val, name,
			))
		}
		boolean = val != 0

	case json.Number:
		num, err := val.Int64()
		if err != nil {
			return exec.returnVerboseError(fmt.Errorf(
				`%w: argument "%v" of jsonpath item method %v is invalid for type boolean`,
				ErrVerbose, val, name,
			))
		}
		boolean = num != 0
	case string:
		var err error
		boolean, err = exec.execBooleanString(val, name)
		if err != nil {
			return exec.returnVerboseError(err)
		}

	default:
		return exec.returnVerboseError(fmt.Errorf(
			"%w: jsonpath item method %v can only be applied to a bool, string, or numeric value",
			ErrVerbose, name,
		))
	}

	return exec.executeNextItem(ctx, node, nil, boolean, found)
}

// execBooleanString converts val to a boolean. The value of val must
// case-insensitively match one of:
//   - t
//   - true
//   - f
//   - false
//   - y
//   - yes
//   - n
//   - no
//   - on
//   - off
//   - 1
//   - 0
func (exec *Executor) execBooleanString(val string, name ast.MethodName) (bool, error) {
	size := len(val)
	if size == 0 {
		return false, fmt.Errorf(
			`%w: argument "%v" of jsonpath item method %v is invalid for type boolean`,
			ErrVerbose, val, name,
		)
	}

	switch val[0] {
	case 't', 'T':
		if size == 1 || strings.EqualFold(val, "true") {
			return true, nil
		}
	case 'f', 'F':
		if size == 1 || strings.EqualFold(val, "false") {
			return false, nil
		}
	case 'y', 'Y':
		if size == 1 || strings.EqualFold(val, "yes") {
			return true, nil
		}
	case 'n', 'N':
		if size == 1 || strings.EqualFold(val, "no") {
			return false, nil
		}
	case 'o', 'O':
		if strings.EqualFold(val, "on") {
			return true, nil
		} else if strings.EqualFold(val, "off") {
			return false, nil
		}
	case '1':
		if size == 1 {
			return true, nil
		}
	case '0':
		if size == 1 {
			return false, nil
		}
	}

	return false, fmt.Errorf(
		`%w: argument "%v" of jsonpath item method %v is invalid for type boolean`,
		ErrVerbose, val, name,
	)
}

// executeNumberMethod implements the numeric() and decimal() methods. It
// varies somewhat from Postgres because Postgres uses its arbitrary precision
// numeric type, which can be huge and precise, while we use only float64 and
// int64 values. If we ever switch to the github.com/shopspring/decimal
// package we could make it more precise and therefore compatible, at least
// when numbers are parsed into [json.Number].
func (exec *Executor) executeNumberMethod(
	ctx context.Context,
	node ast.Node,
	value any,
	found *valueList,
	unwrap bool,
) (resultStatus, error) {
	var (
		num float64
		err error
	)

	switch val := value.(type) {
	case []any:
		if unwrap {
			return exec.executeItemUnwrapTargetArray(ctx, node, val, found)
		}
		return exec.returnVerboseError(fmt.Errorf(
			`%w: jsonpath item method %v can only be applied to a string or numeric value`,
			ErrVerbose, node,
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
		return exec.returnVerboseError(fmt.Errorf(
			`%w: jsonpath item method %v can only be applied to a string or numeric value`,
			ErrVerbose, node,
		))
	}

	if err != nil {
		return exec.returnVerboseError(fmt.Errorf(
			`%w: argument "%v" of jsonpath item method %v is invalid for type numeric`,
			ErrVerbose, value, node,
		))
	}

	if math.IsInf(num, 0) || math.IsNaN(num) {
		return exec.returnVerboseError(fmt.Errorf(
			"%w: NaN or Infinity is not allowed for jsonpath item method %v",
			ErrVerbose, node,
		))
	}

	if node, ok := node.(*ast.BinaryNode); ok {
		num, err = exec.executeDecimalMethod(node, value, num)
		if err != nil {
			return exec.returnError(err)
		}
	}

	return exec.executeNextItem(ctx, node, nil, num, found)
}

// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/include/utils/numeric.h#L32-L35
const (
	numericMaxPrecision = 1000
	numericMinScale     = -1000
	numericMaxScale     = 1000
)

// executeDecimalMethod processes the arguments to the .decimal() method,
// which must have the precision and optional scale. It converts them to
// int32, formats the number as string and then parses back into a float,
// which it returns.
func (exec *Executor) executeDecimalMethod(
	node *ast.BinaryNode,
	value any,
	num float64,
) (float64, error) {
	op := node.Operator()
	if op != ast.BinaryDecimal || node.Left() == nil {
		return num, nil
	}

	precision, err := getNodeInt32(op, node.Left(), "precision")
	if err != nil {
		return 0, err
	}

	// Verify the precision
	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/backend/utils/adt/numeric.c#L1326-L1330
	if precision < 1 || precision > numericMaxPrecision {
		return 0, fmt.Errorf(
			"%w: NUMERIC precision %d must be between 1 and %d",
			ErrExecution, precision, numericMaxPrecision,
		)
	}

	scale := 0
	if right := node.Right(); right != nil {
		var err error
		scale, err = getNodeInt32(op, right, "scale")
		if err != nil {
			return 0, err
		}

		// Verify the scale.
		// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/backend/utils/adt/numeric.c#L1331-L1335
		if scale < numericMinScale || scale > numericMaxScale {
			return 0, fmt.Errorf(
				"%w: NUMERIC scale %d must be between %d and %d",
				ErrExecution, scale, numericMinScale, numericMaxScale,
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
		return 0, fmt.Errorf(
			`%w: argument "%v" of jsonpath item method %v is invalid for type numeric`,
			ErrVerbose, value, op,
		)
	}
	return rounded, nil
}

// intCallback defines a callback to carry out an operation on an int64.
type intCallback func(int64) int64

// floatCallback defines a callback to carry out an operation on a float64.
type floatCallback func(float64) float64

// intAbs returns the absolute value of x. Implements intCallback.
func intAbs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

// intSelf returns x. Implements intCallback.
func intSelf(x int64) int64 { return x }

// floatSelf returns x.  Implements floatCallback.
func floatSelf(x float64) float64 { return x }

// intUMinus applies unary minus to x. Implements intCallback.
func intUMinus(x int64) int64 { return -x }

// floatUMinus applies unary minus to x. Implements floatCallback.
func floatUMinus(x float64) float64 { return -x }

// executeNumericItemMethod executes numeric item methods (.abs(), .floor(),
// .ceil()) using the specified intCallback or floatCallback.
func (exec *Executor) executeNumericItemMethod(
	ctx context.Context,
	node ast.Node,
	value any,
	unwrap bool,
	intCallback intCallback,
	floatCallback floatCallback,
	found *valueList,
) (resultStatus, error) {
	var num any

	switch val := value.(type) {
	case []any:
		if unwrap {
			return exec.executeItemUnwrapTargetArray(ctx, node, value, found)
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
			return exec.returnVerboseError(fmt.Errorf(
				"%w: jsonpath item method %v can only be applied to a numeric value",
				ErrVerbose, node,
			))
		}
	default:
		return exec.returnVerboseError(fmt.Errorf(
			"%w: jsonpath item method %v can only be applied to a numeric value",
			ErrVerbose, node,
		))
	}

	return exec.executeNextItem(ctx, node, node.Next(), num, found)
}
