package exec

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/theory/sqljson/path/ast"
	"github.com/theory/sqljson/path/parser"
)

func TestExecuteIntegerMath(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	r := require.New(t)

	for _, tc := range []struct {
		name  string
		left  int64
		right int64
		op    ast.BinaryOperator
		exp   int64
		err   string
		isErr error
	}{
		{
			name:  "add",
			left:  20,
			right: 22,
			op:    ast.BinaryAdd,
			exp:   42,
		},
		{
			name:  "sub",
			left:  20,
			right: 22,
			op:    ast.BinarySub,
			exp:   -2,
		},
		{
			name:  "mul",
			left:  21,
			right: 2,
			op:    ast.BinaryMul,
			exp:   42,
		},
		{
			name:  "div",
			left:  42,
			right: 2,
			op:    ast.BinaryDiv,
			exp:   21,
		},
		{
			name:  "div_zero",
			left:  42,
			right: 0,
			op:    ast.BinaryDiv,
			err:   "exec: division by zero",
			isErr: ErrVerbose,
		},
		{
			name:  "mod",
			left:  42,
			right: 4,
			op:    ast.BinaryMod,
			exp:   2,
		},
		{
			name:  "mod_zero",
			left:  42,
			right: 0,
			op:    ast.BinaryMod,
			err:   "exec: division by zero",
			isErr: ErrVerbose,
		},
		{
			name:  "not_math",
			op:    ast.BinaryAnd,
			err:   "exec invalid: && is not a binary math operator",
			isErr: ErrInvalid,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			res, err := executeIntegerMath(tc.left, tc.right, tc.op)
			a.Equal(tc.exp, res)
			if tc.isErr == nil {
				r.NoError(err)
			} else {
				r.EqualError(err, tc.err)
				r.ErrorIs(err, tc.isErr)
			}
		})
	}
}

func TestExecuteFloatMath(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	r := require.New(t)

	for _, tc := range []struct {
		name  string
		left  float64
		right float64
		op    ast.BinaryOperator
		exp   float64
		err   string
		isErr error
	}{
		{
			name:  "add",
			left:  98.6,
			right: 0.5,
			op:    ast.BinaryAdd,
			exp:   99.1,
		},
		{
			name:  "sub",
			left:  14.8,
			right: 1.4,
			op:    ast.BinarySub,
			exp:   13.4,
		},
		{
			name:  "mul",
			left:  18,
			right: 2.2,
			op:    ast.BinaryMul,
			exp:   39.6,
		},
		{
			name:  "div",
			left:  12.4,
			right: 4,
			op:    ast.BinaryDiv,
			exp:   3.1,
		},
		{
			name:  "div_zero",
			left:  42,
			right: 0.0,
			op:    ast.BinaryDiv,
			err:   "exec: division by zero",
			isErr: ErrVerbose,
		},
		{
			name:  "mod",
			left:  42.0,
			right: 4.0,
			op:    ast.BinaryMod,
			exp:   2.0,
		},
		{
			name:  "mod_zero",
			left:  42,
			right: 0.0,
			op:    ast.BinaryMod,
			err:   "exec: division by zero",
			isErr: ErrVerbose,
		},
		{
			name:  "not_math",
			op:    ast.BinaryAnd,
			err:   "exec invalid: && is not a binary math operator",
			isErr: ErrInvalid,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			res, err := executeFloatMath(tc.left, tc.right, tc.op)
			//nolint:testifylint
			a.Equal(tc.exp, res)
			if tc.isErr == nil {
				r.NoError(err)
			} else {
				r.EqualError(err, tc.err)
				r.ErrorIs(err, tc.isErr)
			}
		})
	}
}

func TestMathOperandErr(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	for _, tc := range []struct {
		name string
		op   ast.BinaryOperator
		pos  string
	}{
		{
			name: "add_left",
			op:   ast.BinaryAdd,
			pos:  "left",
		},
		{
			name: "sub_right",
			op:   ast.BinarySub,
			pos:  "right",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := mathOperandErr(tc.op, tc.pos)
			r.EqualError(err, fmt.Sprintf(
				"exec: %v operand of jsonpath operator %v is not a single numeric value",
				tc.pos, tc.op,
			))
			r.ErrorIs(err, ErrVerbose)
		})
	}
}

func TestExecUnaryMathExpr(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	r := require.New(t)
	ctx := context.Background()
	path, _ := parser.Parse("$")
	icb := func(i int64) int64 { return i * 2 }
	fcb := func(i float64) float64 { return i * 3 }

	for _, tc := range []struct {
		name     string
		node     ast.Node
		value    any
		exp      resultStatus
		find     []any
		err      string
		isErr    error
		okNoList bool
	}{
		{
			name:  "item_error",
			node:  ast.NewUnary(ast.UnaryPlus, ast.NewVariable("x")),
			exp:   statusFailed,
			err:   `exec: could not find jsonpath variable "x"`,
			isErr: ErrExecution,
		},
		{
			name:  "int",
			node:  ast.NewUnary(ast.UnaryPlus, ast.NewConst(ast.ConstRoot)),
			value: int64(-2),
			exp:   statusOK,
			find:  []any{int64(-4)},
		},
		{
			name:  "ints",
			node:  ast.NewUnary(ast.UnaryPlus, ast.NewConst(ast.ConstRoot)),
			value: []any{int64(-2), int64(5)},
			exp:   statusOK,
			find:  []any{int64(-4), int64(10)},
		},
		{
			name:  "float",
			node:  ast.NewUnary(ast.UnaryPlus, ast.NewConst(ast.ConstRoot)),
			value: []any{float64(-2), float64(5)},
			exp:   statusOK,
			find:  []any{float64(-6), float64(15)},
		},
		{
			name:  "json_int",
			node:  ast.NewUnary(ast.UnaryPlus, ast.NewConst(ast.ConstRoot)),
			value: []any{json.Number("-2"), json.Number("5")},
			exp:   statusOK,
			find:  []any{int64(-4), int64(10)},
		},
		{
			name:  "json_float",
			node:  ast.NewUnary(ast.UnaryPlus, ast.NewConst(ast.ConstRoot)),
			value: []any{json.Number("-2.5"), json.Number("5.5")},
			exp:   statusOK,
			find:  []any{float64(-7.5), float64(16.5)},
		},
		{
			name:     "json_bad",
			node:     ast.NewUnary(ast.UnaryPlus, ast.NewConst(ast.ConstRoot)),
			value:    []any{json.Number("lol")},
			exp:      statusFailed,
			err:      `exec: operand of unary jsonpath operator + is not a numeric value`,
			isErr:    ErrVerbose,
			okNoList: true,
		},
		{
			name:     "nan",
			node:     ast.NewUnary(ast.UnaryMinus, ast.NewConst(ast.ConstRoot)),
			value:    []any{"foo"},
			exp:      statusFailed,
			err:      `exec: operand of unary jsonpath operator - is not a numeric value`,
			isErr:    ErrVerbose,
			okNoList: true,
		},
		{
			name: "next_item",
			node: ast.LinkNodes([]ast.Node{
				ast.NewUnary(ast.UnaryPlus, ast.NewConst(ast.ConstRoot)),
				ast.NewMethod(ast.MethodString),
			}),
			value: []any{int64(21)},
			exp:   statusOK,
			find:  []any{"42"},
		},
		{
			name: "next_item_error",
			node: ast.LinkNodes([]ast.Node{
				ast.NewUnary(ast.UnaryPlus, ast.NewConst(ast.ConstRoot)),
				ast.NewMethod(ast.MethodKeyValue),
			}),
			value: []any{int64(21)},
			exp:   statusFailed,
			err:   `exec: jsonpath item method .keyvalue() can only be applied to an object`,
			isErr: ErrVerbose,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Make sure we have a unary node.
			node, ok := tc.node.(*ast.UnaryNode)
			r.True(ok)

			// Set up an executor.
			e := newTestExecutor(path, nil, true, false)
			e.root = tc.value

			// Test execKeyNode with a list.
			list := newList()
			res, err := e.execUnaryMathExpr(ctx, node, tc.value, icb, fcb, list)
			a.Equal(tc.exp, res)

			// Check the error and list.
			if tc.isErr == nil {
				r.NoError(err)
				a.Equal(tc.find, list.list)
			} else {
				r.EqualError(err, tc.err)
				r.ErrorIs(err, tc.isErr)
				a.Empty(list.list)
			}

			// Try with nil found.
			res, err = e.execUnaryMathExpr(ctx, node, tc.value, icb, fcb, nil)
			if tc.okNoList {
				a.Equal(statusOK, res)
				r.NoError(err)
			} else {
				a.Equal(tc.exp, res)
				if tc.isErr == nil {
					r.NoError(err)
				} else {
					r.EqualError(err, tc.err)
					r.ErrorIs(err, tc.isErr)
				}
			}
		})
	}
}

func TestExecBinaryMathExpr(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	r := require.New(t)
	ctx := context.Background()
	path, _ := parser.Parse("$")

	for _, tc := range []struct {
		name  string
		node  ast.Node
		value any
		exp   resultStatus
		find  []any
		err   string
		isErr error
	}{
		{
			name:  "invalid_left_value",
			node:  ast.NewBinary(ast.BinaryAdd, ast.NewVariable("x"), ast.NewInteger("2")),
			exp:   statusFailed,
			err:   `exec: could not find jsonpath variable "x"`,
			isErr: ErrExecution,
		},
		{
			name:  "invalid_right_value",
			node:  ast.NewBinary(ast.BinaryAdd, ast.NewInteger("2"), ast.NewVariable("x")),
			exp:   statusFailed,
			err:   `exec: could not find jsonpath variable "x"`,
			isErr: ErrExecution,
		},
		{
			name:  "too_many_left",
			node:  ast.NewBinary(ast.BinaryAdd, ast.NewConst(ast.ConstRoot), ast.NewInteger("2")),
			value: []any{int64(4), int64(4)},
			exp:   statusFailed,
			err:   `exec: left operand of jsonpath operator + is not a single numeric value`,
			isErr: ErrExecution,
		},
		{
			name:  "too_many_right",
			node:  ast.NewBinary(ast.BinaryAdd, ast.NewInteger("2"), ast.NewConst(ast.ConstRoot)),
			value: []any{int64(4), int64(4)},
			exp:   statusFailed,
			err:   `exec: right operand of jsonpath operator + is not a single numeric value`,
			isErr: ErrExecution,
		},
		{
			name:  "add_int",
			node:  ast.NewBinary(ast.BinaryAdd, ast.NewConst(ast.ConstRoot), ast.NewInteger("2")),
			value: int64(4),
			exp:   statusOK,
			find:  []any{int64(6)},
		},
		{
			name:  "mul_float",
			node:  ast.NewBinary(ast.BinaryMul, ast.NewConst(ast.ConstRoot), ast.NewInteger("2")),
			value: float64(2.2),
			exp:   statusOK,
			find:  []any{float64(4.4)},
		},
		{
			name:  "invalid_operand",
			node:  ast.NewBinary(ast.BinaryAdd, ast.NewConst(ast.ConstRoot), ast.NewString("hi")),
			value: int64(4),
			exp:   statusFailed,
			err:   `exec: right operand of jsonpath operator + is not a single numeric value`,
			isErr: ErrExecution,
		},
		{
			name: "add_int_next",
			node: ast.LinkNodes([]ast.Node{
				ast.NewBinary(ast.BinaryAdd, ast.NewConst(ast.ConstRoot), ast.NewInteger("2")),
				ast.NewMethod(ast.MethodString),
			}),
			value: int64(4),
			exp:   statusOK,
			find:  []any{"6"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Make sure we have a binary node.
			node, ok := tc.node.(*ast.BinaryNode)
			r.True(ok)

			// Set up an executor.
			e := newTestExecutor(path, nil, true, false)
			e.root = tc.value

			// Test execKeyNode with a list.
			list := newList()
			res, err := e.execBinaryMathExpr(ctx, node, tc.value, list)
			a.Equal(tc.exp, res)

			// Check the error and list.
			if tc.isErr == nil {
				r.NoError(err)
				a.Equal(tc.find, list.list)
			} else {
				r.EqualError(err, tc.err)
				r.ErrorIs(err, tc.isErr)
				a.Empty(list.list)
			}

			// Try with nil found.
			res, err = e.execBinaryMathExpr(ctx, node, tc.value, nil)
			a.Equal(tc.exp, res)
			if tc.isErr == nil {
				r.NoError(err)
			} else {
				r.EqualError(err, tc.err)
				r.ErrorIs(err, tc.isErr)
			}
		})
	}
}

func TestExecMathOp(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	r := require.New(t)

	for _, tc := range []struct {
		name  string
		left  any
		right any
		op    ast.BinaryOperator
		exp   any
		err   string
		isErr error
	}{
		{
			name:  "int_int_add",
			left:  int64(2),
			right: int64(5),
			op:    ast.BinaryAdd,
			exp:   int64(7),
		},
		{
			name:  "int_float_sub",
			left:  int64(7),
			right: float64(2),
			op:    ast.BinarySub,
			exp:   float64(5),
		},
		{
			name:  "int_json_float_mul",
			left:  int64(2),
			right: json.Number("5.2"),
			op:    ast.BinaryMul,
			exp:   float64(10.4),
		},
		{
			name:  "int_json_int_div",
			left:  int64(10),
			right: json.Number("5"),
			op:    ast.BinaryDiv,
			exp:   int64(2),
		},
		{
			name:  "int_json_bad",
			left:  int64(10),
			right: json.Number("hi"),
			op:    ast.BinaryDiv,
			err:   `exec: right operand of jsonpath operator / is not a single numeric value`,
			isErr: ErrVerbose,
		},
		{
			name:  "int_nan",
			left:  int64(10),
			right: "hi",
			op:    ast.BinaryMod,
			err:   `exec: right operand of jsonpath operator % is not a single numeric value`,
			isErr: ErrVerbose,
		},
		{
			name:  "float_int_sub",
			left:  float64(7.2),
			right: int64(2),
			op:    ast.BinarySub,
			exp:   float64(5.2),
		},
		{
			name:  "float_float_add",
			left:  float64(7.2),
			right: float64(1.6),
			op:    ast.BinaryAdd,
			exp:   float64(8.8),
		},
		{
			name:  "float_json_int_sub",
			left:  float64(7.2),
			right: json.Number("2"),
			op:    ast.BinarySub,
			exp:   float64(5.2),
		},
		{
			name:  "float_json_float_add",
			left:  float64(7.2),
			right: json.Number("1.6"),
			op:    ast.BinaryAdd,
			exp:   float64(8.8),
		},
		{
			name:  "float_json_bad",
			left:  float64(10),
			right: json.Number("hi"),
			op:    ast.BinaryMul,
			err:   `exec: right operand of jsonpath operator * is not a single numeric value`,
			isErr: ErrVerbose,
		},
		{
			name:  "float_nan",
			left:  float64(10),
			right: "hi",
			op:    ast.BinaryMod,
			err:   `exec: right operand of jsonpath operator % is not a single numeric value`,
			isErr: ErrVerbose,
		},
		{
			name:  "json_int_int_add",
			left:  json.Number("2"),
			right: int64(5),
			op:    ast.BinaryAdd,
			exp:   int64(7),
		},
		{
			name:  "json_int_float_sub",
			left:  json.Number("10"),
			right: float64(2.2),
			op:    ast.BinarySub,
			exp:   float64(7.8),
		},
		{
			name:  "json_float_int_add",
			left:  json.Number("2.2"),
			right: int64(5),
			op:    ast.BinaryAdd,
			exp:   float64(7.2),
		},
		{
			name:  "json_float_float_sub",
			left:  json.Number("10.4"),
			right: float64(2.2),
			op:    ast.BinarySub,
			exp:   float64(8.2),
		},
		{
			name:  "json_bad",
			left:  json.Number("hi"),
			op:    ast.BinaryMul,
			err:   `exec: left operand of jsonpath operator * is not a single numeric value`,
			isErr: ErrVerbose,
		},
		{
			name:  "bad_left",
			left:  "hi",
			op:    ast.BinaryAdd,
			err:   `exec: left operand of jsonpath operator + is not a single numeric value`,
			isErr: ErrVerbose,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			res, err := execMathOp(tc.left, tc.right, tc.op)
			a.Equal(tc.exp, res)
			if tc.isErr == nil {
				r.NoError(err)
			} else {
				r.EqualError(err, tc.err)
				r.ErrorIs(err, tc.isErr)
			}
		})
	}
}
