package exec

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/theory/sqljson/path/ast"
	"github.com/theory/sqljson/path/parser"
	"github.com/theory/sqljson/path/types"
)

func TestQuery(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	for _, tc := range []struct {
		test  string
		path  string
		value any
		vars  Vars
		throw bool
		useTZ bool
		exp   resultStatus
		find  []any
		err   string
		isErr error
	}{
		{
			test:  "lax_root",
			path:  "$",
			value: "hi",
			exp:   statusOK,
			find:  []any{"hi"},
		},
		{
			test:  "var_method",
			path:  "strict $x.string()",
			value: "hi",
			vars:  Vars{"x": int64(42)},
			exp:   statusOK,
			find:  []any{"42"},
		},
		{
			test:  "no_var",
			path:  "strict $x",
			value: "hi",
			exp:   statusFailed,
			err:   `exec: could not find jsonpath variable "x"`,
			isErr: ErrExecution,
		},
		{
			test:  "use_tz",
			path:  "$.time()",
			value: "12:42:53+01",
			useTZ: true,
			exp:   statusOK,
			find:  []any{types.NewTime(time.Date(0, 1, 1, 12, 42, 53, 0, time.UTC))},
		},
		{
			test:  "no_tz",
			path:  "$.time()",
			value: "12:42:53+01",
			useTZ: false,
			exp:   statusFailed,
			err:   `exec: cannot convert value from timetz to time without time zone usage.` + tzHint,
			isErr: ErrExecution,
		},
		{
			test:  "strict_root",
			path:  "strict $",
			value: "hi",
			exp:   statusOK,
			find:  []any{"hi"},
		},
		{
			test:  "filtered_not_found",
			path:  "$ ?(@ == 1)",
			value: "hi",
			exp:   statusNotFound,
			find:  []any{},
		},
		{
			test:  "strict filtered_not_found",
			path:  "strict $ ?(@ == 1)",
			value: "hi",
			exp:   statusNotFound,
			find:  []any{},
		},
		{
			test:  "filtered_subset",
			path:  "$ ?(@ >= 2)",
			value: []any{int64(1), int64(3), int64(4), int64(2), int64(0), int64(99)},
			exp:   statusOK,
			find:  []any{int64(3), int64(4), int64(2), int64(99)},
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)
			r := require.New(t)

			// Set up executor.
			path, err := parser.Parse(tc.path)
			r.NoError(err)
			e := newTestExecutor(path, tc.vars, tc.throw, tc.useTZ)
			e.root = tc.value
			e.current = tc.value

			// Start with list.
			vals := newList()
			res, err := e.query(ctx, vals, e.path.Root(), tc.value)
			a.Equal(tc.exp, res)

			// Check the error and list.
			if tc.isErr == nil {
				r.NoError(err)
				a.Equal(tc.find, vals.list)
			} else {
				r.EqualError(err, tc.err)
				r.ErrorIs(err, tc.isErr)
				a.Empty(vals.list)
			}

			// Try without list (exists).
			res, err = e.query(ctx, nil, e.path.Root(), tc.value)
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

func TestExecuteItem(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	for _, tc := range []struct {
		test  string
		path  string
		value any
		exp   resultStatus
		find  []any
		err   string
		isErr error
	}{
		{
			test:  "root",
			path:  "$",
			value: true,
			exp:   statusOK,
			find:  []any{true},
		},
		{
			test:  "strict_root",
			path:  "strict $",
			value: true,
			exp:   statusOK,
			find:  []any{true},
		},
		{
			test:  "unwrap",
			path:  "$.string()",
			value: []any{int64(42), true},
			exp:   statusOK,
			find:  []any{"42", "true"},
		},
		{
			test:  "strict_no_unwrap",
			path:  "strict $.string()",
			value: []any{int64(42), true},
			exp:   statusFailed,
			err:   `exec: jsonpath item method .string() can only be applied to a boolean, string, numeric, or datetime value`,
			isErr: ErrVerbose,
		},
		{
			test:  "filtered_subset",
			path:  "$ ?(@ >= 2)",
			value: []any{int64(1), int64(3), int64(4), int64(2), int64(0), int64(99)},
			exp:   statusOK,
			find:  []any{int64(3), int64(4), int64(2), int64(99)},
		},
		{
			test:  "filtered_not_found",
			path:  "$ ?(@ == 1)",
			value: "hi",
			exp:   statusNotFound,
			find:  []any{},
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)
			r := require.New(t)

			// Set up executor.
			path, err := parser.Parse(tc.path)
			r.NoError(err)
			e := newTestExecutor(path, nil, true, false)
			e.root = tc.value
			e.current = tc.value

			// Start with list.
			vals := newList()
			res, err := e.executeItem(ctx, e.path.Root(), tc.value, vals)
			a.Equal(tc.exp, res)

			// Check the error and list.
			if tc.isErr == nil {
				r.NoError(err)
				a.Equal(tc.find, vals.list)
			} else {
				r.EqualError(err, tc.err)
				r.ErrorIs(err, tc.isErr)
				a.Empty(vals.list)
			}

			// Try without list (exists).
			res, err = e.executeItem(ctx, e.path.Root(), tc.value, nil)
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

func TestExecuteItemOptUnwrapResult(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	for _, tc := range []struct {
		test   string
		path   string
		value  any
		unwrap bool
		exp    resultStatus
		find   []any
		err    string
		isErr  error
	}{
		{
			test:  "root",
			path:  "$",
			value: true,
			exp:   statusOK,
			find:  []any{true},
		},
		{
			test:  "strict_root",
			path:  "strict $",
			value: true,
			exp:   statusOK,
			find:  []any{true},
		},
		{
			test:   "unwrap",
			path:   "$.string()",
			value:  []any{int64(42), true},
			unwrap: true,
			exp:    statusOK,
			find:   []any{"42", "true"},
		},
		{
			test:   "unwrap_strict",
			path:   "strict $.string()",
			value:  []any{int64(42), true},
			unwrap: true,
			exp:    statusFailed,
			err:    `exec: jsonpath item method .string() can only be applied to a boolean, string, numeric, or datetime value`,
			isErr:  ErrVerbose,
		},
		{
			test:   "unwrap_error",
			path:   "$.integer()",
			value:  []any{true},
			unwrap: true,
			exp:    statusFailed,
			err:    `exec: jsonpath item method .integer() can only be applied to a string or numeric value`,
			isErr:  ErrVerbose,
		},
		{
			test:  "no_unwrap_lax",
			path:  "$.string()",
			value: []any{int64(42), true},
			exp:   statusOK,
			find:  []any{"42", "true"},
		},
		{
			test:   "nested_unwrap",
			path:   "$",
			value:  []any{int64(42), []any{true, float64(98.6)}},
			unwrap: true,
			exp:    statusOK,
			find:   []any{int64(42), []any{true, float64(98.6)}},
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)
			r := require.New(t)

			// Set up executor.
			path, err := parser.Parse(tc.path)
			r.NoError(err)
			e := newTestExecutor(path, nil, true, false)
			e.root = tc.value
			e.current = tc.value

			// Execute.
			vals := newList()
			res, err := e.executeItemOptUnwrapResult(ctx, e.path.Root(), tc.value, tc.unwrap, vals)
			a.Equal(tc.exp, res)

			// Check the error and list.
			if tc.isErr == nil {
				r.NoError(err)
				a.Equal(tc.find, vals.list)
			} else {
				r.EqualError(err, tc.err)
				r.ErrorIs(err, tc.isErr)
				a.Empty(vals.list)
			}

			// Try silent.
			vals = newList()
			verbose := e.verbose
			res, err = e.executeItemOptUnwrapResultSilent(ctx, e.path.Root(), tc.value, tc.unwrap, vals)
			a.Equal(tc.exp, res)
			a.Equal(verbose, e.verbose)
			r.NoError(err)
		})
	}
}

func TestExecuteItemOptUnwrapTarget(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	rx, _ := ast.NewRegex(ast.NewConst(ast.ConstRoot), "x", "")
	type wrapNode struct{ ast.Node }

	for _, tc := range []struct {
		test   string
		cancel bool
		node   ast.Node
		value  any
		unwrap bool
		vars   Vars
		exp    resultStatus
		find   []any
		err    string
		isErr  error
	}{
		{
			test:   "cancel",
			cancel: true,
			node:   ast.NewConst(ast.ConstRoot),
			value:  true,
			exp:    statusFailed,
			err:    "exec: context canceled",
			isErr:  ErrExecution,
		},
		{
			test:  "const",
			node:  ast.NewConst(ast.ConstRoot),
			value: true,
			exp:   statusOK,
			find:  []any{true},
		},
		{
			test: "string",
			node: ast.NewString("hi"),
			exp:  statusOK,
			find: []any{"hi"},
		},
		{
			test: "integer",
			node: ast.NewInteger("42"),
			exp:  statusOK,
			find: []any{int64(42)},
		},
		{
			test: "numeric",
			node: ast.NewNumeric("98.6"),
			exp:  statusOK,
			find: []any{float64(98.6)},
		},
		{
			test: "variable",
			node: ast.NewVariable("x"),
			vars: Vars{"x": "hi"},
			exp:  statusOK,
			find: []any{"hi"},
		},
		{
			test:  "key",
			node:  ast.NewKey("x"),
			value: map[string]any{"x": "hi"},
			exp:   statusOK,
			find:  []any{"hi"},
		},
		{
			test:  "binary",
			node:  ast.NewBinary(ast.BinaryAdd, ast.NewConst(ast.ConstRoot), ast.NewConst(ast.ConstRoot)),
			value: int64(21),
			exp:   statusOK,
			find:  []any{int64(42)},
		},
		{
			test: "unary",
			node: ast.NewUnary(ast.UnaryMinus, ast.NewInteger("42")),
			exp:  statusOK,
			find: []any{int64(-42)},
		},
		{
			test:  "regex",
			node:  rx,
			value: "hex",
			exp:   statusOK,
			find:  []any{true},
		},
		{
			test:  "method",
			node:  ast.NewMethod(ast.MethodString),
			value: true,
			exp:   statusOK,
			find:  []any{"true"},
		},
		{
			test:  "any",
			node:  ast.NewAny(0, -1),
			value: map[string]any{"x": "y"},
			exp:   statusOK,
			find:  []any{map[string]any{"x": "y"}, "y"},
		},
		{
			test: "array_index",
			node: ast.NewArrayIndex([]ast.Node{
				ast.NewBinary(ast.BinarySubscript, ast.NewInteger("1"), ast.NewInteger("2")),
			}),
			value: []any{"x", "y", "z"},
			exp:   statusOK,
			find:  []any{"y", "z"},
		},
		{
			test:  "unknown_node",
			node:  wrapNode{},
			exp:   statusFailed,
			err:   `exec invalid: Unknown node type exec.wrapNode`,
			isErr: ErrInvalid,
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)
			r := require.New(t)

			// Set up executor.
			e := newTestExecutor(laxRootPath, tc.vars, true, false)
			e.root = tc.value
			e.current = tc.value

			// Execute.
			vals := newList()
			var (
				res resultStatus
				err error
			)
			if tc.cancel {
				canceledCtx, cancel := context.WithCancel(ctx)
				cancel()
				res, err = e.executeItemOptUnwrapTarget(canceledCtx, tc.node, tc.value, vals, tc.unwrap)
				r.ErrorIs(err, context.Canceled)
			} else {
				res, err = e.executeItemOptUnwrapTarget(ctx, tc.node, tc.value, vals, tc.unwrap)
			}
			a.Equal(tc.exp, res)

			// Check the error and list.
			if tc.isErr == nil {
				r.NoError(err)
				a.Equal(tc.find, vals.list)
			} else {
				r.EqualError(err, tc.err)
				r.ErrorIs(err, tc.isErr)
				a.Empty(vals.list)
			}
		})
	}
}

func TestExecuteNextItem(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	for _, tc := range []struct {
		test  string
		cur   ast.Node
		next  ast.Node
		value any
		exp   resultStatus
		find  []any
		err   string
		isErr error
	}{
		{
			test:  "nil_nil",
			value: true,
			exp:   statusOK,
			find:  []any{true},
		},
		{
			test:  "nil_next",
			next:  ast.NewMethod(ast.MethodString),
			value: true,
			exp:   statusOK,
			find:  []any{"true"},
		},
		{
			test:  "current_next",
			cur:   ast.NewMethod(ast.MethodBoolean),
			next:  ast.NewMethod(ast.MethodString),
			value: "t",
			exp:   statusOK,
			find:  []any{"t"},
		},
		{
			test:  "current_next_nil",
			next:  ast.NewConst(ast.ConstRoot),
			value: true,
			exp:   statusOK,
			find:  []any{true},
		},
		{
			test:  "current_next_method",
			next:  ast.LinkNodes([]ast.Node{ast.NewConst(ast.ConstRoot), ast.NewMethod(ast.MethodString)}),
			value: true,
			exp:   statusOK,
			find:  []any{"true"},
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)
			r := require.New(t)

			// Set up executor.
			e := newTestExecutor(laxRootPath, nil, true, false)
			e.root = tc.value
			e.current = tc.value

			// Execute.
			vals := newList()
			res, err := e.executeNextItem(ctx, tc.cur, tc.next, tc.value, vals)
			a.Equal(tc.exp, res)

			// Check the error and list.
			if tc.isErr == nil {
				r.NoError(err)
				a.Equal(tc.find, vals.list)
			} else {
				r.EqualError(err, tc.err)
				r.ErrorIs(err, tc.isErr)
				a.Empty(vals.list)
			}

			// Try without found list.
			res, err = e.executeNextItem(ctx, tc.cur, tc.next, tc.value, nil)
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
