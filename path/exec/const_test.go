package exec

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/theory/sqljson/path/ast"
	"github.com/theory/sqljson/path/parser"
)

func TestExecConstNode(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	r := require.New(t)
	ctx := context.Background()
	path, _ := parser.Parse("$")
	base := kvBaseObject{addr: uintptr(42), id: -1}
	current := []any{"hi", true}

	for _, tc := range []struct {
		name   string
		node   *ast.ConstNode
		value  any
		find   []any
		unwrap bool
		exp    resultStatus
		err    string
		isErr  error
	}{
		{
			name: "null",
			node: ast.NewConst(ast.ConstNull),
			exp:  statusOK,
			find: []any{nil},
		},
		{
			name: "true",
			node: ast.NewConst(ast.ConstTrue),
			exp:  statusOK,
			find: []any{true},
		},
		{
			name: "false",
			node: ast.NewConst(ast.ConstFalse),
			exp:  statusOK,
			find: []any{false},
		},
		{
			name: "root",
			node: ast.NewConst(ast.ConstRoot),
			exp:  statusOK,
			find: []any{path.Root()},
		},
		{
			name: "current",
			node: ast.NewConst(ast.ConstCurrent),
			exp:  statusOK,
			find: []any{current},
		},
		{
			name:  "any_key",
			node:  ast.NewConst(ast.ConstAnyKey),
			value: map[string]any{"hi": "x", "there": "x"},
			exp:   statusOK,
			find:  []any{"x", "x"},
		},
		{
			name:  "any_key_array",
			node:  ast.NewConst(ast.ConstAnyKey),
			value: []any{"hi", "there"},
			exp:   statusNotFound,
			find:  []any{},
		},
		{
			name:   "any_key_array_unwrap",
			node:   ast.NewConst(ast.ConstAnyKey),
			value:  []any{"hi", "there"},
			unwrap: true,
			exp:    statusNotFound,
			find:   []any{},
		},
		{
			name:   "any_key_nested_array_unwrap",
			node:   ast.NewConst(ast.ConstAnyKey),
			value:  []any{"hi", "there", map[string]any{"x": int64(1), "y": int64(1)}},
			unwrap: true,
			exp:    statusOK,
			find:   []any{int64(1), int64(1)},
		},
		{
			name:  "any_array",
			node:  ast.NewConst(ast.ConstAnyArray),
			value: []any{"hi", "there"},
			exp:   statusOK,
			find:  []any{"hi", "there"},
		},
		{
			name: "last",
			node: ast.NewConst(ast.ConstLast),
			exp:  statusOK,
			find: []any{int64(3)},
		},
		{
			name:  "unknown_const",
			node:  ast.NewConst(ast.Constant(-1)),
			exp:   statusFailed,
			find:  []any{},
			err:   "exec invalid: Unknown ConstNode Constant(-1)",
			isErr: ErrInvalid,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Construct found.
			var found *valueList
			if tc.find != nil {
				found = newList()
			}

			// Construct executor.
			e := newTestExecutor(path, nil, true, false)
			e.root = path.Root()
			e.baseObject = base
			e.current = current
			e.innermostArraySize = 4

			// Execute execConstNode.
			res, err := e.execConstNode(ctx, tc.node, tc.value, found, tc.unwrap)
			a.Equal(tc.exp, res)

			// Base and current objects should be reset.
			a.Equal(base, e.baseObject)
			a.Equal(current, e.current)

			// Check found
			if tc.find != nil {
				a.Equal(tc.find, found.list)
			}

			// Check the error.
			if tc.isErr == nil {
				r.NoError(err)
			} else {
				r.EqualError(err, tc.err)
				r.ErrorIs(err, tc.isErr)
			}
		})
	}
}

func TestExecLiteralConst(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	r := require.New(t)
	ctx := context.Background()
	path, _ := parser.Parse("$")

	for _, tc := range []struct {
		name  string
		node  ast.Node
		find  []any
		exp   resultStatus
		err   string
		isErr error
	}{
		{
			name: "no_found",
			node: ast.NewConst(ast.ConstNull),
			exp:  statusOK,
		},
		{
			name: "null",
			node: ast.NewConst(ast.ConstNull),
			exp:  statusOK,
			find: []any{nil},
		},
		{
			name: "true",
			node: ast.NewConst(ast.ConstTrue),
			exp:  statusOK,
			find: []any{true},
		},
		{
			name: "false",
			node: ast.NewConst(ast.ConstFalse),
			exp:  statusOK,
			find: []any{false},
		},
		{
			name: "false_next",
			node: ast.LinkNodes([]ast.Node{ast.NewConst(ast.ConstFalse), ast.NewMethod(ast.MethodString)}),
			exp:  statusOK,
			find: []any{"false"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Construct found.
			var found *valueList
			if tc.find != nil {
				found = newList()
			}

			// Get the constant.
			node, ok := tc.node.(*ast.ConstNode)
			a.True(ok)

			// Construct executor.
			e := newTestExecutor(path, nil, true, false)

			// Execute execLiteralConst.
			res, err := e.execLiteralConst(ctx, node, found)
			a.Equal(tc.exp, res)

			// Check found
			if tc.find != nil {
				a.Equal(tc.find, found.list)
			}

			// Check the error.
			if tc.isErr == nil {
				r.NoError(err)
			} else {
				r.EqualError(err, tc.err)
				r.ErrorIs(err, tc.isErr)
			}
		})
	}
}

func TestExecAnyKey(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	r := require.New(t)
	ctx := context.Background()
	lax, _ := parser.Parse("$")
	strict, _ := parser.Parse("strict $")

	for _, tc := range []struct {
		name   string
		path   *ast.AST
		node   *ast.ConstNode
		value  any
		find   []any
		unwrap bool
		strict bool
		exp    resultStatus
		err    string
		isErr  error
	}{
		{
			name:  "any_key",
			path:  lax,
			node:  ast.NewConst(ast.ConstAnyKey),
			value: map[string]any{"hi": "x", "there": "x"},
			exp:   statusOK,
			find:  []any{"x", "x"},
		},
		{
			name:  "any_key_array",
			path:  lax,
			node:  ast.NewConst(ast.ConstAnyKey),
			value: []any{"hi", "there"},
			exp:   statusNotFound,
			find:  []any{},
		},
		{
			name:  "any_key_array_strict",
			path:  strict,
			node:  ast.NewConst(ast.ConstAnyKey),
			value: []any{"hi", "there"},
			exp:   statusFailed,
			find:  []any{},
			err:   "exec: jsonpath wildcard member accessor can only be applied to an object",
			isErr: ErrVerbose,
		},
		{
			name:   "any_key_array_unwrap",
			path:   lax,
			node:   ast.NewConst(ast.ConstAnyKey),
			value:  []any{"hi", "there"},
			unwrap: true,
			exp:    statusNotFound,
			find:   []any{},
		},
		{
			name:   "any_key_nested_array_unwrap",
			path:   lax,
			node:   ast.NewConst(ast.ConstAnyKey),
			value:  []any{"hi", "there", map[string]any{"x": int64(1), "y": int64(1)}},
			unwrap: true,
			exp:    statusOK,
			find:   []any{int64(1), int64(1)},
		},
		{
			name:  "any_key_scalar",
			path:  lax,
			node:  ast.NewConst(ast.ConstAnyKey),
			value: true,
			exp:   statusNotFound,
			find:  []any{},
		},
		{
			name:  "any_key_scalar_strict",
			path:  strict,
			node:  ast.NewConst(ast.ConstAnyKey),
			value: true,
			exp:   statusFailed,
			find:  []any{},
			err:   "exec: jsonpath wildcard member accessor can only be applied to an object",
			isErr: ErrVerbose,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Construct found.
			var found *valueList
			if tc.find != nil {
				found = newList()
			}

			// Construct executor.
			e := newTestExecutor(tc.path, nil, true, false)

			// Execute execAnyKey.
			res, err := e.execAnyKey(ctx, tc.node, tc.value, found, tc.unwrap)
			a.Equal(tc.exp, res)

			// Check found
			if tc.find != nil {
				a.Equal(tc.find, found.list)
			}

			// Check the error.
			if tc.isErr == nil {
				r.NoError(err)
			} else {
				r.EqualError(err, tc.err)
				r.ErrorIs(err, tc.isErr)
			}
		})
	}
}

func TestExecAnyArray(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	r := require.New(t)
	ctx := context.Background()
	lax, _ := parser.Parse("$")
	strict, _ := parser.Parse("strict $")

	for _, tc := range []struct {
		name   string
		node   ast.Node
		path   *ast.AST
		ignore bool
		value  any
		find   []any
		exp    resultStatus
		err    string
		isErr  error
	}{
		{
			name:  "array",
			node:  ast.NewConst(ast.ConstNull),
			path:  lax,
			value: []any{true, false, nil},
			exp:   statusOK,
			find:  []any{true, false, nil},
		},
		{
			name:  "array_next",
			node:  ast.LinkNodes([]ast.Node{ast.NewConst(ast.ConstNull), ast.NewMethod(ast.MethodString)}),
			path:  lax,
			value: []any{true, false, float64(98.6)},
			exp:   statusOK,
			find:  []any{"true", "false", "98.6"},
		},
		{
			name:  "array_next_err",
			node:  ast.LinkNodes([]ast.Node{ast.NewConst(ast.ConstNull), ast.NewMethod(ast.MethodString)}),
			path:  lax,
			value: []any{true, false, nil},
			exp:   statusFailed,
			find:  []any{"true", "false"},
			err:   "exec: jsonpath item method .string() can only be applied to a bool, string, numeric, or datetime value",
			isErr: ErrVerbose,
		},
		{
			name:  "auto_wrap",
			node:  ast.NewConst(ast.ConstNull),
			path:  lax,
			value: true,
			exp:   statusOK,
			find:  []any{true},
		},
		{
			name:   "no_auto_wrap_no_error",
			node:   ast.NewConst(ast.ConstNull),
			path:   strict,
			ignore: true,
			value:  true,
			exp:    statusNotFound,
			find:   []any{},
		},
		{
			name:  "no_auto_wrap_strict",
			node:  ast.NewConst(ast.ConstNull),
			path:  strict,
			value: true,
			exp:   statusFailed,
			find:  []any{},
			err:   "exec: jsonpath wildcard array accessor can only be applied to an array",
			isErr: ErrVerbose,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Construct found.
			var found *valueList
			if tc.find != nil {
				found = newList()
			}

			// Get the constant.
			node, ok := tc.node.(*ast.ConstNode)
			a.True(ok)

			// Construct executor.
			e := newTestExecutor(tc.path, nil, true, false)
			if tc.ignore {
				e.ignoreStructuralErrors = true
			}

			// Execute execAnyArray.
			res, err := e.execAnyArray(ctx, node, tc.value, found)
			a.Equal(tc.exp, res)

			// Check found
			if tc.find != nil {
				a.Equal(tc.find, found.list)
			}

			// Check the error.
			if tc.isErr == nil {
				r.NoError(err)
			} else {
				r.EqualError(err, tc.err)
				r.ErrorIs(err, tc.isErr)
			}
		})
	}
}

func TestExecLastConst(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	r := require.New(t)
	ctx := context.Background()
	path, _ := parser.Parse("$")

	for _, tc := range []struct {
		name  string
		node  ast.Node
		size  int
		find  []any
		exp   resultStatus
		err   string
		isErr error
	}{
		{
			name:  "outside_array_subscript",
			node:  ast.NewConst(ast.ConstLast),
			size:  -1,
			exp:   statusFailed,
			err:   "exec: evaluating jsonpath LAST outside of array subscript",
			isErr: ErrExecution,
		},
		{
			name: "size_4",
			node: ast.NewConst(ast.ConstLast),
			size: 4,
			exp:  statusOK,
			find: []any{int64(3)},
		},
		{
			name: "no_found",
			node: ast.NewConst(ast.ConstLast),
			size: 4,
			exp:  statusOK,
		},
		{
			name: "size_6",
			node: ast.NewConst(ast.ConstLast),
			size: 6,
			exp:  statusOK,
			find: []any{int64(5)},
		},
		{
			name: "size_4_next",
			node: ast.LinkNodes([]ast.Node{ast.NewConst(ast.ConstLast), ast.NewMethod(ast.MethodString)}),
			size: 4,
			exp:  statusOK,
			find: []any{"3"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Construct found.
			var found *valueList
			if tc.find != nil {
				found = newList()
			}

			// Get the constant.
			node, ok := tc.node.(*ast.ConstNode)
			a.True(ok)

			// Construct executor.
			e := newTestExecutor(path, nil, true, false)
			e.innermostArraySize = tc.size

			// Execute execLastConst.
			res, err := e.execLastConst(ctx, node, found)
			a.Equal(tc.exp, res)

			// Check found
			if tc.find != nil {
				a.Equal(tc.find, found.list)
			}

			// Check the error.
			if tc.isErr == nil {
				r.NoError(err)
			} else {
				r.EqualError(err, tc.err)
				r.ErrorIs(err, tc.isErr)
			}
		})
	}
}
