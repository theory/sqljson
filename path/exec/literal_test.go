package exec

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/theory/sqljson/path/ast"
	"github.com/theory/sqljson/path/parser"
)

func TestExecLiteral(t *testing.T) {
	t.Parallel()
	path, _ := parser.Parse("$")
	ctx := context.Background()

	for _, tc := range []struct {
		name  string
		node  ast.Node
		value any
		exp   resultStatus
		err   string
		isErr error
	}{
		{
			name:  "string",
			node:  ast.NewString("hi"),
			value: "hi",
			exp:   statusOK,
		},
		{
			name:  "integer",
			node:  ast.NewInteger("42"),
			value: int64(42),
			exp:   statusOK,
		},
		{
			name:  "float",
			node:  ast.NewNumeric("98.6"),
			value: float64(98.6),
			exp:   statusOK,
		},
		{
			name:  "error",
			node:  ast.LinkNodes([]ast.Node{ast.NewString("hi"), ast.NewMethod(ast.MethodInteger)}),
			err:   "exec: jsonpath item method .integer() can only be applied to a string or numeric value",
			isErr: ErrExecution,
			exp:   statusFailed,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)
			r := require.New(t)

			e := newTestExecutor(path, nil, true, false)
			list := newList()
			res, err := e.execLiteral(ctx, tc.node, tc.value, list)
			a.Equal(tc.exp, res)

			if tc.isErr == nil {
				r.NoError(err)
				a.Equal([]any{tc.value}, list.list)
			} else {
				r.EqualError(err, tc.err)
				r.ErrorIs(err, tc.isErr)
				a.Empty(list.list)
			}

			// Test with nil found.
			res, err = e.execLiteral(ctx, tc.node, tc.value, nil)
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

func TestExecVariable(t *testing.T) {
	t.Parallel()
	path, _ := parser.Parse("$")
	ctx := context.Background()

	// Offset of object in a slice is non-determinate, so calculate it at runtime.
	vars := Vars{"x": map[string]any{"y": "hi"}}
	xID := 10000000000 + deltaBetween(vars, vars["x"])

	for _, tc := range []struct {
		name  string
		vars  Vars
		node  ast.Node
		exp   resultStatus
		find  any
		err   string
		isErr error
	}{
		{
			name: "var_exists",
			vars: Vars{"x": "hi"},
			node: ast.NewVariable("x"),
			exp:  statusOK,
			find: "hi",
		},
		{
			name:  "var_not_exists",
			vars:  Vars{"x": "hi"},
			node:  ast.NewVariable("y"),
			err:   `exec: could not find jsonpath variable "y"`,
			isErr: ErrExecution,
			exp:   statusFailed,
		},
		{
			name: "var_exists_next",
			vars: Vars{"x": int64(42)},
			node: ast.LinkNodes([]ast.Node{ast.NewVariable("x"), ast.NewMethod(ast.MethodString)}),
			exp:  statusOK,
			find: "42",
		},
		{
			name: "var_exists_next_keyvalue",
			vars: vars,
			node: ast.LinkNodes([]ast.Node{ast.NewVariable("x"), ast.NewMethod(ast.MethodKeyValue)}),
			exp:  statusOK,
			find: map[string]any{"id": xID, "key": "y", "value": "hi"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)
			r := require.New(t)

			// Make sure we have a variable node.
			node, ok := tc.node.(*ast.VariableNode)
			r.True(ok)

			// Set up an executor.
			e := newTestExecutor(path, nil, true, false)
			e.vars = tc.vars

			// Test execVariable with a list.
			list := newList()
			res, err := e.execVariable(ctx, node, list)
			a.Equal(tc.exp, res)
			// Root ID 0 should be restored.
			a.Equal(0, e.baseObject.id)

			// Check the error and list.
			if tc.isErr == nil {
				r.NoError(err)
				a.Equal([]any{tc.find}, list.list)
			} else {
				r.EqualError(err, tc.err)
				r.ErrorIs(err, tc.isErr)
				a.Empty(list.list)
			}

			// Test with nil found.
			res, err = e.execVariable(ctx, node, nil)
			a.Equal(tc.exp, res)
			// Root ID 0 should be restored.
			a.Equal(0, e.baseObject.id)

			// Check the error and list.
			if tc.isErr == nil {
				r.NoError(err)
			} else {
				r.EqualError(err, tc.err)
				r.ErrorIs(err, tc.isErr)
			}
		})
	}
}

func TestExecKeyNode(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	lax, _ := parser.Parse("$")
	strict, _ := parser.Parse("strict $")

	for _, tc := range []struct {
		name   string
		path   *ast.AST
		node   ast.Node
		value  any
		unwrap bool
		silent bool
		exp    resultStatus
		find   []any
		err    string
		isErr  error
	}{
		{
			name:  "find_key_string",
			path:  lax,
			node:  ast.NewKey("x"),
			value: map[string]any{"x": "hi"},
			exp:   statusOK,
			find:  []any{"hi"},
		},
		{
			name:  "find_key_array",
			path:  lax,
			node:  ast.NewKey("y"),
			value: map[string]any{"y": []any{"go"}},
			exp:   statusOK,
			find:  []any{[]any{"go"}},
		},
		{
			name:  "find_key_obj",
			path:  lax,
			node:  ast.NewKey("z"),
			value: map[string]any{"z": map[string]any{"a": "go"}},
			exp:   statusOK,
			find:  []any{map[string]any{"a": "go"}},
		},
		{
			name:  "no_such_key_lax",
			path:  lax,
			node:  ast.NewKey("y"),
			value: map[string]any{"x": "hi"},
			exp:   statusNotFound,
			find:  []any{},
		},
		{
			name:  "no_such_key_strict",
			path:  strict,
			node:  ast.NewKey("y"),
			value: map[string]any{"x": "hi"},
			exp:   statusFailed,
			err:   `exec: JSON object does not contain key "y"`,
			isErr: ErrVerbose,
		},
		{
			name:   "no_such_key_strict_silent",
			path:   strict,
			node:   ast.NewKey("y"),
			silent: true,
			value:  map[string]any{"x": "hi"},
			exp:    statusFailed,
			find:   []any{},
		},
		{
			name:  "not_an_object_lax",
			path:  lax,
			node:  ast.NewKey("y"),
			value: []any{"hi"},
			exp:   statusNotFound,
			find:  []any{},
		},
		{
			name:  "not_an_object_strict",
			path:  strict,
			node:  ast.NewKey("y"),
			value: []any{"hi"},
			exp:   statusFailed,
			err:   `exec: jsonpath member accessor can only be applied to an object`,
			isErr: ErrVerbose,
		},
		{
			name:   "unwrap_array",
			path:   lax,
			node:   ast.NewKey("y"),
			value:  []any{map[string]any{"y": "arg"}},
			unwrap: true,
			exp:    statusOK,
			find:   []any{"arg"},
		},
		{
			name:  "find_key_with_next",
			path:  lax,
			node:  ast.LinkNodes([]ast.Node{ast.NewKey("x"), ast.NewKey("y")}),
			value: map[string]any{"x": map[string]any{"y": "hi"}},
			exp:   statusOK,
			find:  []any{"hi"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)
			r := require.New(t)

			// Make sure we have a key node.
			node, ok := tc.node.(*ast.KeyNode)
			r.True(ok)

			// Set up an executor.
			e := newTestExecutor(tc.path, nil, true, false)
			e.verbose = !tc.silent

			// Test execKeyNode with a list.
			list := newList()
			res, err := e.execKeyNode(ctx, node, tc.value, list, tc.unwrap)
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
