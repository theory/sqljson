package exec

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/theory/sqljson/path/ast"
	"github.com/theory/sqljson/path/types"
)

func TestExecBinaryNode(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	for _, tc := range []struct {
		test   string
		node   *ast.BinaryNode
		value  any
		unwrap bool
		exp    resultStatus
		find   []any
		err    string
		isErr  error
	}{
		{
			test: "and",
			node: ast.NewBinary(
				ast.BinaryAnd,
				ast.NewBinary(ast.BinaryEqual, ast.NewConst(ast.ConstRoot), ast.NewConst(ast.ConstRoot)),
				ast.NewBinary(ast.BinaryEqual, ast.NewConst(ast.ConstRoot), ast.NewConst(ast.ConstRoot)),
			),
			exp:  statusOK,
			find: []any{true},
		},
		{
			test: "or",
			node: ast.NewBinary(
				ast.BinaryOr,
				ast.NewBinary(ast.BinaryEqual, ast.NewConst(ast.ConstRoot), ast.NewConst(ast.ConstRoot)),
				ast.NewBinary(ast.BinaryEqual, ast.NewConst(ast.ConstRoot), ast.NewConst(ast.ConstRoot)),
			),
			exp:  statusOK,
			find: []any{true},
		},
		{
			test: "eq",
			node: ast.NewBinary(ast.BinaryEqual, ast.NewInteger("42"), ast.NewInteger("42")),
			exp:  statusOK,
			find: []any{true},
		},
		{
			test: "ne",
			node: ast.NewBinary(ast.BinaryNotEqual, ast.NewInteger("42"), ast.NewInteger("42")),
			exp:  statusOK,
			find: []any{false},
		},
		{
			test: "lt",
			node: ast.NewBinary(ast.BinaryLess, ast.NewInteger("41"), ast.NewInteger("42")),
			exp:  statusOK,
			find: []any{true},
		},
		{
			test: "gt",
			node: ast.NewBinary(ast.BinaryLess, ast.NewInteger("42"), ast.NewInteger("42")),
			exp:  statusOK,
			find: []any{false},
		},
		{
			test: "le",
			node: ast.NewBinary(ast.BinaryLessOrEqual, ast.NewInteger("42"), ast.NewInteger("42")),
			exp:  statusOK,
			find: []any{true},
		},
		{
			test: "ge",
			node: ast.NewBinary(ast.BinaryGreaterOrEqual, ast.NewInteger("42"), ast.NewInteger("42")),
			exp:  statusOK,
			find: []any{true},
		},
		{
			test: "starts_with",
			node: ast.NewBinary(ast.BinaryStartsWith, ast.NewString("hi there"), ast.NewString("hi")),
			exp:  statusOK,
			find: []any{true},
		},
		{
			test: "add",
			node: ast.NewBinary(ast.BinaryAdd, ast.NewInteger("12"), ast.NewInteger("38")),
			exp:  statusOK,
			find: []any{int64(50)},
		},
		{
			test: "sub",
			node: ast.NewBinary(ast.BinarySub, ast.NewInteger("42"), ast.NewInteger("12")),
			exp:  statusOK,
			find: []any{int64(30)},
		},
		{
			test: "mul",
			node: ast.NewBinary(ast.BinaryMul, ast.NewInteger("5"), ast.NewInteger("6")),
			exp:  statusOK,
			find: []any{int64(30)},
		},
		{
			test: "div",
			node: ast.NewBinary(ast.BinaryDiv, ast.NewInteger("10"), ast.NewInteger("2")),
			exp:  statusOK,
			find: []any{int64(5)},
		},
		{
			test: "mod",
			node: ast.NewBinary(ast.BinaryMod, ast.NewInteger("12"), ast.NewInteger("5")),
			exp:  statusOK,
			find: []any{int64(2)},
		},
		{
			test:  "decimal",
			value: float64(12.233),
			node:  ast.NewBinary(ast.BinaryDecimal, ast.NewInteger("4"), ast.NewInteger("2")),
			exp:   statusOK,
			find:  []any{float64(12.23)},
		},
		{
			test:  "subscript",
			node:  ast.NewBinary(ast.BinarySubscript, nil, nil),
			exp:   statusFailed,
			err:   `exec: evaluating jsonpath subscript expression outside of array subscript`,
			isErr: ErrExecution,
		},
		{
			test: "unknown_op",
			node: ast.NewBinary(ast.BinaryOperator(-1), nil, nil),
			exp:  statusNotFound,
			find: []any{},
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)
			r := require.New(t)

			e := newTestExecutor(laxRootPath, nil, true, false)
			list := newList()
			res, err := e.execBinaryNode(ctx, tc.node, tc.value, list, tc.unwrap)
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
		})
	}
}

func TestExecUnaryNode(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	for _, tc := range []struct {
		test   string
		node   *ast.UnaryNode
		value  any
		unwrap bool
		exp    resultStatus
		find   []any
		err    string
		isErr  error
	}{
		{
			test: "not",
			node: ast.NewUnary(
				ast.UnaryNot,
				ast.NewUnary(ast.UnaryExists, ast.NewConst(ast.ConstRoot)),
			),
			exp:  statusOK,
			find: []any{false},
		},
		{
			test: "filter",
			node: ast.NewUnary(
				ast.UnaryFilter,
				ast.NewBinary(
					ast.BinaryEqual,
					ast.NewConst(ast.ConstTrue),
					ast.NewConst(ast.ConstTrue),
				),
			),
			value: "hi",
			exp:   statusOK,
			find:  []any{"hi"},
		},
		{
			test: "filter_false",
			node: ast.NewUnary(
				ast.UnaryFilter,
				ast.NewBinary(
					ast.BinaryNotEqual,
					ast.NewConst(ast.ConstTrue),
					ast.NewConst(ast.ConstTrue),
				),
			),
			value: "hi",
			exp:   statusNotFound,
			find:  []any{},
		},
		{
			test: "filter_array",
			node: ast.NewUnary(
				ast.UnaryFilter,
				ast.NewBinary(
					ast.BinaryEqual,
					ast.NewConst(ast.ConstTrue),
					ast.NewConst(ast.ConstTrue),
				),
			),
			value: []any{"hi"},
			exp:   statusOK,
			find:  []any{[]any{"hi"}},
		},
		{
			test: "filter_array_unwrap",
			node: ast.NewUnary(
				ast.UnaryFilter,
				ast.NewBinary(
					ast.BinaryEqual,
					ast.NewConst(ast.ConstTrue),
					ast.NewConst(ast.ConstTrue),
				),
			),
			value:  []any{"hi", "there"},
			unwrap: true,
			exp:    statusOK,
			find:   []any{"hi", "there"},
		},
		{
			test: "filter_array_unwrap_false",
			node: ast.NewUnary(
				ast.UnaryFilter,
				ast.NewBinary(
					ast.BinaryNotEqual,
					ast.NewConst(ast.ConstTrue),
					ast.NewConst(ast.ConstTrue),
				),
			),
			value:  []any{"hi", "there"},
			unwrap: true,
			exp:    statusNotFound,
			find:   []any{},
		},
		{
			test:  "plus",
			node:  ast.NewUnary(ast.UnaryPlus, ast.NewConst(ast.ConstRoot)),
			exp:   statusOK,
			value: int64(-42),
			find:  []any{int64(-42)},
		},
		{
			test:  "minus",
			node:  ast.NewUnary(ast.UnaryMinus, ast.NewConst(ast.ConstRoot)),
			exp:   statusOK,
			value: int64(-42),
			find:  []any{int64(42)},
		},
		{
			test:  "datetime",
			node:  ast.NewUnary(ast.UnaryDateTime, nil),
			exp:   statusOK,
			value: "2024-06-14",
			find:  []any{types.NewDate(time.Date(2024, 6, 14, 0, 0, 9, 9, time.UTC))},
		},
		{
			test:  "date",
			node:  ast.NewUnary(ast.UnaryDateTime, nil),
			exp:   statusOK,
			value: "2024-06-14",
			find:  []any{types.NewDate(time.Date(2024, 6, 14, 0, 0, 0, 0, time.UTC))},
		},
		{
			test:  "time",
			node:  ast.NewUnary(ast.UnaryTime, nil),
			exp:   statusOK,
			value: "14:23:54",
			find:  []any{types.NewTime(time.Date(0, 1, 1, 14, 23, 54, 0, time.UTC))},
		},
		{
			test:  "timetz",
			node:  ast.NewUnary(ast.UnaryTimeTZ, nil),
			exp:   statusOK,
			value: "14:23:54+01",
			find:  []any{types.NewTimeTZ(time.Date(0, 1, 1, 14, 23, 54, 0, time.FixedZone("", 60*60)))},
		},
		{
			test:  "timestamp",
			node:  ast.NewUnary(ast.UnaryTimestamp, nil),
			exp:   statusOK,
			value: "2024-06-14T14:23:54",
			find:  []any{types.NewTimestamp(time.Date(2024, 6, 14, 14, 23, 54, 0, time.UTC))},
		},
		{
			test:  "timestamptz",
			node:  ast.NewUnary(ast.UnaryTimestampTZ, nil),
			exp:   statusOK,
			value: "2024-06-14T14:23:54+01",
			find: []any{types.NewTimestampTZ(
				ctx,
				time.Date(2024, 6, 14, 14, 23, 54, 0, time.FixedZone("", 60*60)),
			)},
		},
		{
			test:  "datetime_array",
			node:  ast.NewUnary(ast.UnaryDateTime, nil),
			value: []any{"2024-06-14", "2024-06-14T14:23:54+01"},
			exp:   statusFailed,
			err:   `exec: jsonpath item method .datetime() can only be applied to a string`,
			isErr: ErrVerbose,
		},
		{
			test:   "datetime_array_unwrap",
			node:   ast.NewUnary(ast.UnaryDateTime, nil),
			exp:    statusOK,
			value:  []any{"2024-06-14", "2024-06-14T14:23:54+01"},
			unwrap: true,
			find: []any{
				types.NewDate(time.Date(2024, 6, 14, 0, 0, 0, 0, time.FixedZone("", 0))),
				types.NewTimestampTZ(
					ctx,
					time.Date(2024, 6, 14, 14, 23, 54, 0, time.FixedZone("", 60*60)),
				),
			},
		},
		{
			test: "unknown_op",
			node: ast.NewUnary(ast.UnaryOperator(-1), nil),
			exp:  statusNotFound,
			find: []any{},
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)
			r := require.New(t)

			e := newTestExecutor(laxRootPath, nil, true, false)
			e.root = tc.value
			list := newList()
			res, err := e.execUnaryNode(ctx, tc.node, tc.value, list, tc.unwrap)
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
		})
	}
}

func TestExecRegexNode(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	for _, tc := range []struct {
		test  string
		regex string
		value any
		exp   resultStatus
		find  []any
		err   string
		isErr error
	}{
		{
			test:  "regex_match",
			regex: "^hi",
			value: "hi there",
			exp:   statusOK,
			find:  []any{true},
		},
		{
			test:  "regex_no_match",
			regex: "^hi",
			value: "say hi there",
			exp:   statusOK,
			find:  []any{false},
		},
		{
			test:  "regex_not_string",
			regex: "^hi",
			value: map[string]any{"x": "hi"},
			exp:   statusOK,
			find:  []any{nil},
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)
			r := require.New(t)

			rx, err := ast.NewRegex(ast.NewConst(ast.ConstRoot), tc.regex, "")
			r.NoError(err)

			e := newTestExecutor(laxRootPath, nil, true, false)
			e.root = tc.value
			list := newList()
			res, err := e.execRegexNode(ctx, rx, tc.value, list)
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
		})
	}
}

func TestExecAnyNode(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	for _, tc := range []struct {
		test  string
		node  ast.Node
		value any
		exp   resultStatus
		find  []any
		err   string
		isErr error
	}{
		{
			test:  "map_unbounded",
			node:  ast.NewAny(0, -1),
			value: map[string]any{"x": true, "y": true},
			exp:   statusOK,
			find:  []any{map[string]any{"x": true, "y": true}, true, true},
		},
		{
			test:  "map_2_levels",
			node:  ast.NewAny(0, 1),
			value: map[string]any{"x": map[string]any{"y": map[string]any{"z": "hi"}}},
			exp:   statusOK,
			find: []any{
				map[string]any{"x": map[string]any{"y": map[string]any{"z": "hi"}}},
				map[string]any{"y": map[string]any{"z": "hi"}},
			},
		},
		{
			test:  "map_3_levels",
			node:  ast.NewAny(0, 2),
			value: map[string]any{"x": map[string]any{"y": map[string]any{"z": "hi"}}},
			exp:   statusOK,
			find: []any{
				map[string]any{"x": map[string]any{"y": map[string]any{"z": "hi"}}},
				map[string]any{"y": map[string]any{"z": "hi"}},
				map[string]any{"z": "hi"},
			},
		},
		{
			test:  "map_1_2_levels",
			node:  ast.NewAny(1, 2),
			value: map[string]any{"x": map[string]any{"y": map[string]any{"z": "hi"}}},
			exp:   statusOK,
			find: []any{
				map[string]any{"y": map[string]any{"z": "hi"}},
				map[string]any{"z": "hi"},
			},
		},
		{
			test:  "array_unbounded",
			node:  ast.NewAny(0, -1),
			value: []any{"x", "y", map[string]any{"x": "hi"}},
			exp:   statusOK,
			find: []any{
				[]any{"x", "y", map[string]any{"x": "hi"}},
				"x", "y",
				map[string]any{"x": "hi"},
				"hi",
			},
		},
		{
			test:  "array_2_levels",
			node:  ast.NewAny(0, 1),
			value: []any{"x", "y", map[string]any{"x": "hi"}},
			exp:   statusOK,
			find: []any{
				[]any{"x", "y", map[string]any{"x": "hi"}},
				"x", "y",
				map[string]any{"x": "hi"},
			},
		},
		{
			test:  "array_1_levels",
			node:  ast.NewAny(1, 1),
			value: []any{"x", "y", map[string]any{"x": "hi"}},
			exp:   statusOK,
			find: []any{
				"x", "y",
				map[string]any{"x": "hi"},
			},
		},
		{
			test:  "not_object_or_array",
			node:  ast.NewAny(1, -1),
			value: true,
			exp:   statusNotFound,
			find:  []any{},
		},
		{
			test:  "map_next",
			node:  ast.LinkNodes([]ast.Node{ast.NewAny(1, 1), ast.NewMethod(ast.MethodString)}),
			value: map[string]any{"x": true, "y": true},
			exp:   statusOK,
			find:  []any{"true", "true"},
		},
		{
			test:  "map_next_error",
			node:  ast.LinkNodes([]ast.Node{ast.NewAny(1, 1), ast.NewMethod(ast.MethodFloor)}),
			value: map[string]any{"x": "hi"},
			exp:   statusFailed,
			err:   `exec: jsonpath item method .floor() can only be applied to a numeric value`,
			isErr: ErrVerbose,
		},
		{
			test:  "nested_array",
			node:  ast.NewAny(1, -1),
			value: []any{[]any{"hi", true}},
			exp:   statusOK,
			find:  []any{[]any{"hi", true}, "hi", true},
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)
			r := require.New(t)

			// Should have an AnyNode.
			node, ok := tc.node.(*ast.AnyNode)
			a.True(ok)

			e := newTestExecutor(laxRootPath, nil, true, false)
			e.ignoreStructuralErrors = false
			e.root = tc.value

			// Test with found first and ignore the result.
			list := newList()
			_, err := e.execAnyNode(ctx, node, tc.value, list)
			a.False(e.ignoreStructuralErrors)

			// Check the error and list.
			if tc.isErr == nil {
				r.NoError(err)
				a.Equal(tc.find, list.list)
			} else {
				r.EqualError(err, tc.err)
				r.ErrorIs(err, tc.isErr)
				a.Empty(list.list)
			}

			// Test without found, pay attention to the result.
			res, err := e.execAnyNode(ctx, node, tc.value, nil)
			a.False(e.ignoreStructuralErrors)
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

func TestCollection(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		test  string
		value any
		exp   []any
	}{
		{
			test:  "slice",
			value: []any{"hi", "yo"},
			exp:   []any{"hi", "yo"},
		},
		{
			test:  "map",
			value: map[string]any{"x": "hi", "y": "hi"},
			exp:   []any{"hi", "hi"},
		},
		{
			test:  "int",
			value: int64(42),
		},
		{
			test:  "bool",
			value: true,
		},
		{
			test:  "nil",
			value: nil,
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)

			a.Equal(tc.exp, collection(tc.value))
		})
	}
}

func TestExecuteAnyItem(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	for _, tc := range []struct {
		test   string
		node   ast.Node
		value  []any
		ignore bool
		unwrap bool
		exp    resultStatus
		find   []any
		err    string
		isErr  error
	}{
		{
			test:  "flat_all",
			node:  ast.NewAny(0, -1),
			value: []any{"hi", true},
			exp:   statusOK,
			find:  []any{"hi", true},
		},
		{
			test:  "nest_map_all",
			node:  ast.NewAny(0, -1),
			value: []any{"hi", map[string]any{"x": map[string]any{"y": map[string]any{"z": "yo"}}}},
			exp:   statusOK,
			find: []any{
				"hi",
				map[string]any{"x": map[string]any{"y": map[string]any{"z": "yo"}}},
				map[string]any{"y": map[string]any{"z": "yo"}},
				map[string]any{"z": "yo"},
				"yo",
				map[string]any{"y": map[string]any{"z": "yo"}},
				map[string]any{"z": "yo"},
				"yo",
				map[string]any{"z": "yo"},
				"yo",
				"yo",
			},
		},
		{
			test:  "nest_map_0_2",
			node:  ast.NewAny(0, 2),
			value: []any{"hi", map[string]any{"x": map[string]any{"y": map[string]any{"z": "yo"}}}},
			exp:   statusOK,
			find: []any{
				"hi",
				map[string]any{"x": map[string]any{"y": map[string]any{"z": "yo"}}},
				map[string]any{"y": map[string]any{"z": "yo"}},
				map[string]any{"z": "yo"},
				map[string]any{"y": map[string]any{"z": "yo"}},
				map[string]any{"z": "yo"},
				"yo",
			},
		},
		{
			test:  "nest_map_1_2",
			node:  ast.NewAny(1, 2),
			value: []any{"hi", map[string]any{"x": map[string]any{"y": map[string]any{"z": "yo"}}}},
			exp:   statusOK,
			find: []any{
				map[string]any{"y": map[string]any{"z": "yo"}},
				map[string]any{"z": "yo"},
				map[string]any{"z": "yo"},
				"yo",
			},
		},
		{
			test:  "nest_array_all",
			node:  ast.NewAny(0, -1),
			value: []any{"hi", []any{"yo", []any{"x", []any{true}}}},
			exp:   statusOK,
			find: []any{
				"hi",
				[]any{"yo", []any{"x", []any{true}}},
				"yo",
				[]any{"x", []any{true}},
				"x",
				[]any{true},
				true,
				"yo",
				[]any{"x", []any{true}},
				"x",
				[]any{true},
				true,
				"x",
				[]any{true},
				true,
				true,
			},
		},
		{
			test:  "nest_array_0_2",
			node:  ast.NewAny(0, 2),
			value: []any{"hi", []any{"yo", []any{"x", []any{true}}}},
			exp:   statusOK,
			find: []any{
				"hi",
				[]any{"yo", []any{"x", []any{true}}},
				"yo",
				[]any{"x", []any{true}},
				"x",
				[]any{true},
				"yo",
				[]any{"x", []any{true}},
				"x",
				[]any{true},
				true,
			},
		},
		{
			test:  "nest_array_1_2",
			node:  ast.NewAny(1, 2),
			value: []any{"hi", []any{"yo", []any{"x", []any{true}}}},
			exp:   statusOK,
			find: []any{
				"yo",
				[]any{"x", []any{true}},
				"x",
				[]any{true},
				"x",
				[]any{true},
				true,
			},
		},
		{
			test:  "level_gt_last",
			node:  ast.NewAny(0, 0),
			value: []any{"hi", true},
			exp:   statusNotFound,
			find:  []any{},
		},
		{
			test:  "next_item",
			node:  ast.LinkNodes([]ast.Node{ast.NewAny(0, -1), ast.NewMethod(ast.MethodString)}),
			value: []any{"hi", true},
			exp:   statusOK,
			find:  []any{"hi", "true"},
		},
		{
			test:  "next_item_level",
			node:  ast.LinkNodes([]ast.Node{ast.NewAny(0, -1), ast.NewMethod(ast.MethodString)}),
			value: []any{[]any{"hi", true}},
			exp:   statusOK,
			find:  []any{"hi", "true", "hi", "true", "hi", "true"},
		},
		{
			test:  "next_item_error",
			node:  ast.LinkNodes([]ast.Node{ast.NewAny(0, -1), ast.NewMethod(ast.MethodNumber)}),
			value: []any{"hi", true},
			exp:   statusFailed,
			err:   `exec: argument "hi" of jsonpath item method .number() is invalid for type numeric`,
			isErr: ErrVerbose,
		},
		{
			test:  "next_item_level_error",
			node:  ast.LinkNodes([]ast.Node{ast.NewAny(0, -1), ast.NewMethod(ast.MethodNumber)}),
			value: []any{"hi", []any{"hi", true}},
			exp:   statusFailed,
			err:   `exec: argument "hi" of jsonpath item method .number() is invalid for type numeric`,
			isErr: ErrVerbose,
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)
			r := require.New(t)

			// Should have an AnyNode.
			node, ok := tc.node.(*ast.AnyNode)
			a.True(ok)

			e := newTestExecutor(laxRootPath, nil, true, false)
			e.ignoreStructuralErrors = false
			e.root = tc.value

			// Test with found first and ignore the result.
			list := newList()
			res, err := e.executeAnyItem(ctx, node, tc.value, list, 1, node.First(), node.Last(), tc.ignore, tc.unwrap)
			a.Equal(tc.exp, res)
			a.False(e.ignoreStructuralErrors)

			// Check the error and list.
			if tc.isErr == nil {
				r.NoError(err)
				a.Equal(tc.find, list.list)
			} else {
				r.EqualError(err, tc.err)
				r.ErrorIs(err, tc.isErr)
				a.Empty(list.list)
			}

			// Test without found, pay attention to the result.
			res, err = e.executeAnyItem(ctx, node, tc.value, nil, 1, node.First(), node.Last(), tc.ignore, tc.unwrap)
			a.False(e.ignoreStructuralErrors)
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

// TestExecuteLikeRegex in exec_test.go tests happy paths.
func TestExecuteLikeRegexErrors(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	r := require.New(t)
	ctx := context.Background()

	e := newTestExecutor(laxRootPath, nil, true, false)
	r.PanicsWithValue(
		"Node *ast.ConstNode passed to executeLikeRegex is not an ast.RegexNode",
		func() { _, _ = e.executeLikeRegex(ctx, ast.NewConst(ast.ConstRoot), nil, nil) },
	)

	rx, err := ast.NewRegex(ast.NewConst(ast.ConstRoot), ".", "")
	r.NoError(err)

	res, err := e.executeLikeRegex(ctx, rx, true, nil)
	a.Equal(predUnknown, res)
	a.NoError(err)
}

func TestExecuteStartsWith(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	for _, tc := range []struct {
		test   string
		str    any
		prefix any
		exp    predOutcome
	}{
		{
			test:   "full_string",
			str:    "hi there",
			prefix: "hi there",
			exp:    predTrue,
		},
		{
			test:   "prefix",
			str:    "hi there",
			prefix: "hi ",
			exp:    predTrue,
		},
		{
			test:   "not_prefix",
			str:    "hi there",
			prefix: " hi",
			exp:    predFalse,
		},
		{
			test: "not_string",
			str:  true,
			exp:  predUnknown,
		},
		{
			test:   "not_string_prefix",
			str:    "hi",
			prefix: int64(42),
			exp:    predUnknown,
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)
			r := require.New(t)

			res, err := executeStartsWith(ctx, nil, tc.str, tc.prefix)
			a.Equal(tc.exp, res)
			r.NoError(err)
		})
	}
}
