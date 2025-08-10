package exec

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/theory/sqljson/path/ast"
)

func TestPredOutcome(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		out  predOutcome
	}{
		{"FALSE", predFalse},
		{"TRUE", predTrue},
		{"UNKNOWN", predUnknown},
		{"UNKNOWN_PREDICATE_OUTCOME", predOutcome(255)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)

			a.Equal(tc.name, tc.out.String())
		})
	}

	t.Run("predFrom", func(t *testing.T) {
		t.Parallel()
		a := assert.New(t)

		a.Equal(predTrue, predFrom(true))
		a.Equal(predFalse, predFrom(false))
	})
}

func TestPredicateCallback(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	e := newTestExecutor(laxRootPath, nil, true, false)
	a.IsType((predicateCallback)(nil), predicateCallback(e.compareItems))
	a.IsType((predicateCallback)(nil), predicateCallback(executeStartsWith))
	a.IsType((predicateCallback)(nil), predicateCallback(e.executeLikeRegex))
}

func TestExecutePredicate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	rx, _ := ast.NewRegex(ast.NewConst(ast.ConstRoot), ".", "")

	for _, tc := range []struct {
		name     string
		path     *ast.AST
		pred     ast.Node
		left     ast.Node
		right    ast.Node
		value    any
		unwrap   bool
		callback func(e *Executor) predicateCallback
		exp      predOutcome
		err      string
		isErr    error
	}{
		{
			name:     "left_unknown",
			path:     laxRootPath,
			left:     ast.NewMethod(ast.MethodBigInt),
			value:    "hi",
			callback: func(_ *Executor) predicateCallback { return executeStartsWith },
			exp:      predUnknown,
		},
		{
			name:     "right_unknown",
			path:     laxRootPath,
			left:     ast.NewInteger("42"),
			right:    ast.NewMethod(ast.MethodBigInt),
			value:    "hi",
			callback: func(_ *Executor) predicateCallback { return executeStartsWith },
			exp:      predUnknown,
		},
		{
			name:     "left_and_right_compare",
			path:     laxRootPath,
			pred:     ast.NewBinary(ast.BinaryEqual, nil, nil),
			left:     ast.NewInteger("42"),
			right:    ast.NewInteger("42"),
			callback: func(e *Executor) predicateCallback { return e.compareItems },
			exp:      predTrue,
		},
		{
			name:     "left_and_right_no_compare",
			path:     laxRootPath,
			pred:     ast.NewBinary(ast.BinaryEqual, nil, nil),
			left:     ast.NewInteger("42"),
			right:    ast.NewInteger("43"),
			callback: func(e *Executor) predicateCallback { return e.compareItems },
			exp:      predFalse,
		},
		{
			name:     "left_only_regex",
			path:     laxRootPath,
			pred:     rx,
			left:     ast.NewString("hi"),
			callback: func(e *Executor) predicateCallback { return e.executeLikeRegex },
			exp:      predTrue,
		},
		{
			name:     "compare_error",
			path:     laxRootPath,
			pred:     rx,
			left:     ast.NewString("hi"),
			callback: func(e *Executor) predicateCallback { return e.compareItems },
			exp:      predUnknown,
			err:      `exec invalid: invalid node type *ast.RegexNode passed to compareItems`,
			isErr:    ErrInvalid,
		},
		{
			name:     "unknown_strict",
			path:     strictRootPath,
			pred:     rx,
			left:     ast.NewInteger("42"),
			callback: func(e *Executor) predicateCallback { return e.executeLikeRegex },
			exp:      predUnknown,
		},
		{
			name:     "unknown_lax",
			path:     laxRootPath,
			pred:     rx,
			left:     ast.NewInteger("42"),
			callback: func(e *Executor) predicateCallback { return e.executeLikeRegex },
			exp:      predUnknown,
		},
		{
			name:     "found_strict",
			path:     strictRootPath,
			pred:     rx,
			left:     ast.NewString("hi"),
			callback: func(e *Executor) predicateCallback { return e.executeLikeRegex },
			exp:      predTrue,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)
			r := require.New(t)

			e := newTestExecutor(tc.path, nil, true, false)
			cb := tc.callback(e)
			res, err := e.executePredicate(ctx, tc.pred, tc.left, tc.right, tc.value, tc.unwrap, cb)
			a.Equal(tc.exp, res)

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
