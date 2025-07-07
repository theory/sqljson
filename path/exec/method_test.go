package exec

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/theory/sqljson/path/ast"
	"github.com/theory/sqljson/path/parser"
	"github.com/theory/sqljson/path/types"
)

func TestExecMethodNode(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	r := require.New(t)
	path, _ := parser.Parse("$")
	ctx := context.Background()

	// Offset of object in a slice is non-determinate, so calculate it at runtime.
	value := []any{map[string]any{"x": true, "y": "hi"}}
	offset := deltaBetween(value, value[0])

	for _, tc := range []struct {
		name   string
		node   ast.Node
		value  any
		unwrap bool
		exp    resultStatus
		find   []any
		err    string
		isErr  error
	}{
		{
			name:  "number",
			node:  ast.NewMethod(ast.MethodNumber),
			value: "42",
			exp:   statusOK,
			find:  []any{float64(42)},
		},
		{
			name:   "number_unwrap",
			node:   ast.NewMethod(ast.MethodNumber),
			value:  []any{"42", "98.6"},
			exp:    statusOK,
			unwrap: true,
			find:   []any{float64(42), float64(98.6)},
		},
		{
			name:  "number_no_unwrap",
			node:  ast.NewMethod(ast.MethodNumber),
			value: []any{"42", "98.6"},
			exp:   statusFailed,
			err:   `exec: jsonpath item method .number() can only be applied to a string or numeric value`,
			isErr: ErrVerbose,
		},
		{
			name:  "number_next",
			node:  ast.LinkNodes([]ast.Node{ast.NewMethod(ast.MethodNumber), ast.NewMethod(ast.MethodString)}),
			value: "42",
			exp:   statusOK,
			find:  []any{"42"},
		},
		{
			name:  "abs",
			node:  ast.NewMethod(ast.MethodAbs),
			value: int64(-42),
			exp:   statusOK,
			find:  []any{int64(42)},
		},
		{
			name:   "abs_unwrap",
			node:   ast.NewMethod(ast.MethodAbs),
			value:  []any{int64(-42), float64(98.6)},
			unwrap: true,
			exp:    statusOK,
			find:   []any{int64(42), float64(98.6)},
		},
		{
			name:  "abs_no_unwrap",
			node:  ast.NewMethod(ast.MethodAbs),
			value: []any{int64(-42), float64(98.6)},
			exp:   statusFailed,
			err:   `exec: jsonpath item method .abs() can only be applied to a numeric value`,
			isErr: ErrVerbose,
		},
		{
			name:  "floor",
			node:  ast.NewMethod(ast.MethodFloor),
			value: float64(42.8),
			exp:   statusOK,
			find:  []any{float64(42)},
		},
		{
			name:   "floor_unwrap",
			node:   ast.NewMethod(ast.MethodFloor),
			value:  []any{float64(42.8), float64(99.1)},
			unwrap: true,
			exp:    statusOK,
			find:   []any{float64(42), float64(99)},
		},
		{
			name:  "floor_no_unwrap",
			node:  ast.NewMethod(ast.MethodFloor),
			value: []any{float64(42.8), float64(99.1)},
			exp:   statusFailed,
			err:   `exec: jsonpath item method .floor() can only be applied to a numeric value`,
			isErr: ErrVerbose,
		},
		{
			name:  "ceiling",
			node:  ast.NewMethod(ast.MethodCeiling),
			value: float64(41.2),
			exp:   statusOK,
			find:  []any{float64(42)},
		},
		{
			name:   "ceiling_unwrap",
			node:   ast.NewMethod(ast.MethodCeiling),
			value:  []any{float64(41.2), float64(98.6)},
			unwrap: true,
			exp:    statusOK,
			find:   []any{float64(42), float64(99)},
		},
		{
			name:  "ceiling_no_unwrap",
			node:  ast.NewMethod(ast.MethodCeiling),
			value: []any{float64(41.2), float64(98.6)},
			exp:   statusFailed,
			err:   `exec: jsonpath item method .ceiling() can only be applied to a numeric value`,
			isErr: ErrVerbose,
		},
		{
			name:  "type",
			node:  ast.NewMethod(ast.MethodType),
			value: types.NewDate(time.Now()),
			exp:   statusOK,
			find:  []any{"date"},
		},
		{
			name:   "type_does_not_unwrap",
			node:   ast.NewMethod(ast.MethodType),
			value:  []any{"hi", types.NewDate(time.Now())},
			unwrap: true,
			exp:    statusOK,
			find:   []any{"array"},
		},
		{
			name:  "type_no_unwrap",
			node:  ast.NewMethod(ast.MethodType),
			value: []any{"hi", types.NewDate(time.Now())},
			exp:   statusOK,
			find:  []any{"array"},
		},
		{
			name:  "size",
			node:  ast.NewMethod(ast.MethodSize),
			value: []any{true, false},
			exp:   statusOK,
			find:  []any{int64(2)},
		},
		{
			name:  "size_not_array",
			node:  ast.NewMethod(ast.MethodSize),
			value: "xxx",
			exp:   statusOK,
			find:  []any{int64(1)},
		},
		{
			name:  "double",
			node:  ast.NewMethod(ast.MethodDouble),
			value: "42",
			exp:   statusOK,
			find:  []any{float64(42)},
		},
		{
			name:   "double_unwrap",
			node:   ast.NewMethod(ast.MethodDouble),
			value:  []any{"42", int64(2), float64(98.6)},
			unwrap: true,
			exp:    statusOK,
			find:   []any{float64(42), float64(2), float64(98.6)},
		},
		{
			name:  "double_no_unwrap",
			node:  ast.NewMethod(ast.MethodDouble),
			value: []any{"42", int64(2), float64(98.6)},
			exp:   statusFailed,
			err:   `exec: jsonpath item method .double() can only be applied to a string or numeric value`,
			isErr: ErrVerbose,
		},
		{
			name:  "integer",
			node:  ast.NewMethod(ast.MethodInteger),
			value: "42",
			exp:   statusOK,
			find:  []any{int64(42)},
		},
		{
			name:   "integer_unwrap",
			node:   ast.NewMethod(ast.MethodInteger),
			value:  []any{"42", int64(2)},
			exp:    statusOK,
			unwrap: true,
			find:   []any{int64(42), int64(2)},
		},
		{
			name:  "integer_no_unwrap",
			node:  ast.NewMethod(ast.MethodInteger),
			value: []any{"42", int64(2)},
			exp:   statusFailed,
			err:   `exec: jsonpath item method .integer() can only be applied to a string or numeric value`,
			isErr: ErrVerbose,
		},
		{
			name:  "bigint",
			node:  ast.NewMethod(ast.MethodBigInt),
			value: "42",
			exp:   statusOK,
			find:  []any{int64(42)},
		},
		{
			name:   "bigint_unwrap",
			node:   ast.NewMethod(ast.MethodBigInt),
			value:  []any{"42", int64(2)},
			exp:    statusOK,
			unwrap: true,
			find:   []any{int64(42), int64(2)},
		},
		{
			name:  "bigint_no_unwrap",
			node:  ast.NewMethod(ast.MethodBigInt),
			value: []any{"42", int64(2)},
			exp:   statusFailed,
			err:   `exec: jsonpath item method .bigint() can only be applied to a string or numeric value`,
			isErr: ErrVerbose,
		},
		{
			name:  "string",
			node:  ast.NewMethod(ast.MethodString),
			value: true,
			exp:   statusOK,
			find:  []any{"true"},
		},
		{
			// https://www.postgresql.org/message-id/A64AE04F-4410-42B7-A141-7A7349260F4D@justatheory.com
			name:   "string_does_not_unwrap",
			node:   ast.NewMethod(ast.MethodString),
			value:  []any{true, int64(42)},
			unwrap: true,
			exp:    statusOK,
			find:   []any{"true", "42"},
		},
		{
			name:  "string_no_unwrap",
			node:  ast.NewMethod(ast.MethodString),
			value: []any{true, int64(42)},
			exp:   statusFailed,
			err:   `exec: jsonpath item method .string() can only be applied to a boolean, string, numeric, or datetime value`,
			isErr: ErrVerbose,
		},
		{
			name:  "boolean",
			node:  ast.NewMethod(ast.MethodBoolean),
			value: "t",
			exp:   statusOK,
			find:  []any{true},
		},
		{
			name:   "boolean_unwrap",
			node:   ast.NewMethod(ast.MethodBoolean),
			value:  []any{"t", "n"},
			unwrap: true,
			exp:    statusOK,
			find:   []any{true, false},
		},
		{
			name:  "boolean_no_unwrap",
			node:  ast.NewMethod(ast.MethodBoolean),
			value: []any{"t", "n"},
			exp:   statusFailed,
			err:   `exec: jsonpath item method .boolean() can only be applied to a boolean, string, or numeric value`,
			isErr: ErrVerbose,
		},
		{
			name:  "keyvalue",
			node:  ast.NewMethod(ast.MethodKeyValue),
			value: map[string]any{"x": "hi"},
			exp:   statusOK,
			find:  []any{map[string]any{"id": int64(0), "key": "x", "value": "hi"}},
		},
		{
			name:   "keyvalue_wrap",
			node:   ast.NewMethod(ast.MethodKeyValue),
			value:  value,
			unwrap: true,
			exp:    statusOK,
			find: []any{
				map[string]any{"id": offset, "key": "x", "value": true},
				map[string]any{"id": offset, "key": "y", "value": "hi"},
			},
		},
		{
			name:  "keyvalue_no_wrap",
			node:  ast.NewMethod(ast.MethodKeyValue),
			value: value,
			exp:   statusFailed,
			err:   `exec: jsonpath item method .keyvalue() can only be applied to an object`,
			isErr: ErrVerbose,
		},
		{
			name:  "unknown_method",
			node:  ast.NewMethod(ast.MethodName(-1)),
			value: struct{}{},
			exp:   statusFailed,
			err:   `exec invalid: unknown method MethodName(-1)`,
			isErr: ErrInvalid,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Make sure we have a method node.
			node, ok := tc.node.(*ast.MethodNode)
			r.True(ok)

			// Set up an executor.
			e := newTestExecutor(path, nil, true, false)
			e.root = tc.value
			_ = e.setTempBaseObject(e.root, 0)

			// Test execKeyNode with a list.
			list := newList()
			res, err := e.execMethodNode(ctx, node, tc.value, list, tc.unwrap)
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
			res, err = e.execMethodNode(ctx, node, tc.value, nil, tc.unwrap)
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

type methodTestCase struct {
	name   string
	path   *ast.AST
	silent bool
	node   ast.Node
	value  any
	unwrap bool
	exp    resultStatus
	find   []any
	err    string
	isErr  error
}

func (tc methodTestCase) checkResults(t *testing.T, res resultStatus, found *valueList, err error) {
	t.Helper()
	a := assert.New(t)
	r := require.New(t)

	a.Equal(tc.exp, res)
	if tc.isErr == nil {
		r.NoError(err)
		a.Equal(tc.find, found.list)
	} else {
		r.EqualError(err, tc.err)
		r.ErrorIs(err, tc.isErr)
		a.Empty(found.list)
	}
}

//nolint:gochecknoglobals
var (
	laxRootPath, _    = parser.Parse("$")
	strictRootPath, _ = parser.Parse("strict $")
)

func (tc methodTestCase) prep() (*Executor, *valueList) {
	if tc.path == nil {
		tc.path = laxRootPath
	}
	return newTestExecutor(tc.path, nil, !tc.silent, false), newList()
}

func (tc methodTestCase) checkNode(t *testing.T, ok bool, meth *ast.MethodNode, name ast.MethodName) {
	t.Helper()
	assert.True(t, ok)
	assert.Equal(t, name, meth.Name())
}

func TestExecMethodType(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	meth := ast.NewMethod(ast.MethodType)

	for _, tc := range []methodTestCase{
		{
			name:  "object",
			node:  meth,
			value: map[string]any{},
			exp:   statusOK,
			find:  []any{"object"},
		},
		{
			name:  "object_next",
			node:  ast.LinkNodes([]ast.Node{ast.NewMethod(ast.MethodType), ast.NewMethod(ast.MethodSize)}),
			value: map[string]any{},
			exp:   statusOK,
			find:  []any{int64(1)},
		},
		{
			name:  "array",
			node:  meth,
			value: []any{},
			exp:   statusOK,
			find:  []any{"array"},
		},
		{
			name:  "string",
			node:  meth,
			value: "hi",
			exp:   statusOK,
			find:  []any{"string"},
		},
		{
			name:  "int_number",
			node:  meth,
			value: int64(1),
			exp:   statusOK,
			find:  []any{"number"},
		},
		{
			name:  "float_number",
			node:  meth,
			value: float64(1),
			exp:   statusOK,
			find:  []any{"number"},
		},
		{
			name:  "json_number",
			node:  meth,
			value: json.Number("1"),
			exp:   statusOK,
			find:  []any{"number"},
		},
		{
			name:  "bool",
			node:  meth,
			value: true,
			exp:   statusOK,
			find:  []any{"boolean"},
		},
		{
			name:  "date",
			node:  meth,
			value: types.NewDate(time.Now()),
			exp:   statusOK,
			find:  []any{"date"},
		},
		{
			name:  "time",
			node:  meth,
			value: types.NewTime(time.Now()),
			exp:   statusOK,
			find:  []any{"time without time zone"},
		},
		{
			name:  "timetz",
			node:  meth,
			value: types.NewTimeTZ(time.Now()),
			exp:   statusOK,
			find:  []any{"time with time zone"},
		},
		{
			name:  "timestamp",
			node:  meth,
			value: types.NewTimestamp(time.Now()),
			exp:   statusOK,
			find:  []any{"timestamp without time zone"},
		},
		{
			name:  "timestampTZ",
			node:  meth,
			value: types.NewTimestampTZ(context.Background(), time.Now()),
			exp:   statusOK,
			find:  []any{"timestamp with time zone"},
		},
		{
			name:  "nil",
			node:  meth,
			value: nil,
			exp:   statusOK,
			find:  []any{"null"},
		},
		{
			name:  "struct",
			node:  meth,
			value: struct{}{},
			exp:   statusFailed,
			err:   `exec invalid: unsupported data type struct {}`,
			isErr: ErrInvalid,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Make sure we have a .type() node.
			node, ok := tc.node.(*ast.MethodNode)
			tc.checkNode(t, ok, node, ast.MethodType)

			// Test execMethodType
			e, list := tc.prep()
			res, err := e.execMethodType(ctx, node, tc.value, list)
			tc.checkResults(t, res, list, err)
		})
	}
}

func TestExecMethodSize(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	meth := ast.NewMethod(ast.MethodSize)

	for _, tc := range []methodTestCase{
		{
			name:  "array_size_2",
			node:  meth,
			value: []any{1, 3},
			exp:   statusOK,
			find:  []any{int64(2)},
		},
		{
			name:  "array_size_6",
			node:  meth,
			value: []any{1, 3, 2, 4, 6, 8},
			exp:   statusOK,
			find:  []any{int64(6)},
		},
		{
			name:  "bool",
			node:  meth,
			value: true,
			exp:   statusOK,
			find:  []any{int64(1)},
		},
		{
			name:  "nil",
			node:  meth,
			value: nil,
			exp:   statusOK,
			find:  []any{int64(1)},
		},
		{
			name:  "object",
			node:  meth,
			value: map[string]any{"x": true, "y": false},
			exp:   statusOK,
			find:  []any{int64(1)},
		},
		{
			name:  "strict_not_array",
			path:  strictRootPath,
			node:  meth,
			value: true,
			exp:   statusFailed,
			err:   `exec: jsonpath item method .size() can only be applied to an array`,
			isErr: ErrVerbose,
		},
		{
			name:   "strict_not_array_silent",
			node:   meth,
			value:  true,
			silent: true,
			exp:    statusOK,
			find:   []any{int64(1)},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Make sure we have a .size() node.
			node, ok := tc.node.(*ast.MethodNode)
			tc.checkNode(t, ok, node, ast.MethodSize)

			// Test execMethodSize
			e, list := tc.prep()
			res, err := e.execMethodSize(ctx, node, tc.value, list)
			tc.checkResults(t, res, list, err)
		})
	}
}

func TestExecMethodDouble(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	meth := ast.NewMethod(ast.MethodDouble)

	for _, tc := range []methodTestCase{
		{
			name:   "array_unwrap",
			node:   meth,
			value:  []any{"1", "3.2"},
			unwrap: true,
			exp:    statusOK,
			find:   []any{float64(1), float64(3.2)},
		},
		{
			name:  "array_no_unwrap",
			node:  meth,
			value: []any{"1", "3.2"},
			exp:   statusFailed,
			err:   `exec: jsonpath item method .double() can only be applied to a string or numeric value`,
			isErr: ErrVerbose,
		},
		{
			name:  "int",
			node:  meth,
			value: int64(42),
			exp:   statusOK,
			find:  []any{float64(42)},
		},
		{
			name:  "max_int",
			node:  meth,
			value: int64(math.MaxInt64),
			exp:   statusOK,
			find:  []any{float64(math.MaxInt64)},
		},
		{
			name:  "min_int",
			node:  meth,
			value: int64(math.MinInt64),
			exp:   statusOK,
			find:  []any{float64(math.MinInt64)},
		},
		{
			name:  "float",
			node:  meth,
			value: float64(98.6),
			exp:   statusOK,
			find:  []any{float64(98.6)},
		},
		{
			name:  "max_float",
			node:  meth,
			value: float64(math.MaxFloat64),
			exp:   statusOK,
			find:  []any{float64(math.MaxFloat64)},
		},
		{
			name:  "min_float",
			node:  meth,
			value: float64(math.SmallestNonzeroFloat64),
			exp:   statusOK,
			find:  []any{float64(math.SmallestNonzeroFloat64)},
		},
		{
			name:  "json",
			node:  meth,
			value: json.Number("98.6"),
			exp:   statusOK,
			find:  []any{float64(98.6)},
		},
		{
			name:  "json_invalid",
			node:  meth,
			value: json.Number("hi"),
			exp:   statusFailed,
			err:   `exec: argument "hi" of jsonpath item method .double() is invalid for type double precision`,
			isErr: ErrExecution,
		},
		{
			name:  "string",
			node:  meth,
			value: "98.6",
			exp:   statusOK,
			find:  []any{float64(98.6)},
		},
		{
			name:  "string_invalid",
			node:  meth,
			value: "hi",
			exp:   statusFailed,
			err:   `exec: argument "hi" of jsonpath item method .double() is invalid for type double precision`,
			isErr: ErrExecution,
		},
		{
			name:  "bool",
			node:  meth,
			value: true,
			exp:   statusFailed,
			err:   `exec: jsonpath item method .double() can only be applied to a string or numeric value`,
			isErr: ErrExecution,
		},
		{
			name:  "inf",
			node:  meth,
			value: "inf",
			exp:   statusFailed,
			err:   `exec: NaN or Infinity is not allowed for jsonpath item method .double()`,
			isErr: ErrVerbose,
		},
		{
			name:  "neg_inf",
			node:  meth,
			value: "-inf",
			exp:   statusFailed,
			err:   `exec: NaN or Infinity is not allowed for jsonpath item method .double()`,
			isErr: ErrVerbose,
		},
		{
			name:  "nan",
			node:  meth,
			value: "nan",
			exp:   statusFailed,
			err:   `exec: NaN or Infinity is not allowed for jsonpath item method .double()`,
			isErr: ErrVerbose,
		},
		{
			name:  "json_next",
			node:  ast.LinkNodes([]ast.Node{ast.NewMethod(ast.MethodDouble), ast.NewMethod(ast.MethodString)}),
			value: json.Number("98.6"),
			exp:   statusOK,
			find:  []any{"98.6"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Make sure we have a .double() node.
			node, ok := tc.node.(*ast.MethodNode)
			tc.checkNode(t, ok, node, ast.MethodDouble)

			// Test execMethodDouble
			e, list := tc.prep()
			res, err := e.execMethodDouble(ctx, node, tc.value, list, tc.unwrap)
			tc.checkResults(t, res, list, err)
		})
	}
}

func TestExecMethodInteger(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	meth := ast.NewMethod(ast.MethodInteger)

	for _, tc := range []methodTestCase{
		{
			name:  "int",
			node:  meth,
			value: int64(42),
			exp:   statusOK,
			find:  []any{int64(42)},
		},
		{
			name:  "max_int",
			node:  meth,
			value: int64(math.MaxInt32),
			exp:   statusOK,
			find:  []any{int64(math.MaxInt32)},
		},
		{
			name:  "min_int",
			node:  meth,
			value: int64(math.MinInt32),
			exp:   statusOK,
			find:  []any{int64(math.MinInt32)},
		},
		{
			name:  "over_max_int",
			node:  meth,
			value: int64(math.MaxInt32 + 1),
			exp:   statusFailed,
			err: fmt.Sprintf(
				`exec: argument "%v" of jsonpath item method .integer() is invalid for type integer`,
				int64(math.MaxInt32+1),
			),
			isErr: ErrVerbose,
		},
		{
			name:  "under_min_int",
			node:  meth,
			value: int64(math.MinInt32 - 1),
			exp:   statusFailed,
			err: fmt.Sprintf(
				`exec: argument "%v" of jsonpath item method .integer() is invalid for type integer`,
				int64(math.MinInt32-1),
			),
			isErr: ErrVerbose,
		},
		{
			name:  "float_round_up",
			node:  meth,
			value: float64(98.6),
			exp:   statusOK,
			find:  []any{int64(99)},
		},
		{
			name:  "float_round_down",
			node:  meth,
			value: float64(42.3),
			exp:   statusOK,
			find:  []any{int64(42)},
		},
		{
			name:  "json_number_int",
			node:  meth,
			value: json.Number("42"),
			exp:   statusOK,
			find:  []any{int64(42)},
		},
		{
			name:  "json_number_float_down",
			node:  meth,
			value: json.Number("42.3"),
			exp:   statusOK,
			find:  []any{int64(42)},
		},
		{
			name:  "json_number_float_up",
			node:  meth,
			value: json.Number("42.5"),
			exp:   statusOK,
			find:  []any{int64(43)},
		},
		{
			name:  "json_number_invalid",
			node:  meth,
			value: json.Number("hi"),
			exp:   statusFailed,
			err:   `exec: argument "hi" of jsonpath item method .integer() is invalid for type integer`,
			isErr: ErrVerbose,
		},
		{
			name:  "string",
			node:  meth,
			value: "42",
			exp:   statusOK,
			find:  []any{int64(42)},
		},
		{
			name:  "string_float",
			node:  meth,
			value: "42.3",
			exp:   statusFailed,
			err:   `exec: argument "42.3" of jsonpath item method .integer() is invalid for type integer`,
			isErr: ErrVerbose,
		},
		{
			name:  "invalid_string",
			node:  meth,
			value: "hi",
			exp:   statusFailed,
			err:   `exec: argument "hi" of jsonpath item method .integer() is invalid for type integer`,
			isErr: ErrVerbose,
		},
		{
			name:  "inf",
			node:  meth,
			value: "inf",
			exp:   statusFailed,
			err:   `exec: argument "inf" of jsonpath item method .integer() is invalid for type integer`,
			isErr: ErrVerbose,
		},
		{
			name:  "neg_inf",
			node:  meth,
			value: "-inf",
			exp:   statusFailed,
			err:   `exec: argument "-inf" of jsonpath item method .integer() is invalid for type integer`,
			isErr: ErrVerbose,
		},
		{
			name:  "nan",
			node:  meth,
			value: "nan",
			exp:   statusFailed,
			err:   `exec: argument "nan" of jsonpath item method .integer() is invalid for type integer`,
			isErr: ErrVerbose,
		},
		{
			name:  "int_next",
			node:  ast.LinkNodes([]ast.Node{ast.NewMethod(ast.MethodInteger), ast.NewMethod(ast.MethodString)}),
			value: int64(42),
			exp:   statusOK,
			find:  []any{"42"},
		},
		{
			name:  "invalid_value",
			node:  meth,
			value: true,
			exp:   statusFailed,
			err:   `exec: jsonpath item method .integer() can only be applied to a string or numeric value`,
			isErr: ErrVerbose,
		},
		{
			name:  "int_next",
			node:  ast.LinkNodes([]ast.Node{ast.NewMethod(ast.MethodInteger), ast.NewMethod(ast.MethodString)}),
			value: int64(42),
			exp:   statusOK,
			find:  []any{"42"},
		},
		{
			name:  "array",
			node:  meth,
			value: []any{int64(42)},
			exp:   statusFailed,
			err:   `exec: jsonpath item method .integer() can only be applied to a string or numeric value`,
			isErr: ErrVerbose,
		},
		{
			name:   "array_unwrap",
			node:   meth,
			value:  []any{float64(42.2), "88"},
			unwrap: true,
			exp:    statusOK,
			find:   []any{int64(42), int64(88)},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Make sure we have a .Integer() node.
			node, ok := tc.node.(*ast.MethodNode)
			tc.checkNode(t, ok, node, ast.MethodInteger)

			// Test execMethodInteger
			e, list := tc.prep()
			res, err := e.execMethodInteger(ctx, node, tc.value, list, tc.unwrap)
			tc.checkResults(t, res, list, err)
		})
	}
}

func TestExecMethodBigInt(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	meth := ast.NewMethod(ast.MethodBigInt)

	for _, tc := range []methodTestCase{
		{
			name:  "int",
			node:  meth,
			value: int64(42),
			exp:   statusOK,
			find:  []any{int64(42)},
		},
		{
			name:  "max_int",
			node:  meth,
			value: int64(math.MaxInt64),
			exp:   statusOK,
			find:  []any{int64(math.MaxInt64)},
		},
		{
			name:  "min_int",
			node:  meth,
			value: int64(math.MinInt64),
			exp:   statusOK,
			find:  []any{int64(math.MinInt64)},
		},
		{
			name:  "float_up",
			node:  meth,
			value: float64(98.6),
			exp:   statusOK,
			find:  []any{int64(99)},
		},
		{
			name:  "float_down",
			node:  meth,
			value: float64(98.4),
			exp:   statusOK,
			find:  []any{int64(98)},
		},
		{
			name:  "float_upper_bound",
			node:  meth,
			value: float64(math.MaxUint64),
			exp:   statusFailed,
			err: fmt.Sprintf(
				`exec: argument "%v" of jsonpath item method .bigint() is invalid for type bigint`,
				float64(math.MaxUint64),
			),
			isErr: ErrVerbose,
		},
		{
			name:  "float_lower_bound",
			node:  meth,
			value: float64(-math.MaxUint64),
			exp:   statusFailed,
			err: fmt.Sprintf(
				`exec: argument "%v" of jsonpath item method .bigint() is invalid for type bigint`,
				float64(-math.MaxUint64),
			),
			isErr: ErrVerbose,
		},
		{
			name:  "json_int",
			node:  meth,
			value: json.Number("42"),
			exp:   statusOK,
			find:  []any{int64(42)},
		},
		{
			name:  "json_float_down",
			node:  meth,
			value: json.Number("-42.3"),
			exp:   statusOK,
			find:  []any{int64(-42)},
		},
		{
			name:  "json_float_up",
			node:  meth,
			value: json.Number("98.6"),
			exp:   statusOK,
			find:  []any{int64(99)},
		},
		{
			name:  "json_float_upper_bound",
			node:  meth,
			value: json.Number("18446744073709551615.123"),
			exp:   statusFailed,
			err:   `exec: argument "18446744073709551615.123" of jsonpath item method .bigint() is invalid for type bigint`,
			isErr: ErrVerbose,
		},
		{
			name:  "json_float_lower_bound",
			node:  meth,
			value: json.Number("-18446744073709551615.123"),
			exp:   statusFailed,
			err:   `exec: argument "-18446744073709551615.123" of jsonpath item method .bigint() is invalid for type bigint`,
			isErr: ErrVerbose,
		},
		{
			name:  "invalid_json",
			node:  meth,
			value: json.Number("hi"),
			exp:   statusFailed,
			err:   `exec: argument "hi" of jsonpath item method .bigint() is invalid for type bigint`,
			isErr: ErrVerbose,
		},
		{
			name:  "string_int",
			node:  meth,
			value: "42",
			exp:   statusOK,
			find:  []any{int64(42)},
		},
		{
			name:  "string_max_big_int",
			node:  meth,
			value: strconv.FormatInt(math.MaxInt64, 10),
			exp:   statusOK,
			find:  []any{int64(math.MaxInt64)},
		},
		{
			name:  "string_min_big_int",
			node:  meth,
			value: strconv.FormatInt(math.MinInt64, 10),
			exp:   statusOK,
			find:  []any{int64(math.MinInt64)},
		},
		{
			name:  "string_float",
			node:  meth,
			value: "42.8",
			exp:   statusFailed,
			err:   `exec: argument "42.8" of jsonpath item method .bigint() is invalid for type bigint`,
			isErr: ErrVerbose,
		},
		{
			name:  "invalid_string",
			node:  meth,
			value: "hi",
			exp:   statusFailed,
			err:   `exec: argument "hi" of jsonpath item method .bigint() is invalid for type bigint`,
			isErr: ErrVerbose,
		},
		{
			name:  "inf",
			node:  meth,
			value: "inf",
			exp:   statusFailed,
			err:   `exec: argument "inf" of jsonpath item method .bigint() is invalid for type bigint`,
			isErr: ErrVerbose,
		},
		{
			name:  "neg_inf",
			node:  meth,
			value: "-inf",
			exp:   statusFailed,
			err:   `exec: argument "-inf" of jsonpath item method .bigint() is invalid for type bigint`,
			isErr: ErrVerbose,
		},
		{
			name:  "nan",
			node:  meth,
			value: "nan",
			exp:   statusFailed,
			err:   `exec: argument "nan" of jsonpath item method .bigint() is invalid for type bigint`,
			isErr: ErrVerbose,
		},
		{
			name:  "int_next",
			node:  ast.LinkNodes([]ast.Node{ast.NewMethod(ast.MethodBigInt), ast.NewMethod(ast.MethodString)}),
			value: int64(42),
			exp:   statusOK,
			find:  []any{"42"},
		},
		{
			name:  "invalid_value",
			node:  meth,
			value: true,
			exp:   statusFailed,
			err:   `exec: jsonpath item method .bigint() can only be applied to a string or numeric value`,
			isErr: ErrVerbose,
		},
		{
			name:  "array",
			node:  meth,
			value: []any{int64(42)},
			exp:   statusFailed,
			err:   `exec: jsonpath item method .bigint() can only be applied to a string or numeric value`,
			isErr: ErrVerbose,
		},
		{
			name:   "array_unwrap",
			node:   meth,
			value:  []any{int64(42), "1024"},
			unwrap: true,
			exp:    statusOK,
			find:   []any{int64(42), int64(1024)},
		},
		{
			name:   "array_unwrap_next",
			node:   ast.LinkNodes([]ast.Node{ast.NewMethod(ast.MethodBigInt), ast.NewMethod(ast.MethodString)}),
			value:  []any{int64(42), "1024"},
			unwrap: true,
			exp:    statusOK,
			find:   []any{"42", "1024"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Make sure we have a .BigInt() node.
			node, ok := tc.node.(*ast.MethodNode)
			tc.checkNode(t, ok, node, ast.MethodBigInt)

			// Test execMethodBigInt
			e, list := tc.prep()
			res, err := e.execMethodBigInt(ctx, node, tc.value, list, tc.unwrap)
			tc.checkResults(t, res, list, err)
		})
	}
}

func TestExecMethodString(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	meth := ast.NewMethod(ast.MethodString)
	now := time.Now()

	for _, tc := range []methodTestCase{
		{
			name:  "string",
			node:  meth,
			value: "hi",
			exp:   statusOK,
			find:  []any{"hi"},
		},
		{
			name:  "date",
			node:  meth,
			value: types.NewDate(now),
			exp:   statusOK,
			find:  []any{types.NewDate(now).String()},
		},
		{
			name:  "time",
			node:  meth,
			value: types.NewTime(now),
			exp:   statusOK,
			find:  []any{types.NewTime(now).String()},
		},
		{
			name:  "timetz",
			node:  meth,
			value: types.NewTimeTZ(now),
			exp:   statusOK,
			find:  []any{types.NewTimeTZ(now).String()},
		},
		{
			name:  "timestamp",
			node:  meth,
			value: types.NewTimestamp(now),
			exp:   statusOK,
			find:  []any{types.NewTimestamp(now).String()},
		},
		{
			name:  "timestamptz",
			node:  meth,
			value: types.NewTimestampTZ(ctx, now),
			exp:   statusOK,
			find:  []any{types.NewTimestampTZ(ctx, now).String()},
		},
		{
			name:  "stringer_json_number",
			node:  meth,
			value: json.Number("188.2"),
			exp:   statusOK,
			find:  []any{"188.2"},
		},
		{
			name:  "int",
			node:  meth,
			value: int64(42),
			exp:   statusOK,
			find:  []any{"42"},
		},
		{
			name:  "float",
			node:  meth,
			value: float64(98.6),
			exp:   statusOK,
			find:  []any{"98.6"},
		},
		{
			name:  "true",
			node:  meth,
			value: true,
			exp:   statusOK,
			find:  []any{"true"},
		},
		{
			name:  "false",
			node:  meth,
			value: false,
			exp:   statusOK,
			find:  []any{"false"},
		},
		{
			name:  "nil",
			node:  meth,
			value: nil,
			exp:   statusFailed,
			err:   `exec: jsonpath item method .string() can only be applied to a boolean, string, numeric, or datetime value`,
			isErr: ErrVerbose,
		},
		{
			name:  "obj",
			node:  meth,
			value: map[string]any{},
			exp:   statusFailed,
			err:   `exec: jsonpath item method .string() can only be applied to a boolean, string, numeric, or datetime value`,
			isErr: ErrVerbose,
		},
		{
			name:  "array",
			node:  meth,
			value: []any{int64(42), true},
			exp:   statusFailed,
			err:   `exec: jsonpath item method .string() can only be applied to a boolean, string, numeric, or datetime value`,
			isErr: ErrVerbose,
		},
		{
			name:   "array_unwrap",
			node:   meth,
			value:  []any{int64(42), true},
			unwrap: true,
			exp:    statusOK,
			find:   []any{"42", "true"},
		},
		{
			name:  "string_next",
			node:  ast.LinkNodes([]ast.Node{ast.NewMethod(ast.MethodString), ast.NewMethod(ast.MethodInteger)}),
			value: "42",
			exp:   statusOK,
			find:  []any{int64(42)},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Make sure we have a .String() node.
			node, ok := tc.node.(*ast.MethodNode)
			tc.checkNode(t, ok, node, ast.MethodString)

			// Test execMethodString
			e, list := tc.prep()
			res, err := e.execMethodString(ctx, node, tc.value, list, tc.unwrap)
			tc.checkResults(t, res, list, err)
		})
	}
}

func TestExecMethodBoolean(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	meth := ast.NewMethod(ast.MethodBoolean)

	for _, tc := range []methodTestCase{
		{
			name:  "true",
			node:  meth,
			value: true,
			exp:   statusOK,
			find:  []any{true},
		},
		{
			name:  "false",
			node:  meth,
			value: false,
			exp:   statusOK,
			find:  []any{false},
		},
		{
			name:  "int1",
			node:  meth,
			value: int64(1),
			exp:   statusOK,
			find:  []any{true},
		},
		{
			name:  "int1000",
			node:  meth,
			value: int64(1000),
			exp:   statusOK,
			find:  []any{true},
		},
		{
			name:  "int_neg10",
			node:  meth,
			value: int64(-10),
			exp:   statusOK,
			find:  []any{true},
		},
		{
			name:  "int0",
			node:  meth,
			value: int64(0),
			exp:   statusOK,
			find:  []any{false},
		},
		{
			name:  "int_neg0",
			node:  meth,
			value: int64(-0),
			exp:   statusOK,
			find:  []any{false},
		},
		{
			name:  "float1",
			node:  meth,
			value: float64(1.0),
			exp:   statusOK,
			find:  []any{true},
		},
		{
			name:  "float1000",
			node:  meth,
			value: float64(1000.0),
			exp:   statusOK,
			find:  []any{true},
		},
		{
			name:  "float_neg0",
			node:  meth,
			value: float64(-10),
			exp:   statusOK,
			find:  []any{true},
		},
		{
			name:  "float0",
			node:  meth,
			value: float64(0),
			exp:   statusOK,
			find:  []any{false},
		},
		{
			name:  "float_dot_one",
			node:  meth,
			value: float64(1.1),
			exp:   statusFailed,
			err:   `exec: argument "1.1" of jsonpath item method .boolean() is invalid for type boolean`,
			isErr: ErrVerbose,
		},
		{
			name:  "float_dot_nine",
			node:  meth,
			value: float64(1.9),
			exp:   statusFailed,
			err:   `exec: argument "1.9" of jsonpath item method .boolean() is invalid for type boolean`,
			isErr: ErrVerbose,
		},
		{
			name:  "float_neg000_dot_nine",
			node:  meth,
			value: float64(-1000.9),
			exp:   statusFailed,
			err:   `exec: argument "-1000.9" of jsonpath item method .boolean() is invalid for type boolean`,
			isErr: ErrVerbose,
		},
		{
			name:  "json_int1",
			node:  meth,
			value: json.Number("1"),
			exp:   statusOK,
			find:  []any{true},
		},
		{
			name:  "json_int0",
			node:  meth,
			value: json.Number("0"),
			exp:   statusOK,
			find:  []any{false},
		},
		{
			name:  "json_int1_dot0",
			node:  meth,
			value: json.Number("1.0"),
			exp:   statusOK,
			find:  []any{true},
		},
		{
			name:  "json_int0_dot0",
			node:  meth,
			value: json.Number("0.0"),
			exp:   statusOK,
			find:  []any{false},
		},
		{
			name:  "json_float1000",
			node:  meth,
			value: json.Number("1000.0"),
			exp:   statusOK,
			find:  []any{true},
		},
		{
			name:  "json_float_neg10",
			node:  meth,
			value: json.Number("-10.0"),
			exp:   statusOK,
			find:  []any{true},
		},
		{
			name:  "json_float_0",
			node:  meth,
			value: json.Number("0.0"),
			exp:   statusOK,
			find:  []any{false},
		},
		{
			name:  "json_float_neg0",
			node:  meth,
			value: json.Number("-0.0"),
			exp:   statusOK,
			find:  []any{false},
		},
		{
			name:  "json_float_dot_one",
			node:  meth,
			value: json.Number("1.1"),
			exp:   statusFailed,
			err:   `exec: argument "1.1" of jsonpath item method .boolean() is invalid for type boolean`,
			isErr: ErrVerbose,
		},
		{
			name:  "float_dot_nine",
			node:  meth,
			value: json.Number("1.9"),
			exp:   statusFailed,
			err:   `exec: argument "1.9" of jsonpath item method .boolean() is invalid for type boolean`,
			isErr: ErrVerbose,
		},
		{
			name:  "json_float_neg_1000_dot_nine",
			node:  meth,
			value: json.Number("-1000.9"),
			exp:   statusFailed,
			err:   `exec: argument "-1000.9" of jsonpath item method .boolean() is invalid for type boolean`,
			isErr: ErrVerbose,
		},
		{
			name:  "string_t",
			node:  meth,
			value: "t",
			exp:   statusOK,
			find:  []any{true},
		},
		{
			name:  "string_f",
			node:  meth,
			value: "f",
			exp:   statusOK,
			find:  []any{false},
		},
		{
			name:  "string_y",
			node:  meth,
			value: "y",
			exp:   statusOK,
			find:  []any{true},
		},
		{
			name:  "string_n",
			node:  meth,
			value: "n",
			exp:   statusOK,
			find:  []any{false},
		},
		{
			name:  "invalid_string",
			node:  meth,
			value: "nope",
			exp:   statusFailed,
			err:   `exec: argument "nope" of jsonpath item method .boolean() is invalid for type boolean`,
			isErr: ErrVerbose,
		},
		{
			name:  "object",
			node:  meth,
			value: map[string]any{"x": true},
			exp:   statusFailed,
			err:   `exec: jsonpath item method .boolean() can only be applied to a boolean, string, or numeric value`,
			isErr: ErrVerbose,
		},
		{
			name:  "array",
			node:  meth,
			value: []any{true, false},
			exp:   statusFailed,
			err:   `exec: jsonpath item method .boolean() can only be applied to a boolean, string, or numeric value`,
			isErr: ErrVerbose,
		},
		{
			name:   "array_unwrap",
			node:   meth,
			value:  []any{true, false},
			unwrap: true,
			exp:    statusOK,
			find:   []any{true, false},
		},
		{
			name:  "bool_next",
			node:  ast.LinkNodes([]ast.Node{ast.NewMethod(ast.MethodBoolean), ast.NewMethod(ast.MethodString)}),
			value: true,
			exp:   statusOK,
			find:  []any{"true"},
		},
		{
			name:   "array_unwrap_next",
			node:   ast.LinkNodes([]ast.Node{ast.NewMethod(ast.MethodBoolean), ast.NewMethod(ast.MethodString)}),
			value:  []any{"t", "f"},
			unwrap: true,
			exp:    statusOK,
			find:   []any{"true", "false"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Make sure we have a .Boolean() node.
			node, ok := tc.node.(*ast.MethodNode)
			tc.checkNode(t, ok, node, ast.MethodBoolean)

			// Test execMethodBoolean
			e, list := tc.prep()
			res, err := e.execMethodBoolean(ctx, node, tc.value, list, tc.unwrap)
			tc.checkResults(t, res, list, err)
		})
	}
}

func TestExecBooleanString(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	r := require.New(t)

	for _, tc := range []struct {
		name  string
		val   string
		exp   bool
		err   string
		isErr error
	}{
		{
			name:  "empty_string",
			val:   "",
			err:   `exec: argument "" of jsonpath item method .boolean() is invalid for type boolean`,
			isErr: ErrVerbose,
		},
		{
			name: "t",
			val:  "t",
			exp:  true,
		},
		{
			name: "T",
			val:  "T",
			exp:  true,
		},
		{
			name: "true",
			val:  "true",
			exp:  true,
		},
		{
			name: "TRUE",
			val:  "TRUE",
			exp:  true,
		},
		{
			name: "TruE",
			val:  "TruE",
			exp:  true,
		},
		{
			name:  "tru",
			val:   "tru",
			err:   `exec: argument "tru" of jsonpath item method .boolean() is invalid for type boolean`,
			isErr: ErrVerbose,
		},
		{
			name: "f",
			val:  "f",
			exp:  false,
		},
		{
			name: "F",
			val:  "F",
			exp:  false,
		},
		{
			name: "false",
			val:  "false",
			exp:  false,
		},
		{
			name: "FALSE",
			val:  "FALSE",
			exp:  false,
		},
		{
			name: "FalSe",
			val:  "FalSe",
			exp:  false,
		},
		{
			name:  "fal",
			val:   "fal",
			err:   `exec: argument "fal" of jsonpath item method .boolean() is invalid for type boolean`,
			isErr: ErrVerbose,
		},
		{
			name: "y",
			val:  "y",
			exp:  true,
		},
		{
			name: "Y",
			val:  "Y",
			exp:  true,
		},
		{
			name: "yes",
			val:  "yes",
			exp:  true,
		},
		{
			name: "YES",
			val:  "YES",
			exp:  true,
		},
		{
			name: "Yes",
			val:  "Yes",
			exp:  true,
		},
		{
			name:  "ye",
			val:   "ye",
			err:   `exec: argument "ye" of jsonpath item method .boolean() is invalid for type boolean`,
			isErr: ErrVerbose,
		},
		{
			name: "n",
			val:  "n",
			exp:  false,
		},
		{
			name: "N",
			val:  "N",
			exp:  false,
		},
		{
			name: "no",
			val:  "no",
			exp:  false,
		},
		{
			name: "NO",
			val:  "NO",
			exp:  false,
		},
		{
			name:  "non",
			val:   "non",
			err:   `exec: argument "non" of jsonpath item method .boolean() is invalid for type boolean`,
			isErr: ErrVerbose,
		},
		{
			name: "on",
			val:  "on",
			exp:  true,
		},
		{
			name: "ON",
			val:  "ON",
			exp:  true,
		},
		{
			name: "oN",
			val:  "oN",
			exp:  true,
		},
		{
			name: "off",
			val:  "off",
			exp:  false,
		},
		{
			name: "OFF",
			val:  "OFF",
			exp:  false,
		},
		{
			name: "Off",
			val:  "Off",
			exp:  false,
		},
		{
			name:  "oof",
			val:   "oof",
			err:   `exec: argument "oof" of jsonpath item method .boolean() is invalid for type boolean`,
			isErr: ErrVerbose,
		},
		{
			name: "1",
			val:  "1",
			exp:  true,
		},
		{
			name: "0",
			val:  "0",
			exp:  false,
		},
		{
			name:  "1_space",
			val:   "1 ",
			err:   `exec: argument "1 " of jsonpath item method .boolean() is invalid for type boolean`,
			isErr: ErrVerbose,
		},
		{
			name:  "0_space",
			val:   "0 ",
			err:   `exec: argument "0 " of jsonpath item method .boolean() is invalid for type boolean`,
			isErr: ErrVerbose,
		},
		{
			name:  "t_space",
			val:   "t ",
			err:   `exec: argument "t " of jsonpath item method .boolean() is invalid for type boolean`,
			isErr: ErrVerbose,
		},
		{
			name:  "f_space",
			val:   " f",
			err:   `exec: argument " f" of jsonpath item method .boolean() is invalid for type boolean`,
			isErr: ErrVerbose,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			res, err := execBooleanString(tc.val, ast.MethodBoolean)
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

func TestExecuteNumberMethod(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	number := ast.NewMethod(ast.MethodNumber)
	decimal := ast.NewBinary(ast.BinaryDecimal, nil, nil)

	for _, tc := range []methodTestCase{
		{
			name:  "float",
			node:  number,
			value: float64(98.6),
			exp:   statusOK,
			find:  []any{float64(98.6)},
		},
		{
			name:  "int",
			node:  number,
			value: int64(42),
			exp:   statusOK,
			find:  []any{float64(42)},
		},
		{
			name:  "max_int",
			node:  number,
			value: int64(math.MaxInt64),
			exp:   statusOK,
			find:  []any{float64(math.MaxInt64)},
		},
		{
			name:  "min_int",
			node:  number,
			value: int64(math.MinInt64),
			exp:   statusOK,
			find:  []any{float64(math.MinInt64)},
		},
		{
			name:  "json_int",
			node:  number,
			value: json.Number("42"),
			exp:   statusOK,
			find:  []any{float64(42)},
		},
		{
			name:  "json_float",
			node:  number,
			value: json.Number("98.6"),
			exp:   statusOK,
			find:  []any{float64(98.6)},
		},
		{
			name:  "number_invalid_json",
			node:  number,
			value: json.Number("hi"),
			exp:   statusFailed,
			err:   `exec: argument "hi" of jsonpath item method .number() is invalid for type numeric`,
			isErr: ErrVerbose,
		},
		{
			name:  "invalid_json_decimal",
			node:  decimal,
			value: json.Number("hi"),
			exp:   statusFailed,
			err:   `exec: argument "hi" of jsonpath item method .decimal() is invalid for type numeric`,
			isErr: ErrVerbose,
		},
		{
			name:  "string_int",
			node:  number,
			value: "42",
			exp:   statusOK,
			find:  []any{float64(42)},
		},
		{
			name:  "string_float",
			node:  number,
			value: "98.6",
			exp:   statusOK,
			find:  []any{float64(98.6)},
		},
		{
			name:  "string_max_int",
			node:  number,
			value: strconv.FormatInt(math.MaxInt64, 10),
			exp:   statusOK,
			find:  []any{float64(math.MaxInt64)},
		},
		{
			name:  "string_max_float",
			node:  number,
			value: fmt.Sprintf("%v", math.MaxFloat64),
			exp:   statusOK,
			find:  []any{float64(math.MaxFloat64)},
		},
		{
			name:  "object_number",
			node:  number,
			value: map[string]any{"x": "42"},
			exp:   statusFailed,
			err:   `exec: jsonpath item method .number() can only be applied to a string or numeric value`,
			isErr: ErrVerbose,
		},
		{
			name:  "decimal_number",
			node:  decimal,
			value: map[string]any{"x": "42"},
			exp:   statusFailed,
			err:   `exec: jsonpath item method .decimal() can only be applied to a string or numeric value`,
			isErr: ErrVerbose,
		},
		{
			name:  "array",
			node:  number,
			value: []any{"42", float64(98.6)},
			exp:   statusFailed,
			err:   `exec: jsonpath item method .number() can only be applied to a string or numeric value`,
			isErr: ErrVerbose,
		},
		{
			name:   "array_unwrap",
			node:   number,
			value:  []any{"42", float64(98.6)},
			unwrap: true,
			exp:    statusOK,
			find:   []any{float64(42), float64(98.6)},
		},
		{
			name:  "inf",
			node:  number,
			value: "inf",
			exp:   statusFailed,
			err:   `exec: NaN or Infinity is not allowed for jsonpath item method .number()`,
			isErr: ErrVerbose,
		},
		{
			name:  "neg_inf",
			node:  number,
			value: "-inf",
			exp:   statusFailed,
			err:   `exec: NaN or Infinity is not allowed for jsonpath item method .number()`,
			isErr: ErrVerbose,
		},
		{
			name:  "nan",
			node:  number,
			value: "nan",
			exp:   statusFailed,
			err:   `exec: NaN or Infinity is not allowed for jsonpath item method .number()`,
			isErr: ErrVerbose,
		},
		{
			name:  "inf_decimal",
			node:  decimal,
			value: "inf",
			exp:   statusFailed,
			err:   `exec: NaN or Infinity is not allowed for jsonpath item method .decimal()`,
			isErr: ErrVerbose,
		},
		{
			name:  "float_decimal",
			node:  decimal,
			value: float64(98.6),
			exp:   statusOK,
			find:  []any{float64(98.6)},
		},
		{
			name:  "float_decimal_precision",
			node:  ast.NewBinary(ast.BinaryDecimal, ast.NewInteger("4"), nil),
			value: float64(12.2),
			exp:   statusOK,
			find:  []any{float64(12)},
		},
		{
			name:  "float_decimal_precision_scale",
			node:  ast.NewBinary(ast.BinaryDecimal, ast.NewInteger("4"), ast.NewInteger("2")),
			value: float64(12.233),
			exp:   statusOK,
			find:  []any{float64(12.23)},
		},
		{
			name:  "float_decimal_error",
			node:  ast.NewBinary(ast.BinaryDecimal, ast.NewInteger("3"), ast.NewInteger("2")),
			value: float64(12.233),
			exp:   statusFailed,
			err:   `exec: argument "12.233" of jsonpath item method .decimal() is invalid for type numeric`,
			isErr: ErrVerbose,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Determine the method.
			var meth any
			meth = tc.node
			if bin, ok := tc.node.(*ast.BinaryNode); ok {
				meth = bin.Operator()
			}

			// Test execMethodNumber
			e, list := tc.prep()
			res, err := e.executeNumberMethod(ctx, tc.node, tc.value, list, tc.unwrap, meth)
			tc.checkResults(t, res, list, err)
		})
	}
}

func TestExecuteDecimalMethod(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	r := require.New(t)

	for _, tc := range []struct {
		name  string
		node  *ast.BinaryNode
		value any
		num   float64
		exp   float64
		err   string
		isErr error
	}{
		{
			name: "not_decimal",
			node: ast.NewBinary(ast.BinaryAdd, nil, nil),
			num:  float64(98.6),
			exp:  float64(98.6),
		},
		{
			name: "no_args",
			node: ast.NewBinary(ast.BinaryDecimal, nil, nil),
			num:  float64(98.6),
			exp:  float64(98.6),
		},
		{
			name:  "invalid_precision",
			node:  ast.NewBinary(ast.BinaryDecimal, ast.NewString("hi"), nil),
			err:   `exec: invalid jsonpath item type for .decimal() precision`,
			isErr: ErrExecution,
		},
		{
			name:  "precision_zero",
			node:  ast.NewBinary(ast.BinaryDecimal, ast.NewInteger("0"), nil),
			err:   `exec: NUMERIC precision 0 must be between 1 and 1000`,
			isErr: ErrExecution,
		},
		{
			name:  "precision_1001",
			node:  ast.NewBinary(ast.BinaryDecimal, ast.NewInteger("1001"), nil),
			err:   `exec: NUMERIC precision 1001 must be between 1 and 1000`,
			isErr: ErrExecution,
		},
		{
			name: "precision_1000",
			node: ast.NewBinary(ast.BinaryDecimal, ast.NewInteger("1000"), nil),
			num:  float64(98.6),
			exp:  float64(99),
		},
		{
			name: "precision_10",
			node: ast.NewBinary(ast.BinaryDecimal, ast.NewInteger("10"), nil),
			num:  float64(98.6),
			exp:  float64(99),
		},
		{
			name:  "precision_too_small",
			node:  ast.NewBinary(ast.BinaryDecimal, ast.NewInteger("1"), nil),
			value: float64(98.6),
			num:   float64(98.6),
			err:   `exec: argument "98.6" of jsonpath item method .decimal() is invalid for type numeric`,
			isErr: ErrExecution,
		},
		{
			name:  "invalid_scale",
			node:  ast.NewBinary(ast.BinaryDecimal, ast.NewInteger("10"), ast.NewString("hi")),
			err:   `exec: invalid jsonpath item type for .decimal() scale`,
			isErr: ErrExecution,
		},
		{
			name:  "scale_neg_1001",
			node:  ast.NewBinary(ast.BinaryDecimal, ast.NewInteger("10"), ast.NewInteger("-1001")),
			err:   `exec: NUMERIC scale -1001 must be between -1000 and 1000`,
			isErr: ErrExecution,
		},
		{
			name:  "scale_1001",
			node:  ast.NewBinary(ast.BinaryDecimal, ast.NewInteger("10"), ast.NewInteger("1001")),
			err:   `exec: NUMERIC scale 1001 must be between -1000 and 1000`,
			isErr: ErrExecution,
		},
		{
			name: "precision_scale_ok",
			node: ast.NewBinary(ast.BinaryDecimal, ast.NewInteger("5"), ast.NewInteger("3")),
			num:  float64(12.333),
			exp:  float64(12.333),
		},
		{
			name: "scale_down",
			node: ast.NewBinary(ast.BinaryDecimal, ast.NewInteger("5"), ast.NewInteger("2")),
			num:  float64(12.333),
			exp:  float64(12.33),
		},
		{
			name:  "scale_short",
			node:  ast.NewBinary(ast.BinaryDecimal, ast.NewInteger("3"), ast.NewInteger("2")),
			value: float64(12.333),
			num:   float64(12.333),
			err:   `exec: argument "12.333" of jsonpath item method .decimal() is invalid for type numeric`,
			isErr: ErrExecution,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			e := newTestExecutor(laxRootPath, nil, true, false)
			res, err := e.executeDecimalMethod(tc.node, tc.value, tc.num)

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

func TestNumericCallbacks(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	t.Run("intAbs", func(t *testing.T) {
		t.Parallel()
		a.IsType((intCallback)(nil), intCallback(intAbs))
		for i, n := range []int64{0, -1, 2, -3, 4, 5} {
			a.Equal(int64(i), intAbs(n))
		}
	})

	t.Run("intSelf", func(t *testing.T) {
		t.Parallel()
		a.IsType((intCallback)(nil), intCallback(intSelf))
		for _, n := range []int64{4, 42, -99, -100323, 4, 10030} {
			a.Equal(n, intSelf(n))
		}
	})

	t.Run("floatSelf", func(t *testing.T) {
		t.Parallel()
		a.IsType((floatCallback)(nil), floatCallback(floatSelf))
		for _, n := range []float64{-1, 12, 53, 98.6, 42.3, 100.99} {
			//nolint:testifylint
			a.Equal(n, floatSelf(n))
		}
	})

	t.Run("intUMinus", func(t *testing.T) {
		t.Parallel()
		a.IsType((intCallback)(nil), intCallback(intUMinus))
		for _, n := range []int64{4, 42, -99, -100323, 4, 10030} {
			a.Equal(-n, intUMinus(n))
		}
	})

	t.Run("floatSelf", func(t *testing.T) {
		t.Parallel()
		a.IsType((floatCallback)(nil), floatCallback(floatUMinus))
		for _, n := range []float64{-1, 12, 53, 98.6, 42.3, 100.99} {
			//nolint:testifylint
			a.Equal(-n, floatUMinus(n))
		}
	})
}

func TestExecuteNumericItemMethod(t *testing.T) {
	t.Parallel()
	abs := ast.NewMethod(ast.MethodAbs)
	floor := ast.NewMethod(ast.MethodFloor)
	ceil := ast.NewMethod(ast.MethodCeiling)
	ctx := context.Background()

	for _, tc := range []struct {
		methodTestCase

		intCB   intCallback
		floatCB floatCallback
	}{
		{
			methodTestCase: methodTestCase{
				name:  "int_abs",
				path:  laxRootPath,
				node:  abs,
				value: int64(-42),
				exp:   statusOK,
				find:  []any{int64(42)},
			},
			intCB: intAbs,
		},
		{
			methodTestCase: methodTestCase{
				name:  "float_abs",
				path:  laxRootPath,
				node:  abs,
				value: float64(-42.2),
				exp:   statusOK,
				find:  []any{float64(42.2)},
			},
			floatCB: math.Abs,
		},
		{
			methodTestCase: methodTestCase{
				name:  "json_int_abs",
				path:  laxRootPath,
				node:  abs,
				value: json.Number("-42"),
				exp:   statusOK,
				find:  []any{int64(42)},
			},
			intCB: intAbs,
		},
		{
			methodTestCase: methodTestCase{
				name:  "json_float_abs",
				path:  laxRootPath,
				node:  abs,
				value: json.Number("-42.2"),
				exp:   statusOK,
				find:  []any{float64(42.2)},
			},
			floatCB: math.Abs,
		},
		{
			methodTestCase: methodTestCase{
				name:  "invalid_json_number",
				path:  laxRootPath,
				node:  abs,
				value: json.Number("hi"),
				exp:   statusFailed,
				err:   `exec: jsonpath item method .abs() can only be applied to a numeric value`,
				isErr: ErrVerbose,
			},
		},
		{
			methodTestCase: methodTestCase{
				name:  "object",
				path:  laxRootPath,
				node:  abs,
				value: map[string]any{"hi": true},
				exp:   statusFailed,
				err:   `exec: jsonpath item method .abs() can only be applied to a numeric value`,
				isErr: ErrVerbose,
			},
		},
		{
			methodTestCase: methodTestCase{
				name:  "array",
				path:  laxRootPath,
				node:  abs,
				value: []any{int64(-42), float64(-42.2)},
				exp:   statusFailed,
				err:   `exec: jsonpath item method .abs() can only be applied to a numeric value`,
				isErr: ErrVerbose,
			},
		},
		{
			methodTestCase: methodTestCase{
				name:   "abs_array_unwrap",
				path:   laxRootPath,
				node:   abs,
				value:  []any{int64(-42), float64(-42.2)},
				unwrap: true,
				exp:    statusOK,
				find:   []any{int64(42), float64(42.2)},
			},
			intCB:   intAbs,
			floatCB: math.Abs,
		},
		{
			methodTestCase: methodTestCase{
				name:  "int_floor",
				path:  laxRootPath,
				node:  floor,
				value: int64(-42),
				exp:   statusOK,
				find:  []any{int64(-42)},
			},
			intCB: intSelf,
		},
		{
			methodTestCase: methodTestCase{
				name:  "float_floor",
				path:  laxRootPath,
				node:  floor,
				value: float64(-42.2),
				exp:   statusOK,
				find:  []any{float64(-43)},
			},
			floatCB: math.Floor,
		},
		{
			methodTestCase: methodTestCase{
				name:  "json_int_floor",
				path:  laxRootPath,
				node:  floor,
				value: json.Number("42"),
				exp:   statusOK,
				find:  []any{int64(42)},
			},
			intCB: intSelf,
		},
		{
			methodTestCase: methodTestCase{
				name:  "json_float_floor",
				path:  laxRootPath,
				node:  floor,
				value: json.Number("42.2"),
				exp:   statusOK,
				find:  []any{float64(42)},
			},
			floatCB: math.Floor,
		},
		{
			methodTestCase: methodTestCase{
				name:  "invalid_json_number",
				path:  laxRootPath,
				node:  floor,
				value: json.Number("hi"),
				exp:   statusFailed,
				err:   `exec: jsonpath item method .floor() can only be applied to a numeric value`,
				isErr: ErrVerbose,
			},
		},
		{
			methodTestCase: methodTestCase{
				name:   "floor_array_unwrap",
				path:   laxRootPath,
				node:   floor,
				value:  []any{int64(42), float64(42.8)},
				unwrap: true,
				exp:    statusOK,
				find:   []any{int64(42), float64(42)},
			},
			intCB:   intSelf,
			floatCB: math.Floor,
		},

		{
			methodTestCase: methodTestCase{
				name:  "int_ceil",
				path:  laxRootPath,
				node:  ceil,
				value: int64(-42),
				exp:   statusOK,
				find:  []any{int64(-42)},
			},
			intCB: intSelf,
		},
		{
			methodTestCase: methodTestCase{
				name:  "float_ceil",
				path:  laxRootPath,
				node:  ceil,
				value: float64(-42.2),
				exp:   statusOK,
				find:  []any{float64(-42)},
			},
			floatCB: math.Ceil,
		},
		{
			methodTestCase: methodTestCase{
				name:  "json_int_ceil",
				path:  laxRootPath,
				node:  ceil,
				value: json.Number("42"),
				exp:   statusOK,
				find:  []any{int64(42)},
			},
			intCB: intSelf,
		},
		{
			methodTestCase: methodTestCase{
				name:  "json_float_ceil",
				path:  laxRootPath,
				node:  ceil,
				value: json.Number("42.2"),
				exp:   statusOK,
				find:  []any{float64(43)},
			},
			floatCB: math.Ceil,
		},
		{
			methodTestCase: methodTestCase{
				name:  "invalid_json_number",
				path:  laxRootPath,
				node:  ceil,
				value: json.Number("hi"),
				exp:   statusFailed,
				err:   `exec: jsonpath item method .ceiling() can only be applied to a numeric value`,
				isErr: ErrVerbose,
			},
		},
		{
			methodTestCase: methodTestCase{
				name:   "ceil_array_unwrap",
				path:   laxRootPath,
				node:   ceil,
				value:  []any{int64(42), float64(42.8)},
				unwrap: true,
				exp:    statusOK,
				find:   []any{int64(42), float64(43)},
			},
			intCB:   intSelf,
			floatCB: math.Ceil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			e, list := tc.prep()
			res, err := e.executeNumericItemMethod(ctx, tc.node, tc.value, tc.unwrap, tc.intCB, tc.floatCB, list)
			tc.checkResults(t, res, list, err)
		})
	}
}
