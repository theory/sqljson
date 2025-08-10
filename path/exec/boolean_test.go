package exec

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/theory/sqljson/path/ast"
	"github.com/theory/sqljson/path/parser"
)

//nolint:dupl
func TestExecuteBinaryBoolItem(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	for _, tc := range []struct {
		name  string
		path  string
		op    ast.BinaryOperator
		value any
		exp   predOutcome
		err   string
		isErr error
	}{
		{
			name:  "binary_and",
			path:  "$ == $ && $ == $",
			value: true,
			op:    ast.BinaryAnd,
			exp:   predTrue,
		},
		{
			name:  "binary_and_unknown_left",
			path:  "$ == $x && $ == $",
			value: true,
			op:    ast.BinaryAnd,
			exp:   predUnknown,
			err:   `exec: could not find jsonpath variable "x"`,
			isErr: ErrExecution,
		},
		{
			name:  "binary_and_unknown_right",
			path:  "$ == $ && $ == $x",
			value: true,
			op:    ast.BinaryAnd,
			exp:   predUnknown,
			err:   `exec: could not find jsonpath variable "x"`,
			isErr: ErrExecution,
		},
		{
			name:  "binary_or",
			path:  "$ == $ || $ == $",
			value: true,
			op:    ast.BinaryOr,
			exp:   predTrue,
		},
		{
			name:  "binary_or_both",
			path:  "$ == false || $ == $",
			value: true,
			op:    ast.BinaryOr,
			exp:   predTrue,
		},
		{
			name:  "binary_or_false",
			path:  "$ == false || $ == false",
			value: true,
			op:    ast.BinaryOr,
			exp:   predFalse,
		},
		{
			name:  "binary_or_unknown_left",
			path:  "$ == $x || $ == $",
			value: true,
			op:    ast.BinaryOr,
			exp:   predUnknown,
			err:   `exec: could not find jsonpath variable "x"`,
			isErr: ErrExecution,
		},
		{
			name:  "binary_or_unknown_right",
			path:  "$ == false || $ == $x",
			value: true,
			op:    ast.BinaryOr,
			exp:   predUnknown,
			err:   `exec: could not find jsonpath variable "x"`,
			isErr: ErrExecution,
		},
		{
			name:  "binary_eq_true",
			path:  "$ == $",
			value: true,
			op:    ast.BinaryEqual,
			exp:   predTrue,
		},
		{
			name:  "binary_eq_false",
			path:  "$ == false",
			value: true,
			op:    ast.BinaryEqual,
			exp:   predFalse,
		},
		{
			name:  "binary_eq_unknown",
			path:  "$ == $x",
			value: true,
			op:    ast.BinaryEqual,
			exp:   predUnknown,
			err:   `exec: could not find jsonpath variable "x"`,
			isErr: ErrExecution,
		},
		{
			name:  "binary_ne_false",
			path:  "$ != $",
			value: true,
			op:    ast.BinaryNotEqual,
			exp:   predFalse,
		},
		{
			name:  "binary_ne_true",
			path:  "$ != false",
			value: true,
			op:    ast.BinaryNotEqual,
			exp:   predTrue,
		},
		{
			name:  "binary_ne_unknown",
			path:  "$ != $x",
			value: true,
			op:    ast.BinaryNotEqual,
			exp:   predUnknown,
			err:   `exec: could not find jsonpath variable "x"`,
			isErr: ErrExecution,
		},
		{
			name:  "binary_lt_true",
			path:  "$ < 3",
			value: int64(1),
			op:    ast.BinaryLess,
			exp:   predTrue,
		},
		{
			name:  "binary_lt_false",
			path:  "$ < 3",
			value: int64(3),
			op:    ast.BinaryLess,
			exp:   predFalse,
		},
		{
			name:  "binary_lt_unknown",
			path:  "$ < $x",
			value: int64(3),
			op:    ast.BinaryLess,
			exp:   predUnknown,
			err:   `exec: could not find jsonpath variable "x"`,
			isErr: ErrExecution,
		},
		{
			name:  "binary_gt_true",
			path:  "$ > 3",
			value: int64(5),
			op:    ast.BinaryGreater,
			exp:   predTrue,
		},
		{
			name:  "binary_gt_false",
			path:  "$ > 3",
			value: int64(3),
			op:    ast.BinaryGreater,
			exp:   predFalse,
		},
		{
			name:  "binary_gt_unknown",
			path:  "$ > $x",
			value: int64(3),
			op:    ast.BinaryGreater,
			exp:   predUnknown,
			err:   `exec: could not find jsonpath variable "x"`,
			isErr: ErrExecution,
		},
		{
			name:  "binary_le_true",
			path:  "$ <= 3",
			value: int64(2),
			op:    ast.BinaryLessOrEqual,
			exp:   predTrue,
		},
		{
			name:  "binary_le_true_2",
			path:  "$ <= 3",
			value: int64(3),
			op:    ast.BinaryLessOrEqual,
			exp:   predTrue,
		},
		{
			name:  "binary_le_false",
			path:  "$ <= 3",
			value: int64(4),
			op:    ast.BinaryLessOrEqual,
			exp:   predFalse,
		},
		{
			name:  "binary_le_unknown",
			path:  "$ <= $x",
			value: int64(3),
			op:    ast.BinaryLessOrEqual,
			exp:   predUnknown,
			err:   `exec: could not find jsonpath variable "x"`,
			isErr: ErrExecution,
		},
		{
			name:  "binary_ge_true",
			path:  "$ >= 3",
			value: int64(4),
			op:    ast.BinaryGreaterOrEqual,
			exp:   predTrue,
		},
		{
			name:  "binary_le_true_2",
			path:  "$ >= 3",
			value: int64(3),
			op:    ast.BinaryGreaterOrEqual,
			exp:   predTrue,
		},
		{
			name:  "binary_le_false",
			path:  "$ >= 3",
			value: int64(2),
			op:    ast.BinaryGreaterOrEqual,
			exp:   predFalse,
		},
		{
			name:  "binary_le_unknown",
			path:  "$ >= $x",
			value: int64(3),
			op:    ast.BinaryGreaterOrEqual,
			exp:   predUnknown,
			err:   `exec: could not find jsonpath variable "x"`,
			isErr: ErrExecution,
		},
		{
			name:  "starts_with_true",
			path:  `$ starts with "a"`,
			value: "abc",
			op:    ast.BinaryStartsWith,
			exp:   predTrue,
		},
		{
			name:  "starts_with_false",
			path:  `$ starts with "b"`,
			value: "abc",
			op:    ast.BinaryStartsWith,
			exp:   predFalse,
		},
		{
			name:  "starts_with_unknown",
			path:  "$ starts with $x",
			value: true,
			op:    ast.BinaryStartsWith,
			exp:   predUnknown,
			err:   `exec: could not find jsonpath variable "x"`,
			isErr: ErrExecution,
		},
		{
			name:  "unsupported_binary",
			path:  "$ + 4",
			value: true,
			op:    ast.BinaryAdd,
			exp:   predUnknown,
			err:   `exec invalid: invalid jsonpath boolean operator +`,
			isErr: ErrInvalid,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)
			r := require.New(t)

			// Parse the path and make sure the root node is what we expect to
			// be testing.
			path, err := parser.Parse(tc.path)
			r.NoError(err)
			node, ok := path.Root().(*ast.BinaryNode)
			r.True(ok)
			a.Equal(tc.op, node.Operator())

			// Test executeBinaryBoolItem with the root node set to tc.value.
			e := newTestExecutor(path, nil, true, false)
			e.root = tc.value
			res, err := e.executeBinaryBoolItem(ctx, node, tc.value)
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

//nolint:dupl
func TestExecuteUnaryBoolItem(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	for _, tc := range []struct {
		name  string
		path  string
		op    ast.UnaryOperator
		value any
		exp   predOutcome
		err   string
		isErr error
	}{
		{
			name:  "unary_not_true",
			path:  "!($ == false)",
			value: true,
			op:    ast.UnaryNot,
			exp:   predTrue,
		},
		{
			name:  "unary_not_false",
			path:  "!($ == true)",
			value: true,
			op:    ast.UnaryNot,
			exp:   predFalse,
		},
		{
			name:  "unary_not_unknown",
			path:  "!($ == $x)",
			value: true,
			op:    ast.UnaryNot,
			exp:   predUnknown,
			err:   `exec: could not find jsonpath variable "x"`,
			isErr: ErrExecution,
		},
		{
			name:  "unary_is_unknown_true",
			path:  "($ == $x) is unknown",
			value: true,
			op:    ast.UnaryIsUnknown,
			exp:   predTrue,
		},
		{
			name:  "unary_is_unknown_false",
			path:  "($ == $) is unknown",
			value: true,
			op:    ast.UnaryIsUnknown,
			exp:   predFalse,
		},
		{
			name:  "unary_is_unknown_false_false",
			path:  "($ == $) is unknown",
			value: false,
			op:    ast.UnaryIsUnknown,
			exp:   predFalse,
		},
		{
			name:  "unary_exists_true",
			path:  "exists ($)",
			value: true,
			op:    ast.UnaryExists,
			exp:   predTrue,
		},
		{
			name:  "unary_exists_false",
			path:  "exists ($.x)",
			value: true,
			op:    ast.UnaryExists,
			exp:   predFalse,
		},
		{
			name:  "unary_exists_unknown",
			path:  "exists ($x)",
			value: true,
			op:    ast.UnaryExists,
			exp:   predUnknown,
			err:   `exec: could not find jsonpath variable "x"`,
			isErr: ErrExecution,
		},
		{
			name:  "unary_exists_strict_true",
			path:  "strict exists ($[*])",
			value: []any{"x", "y"},
			op:    ast.UnaryExists,
			exp:   predTrue,
		},
		{
			name:  "unary_exists_strict_false",
			path:  "strict exists ($[*])",
			value: []any{},
			op:    ast.UnaryExists,
			exp:   predFalse,
		},
		{
			name:  "unary_exists_strict_unknown",
			path:  "strict exists ($x[*])",
			value: []any{},
			op:    ast.UnaryExists,
			exp:   predUnknown,
			err:   `exec: could not find jsonpath variable "x"`,
			isErr: ErrExecution,
		},
		{
			name:  "unary_not_boolean",
			path:  "-$",
			op:    ast.UnaryMinus,
			exp:   predUnknown,
			err:   `exec invalid: invalid jsonpath boolean operator -`,
			isErr: ErrInvalid,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)
			r := require.New(t)

			// Parse the path and make sure the root node is what we expect to
			// be testing.
			path, err := parser.Parse(tc.path)
			r.NoError(err)
			node, ok := path.Root().(*ast.UnaryNode)
			r.True(ok)
			a.Equal(tc.op, node.Operator())

			// Test executeUnaryBoolItem with the root node set to tc.value.
			e := newTestExecutor(path, nil, true, false)
			e.root = tc.value
			res, err := e.executeUnaryBoolItem(ctx, node, tc.value)
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

func TestExecuteBoolItem(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	for _, tc := range []struct {
		name        string
		path        string
		value       any
		canHaveNext bool
		exp         predOutcome
		err         string
		isErr       error
	}{
		{
			name:  "no_next",
			path:  "$ ?($ == $).x",
			value: true,
			exp:   predUnknown,
			err:   `exec invalid: boolean jsonpath item cannot have next item`,
			isErr: ErrInvalid,
		},
		{
			name:        "next_ok_true",
			path:        "($.x == $.x).x",
			value:       map[string]any{"x": true},
			canHaveNext: true,
			exp:         predTrue,
		},
		{
			name:        "next_ok_false",
			path:        "($.x != $.x).x",
			value:       map[string]any{"x": true},
			canHaveNext: true,
			exp:         predFalse,
		},
		{
			name:        "next_ok_unknown",
			path:        "($.x == $x).x",
			value:       map[string]any{"x": true},
			canHaveNext: true,
			exp:         predUnknown,
			err:         `exec: could not find jsonpath variable "x"`,
			isErr:       ErrExecution,
		},
		{
			name:  "binary",
			path:  "$ == $",
			value: true,
			exp:   predTrue,
		},
		{
			name:  "unary",
			path:  "exists ($)",
			value: true,
			exp:   predTrue,
		},
		{
			name:  "regex",
			path:  `$ like_regex "^a"`,
			value: "abc",
			exp:   predTrue,
		},
		{
			name:  "invalid_boolean",
			path:  `$`,
			value: true,
			exp:   predUnknown,
			err:   `exec invalid: invalid boolean jsonpath item type: $`,
			isErr: ErrInvalid,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)
			r := require.New(t)

			// Parse the path.
			path, err := parser.Parse(tc.path)
			r.NoError(err)

			// Test executeBoolItem with the root node set to tc.value.
			e := newTestExecutor(path, nil, true, false)
			e.root = tc.value
			res, err := e.executeBoolItem(ctx, path.Root(), tc.value, tc.canHaveNext)
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

func TestAppendBoolResult(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	for _, tc := range []struct {
		name    string
		path    string
		found   []any
		passOut predOutcome
		passErr error
		exp     resultStatus
		err     string
		isErr   error
	}{
		{
			name:    "passed_error",
			path:    "$",
			passErr: fmt.Errorf("%w: OOPS", ErrExecution),
			exp:     statusFailed,
			err:     `exec: OOPS`,
			isErr:   ErrExecution,
		},
		{
			name:    "pass_unknown",
			path:    "$",
			passOut: predUnknown,
			exp:     statusOK,
		},
		{
			name:    "pass_unknown_found",
			path:    "$",
			passOut: predUnknown,
			found:   []any{nil},
			exp:     statusOK,
		},
		{
			name: "no_found_ok",
			path: "$",
			exp:  statusOK,
		},
		{
			name:    "true_no_next",
			path:    "$",
			passOut: predTrue,
			exp:     statusOK,
		},
		{
			name:    "false_no_next",
			path:    "$",
			passOut: predFalse,
			exp:     statusOK,
		},
		{
			name:    "okay_next",
			path:    "($ == $).x",
			passOut: predTrue,
			exp:     statusNotFound,
		},
		{
			name:    "add_ok",
			path:    "$ + $",
			passOut: predTrue,
			found:   []any{true},
			exp:     statusOK,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)
			r := require.New(t)

			// Parse the path.
			path, err := parser.Parse(tc.path)
			r.NoError(err)

			// Construct found.
			var found *valueList
			if tc.found != nil {
				found = newList()
			}

			// Execute appendBoolResult.
			e := newTestExecutor(path, nil, true, false)
			res, err := e.appendBoolResult(ctx, path.Root(), found, tc.passOut, tc.passErr)
			a.Equal(tc.exp, res)
			if tc.found != nil {
				a.Equal(tc.found, found.list)
			}

			if tc.isErr == nil {
				r.NoError(err)
			} else {
				r.EqualError(err, tc.err)
				r.ErrorIs(err, tc.isErr)
			}
		})
	}
}

func TestExecuteNestedBoolItem(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	for _, tc := range []struct {
		name    string
		path    string
		root    any
		current any
		value   any
		exp     predOutcome
		err     string
		isErr   error
	}{
		{
			name:    "switch_current",
			path:    "$ == $",
			root:    true,
			current: "foo",
			value:   "bar",
			exp:     predTrue,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)
			r := require.New(t)

			// Parse the path.
			path, err := parser.Parse(tc.path)
			r.NoError(err)

			// Execute executeNestedBoolItem.
			e := newTestExecutor(path, nil, true, false)
			e.root = tc.root
			e.current = tc.current
			res, err := e.executeNestedBoolItem(ctx, path.Root(), tc.value)
			a.Equal(tc.exp, res)
			a.Equal(tc.current, e.current)
			if tc.isErr == nil {
				r.NoError(err)
			} else {
				r.EqualError(err, tc.err)
				r.ErrorIs(err, tc.isErr)
			}
		})
	}
}
