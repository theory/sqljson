package exec

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"github.com/theory/sqljson/path/ast"
)

// executeIntegerMath compares lhs to rhs using op and returns the resulting
// value. op must be a binary math operator. Returns an error for an attempt
// to divide by zero.
func executeIntegerMath(lhs, rhs int64, op ast.BinaryOperator) (int64, error) {
	switch op {
	case ast.BinaryAdd:
		return lhs + rhs, nil
	case ast.BinarySub:
		return lhs - rhs, nil
	case ast.BinaryMul:
		return lhs * rhs, nil
	case ast.BinaryDiv:
		if rhs == 0 {
			return 0, fmt.Errorf("%w: division by zero", ErrVerbose)
		}
		return lhs / rhs, nil
	case ast.BinaryMod:
		if rhs == 0 {
			return 0, fmt.Errorf("%w: division by zero", ErrVerbose)
		}
		return lhs % rhs, nil
	default:
		// We process only the binary math operators here.
		return 0, fmt.Errorf("%w: %v is not a binary math operator", ErrInvalid, op)
	}
}

// executeIntegerMath compares lhs to rhs using op and returns the resulting
// value. op must be a binary math operator. Returns an error for an attempt
// to divide by zero.
func executeFloatMath(lhs, rhs float64, op ast.BinaryOperator) (float64, error) {
	switch op {
	case ast.BinaryAdd:
		return lhs + rhs, nil
	case ast.BinarySub:
		return lhs - rhs, nil
	case ast.BinaryMul:
		return lhs * rhs, nil
	case ast.BinaryDiv:
		if rhs == 0 {
			return 0, fmt.Errorf("%w: division by zero", ErrVerbose)
		}
		return lhs / rhs, nil
	case ast.BinaryMod:
		if rhs == 0 {
			return 0, fmt.Errorf("%w: division by zero", ErrVerbose)
		}
		return math.Mod(lhs, rhs), nil
	default:
		// We process only the binary math operators here.
		return 0, fmt.Errorf("%w: %v is not a binary math operator", ErrInvalid, op)
	}
}

// mathOperandErr creates an error for an invalid operand to op. pos is the
// position of the operand, either "left" or "right".
func mathOperandErr(op ast.BinaryOperator, pos string) error {
	return fmt.Errorf(
		"%w: %v operand of jsonpath operator %v is not a single numeric value",
		ErrVerbose, pos, op,
	)
}

// execUnaryMathExpr executes a unary arithmetic expression for each numeric
// item in its operand's sequence. An array operand is automatically unwrapped
// in lax mode.
func (exec *Executor) execUnaryMathExpr(
	ctx context.Context,
	node *ast.UnaryNode,
	value any,
	intCallback intCallback,
	floatCallback floatCallback,
	found *valueList,
) (resultStatus, error) {
	seq := newList()
	res, err := exec.executeItemOptUnwrapResult(ctx, node.Operand(), value, true, seq)
	if res == statusFailed {
		return res, err
	}

	res = statusNotFound
	next := node.Next()
	var val any

	for _, v := range seq.list {
		val = v
		ok := true
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
			val, ok = castJSONNumber(v, intCallback, floatCallback)
		default:
			ok = found == nil && next == nil
		}

		if !ok {
			return exec.returnVerboseError(fmt.Errorf(
				"%w: operand of unary jsonpath operator %v is not a numeric value",
				ErrVerbose, node.Operator(),
			))
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

// execBinaryMathExpr executes a binary arithmetic expression on singleton
// numeric operands. Array operands are automatically unwrapped in lax mode.
func (exec *Executor) execBinaryMathExpr(
	ctx context.Context,
	node *ast.BinaryNode,
	value any,
	op ast.BinaryOperator,
	found *valueList,
) (resultStatus, error) {
	// Get the left node.
	// XXX: The standard says only operands of multiplicative expressions are
	// unwrapped. We extend it to other binary arithmetic expressions too.
	lSeq := newList()
	res, err := exec.executeItemOptUnwrapResult(ctx, node.Left(), value, true, lSeq)
	if res == statusFailed {
		return res, err
	}

	if len(lSeq.list) != 1 {
		return exec.returnVerboseError(mathOperandErr(op, "left"))
	}

	rSeq := newList()
	res, err = exec.executeItemOptUnwrapResult(ctx, node.Right(), value, true, rSeq)
	if res == statusFailed {
		return res, err
	}

	if len(rSeq.list) != 1 {
		return exec.returnVerboseError(mathOperandErr(op, "right"))
	}

	val, err := execMathOp(lSeq.list[0], rSeq.list[0], op)
	if err != nil {
		return exec.returnVerboseError(err)
	}

	next := node.Next()
	if next == nil && found == nil {
		return statusOK, nil
	}

	return exec.executeNextItem(ctx, node, next, val, found)
}

// execMathOp casts left and right into numbers and, if it succeeds, applies
// the binary math op to left and right. left and right must be an int64, a
// float64, or a [json.Number]. In the latter case, execMathOp tries to cast
// values to int64, and falls back on float64.
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
