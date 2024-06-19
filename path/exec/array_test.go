package exec

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/theory/sqljson/path/ast"
	"github.com/theory/sqljson/path/parser"
)

func TestExecSubscript(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	r := require.New(t)
	ctx := context.Background()
	lax, _ := parser.Parse("$")
	strict, _ := parser.Parse("strict $")

	for _, tc := range []struct {
		name  string
		path  *ast.AST
		node  ast.Node
		size  int
		from  int
		to    int
		err   string
		errIs error
	}{
		{
			name:  "not_binary_node",
			path:  lax,
			node:  ast.NewString("hi"),
			err:   `exec: jsonpath array subscript is not a single numeric value`,
			errIs: ErrExecution,
		},
		{
			name:  "not_subscript",
			path:  lax,
			node:  ast.NewBinary(ast.BinaryAdd, ast.NewInteger("1"), ast.NewInteger("2")),
			err:   `exec: jsonpath array subscript is not a single numeric value`,
			errIs: ErrExecution,
		},
		{
			name:  "left_not_number",
			path:  lax,
			node:  ast.NewBinary(ast.BinarySubscript, ast.NewString("1"), ast.NewInteger("2")),
			err:   `exec: jsonpath array subscript is not a single numeric value`,
			errIs: ErrVerbose,
		},
		{
			name:  "right_not_number",
			path:  lax,
			node:  ast.NewBinary(ast.BinarySubscript, ast.NewInteger("1"), ast.NewString("2")),
			err:   `exec: jsonpath array subscript is not a single numeric value`,
			errIs: ErrVerbose,
		},
		{
			name:  "from_lt_0_strict",
			path:  strict,
			node:  ast.NewBinary(ast.BinarySubscript, ast.NewInteger("-1"), nil),
			err:   `exec: jsonpath array subscript is out of bounds`,
			errIs: ErrVerbose,
		},
		{
			name:  "from_gt_to_strict",
			path:  strict,
			node:  ast.NewBinary(ast.BinarySubscript, ast.NewInteger("2"), ast.NewInteger("1")),
			err:   `exec: jsonpath array subscript is out of bounds`,
			errIs: ErrVerbose,
		},
		{
			name:  "to_gt_size_strict",
			path:  strict,
			node:  ast.NewBinary(ast.BinarySubscript, ast.NewInteger("1"), ast.NewInteger("4")),
			size:  2,
			err:   `exec: jsonpath array subscript is out of bounds`,
			errIs: ErrVerbose,
		},
		{
			name: "from_lt_0_lax",
			path: lax,
			node: ast.NewBinary(ast.BinarySubscript, ast.NewInteger("-1"), ast.NewInteger("1")),
			size: 3,
			from: 0,
			to:   1,
		},
		{
			name: "from_gt_to_lax",
			path: lax,
			node: ast.NewBinary(ast.BinarySubscript, ast.NewInteger("2"), ast.NewInteger("4")),
			size: 7,
			from: 2,
			to:   4,
		},
		{
			name: "to_gt_size_lax",
			path: lax,
			node: ast.NewBinary(ast.BinarySubscript, ast.NewInteger("1"), ast.NewInteger("14")),
			size: 7,
			from: 1,
			to:   6,
		},
		{
			name: "no_right_operand",
			path: lax,
			node: ast.NewBinary(ast.BinarySubscript, ast.NewInteger("1"), nil),
			size: 10,
			from: 1,
			to:   1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			e := newTestExecutor(tc.path, nil, true, false)
			from, to, err := e.execSubscript(ctx, tc.node, nil, tc.size)
			a.Equal(tc.from, from)
			a.Equal(tc.to, to)

			if tc.errIs == nil {
				r.NoError(err)
			} else {
				r.EqualError(err, tc.err)
				r.ErrorIs(err, tc.errIs)
			}
		})
	}
}

func TestExecArrayIndex(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	r := require.New(t)
	ctx := context.Background()
	lax, _ := parser.Parse("$")
	strict, _ := parser.Parse("strict $")
	linked, _ := ast.LinkNodes([]ast.Node{
		ast.NewArrayIndex([]ast.Node{
			ast.NewBinary(ast.BinarySubscript, ast.NewInteger("0"), ast.NewConst(ast.ConstLast)),
		}),
		ast.NewMethod(ast.MethodString),
	}).(*ast.ArrayIndexNode)
	nextErr, _ := ast.LinkNodes([]ast.Node{
		ast.NewArrayIndex([]ast.Node{
			ast.NewBinary(ast.BinarySubscript, ast.NewInteger("0"), ast.NewConst(ast.ConstLast)),
		}),
		ast.NewVariable("foo"),
	}).(*ast.ArrayIndexNode)

	for _, tc := range []struct {
		name   string
		path   *ast.AST
		node   *ast.ArrayIndexNode
		value  any
		unwrap bool
		exp    resultStatus
		found  []any
		err    string
		errIs  error
	}{
		{
			name:  "not_array_strict",
			path:  strict,
			value: "hi",
			exp:   statusFailed,
			found: []any{},
			err:   `exec: jsonpath array accessor can only be applied to an array`,
			errIs: ErrVerbose,
		},
		{
			name:  "not_array_lax",
			path:  lax,
			node:  ast.NewArrayIndex([]ast.Node{ast.NewBinary(ast.BinarySubscript, ast.NewInteger("0"), nil)}),
			value: "hi",
			exp:   statusOK,
			found: []any{"hi"},
		},
		{
			name:  "not_found_lax",
			path:  lax,
			node:  ast.NewArrayIndex([]ast.Node{ast.NewBinary(ast.BinarySubscript, ast.NewInteger("1"), nil)}),
			value: "hi",
			exp:   statusNotFound,
			found: []any{},
		},
		{
			name:  "is_array",
			path:  strict,
			node:  ast.NewArrayIndex([]ast.Node{ast.NewBinary(ast.BinarySubscript, ast.NewInteger("0"), nil)}),
			value: []any{"hi"},
			exp:   statusOK,
			found: []any{"hi"},
		},
		{
			name:  "is_array_second_item",
			path:  strict,
			node:  ast.NewArrayIndex([]ast.Node{ast.NewBinary(ast.BinarySubscript, ast.NewInteger("1"), nil)}),
			value: []any{"hi", "go"},
			exp:   statusOK,
			found: []any{"go"},
		},
		{
			name: "is_array_range",
			path: strict,
			node: ast.NewArrayIndex([]ast.Node{
				ast.NewBinary(ast.BinarySubscript, ast.NewInteger("0"), ast.NewInteger("1")),
			}),
			value: []any{"hi", "go", "on"},
			exp:   statusOK,
			found: []any{"hi", "go"},
		},
		{
			name: "is_array_sub_range",
			path: strict,
			node: ast.NewArrayIndex([]ast.Node{
				ast.NewBinary(ast.BinarySubscript, ast.NewInteger("2"), ast.NewInteger("5")),
			}),
			value: []any{"hi", "go", "on", true, "12", false, "nope"},
			exp:   statusOK,
			found: []any{"on", true, "12", false},
		},
		{
			name: "is_array_last",
			path: strict,
			node: ast.NewArrayIndex([]ast.Node{
				ast.NewBinary(ast.BinarySubscript, ast.NewInteger("0"), ast.NewConst(ast.ConstLast)),
			}),
			value: []any{"hi", "go", "on"},
			exp:   statusOK,
			found: []any{"hi", "go", "on"},
		},
		{
			name:  "not_a_subscript",
			path:  strict,
			node:  ast.NewArrayIndex([]ast.Node{ast.NewConst(ast.ConstRoot)}),
			value: []any{"hi"},
			exp:   statusFailed,
			found: []any{},
			err:   `exec: jsonpath array subscript is not a single numeric value`,
			errIs: ErrExecution,
		},
		{
			name: "skip_nil",
			path: strict,
			node: ast.NewArrayIndex([]ast.Node{
				ast.NewBinary(ast.BinarySubscript, ast.NewInteger("0"), ast.NewConst(ast.ConstLast)),
			}),
			value: []any{"hi", nil, "go", "on"},
			exp:   statusOK,
			found: []any{"hi", "go", "on"},
		},
		{
			name: "no_found_param",
			path: strict,
			node: ast.NewArrayIndex([]ast.Node{
				ast.NewBinary(ast.BinarySubscript, ast.NewInteger("0"), ast.NewConst(ast.ConstLast)),
			}),
			value: []any{"hi", "go", "on"},
			exp:   statusOK,
		},
		{
			name:  "next_item",
			path:  strict,
			node:  linked,
			value: []any{int64(2), true},
			exp:   statusOK,
			found: []any{"2", "true"},
		},
		{
			name:  "next_item_fail",
			path:  strict,
			node:  nextErr,
			value: []any{int64(2), true},
			exp:   statusFailed,
			found: []any{},
			err:   `exec: could not find jsonpath variable "foo"`,
			errIs: ErrExecution,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			e := newTestExecutor(tc.path, nil, true, false)
			e.innermostArraySize = 12
			found := newList()
			if tc.found == nil {
				found = nil
			}
			res, err := e.execArrayIndex(ctx, tc.node, tc.value, found)
			a.Equal(tc.exp, res)
			a.Equal(12, e.innermostArraySize)
			if tc.found == nil {
				a.Nil(found)
			} else {
				a.Equal(tc.found, found.list)
			}
			if tc.errIs == nil {
				r.NoError(err)
			} else {
				r.EqualError(err, tc.err)
				r.ErrorIs(err, tc.errIs)
			}
		})
	}
}

func TestExecuteItemUnwrapTargetArray(t *testing.T) {
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
		found []any
		err   string
		errIs error
	}{
		{
			name:  "not_array",
			value: "hi",
			exp:   statusFailed,
			found: []any{},
			err:   `exec invalid: invalid json array value type: string`,
			errIs: ErrInvalid,
		},
		{
			name:  "invalid_array",
			value: []string{"hi"},
			exp:   statusFailed,
			found: []any{},
			err:   `exec invalid: invalid json array value type: []string`,
			errIs: ErrInvalid,
		},
		{
			name:  "is_array_no_node",
			value: []any{float64(1), float64(2)},
			exp:   statusOK,
			found: []any{float64(1), float64(2)},
		},
		{
			name:  "exec_node",
			value: []any{float64(1), float64(2)},
			node:  ast.NewMethod(ast.MethodString),
			exp:   statusOK,
			found: []any{"1", "2"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			e := newTestExecutor(path, nil, true, false)
			found := newList()
			res, err := e.executeItemUnwrapTargetArray(ctx, tc.node, tc.value, found)
			a.Equal(tc.exp, res)
			a.Equal(tc.found, found.list)
			if tc.errIs == nil {
				r.NoError(err)
			} else {
				r.EqualError(err, tc.err)
				r.ErrorIs(err, tc.errIs)
			}
		})
	}
}

func TestGetArrayIndex(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	r := require.New(t)
	ctx := context.Background()
	path, _ := parser.Parse("$[*]")

	for _, tc := range []struct {
		name  string
		node  ast.Node
		value any
		exp   int
		err   string
		errIs error
	}{
		{
			name:  "exec_item_fail",
			node:  ast.NewVariable("foo"),
			err:   `exec: could not find jsonpath variable "foo"`,
			errIs: ErrExecution,
		},
		{
			name:  "too_many_found",
			node:  path.Root(),
			value: []any{1, 2},
			err:   `exec: jsonpath array subscript is not a single numeric value`,
			errIs: ErrExecution,
		},
		{
			name:  "success",
			node:  path.Root(),
			value: []any{int64(1)},
			exp:   1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			e := newTestExecutor(path, nil, true, false)
			e.root = tc.value
			integer, err := e.getArrayIndex(ctx, tc.node, tc.value)
			a.Equal(tc.exp, integer)
			if tc.errIs == nil {
				r.NoError(err)
			} else {
				r.EqualError(err, tc.err)
				r.ErrorIs(err, tc.errIs)
			}
		})
	}
}
