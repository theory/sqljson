package exec

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/theory/sqljson/path/ast"
	"github.com/theory/sqljson/path/parser"
	"github.com/theory/sqljson/path/types"
)

func TestResultStatus(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		test string
		res  resultStatus
	}{
		{"OK", statusOK},
		{"NOT_FOUND", statusNotFound},
		{"FAILED", statusFailed},
		{"UNKNOWN_RESULT_STATUS", resultStatus(255)},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)

			a.Equal(tc.test, tc.res.String())
			a.Equal(tc.res == statusFailed, tc.res.failed())
		})
	}
}

func TestValueList(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	list := newList()
	a.NotNil(list)
	a.True(list.isEmpty())
	a.Equal(1, cap(list.list))

	list.append("foo")
	a.False(list.isEmpty())
	a.Len(list.list, 1)
	a.Equal(1, cap(list.list))

	list.append(42)
	a.False(list.isEmpty())
	a.Len(list.list, 2)
	a.Equal(2, cap(list.list))
}

func TestOptions(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		test string
		opt  Option
		exp  *Executor
	}{
		{
			test: "vars",
			opt:  WithVars(Vars{"foo": 1}),
			exp:  &Executor{verbose: true, vars: Vars{"foo": 1}},
		},
		{
			test: "vars_nested",
			opt:  WithVars(Vars{"foo": 1, "bar": []any{1, 2}}),
			exp:  &Executor{verbose: true, vars: Vars{"foo": 1, "bar": []any{1, 2}}},
		},
		{
			test: "tz",
			opt:  WithTZ(),
			exp:  &Executor{verbose: true, useTZ: true},
		},
		{
			test: "silent",
			opt:  WithSilent(),
			exp:  &Executor{verbose: false},
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)

			e := &Executor{verbose: true}
			tc.opt(e)
			a.Equal(tc.exp, e)
		})
	}
}

func TestNewExec(t *testing.T) {
	t.Parallel()
	lax, _ := parser.Parse("$")
	strict, _ := parser.Parse("strict $")

	for _, tc := range []struct {
		test string
		path *ast.AST
		opts []Option
		exp  *Executor
	}{
		{
			test: "lax_default",
			path: lax,
			exp: &Executor{
				path:                   lax,
				innermostArraySize:     -1,
				ignoreStructuralErrors: true,
				lastGeneratedObjectID:  1,
				verbose:                true,
			},
		},
		{
			test: "strict_default",
			path: strict,
			exp: &Executor{
				path:                   strict,
				innermostArraySize:     -1,
				ignoreStructuralErrors: false,
				lastGeneratedObjectID:  1,
				verbose:                true,
			},
		},
		{
			test: "lax_vars_silent",
			path: lax,
			opts: []Option{WithVars(Vars{"x": 1}), WithSilent()},
			exp: &Executor{
				path:                   lax,
				innermostArraySize:     -1,
				ignoreStructuralErrors: true,
				lastGeneratedObjectID:  1,
				verbose:                false,
				vars:                   Vars{"x": 1},
			},
		},
		{
			test: "strict_tz_silent",
			path: strict,
			opts: []Option{WithTZ(), WithSilent()},
			exp: &Executor{
				path:                   strict,
				innermostArraySize:     -1,
				ignoreStructuralErrors: false,
				lastGeneratedObjectID:  1,
				verbose:                false,
				useTZ:                  true,
			},
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)

			e := newExec(tc.path, tc.opts...)
			a.Equal(tc.exp, e)
		})
	}
}

func TestQueryAndFirstAndExists(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	for _, tc := range []struct {
		test  string
		path  string
		value any
		opts  []Option
		exp   []any
		err   string
		isErr error
		null  bool
	}{
		{
			test:  "root",
			path:  "$",
			value: []any{1, 2},
			exp:   []any{[]any{1, 2}},
		},
		{
			test:  "empty",
			path:  "$[3]",
			value: []any{1, 2},
			exp:   []any{},
		},
		{
			test:  "error",
			path:  "$.string()",
			value: []any{1, 2},
			err:   "exec: jsonpath item method .string() can only be applied to a boolean, string, numeric, or datetime value",
			isErr: ErrVerbose,
		},
		{
			test:  "silent_no_error",
			path:  "$.string()",
			opts:  []Option{WithSilent()},
			value: []any{1, 2},
			exp:   []any{},
			null:  true,
		},
		{
			test:  "like_regex_object",
			path:  `$ like_regex "^hi"`,
			value: map[string]any{"x": "HIGH"},
			exp:   []any{nil},
		},
		{
			test:  "like_regex_object_filter",
			path:  `$ ?(@ like_regex "^hi")`,
			value: map[string]any{"x": "HIGH"},
			exp:   []any{},
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			r := require.New(t)

			// Parse the path.
			path, err := parser.Parse(tc.path)
			r.NoError(err)

			t.Run("query", func(t *testing.T) {
				t.Parallel()
				a := assert.New(t)

				// Run the query.
				res, err := Query(ctx, path, tc.value, tc.opts...)
				a.Equal(tc.exp, res)

				// Check the error.
				if tc.isErr == nil {
					r.NoError(err)
				} else {
					r.EqualError(err, tc.err)
					r.ErrorIs(err, tc.isErr)
				}
			})

			t.Run("first", func(t *testing.T) {
				t.Parallel()
				a := assert.New(t)

				// Run the query.
				res, err := First(ctx, path, tc.value, tc.opts...)
				if len(tc.exp) > 0 {
					a.Equal(tc.exp[0], res)
				} else {
					a.Nil(res)
				}

				// Check the error.
				if tc.isErr == nil {
					r.NoError(err)
				} else {
					r.EqualError(err, tc.err)
					r.ErrorIs(err, tc.isErr)
				}
			})

			t.Run("exists", func(t *testing.T) {
				t.Parallel()
				a := assert.New(t)

				// Run the query.
				res, err := Exists(ctx, path, tc.value, tc.opts...)
				a.Equal(len(tc.exp) > 0, res)

				// Check the error.
				if tc.isErr == nil {
					if tc.null {
						r.EqualError(err, "NULL")
						r.ErrorIs(err, NULL)
					} else {
						r.NoError(err)
					}
				} else {
					r.EqualError(err, tc.err)
					r.ErrorIs(err, tc.isErr)
				}
			})
		})
	}
}

func TestMatch(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	for _, tc := range []struct {
		test  string
		path  string
		value any
		opts  []Option
		exp   bool
		err   string
		isErr error
	}{
		{
			test:  "root_eq",
			path:  "$ == 42",
			value: int64(42),
			exp:   true,
		},
		{
			test:  "root_ne",
			path:  "$ != 42",
			value: int64(42),
			exp:   false,
		},
		{
			test:  "null",
			path:  "$.string() == 12",
			value: []any{1, 2},
			err:   "NULL",
			isErr: NULL,
		},
		{
			test:  "strict_null",
			path:  "strict $.string() == 12",
			value: []any{1, 2},
			err:   "NULL",
			isErr: NULL,
		},
		{
			test:  "not_boolean",
			path:  "$",
			value: []any{1, 2},
			err:   "exec: single boolean result is expected",
			isErr: ErrVerbose,
		},
		{
			test:  "not_boolean_silent",
			path:  "$",
			opts:  []Option{WithSilent()},
			value: []any{1, 2},
			err:   "NULL",
			isErr: NULL,
		},
		{
			test:  "single_boolean_non_predicate",
			path:  "$",
			value: true,
			exp:   true,
		},
		{
			test:  "error",
			path:  `strict $.a`,
			value: map[string]any{},
			err:   `exec: JSON object does not contain key "a"`,
			isErr: ErrVerbose,
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)
			r := require.New(t)

			// Parse the path.
			path, err := parser.Parse(tc.path)
			r.NoError(err)

			// Run the query.
			res, err := Match(ctx, path, tc.value, tc.opts...)
			a.Equal(tc.exp, res)

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

func TestExecAccessors(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	// Test lax.
	lax, _ := parser.Parse("$")
	e := newExec(lax)
	a.False(e.strictAbsenceOfErrors())
	a.True(e.autoWrap())
	a.True(e.autoUnwrap())

	// Test strict.
	strict, _ := parser.Parse("strict $")
	e = newExec(strict)
	a.True(e.strictAbsenceOfErrors())
	a.False(e.autoWrap())
	a.False(e.autoUnwrap())
}

func TestReturnError(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	r := require.New(t)

	// Verbose.
	e := &Executor{verbose: true}
	res, err := e.returnVerboseError(ErrVerbose)
	a.Equal(statusFailed, res)
	r.ErrorIs(err, ErrVerbose)
	res, err = e.returnError(ErrVerbose)
	a.Equal(statusFailed, res)
	r.ErrorIs(err, ErrVerbose)
	res, err = e.returnError(ErrExecution)
	a.Equal(statusFailed, res)
	r.ErrorIs(err, ErrExecution)

	// Silent
	e.verbose = false
	res, err = e.returnVerboseError(ErrVerbose)
	a.Equal(statusFailed, res)
	r.NoError(err)
	res, err = e.returnError(ErrVerbose)
	a.Equal(statusFailed, res)
	r.NoError(err)
	res, err = e.returnError(ErrExecution)
	a.Equal(statusFailed, res)
	r.ErrorIs(err, ErrExecution)
}

// The tests below are admittedly duplicate unit tests for methods in other
// files, but came first while writing the first pass at the implementation.

type execTestCase struct {
	test   string
	path   string
	vars   Vars
	useTZ  bool
	silent bool
	result resultStatus
	json   any
	exp    []any
	err    string
	rand   bool
}

func newTestExecutor(path *ast.AST, vars Vars, throwErrors, useTZ bool) *Executor {
	return &Executor{
		path:                   path,
		vars:                   vars,
		innermostArraySize:     -1,
		useTZ:                  useTZ,
		ignoreStructuralErrors: path.IsLax(),
		verbose:                throwErrors,
		lastGeneratedObjectID:  1,
	}
}

func (tc execTestCase) run(t *testing.T) {
	t.Helper()
	a := assert.New(t)
	r := require.New(t)

	path, err := parser.Parse(tc.path)
	r.NoError(err)
	exec := newTestExecutor(path, tc.vars, !tc.silent, tc.useTZ)
	list, err := exec.execute(context.Background(), tc.json)
	if tc.err != "" {
		r.EqualError(err, tc.err)
		r.ErrorIs(err, ErrExecution)
		a.Empty(list.list)
	} else {
		r.NoError(err)
		a.NotNil(list)
		if tc.rand {
			a.ElementsMatch(tc.exp, list.list)
		} else {
			a.Equal(tc.exp, list.list)
		}
	}

	result, err := exec.exists(context.Background(), tc.json)
	if tc.err != "" {
		r.EqualError(err, tc.err)
		r.ErrorIs(err, ErrExecution)
		a.Equal(statusFailed, result)
	} else {
		r.NoError(err)
		exp := tc.result
		if exp == statusOK && len(tc.exp) == 0 {
			exp = statusNotFound
		}
		a.Equal(exp, result)
	}
}

func TestExecuteRoot(t *testing.T) {
	t.Parallel()
	for _, tc := range []execTestCase{
		{
			test: "root_obj",
			path: "$",
			json: map[string]any{"x": 42},
			exp:  []any{map[string]any{"x": 42}},
		},
		{
			test: "root_num",
			path: "$",
			json: 42.0,
			exp:  []any{42.0},
		},
		{
			test: "root_bool",
			path: "$",
			json: true,
			exp:  []any{true},
		},
		{
			test: "root_array",
			path: "$",
			json: []any{42, true, "hi"},
			exp:  []any{[]any{42, true, "hi"}},
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestExecuteLiteral(t *testing.T) {
	t.Parallel()
	for _, tc := range []execTestCase{
		{
			test: "null_only",
			path: "null",
			json: `""`,
			exp:  []any{nil},
		},
		{
			test: "true_only",
			path: "true",
			json: `""`,
			exp:  []any{true},
		},
		{
			test: "false_only",
			path: "false",
			json: `""`,
			exp:  []any{false},
		},
		{
			test: "string",
			path: `"yes"`,
			json: []any{1, 2, 3},
			exp:  []any{"yes"},
		},
		{
			test: "int",
			path: `42`,
			json: nil,
			exp:  []any{int64(42)},
		},
		{
			test: "float",
			path: `42.0`,
			json: nil,
			exp:  []any{float64(42.0)},
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestExecutePathKeys(t *testing.T) {
	t.Parallel()
	for _, tc := range []execTestCase{
		{
			test: "path_x",
			path: "$.x",
			json: map[string]any{"x": 42},
			exp:  []any{42},
		},
		{
			test: "path_xy",
			path: "$.x.y",
			json: map[string]any{"x": map[string]any{"y": "hi"}},
			exp:  []any{"hi"},
		},
		{
			test: "path_xyz",
			path: "$.x.y.z",
			json: map[string]any{"x": map[string]any{"y": map[string]any{"z": "yep"}}},
			exp:  []any{"yep"},
		},
		{
			test: "path_xy_obj",
			path: "$.x.y",
			json: map[string]any{"x": map[string]any{"y": map[string]any{"z": "yep"}}},
			exp:  []any{map[string]any{"z": "yep"}},
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestExecuteAny(t *testing.T) {
	t.Parallel()
	for _, tc := range []execTestCase{
		{
			test: "any_key",
			path: "$.*",
			json: map[string]any{"x": "hi", "y": 42},
			exp:  []any{"hi", 42},
			rand: true, // Results can be in any order
		},
		{
			test: "any_key_mixed",
			path: "$.*",
			json: map[string]any{"x": map[string]any{"y": 42}, "z": false},
			exp:  []any{map[string]any{"y": 42}, false},
			rand: true, // Results can be in any order
		},
		{
			test: "any_array",
			path: "$[*]",
			json: []any{"hi", 42},
			exp:  []any{"hi", 42},
		},
		{
			test: "any_array_mixed",
			path: "$[*]",
			json: []any{"hi", 42, true, map[string]any{"x": 1}, nil},
			exp:  []any{"hi", 42, true, map[string]any{"x": 1}, nil},
		},
		{
			test: "path_x_any_array",
			path: "$.x[*]",
			json: map[string]any{"x": []any{"hi", 42}},
			exp:  []any{"hi", 42},
		},
		{
			test: "path_xy_any_array",
			path: "$.x.y[*]",
			json: map[string]any{"x": map[string]any{"y": []any{"hi", 42}}},
			exp:  []any{"hi", 42},
		},
		{
			test: "any",
			path: "$.**",
			json: map[string]any{"x": "hi", "y": 42},
			exp:  []any{map[string]any{"x": "hi", "y": 42}, "hi", 42},
			rand: true, // Results can be in any order
		},
		{
			test: "any_nested",
			path: "$.**",
			json: map[string]any{"x": map[string]any{"y": 42}, "z": map[string]any{}},
			exp: []any{
				map[string]any{"x": map[string]any{"y": 42}, "z": map[string]any{}},
				map[string]any{"y": 42},
				42,
				map[string]any{},
			},
			rand: true, // Results can be in any order
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestExecuteMath(t *testing.T) {
	t.Parallel()
	for _, tc := range []execTestCase{
		{
			test: "add_ints",
			path: "$ + 1",
			json: int64(2),
			exp:  []any{int64(3)},
		},
		{
			test: "add_floats",
			path: "$ + 3.2",
			json: float64(98.6),
			exp:  []any{float64(101.8)},
		},
		{
			test: "add_int_flat",
			path: "$ + 3",
			json: float64(98.6),
			exp:  []any{float64(101.6)},
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestExecuteAndOr(t *testing.T) {
	t.Parallel()
	for _, tc := range []execTestCase{
		{
			test: "binary_or_ints",
			path: "$.x == 3 || $.x == 4",
			json: map[string]any{"x": int64(4)},
			exp:  []any{true},
		},
		{
			test: "binary_or_int_float",
			path: "$.x == 3 || $.y == 4.0",
			json: map[string]any{"x": int64(4), "y": float64(4.0)},
			exp:  []any{true},
		},
		{
			test: "binary_and_strings",
			path: `$.x == "hi" && $.y starts with "good"`,
			json: map[string]any{"x": "hi", "y": "good bye"},
			exp:  []any{true},
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestExecuteNumberMethods(t *testing.T) {
	t.Parallel()
	for _, tc := range []execTestCase{
		{
			test: "number_method",
			path: `$.x.number()`,
			json: map[string]any{"x": int64(3)},
			exp:  []any{float64(3)},
		},
		{
			test: "number_method_string",
			path: `$.x.number()`,
			json: map[string]any{"x": "3.4"},
			exp:  []any{float64(3.4)},
		},
		{
			test: "number_method_json_number",
			path: `$.x.number()`,
			json: map[string]any{"x": json.Number("3.4")},
			exp:  []any{float64(3.4)},
		},
		{
			test: "number_method_json_number_int",
			path: `$.x.number()`,
			json: map[string]any{"x": json.Number("1714004682")},
			exp:  []any{float64(1714004682)},
		},
		{
			test: "decimal_method",
			path: `$.x.decimal()`,
			json: map[string]any{"x": "12.2"},
			exp:  []any{float64(12.2)},
		},
		{
			test: "decimal_method_precision",
			path: `$.x.decimal(4)`,
			json: map[string]any{"x": "12.2"},
			exp:  []any{float64(12)},
		},
		{
			test: "decimal_method_precision_short",
			path: `$.x.decimal(1)`,
			json: map[string]any{"x": "12.233"},
			// exp:  []any{float64(12)},
			err: `exec: argument "12.233" of jsonpath item method .decimal() is invalid for type numeric`,
		},
		{
			test: "decimal_method_precision_ok",
			path: `$.x.decimal(5,3)`,
			json: map[string]any{"x": "12.233"},
			exp:  []any{float64(12.233)},
		},
		{
			test: "decimal_method_precision_scale",
			path: `$.x.decimal(4, 2)`,
			json: map[string]any{"x": "12.233"},
			exp:  []any{float64(12.23)},
		},
		{
			test: "decimal_method_precision_scale_short",
			path: `$.x.decimal(3, 2)`,
			json: map[string]any{"x": "12.233"},
			err:  `exec: argument "12.233" of jsonpath item method .decimal() is invalid for type numeric`,
		},
		{
			test: "abs_int",
			path: `$.x.abs()`,
			json: map[string]any{"x": int64(-42)},
			exp:  []any{int64(42)},
		},
		{
			test: "abs_float",
			path: `$.x.abs()`,
			json: map[string]any{"x": float64(-42.22)},
			exp:  []any{float64(42.22)},
		},
		{
			test: "abs_json_number_int",
			path: `$.x.abs()`,
			json: map[string]any{"x": json.Number("-99")},
			exp:  []any{int64(99)},
		},
		{
			test: "abs_json_number_float",
			path: `$.x.abs()`,
			json: map[string]any{"x": json.Number("-42.22")},
			exp:  []any{float64(42.22)},
		},
		{
			test: "floor_int",
			path: `$.x.floor()`,
			json: map[string]any{"x": int64(42)},
			exp:  []any{int64(42)},
		},
		{
			test: "floor_float",
			path: `$.x.floor()`,
			json: map[string]any{"x": float64(42.22)},
			exp:  []any{float64(42)},
		},
		{
			test: "floor_json_number_int",
			path: `$.x.floor()`,
			json: map[string]any{"x": json.Number("99")},
			exp:  []any{int64(99)},
		},
		{
			test: "floor_json_number_float",
			path: `$.x.floor()`,
			json: map[string]any{"x": json.Number("88.88")},
			exp:  []any{float64(88)},
		},
		{
			test: "ceiling_int",
			path: `$.x.ceiling()`,
			json: map[string]any{"x": int64(42)},
			exp:  []any{int64(42)},
		},
		{
			test: "ceiling_float",
			path: `$.x.ceiling()`,
			json: map[string]any{"x": float64(42.22)},
			exp:  []any{float64(43)},
		},
		{
			test: "ceiling_json_number_int",
			path: `$.x.ceiling()`,
			json: map[string]any{"x": json.Number("99")},
			exp:  []any{int64(99)},
		},
		{
			test: "ceiling_json_number_float",
			path: `$.x.ceiling()`,
			json: map[string]any{"x": json.Number("88.88")},
			exp:  []any{float64(89)},
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestExecuteArraySubscripts(t *testing.T) {
	t.Parallel()
	for _, tc := range []execTestCase{
		{
			test: "array_subscript_0",
			path: `$.x[0]`,
			json: map[string]any{"x": []any{"hi"}},
			exp:  []any{"hi"},
		},
		{
			test: "array_subscript_2",
			path: `$.x[2]`,
			json: map[string]any{"x": []any{"hi", "", true}},
			exp:  []any{true},
		},
		{
			test: "array_subscript_from_to",
			path: `$.x[1 to 2]`,
			json: map[string]any{"x": []any{"xx", "hi", true}},
			exp:  []any{"hi", true},
		},
		{
			test: "array_subscript_last",
			path: `$.x[last]`,
			json: map[string]any{"x": []any{"hi", "", true}},
			exp:  []any{true},
		},
		{
			test: "array_subscript_to_last",
			path: `$.x[1 to last]`,
			json: map[string]any{"x": []any{"hi", "", true}},
			exp:  []any{"", true},
		},
		{
			test: "array_subscript_multi",
			path: `$.x[0, 3 to 4]`,
			json: map[string]any{"x": []any{"hi", "", true, "x", "y"}},
			exp:  []any{"hi", "x", "y"},
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestExecuteLikeRegex(t *testing.T) {
	t.Parallel()
	for _, tc := range []execTestCase{
		{
			test: "like_regex",
			path: `$.x like_regex "."`,
			json: map[string]any{"x": "x"},
			exp:  []any{true},
		},
		{
			test: "like_regex_prefix",
			path: `$.x like_regex "^hi"`,
			json: map[string]any{"x": "hi there"},
			exp:  []any{true},
		},
		{
			test: "like_regex_false",
			path: `$.x like_regex "^hi"`,
			json: map[string]any{"x": "say hi there"},
			exp:  []any{false},
		},
		{
			test: "like_regex_flag",
			path: `$.x like_regex "^hi" flag "i"`,
			json: map[string]any{"x": "HIGH"},
			exp:  []any{true},
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestExecuteFilter(t *testing.T) {
	t.Parallel()
	for _, tc := range []execTestCase{
		{
			test: "filter_true",
			path: `$.x ?(@ == "hi")`,
			json: map[string]any{"x": "hi"},
			exp:  []any{"hi"},
		},
		{
			test: "filter_false",
			path: `$.x ?(@ != "hi")`,
			json: map[string]any{"x": "hi"},
			exp:  []any{},
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestExecuteTypeSizeMethods(t *testing.T) {
	t.Parallel()
	for _, tc := range []execTestCase{
		{
			test: "type_method_string",
			path: `$.x.type()`,
			json: map[string]any{"x": "hi"},
			exp:  []any{"string"},
		},
		{
			test: "type_method_multi",
			path: `$[*].type()`,
			json: []any{int64(1), "2", map[string]any{}},
			exp:  []any{"number", "string", "object"},
		},
		{
			test: "size_method_array",
			path: `$.x.size()`,
			json: map[string]any{"x": []any{1, 2, 3}},
			exp:  []any{int64(3)},
		},
		{
			test: "size_method_other",
			path: `$.x.size()`,
			json: map[string]any{"x": true},
			exp:  []any{int64(1)},
		},
		{
			test: "size_method_error",
			path: `strict $.x.size()`,
			json: map[string]any{"x": true},
			err:  `exec: jsonpath item method .size() can only be applied to an array`,
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestExecuteUnaryPlusMinus(t *testing.T) {
	t.Parallel()
	for _, tc := range []execTestCase{
		{
			test: "unary_plus",
			path: `+$.x`,
			json: map[string]any{"x": int64(42)},
			exp:  []any{int64(42)},
		},
		{
			test: "unary_minus_pos",
			path: `-$.x`,
			json: map[string]any{"x": int64(42)},
			exp:  []any{int64(-42)},
		},
		{
			test: "unary_minus_neg",
			path: `-$.x`,
			json: map[string]any{"x": int64(-42)},
			exp:  []any{int64(42)},
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestExecuteDateTime(t *testing.T) {
	t.Parallel()
	offsetZero := time.FixedZone("", 0)
	ctx := context.Background()

	for _, tc := range []execTestCase{
		{
			test: "date",
			path: `$.x.date()`,
			json: map[string]any{"x": "2009-10-03"},
			exp: []any{types.NewDate(
				time.Date(2009, 10, 3, 0, 0, 0, 0, offsetZero),
			)},
		},
		{
			test: "time",
			path: `$.x.time()`,
			json: map[string]any{"x": "20:59:19.79142"},
			exp: []any{types.NewTime(
				time.Date(0, 1, 1, 20, 59, 19, 791420000, offsetZero),
			)},
		},
		{
			test: "time_tz",
			path: `$.x.time_tz()`,
			json: map[string]any{"x": "20:59:19.79142-04"},
			exp: []any{types.NewTimeTZ(
				time.Date(0, 1, 1, 20, 59, 19, 791420000, time.FixedZone("", -4*60*60)),
			)},
		},
		{
			test: "timestamp_T",
			path: `$.x.timestamp()`,
			json: map[string]any{"x": "2024-05-05T20:59:19.79142"},
			exp: []any{types.NewTimestamp(
				time.Date(2024, 5, 5, 20, 59, 19, 791420000, offsetZero),
			)},
		},
		{
			test: "timestamp_space",
			path: `$.x.timestamp()`,
			json: map[string]any{"x": "2024-05-05 20:59:19.79142"},
			exp: []any{types.NewTimestamp(
				time.Date(2024, 5, 5, 20, 59, 19, 791420000, offsetZero),
			)},
		},
		{
			test: "timestamp_T_tz",
			path: `$.x.timestamp_tz()`,
			json: map[string]any{"x": "2024-05-05T20:59:19.79142-05"},
			exp: []any{types.NewTimestampTZ(
				ctx,
				time.Date(2024, 5, 5, 20, 59, 19, 791420000, time.FixedZone("", -5*60*60)),
			)},
		},
		{
			test: "timestamp_space_tz",
			path: `$.x.timestamp_tz()`,
			json: map[string]any{"x": "2024-05-05 20:59:19.79142-05"},
			exp: []any{types.NewTimestampTZ(
				ctx,
				time.Date(2024, 5, 5, 20, 59, 19, 791420000, time.FixedZone("", -5*60*60)),
			)},
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
			// .datetime() should also work
			tc.test += "_datetime"
			tc.path = `$.x.datetime()`
			tc.run(t)
		})
	}
}

func TestExecuteDateTimeErrors(t *testing.T) {
	t.Parallel()
	for _, tc := range []execTestCase{
		{
			test: "not_a_string",
			path: `$.x.timestamp_tz()`,
			json: map[string]any{"x": int64(42)},
			err:  "exec: jsonpath item method .timestamp_tz() can only be applied to a string",
		},
		{
			test: "datetime_template_not_supported",
			path: `$.x.datetime("HH24:MI")`,
			json: map[string]any{"x": "2024-05-05 20:59:19.79142-05"},
			err:  "exec: .datetime(template) is not yet supported",
		},
		{
			test: "invalid_precision",
			path: fmt.Sprintf(`$.x.time(%v)`, int64(math.MaxInt32+1)),
			json: map[string]any{"x": "2024-05-05 20:59:19.79142-05"},
			err:  `exec: time precision of jsonpath item method .time() is out of integer range`,
		},
		{
			test: "not_a_timestamp",
			path: `$.x.time()`,
			json: map[string]any{"x": "NOT A TIMESTAMP"},
			err:  `exec: time format is not recognized: "NOT A TIMESTAMP"`,
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

const tzHint = " HINT: Use WithTZ() option for time zone support"

func TestExecuteDateTimeCast(t *testing.T) {
	t.Parallel()
	offsetZero := time.FixedZone("", 0)
	ctx := context.Background()

	for _, tc := range []execTestCase{
		// Cast to date
		{
			test: "date_to_date",
			path: `$.x.date()`,
			json: map[string]any{"x": "2009-10-03"},
			exp: []any{types.NewDate(
				time.Date(2009, 10, 3, 0, 0, 0, 0, offsetZero),
			)},
		},
		{
			test: "timestamp_to_date",
			path: `$.x.date()`,
			json: map[string]any{"x": "2009-10-03 20:59:19.79142"},
			exp: []any{types.NewDate(
				time.Date(2009, 10, 3, 0, 0, 0, 0, offsetZero),
			)},
		},
		{
			test: "timestamp_tz_to_date",
			path: `$.x.date()`,
			json: map[string]any{"x": "2009-10-03 20:59:19.79142-01"},
			err:  "exec: cannot convert value from timestamptz to date without time zone usage." + tzHint,
		},
		{
			test:  "timestamp_with_tz_to_date",
			path:  `$.x.date()`,
			useTZ: true,
			json:  map[string]any{"x": "2009-10-03 20:59:19.79142-01"},
			exp: []any{types.NewDate(
				time.Date(2009, 10, 3, 0, 0, 0, 0, offsetZero),
			)},
		},
		{
			test: "time_to_date",
			path: `$.x.date()`,
			json: map[string]any{"x": "20:59:19.79142"},
			err:  `exec: date format is not recognized: "20:59:19.79142"`,
		},
		{
			test: "time_tz_to_date",
			path: `$.x.date()`,
			json: map[string]any{"x": "20:59:19.79142-01"},
			err:  `exec: date format is not recognized: "20:59:19.79142-01"`,
		},
		// Cast to time
		{
			test: "date_to_time",
			path: `$.x.time()`,
			json: map[string]any{"x": "2009-10-03"},
			err:  `exec: time format is not recognized: "2009-10-03"`,
		},
		{
			test: "time_to_time",
			path: `$.x.time()`,
			json: map[string]any{"x": "20:59:19.79142"},
			exp: []any{types.NewTime(
				time.Date(0, 1, 1, 20, 59, 19, 791420000, offsetZero),
			)},
		},
		{
			test: "time_tz_to_time",
			path: `$.x.time()`,
			json: map[string]any{"x": "20:59:19.79142-01"},
			err:  "exec: cannot convert value from timetz to time without time zone usage." + tzHint,
		},
		{
			test:  "time_with_tz_to_time",
			path:  `$.x.time()`,
			useTZ: true,
			json:  map[string]any{"x": "20:59:19.79142-01"},
			exp: []any{types.NewTime(
				time.Date(0, 1, 1, 20, 59, 19, 791420000, offsetZero),
			)},
		},
		{
			test: "timestamp_to_time",
			path: `$.x.time()`,
			json: map[string]any{"x": "2009-10-03 20:59:19.79142"},
			exp: []any{types.NewTime(
				time.Date(0, 1, 1, 20, 59, 19, 791420000, offsetZero),
			)},
		},
		{
			test: "timestamp_tz_to_time",
			path: `$.x.time()`,
			json: map[string]any{"x": "2009-10-03 20:59:19.79142+01"},
			err:  "exec: cannot convert value from timestamptz to time without time zone usage." + tzHint,
		},
		{
			test:  "timestamp_with_tz_to_time",
			path:  `$.x.time()`,
			useTZ: true,
			json:  map[string]any{"x": "2009-10-03 20:59:19.79142+01"},
			exp: []any{types.NewTime(types.NewTimestampTZ(
				ctx,
				time.Date(2009, 10, 3, 20, 59, 19, 791420000, time.FixedZone("", 3600)),
			).In(offsetZero))},
		},
		// Cast to timetz
		{
			test: "date_to_timetz",
			path: `$.x.time_tz()`,
			json: map[string]any{"x": "2009-10-03"},
			err:  `exec: time_tz format is not recognized: "2009-10-03"`,
		},
		{
			test: "time_to_timetz",
			path: `$.x.time_tz()`,
			json: map[string]any{"x": "20:59:19.79142"},
			err:  "exec: cannot convert value from time to timetz without time zone usage." + tzHint,
		},
		{
			test:  "time_to_time_with_tz",
			path:  `$.x.time_tz()`,
			useTZ: true,
			json:  map[string]any{"x": "20:59:19.79142"},
			exp: []any{types.NewTimeTZ(
				time.Date(0, 1, 1, 20, 59, 19, 791420000, offsetZero),
			)},
		},
		{
			test: "timetz_to_timetz",
			path: `$.x.time_tz()`,
			json: map[string]any{"x": "20:59:19.79142Z"},
			exp:  []any{types.NewTimeTZ(time.Date(0, 1, 1, 20, 59, 19, 791420000, offsetZero))},
		},
		{
			test: "timestamp_to_timetz",
			path: `$.x.time_tz()`,
			json: map[string]any{"x": "2009-10-03 20:59:19.79142"},
			err:  `exec: time_tz format is not recognized: "2009-10-03 20:59:19.79142"`,
		},
		{
			test: "timestamp_tz_to_timetz",
			path: `$.x.time_tz()`,
			json: map[string]any{"x": "2009-10-03 20:59:19.79142Z"},
			exp: []any{types.NewTimestampTZ(
				ctx,
				time.Date(2009, 10, 3, 20, 59, 19, 791420000, offsetZero),
			).ToTimeTZ(ctx)},
		},
		// Cast to timestamp
		{
			test: "date_to_timestamp",
			path: `$.x.timestamp()`,
			json: map[string]any{"x": "2009-10-03"},
			exp:  []any{types.NewTimestamp(time.Date(2009, 10, 3, 0, 0, 0, 0, offsetZero))},
		},
		{
			test: "time_to_timestamp",
			path: `$.x.timestamp()`,
			json: map[string]any{"x": "20:59:19.79142"},
			err:  `exec: timestamp format is not recognized: "20:59:19.79142"`,
		},
		{
			test: "time_tz_to_timestamp",
			path: `$.x.timestamp()`,
			json: map[string]any{"x": "20:59:19.79142-01"},
			err:  `exec: timestamp format is not recognized: "20:59:19.79142-01"`,
		},
		{
			test: "timestamp_to_timestamp",
			path: `$.x.timestamp()`,
			json: map[string]any{"x": "2009-10-03 20:59:19.79142"},
			exp:  []any{types.NewTimestamp(time.Date(2009, 10, 3, 20, 59, 19, 791420000, offsetZero))},
		},
		{
			test: "timestamp_tz_to_timestamp",
			path: `$.x.timestamp()`,
			json: map[string]any{"x": "2009-10-03 20:59:19.79142Z"},
			err:  "exec: cannot convert value from timestamptz to timestamp without time zone usage." + tzHint,
		},
		{
			test:  "timestamp_with_tz_to_timestamp",
			path:  `$.x.timestamp()`,
			useTZ: true,
			json:  map[string]any{"x": "2009-10-03 20:59:19.79142Z"},
			exp: []any{types.NewTimestamp(
				time.Date(2009, 10, 3, 20, 59, 19, 791420000, offsetZero),
			)},
		},
		// Cast to timestamptz
		{
			test: "date_to_timestamptz",
			path: `$.x.timestamp_tz()`,
			json: map[string]any{"x": "2009-10-03"},
			err:  "exec: cannot convert value from date to timestamptz without time zone usage." + tzHint,
		},
		{
			test:  "date_to_timestamp_with_tz",
			path:  `$.x.timestamp_tz()`,
			useTZ: true,
			json:  map[string]any{"x": "2009-10-03"},
			exp: []any{types.NewDate(
				time.Date(2009, 10, 3, 0, 0, 0, 0, offsetZero),
			).ToTimestampTZ(ctx)},
		},
		{
			test: "time_to_timestamptz",
			path: `$.x.timestamp_tz()`,
			json: map[string]any{"x": "20:59:19.79142"},
			err:  `exec: timestamp_tz format is not recognized: "20:59:19.79142"`,
		},
		{
			test: "time_tz_to_timestamptz",
			path: `$.x.timestamp_tz()`,
			json: map[string]any{"x": "20:59:19.79142-01"},
			err:  `exec: timestamp_tz format is not recognized: "20:59:19.79142-01"`,
		},
		{
			test: "timestamp_to_timestamptz",
			path: `$.x.timestamp_tz()`,
			json: map[string]any{"x": "2009-10-03 20:59:19.79142"},
			err:  "exec: cannot convert value from timestamp to timestamptz without time zone usage." + tzHint,
		},
		{
			test:  "timestamp_to_timestamp_with_tz",
			path:  `$.x.timestamp_tz()`,
			useTZ: true,
			json:  map[string]any{"x": "2009-10-03 20:59:19.79142"},
			exp: []any{types.NewTimestampTZ(
				ctx,
				time.Date(2009, 10, 3, 20, 59, 19, 791420000, offsetZero),
			)},
		},
		{
			test: "timestamp_tz_to_timestamptz",
			path: `$.x.timestamp_tz()`,
			json: map[string]any{"x": "2009-10-03 20:59:19.79142Z"},
			exp: []any{types.NewTimestampTZ(
				ctx,
				time.Date(2009, 10, 3, 20, 59, 19, 791420000, offsetZero),
			)},
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestExecuteTimePrecision(t *testing.T) {
	t.Parallel()
	offsetZero := time.FixedZone("", 0)

	for _, tc := range []execTestCase{
		{
			test: "time_precision",
			path: `$.x.time(3)`,
			json: map[string]any{"x": "20:59:19.79142"},
			exp:  []any{types.NewTime(time.Date(0, 1, 1, 20, 59, 19, 791000000, offsetZero))},
		},
		{
			test: "time_tz_precision",
			path: `$.x.time_tz(4)`,
			json: map[string]any{"x": "20:59:19.79142+01"},
			exp: []any{types.NewTimeTZ(
				time.Date(0, 1, 1, 20, 59, 19, 791400000, time.FixedZone("", 1*60*60)),
			)},
		},
		{
			test: "timestamp_precision",
			path: `$.x.timestamp(2)`,
			json: map[string]any{"x": "2024-05-05T20:59:19.791423"},
			exp:  []any{types.NewTimestamp(time.Date(2024, 5, 5, 20, 59, 19, 790000000, offsetZero))},
		},
		{
			test: "timestamp_tz_precision",
			path: `$.x.timestamp_tz(5)`,
			json: map[string]any{"x": "2024-05-05T20:59:19.791423+02:30"},
			exp: []any{types.NewTimestampTZ(
				context.Background(),
				time.Date(2024, 5, 5, 20, 59, 19, 791420000, time.FixedZone("", 2*60*60+30*60)),
			)},
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestExecuteDateComparison(t *testing.T) {
	t.Parallel()
	for _, tc := range []execTestCase{
		{
			test: "date_eq_date",
			path: `$.date() == $.date()`,
			json: "2024-05-03",
			exp:  []any{true},
		},
		{
			test: "date_ne_date",
			path: `$.date() != $.date()`,
			json: "2024-05-03",
			exp:  []any{false},
		},
		{
			test: "unequal_dates",
			path: `$.x.date() == $.y.date()`,
			json: map[string]any{"x": "2024-05-03", "y": "2024-05-04"},
			exp:  []any{false},
		},
		{
			test: "gt_date",
			path: `$.y.date() >= $.x.date()`,
			json: map[string]any{"x": "2024-05-03", "y": "2024-05-04"},
			exp:  []any{true},
		},
		{
			test: "same_date",
			path: `$.date() == $.date()`,
			json: "2024-05-03",
			exp:  []any{true},
		},
		{
			test: "date_eq_timestamp",
			path: `$.x.date() == $.y.timestamp()`,
			json: map[string]any{"x": "2024-05-03", "y": "2024-05-03 23:53:42.232"},
			exp:  []any{false},
		},
		{
			test: "date_lt_timestamp",
			path: `$.x.date() < $.y.timestamp()`,
			json: map[string]any{"x": "2024-05-03", "y": "2024-05-03 23:53:42.232"},
			exp:  []any{true},
		},
		{
			test: "date_eq_timestamp_midnight",
			path: `$.x.date() == $.y.timestamp()`,
			json: map[string]any{"x": "2024-05-03", "y": "2024-05-03 00:00:00"},
			exp:  []any{true},
		},
		{
			test: "date_eq_timestamp_tz",
			path: `$.x.date() == $.y.timestamp_tz()`,
			json: map[string]any{"x": "2024-05-03", "y": "2024-05-03 23:53:42.232Z"},
			err:  "exec: cannot convert value from date to timestamptz without time zone usage." + tzHint,
		},
		{
			test:  "date_eq_timestamp_with_tz",
			path:  `$.x.date() == $.y.timestamp_tz()`,
			useTZ: true,
			json:  map[string]any{"x": "2024-05-03", "y": "2024-05-03 23:53:42.232Z"},
			exp:   []any{false},
		},
		{
			test:  "date_le_timestamp_with_tz",
			path:  `$.x.date() <= $.y.timestamp_tz()`,
			useTZ: true,
			json:  map[string]any{"x": "2024-05-03", "y": "2024-05-03 23:53:42.232Z"},
			exp:   []any{true},
		},
		{
			test: "date_eq_time",
			path: `$.x.date() == $.y.time()`,
			json: map[string]any{"x": "2024-05-03", "y": "23:53:42.232"},
			exp:  []any{nil},
		},
		{
			test: "date_eq_time_tz",
			path: `$.x.date() == $.y.time_tz()`,
			json: map[string]any{"x": "2024-05-03", "y": "23:53:42.232Z"},
			exp:  []any{nil},
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestExecuteTimeComparison(t *testing.T) {
	t.Parallel()
	for _, tc := range []execTestCase{
		{
			test: "time_eq_time",
			path: `$.time() == $.time()`,
			json: "14:32:43.123345",
			exp:  []any{true},
		},
		{
			test: "time_ne_time",
			path: `$.time() != $.time()`,
			json: "14:32:43.123345",
			exp:  []any{false},
		},
		{
			test: "time_ne_time_true",
			path: `$.time(3) != $.time(4)`,
			json: "14:32:43.123345",
			exp:  []any{true},
		},
		{
			test: "time_eq_time_tz",
			path: `$.x.time() == $.y.time_tz()`,
			json: map[string]any{"x": "14:32:43.123345", "y": "14:32:43.123345Z"},
			err:  "exec: cannot convert value from time to timetz without time zone usage." + tzHint,
		},
		{
			test:  "time_eq_time_with_tz",
			path:  `$.x.time() == $.y.time_tz()`,
			useTZ: true,
			json:  map[string]any{"x": "14:32:43.123345", "y": "14:32:43.123345Z"},
			exp:   []any{true},
		},
		{
			test:  "time_eq_time_with_tz_conv",
			path:  `$.x.time() != $.y.time_tz()`,
			useTZ: true,
			json:  map[string]any{"x": "14:32:43.123345", "y": "14:32:43.123345-01"},
			exp:   []any{true},
		},
		{
			test: "time_eq_date",
			path: `$.x.time() == $.y.date()`,
			json: map[string]any{"x": "14:32:43", "y": "2024-05-05"},
			exp:  []any{nil},
		},
		{
			test: "time_eq_timestamp",
			path: `$.x.time() == $.y.timestamp()`,
			json: map[string]any{"x": "14:32:43", "y": "2024-05-05 14:32:43"},
			exp:  []any{nil},
		},
		{
			test: "time_eq_timestamp_tz",
			path: `$.x.time() == $.y.timestamp_tz()`,
			json: map[string]any{"x": "14:32:43", "y": "2024-05-05 14:32:43Z"},
			exp:  []any{nil},
		},
		{
			test: "timetz_eq_timetz",
			path: `$.time_tz() == $.time_tz()`,
			json: "14:32:43.123345Z",
			exp:  []any{true},
		},
		{
			test: "timetz_ne_timetz",
			path: `$.time_tz() != $.time_tz()`,
			json: "14:32:43.123345Z",
			exp:  []any{false},
		},
		{
			test: "timetz_ne_timetz_true",
			path: `$.time_tz(3) != $.time_tz(4)`,
			json: "14:32:43.123345Z",
			exp:  []any{true},
		},
		{
			test: "timetz_eq_time",
			path: `$.y.time_tz() == $.x.time()`,
			json: map[string]any{"x": "14:32:43.123345", "y": "14:32:43.123345Z"},
			err:  "exec: cannot convert value from time to timetz without time zone usage." + tzHint,
		},
		{
			test:  "time_with_tz_eq_time",
			path:  `$.y.time_tz() == $.x.time()`,
			useTZ: true,
			json:  map[string]any{"x": "14:32:43.123345", "y": "14:32:43.123345Z"},
			exp:   []any{true},
		},
		{
			test:  "time_with_tz_conv_eq_time",
			path:  `$.y.time_tz() != $.x.time()`,
			useTZ: true,
			json:  map[string]any{"x": "14:32:43.123345", "y": "14:32:43.123345-01"},
			exp:   []any{true},
		},
		{
			test: "timetz_eq_date",
			path: `$.x.time_tz() == $.y.date()`,
			json: map[string]any{"x": "14:32:43Z", "y": "2024-05-05"},
			exp:  []any{nil},
		},
		{
			test: "timetz_eq_timestamp",
			path: `$.x.time_tz() == $.y.timestamp()`,
			json: map[string]any{"x": "14:32:43Z", "y": "2024-05-05 14:32:43"},
			exp:  []any{nil},
		},
		{
			test: "timetz_eq_timestamp_tz",
			path: `$.x.time_tz() == $.y.timestamp_tz()`,
			json: map[string]any{"x": "14:32:43Z", "y": "2024-05-05 14:32:43Z"},
			exp:  []any{nil},
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestExecuteTimestampComparison(t *testing.T) {
	t.Parallel()
	for _, tc := range []execTestCase{
		{
			test: "ts_eq_ts",
			path: `$.timestamp() == $.timestamp()`,
			json: "2024-05-05 14:32:43.123345",
			exp:  []any{true},
		},
		{
			test: "ts_ne_ts",
			path: `$.timestamp() != $.timestamp()`,
			json: "2024-05-05 14:32:43.123345",
			exp:  []any{false},
		},
		{
			test: "ts_eq_ts_precision",
			path: `$.timestamp(3) == $.timestamp(4)`,
			json: "2024-05-05 14:32:43.123345",
			exp:  []any{false},
		},
		{
			test: "ts_ne_date",
			path: `$[0].timestamp() != $[1].date()`,
			json: []any{"2024-05-05 14:32:43.123345", "2024-05-05"},
			exp:  []any{true},
		},
		{
			test: "ts_eq_date",
			path: `$[0].timestamp() == $[1].date()`,
			json: []any{"2024-05-05 00:00:00", "2024-05-05"},
			exp:  []any{true},
		},
		{
			test: "ts_eq_ts_tz",
			path: `$[0].timestamp() == $[1].timestamp_tz()`,
			json: []any{"2024-05-05 00:00:00", "2024-05-05 00:00:00Z"},
			err:  "exec: cannot convert value from timestamp to timestamptz without time zone usage." + tzHint,
		},
		{
			test:  "ts_eq_ts_with_tz",
			path:  `$[0].timestamp() == $[1].timestamp_tz()`,
			useTZ: true,
			json:  []any{"2024-05-05 00:00:00", "2024-05-05 00:00:00Z"},
			exp:   []any{true},
		},
		{
			test: "ts_eq_time",
			path: `$[0].timestamp() == $[1].time()`,
			json: []any{"2024-05-05 00:00:00", "00:00:00"},
			exp:  []any{nil},
		},
		{
			test: "ts_eq_time",
			path: `$[0].timestamp() == $[1].time_tz()`,
			json: []any{"2024-05-05 00:00:00", "00:00:00Z"},
			exp:  []any{nil},
		},
		{
			test: "ts_tz_eq_ts_tz",
			path: `$.timestamp_tz() == $.timestamp_tz()`,
			json: "2024-05-05 14:32:43.123345Z",
			exp:  []any{true},
		},
		{
			test: "ts_tz_ne_ts_tz",
			path: `$.timestamp_tz() != $.timestamp_tz()`,
			json: "2024-05-05 14:32:43.123345Z",
			exp:  []any{false},
		},
		{
			test: "ts_tz_eq_ts_tz_precision",
			path: `$.timestamp_tz(2) == $.timestamp_tz(2)`,
			json: "2024-05-05 14:32:43.123345Z",
			exp:  []any{true},
		},
		{
			test: "ts_tz_ne_ts_tz_precision",
			path: `$.timestamp_tz(2) != $.timestamp_tz(3)`,
			json: "2024-05-05 14:32:43.123345Z",
			exp:  []any{true},
		},
		{
			test: "ts_tz_eq_date",
			path: `$[0].timestamp_tz() == $[1].date()`,
			json: []any{"2024-05-05 14:32:43.123345Z", "2024-05-05"},
			err:  "exec: cannot convert value from date to timestamptz without time zone usage." + tzHint,
		},
		{
			test:  "ts_with_tz_ne_date",
			path:  `$[0].timestamp_tz() != $[1].date()`,
			useTZ: true,
			json:  []any{"2024-05-05 14:32:43.123345Z", "2024-05-05"},
			exp:   []any{true},
		},
		{
			test:  "ts_with_tz_eq_date",
			path:  `$[0].timestamp_tz() == $[1].date()`,
			useTZ: true,
			json:  []any{"2024-05-05 00:00:00Z", "2024-05-05"},
			exp:   []any{true},
		},
		{
			test: "ts_tz_eq_timestamp",
			path: `$[0].timestamp_tz() == $[1].timestamp()`,
			json: []any{"2024-05-05 14:32:43.123345Z", "2024-05-05 14:32:43.123345"},
			err:  "exec: cannot convert value from timestamp to timestamptz without time zone usage." + tzHint,
		},
		{
			test:  "ts_with_tz_eq_timestamp",
			path:  `$[0].timestamp_tz() == $[1].timestamp()`,
			useTZ: true,
			json:  []any{"2024-05-05 14:32:43.123345Z", "2024-05-05 14:32:43.123345"},
			exp:   []any{true},
		},
		{
			test: "ts_tz_eq_time",
			path: `$[0].timestamp_tz() == $[1].time()`,
			json: []any{"2024-05-05 00:00:00Z", "00:00:00"},
			exp:  []any{nil},
		},
		{
			test: "ts_tz_eq_time",
			path: `$[0].timestamp_tz() == $[1].time_tz()`,
			json: []any{"2024-05-05 00:00:00Z", "00:00:00Z"},
			exp:  []any{nil},
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestExecuteDoubleMethod(t *testing.T) {
	t.Parallel()
	for _, tc := range []execTestCase{
		{
			test: "double_int",
			path: `$.x.double()`,
			json: map[string]any{"x": int64(42)},
			exp:  []any{float64(42)},
		},
		{
			test: "double_float",
			path: `$.x.double()`,
			json: map[string]any{"x": float64(98.6)},
			exp:  []any{float64(98.6)},
		},
		{
			test: "double_json_number",
			path: `$.x.double()`,
			json: map[string]any{"x": json.Number("1024.3")},
			exp:  []any{float64(1024.3)},
		},
		{
			test: "double_invalid_json_number",
			path: `$.x.double()`,
			json: map[string]any{"x": json.Number("hi")},
			err:  `exec: argument "hi" of jsonpath item method .double() is invalid for type double precision`,
		},
		{
			test: "double_string",
			path: `$.x.double()`,
			json: map[string]any{"x": "1024.3"},
			exp:  []any{float64(1024.3)},
		},
		{
			test: "double_invalid_string",
			path: `$.x.double()`,
			json: map[string]any{"x": "lol"},
			err:  `exec: argument "lol" of jsonpath item method .double() is invalid for type double precision`,
		},
		{
			test: "double_array",
			path: `$.x.double()`,
			json: map[string]any{"x": []any{"1024.3", int64(42)}},
			exp:  []any{float64(1024.3), float64(42)},
		},
		{
			test: "strict_double_array",
			path: `strict $.x.double()`,
			json: map[string]any{"x": []any{"1024.3", int64(42)}},
			err:  "exec: jsonpath item method .double() can only be applied to a string or numeric value",
		},
		{
			test: "double_bool",
			path: `strict $.x.double()`,
			json: map[string]any{"x": true},
			err:  "exec: jsonpath item method .double() can only be applied to a string or numeric value",
		},
		{
			test: "double_infinity",
			path: `strict $.x.double()`,
			json: map[string]any{"x": "infinity"},
			err:  "exec: NaN or Infinity is not allowed for jsonpath item method .double()",
		},
		{
			test: "double_nan",
			path: `strict $.x.double()`,
			json: map[string]any{"x": "NaN"},
			err:  "exec: NaN or Infinity is not allowed for jsonpath item method .double()",
		},
		{
			test: "double_null",
			path: `strict $.x.double()`,
			json: map[string]any{"x": nil},
			err:  "exec: jsonpath item method .double() can only be applied to a string or numeric value",
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestExecuteBigintMethod(t *testing.T) {
	t.Parallel()
	for _, tc := range []execTestCase{
		{
			test: "int_int",
			path: `$.x.bigint()`,
			json: map[string]any{"x": int64(9876543219)},
			exp:  []any{int64(9876543219)},
		},
		{
			test: "int_float",
			path: `$.x.bigint()`,
			json: map[string]any{"x": float64(42.3)},
			exp:  []any{int64(42)},
		},
		{
			test: "int_json_number",
			path: `$.x.bigint()`,
			json: map[string]any{"x": json.Number("9876543219")},
			exp:  []any{int64(9876543219)},
		},
		{
			test: "int_json_number_float",
			path: `$.x.bigint()`,
			json: map[string]any{"x": json.Number("9876543219.2")},
			exp:  []any{int64(9876543219)},
		},
		{
			test: "int_string",
			path: `$.x.bigint()`,
			json: map[string]any{"x": "99"},
			exp:  []any{int64(99)},
		},
		{
			test: "int_array",
			path: `$.x.bigint()`,
			json: map[string]any{"x": []any{"99", int64(1024)}},
			exp:  []any{int64(99), int64(1024)},
		},
		{
			test: "int_array_strict",
			path: `strict $.x.bigint()`,
			json: map[string]any{"x": []any{"99", int64(1024)}},
			err:  "exec: jsonpath item method .bigint() can only be applied to a string or numeric value",
		},
		{
			test: "int_obj",
			path: `$.x.bigint()`,
			json: map[string]any{"x": map[string]any{"99": int64(1024)}},
			err:  "exec: jsonpath item method .bigint() can only be applied to a string or numeric value",
		},
		{
			test: "int_next",
			path: "$.x.bigint().abs()",
			json: map[string]any{"x": int64(-9876543219)},
			exp:  []any{int64(9876543219)},
		},
		{
			test: "int_null",
			path: "$.x.bigint()",
			json: map[string]any{"x": nil},
			err:  "exec: jsonpath item method .bigint() can only be applied to a string or numeric value",
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestExecuteIntegerMethod(t *testing.T) {
	t.Parallel()
	for _, tc := range []execTestCase{
		{
			test: "int_int",
			path: `$.x.integer()`,
			json: map[string]any{"x": int64(42)},
			exp:  []any{int64(42)},
		},
		{
			test: "int_bigint",
			path: `$.x.integer()`,
			json: map[string]any{"x": int64(9876543219)},
			err:  `exec: argument "9876543219" of jsonpath item method .integer() is invalid for type integer`,
		},
		{
			test: "int_bigint_neg",
			path: `$.x.integer()`,
			json: map[string]any{"x": int64(-3147483648)},
			err:  `exec: argument "-3147483648" of jsonpath item method .integer() is invalid for type integer`,
		},
		{
			test: "int_float",
			path: `$.x.integer()`,
			json: map[string]any{"x": float64(42.3)},
			exp:  []any{int64(42)},
		},
		{
			test: "int_json_number",
			path: `$.x.integer()`,
			json: map[string]any{"x": json.Number("42")},
			exp:  []any{int64(42)},
		},
		{
			test: "int_json_number_float",
			path: `$.x.integer()`,
			json: map[string]any{"x": json.Number("42.2")},
			exp:  []any{int64(42)},
		},
		{
			test: "int_json_number_big",
			path: `$.x.integer()`,
			json: map[string]any{"x": json.Number("9876543219")},
			err:  `exec: argument "9876543219" of jsonpath item method .integer() is invalid for type integer`,
		},
		{
			test: "int_json_number_big_neg",
			path: `$.x.integer()`,
			json: map[string]any{"x": json.Number("-3147483648")},
			err:  `exec: argument "-3147483648" of jsonpath item method .integer() is invalid for type integer`,
		},
		{
			test: "int_string",
			path: `$.x.integer()`,
			json: map[string]any{"x": "99"},
			exp:  []any{int64(99)},
		},
		{
			test: "int_string_big",
			path: `$.x.integer()`,
			json: map[string]any{"x": "9876543219"},
			err:  `exec: argument "9876543219" of jsonpath item method .integer() is invalid for type integer`,
		},
		{
			test: "int_string_big_neg",
			path: `$.x.integer()`,
			json: map[string]any{"x": "-3147483648"},
			err:  `exec: argument "-3147483648" of jsonpath item method .integer() is invalid for type integer`,
		},
		{
			test: "int_array",
			path: `$.x.integer()`,
			json: map[string]any{"x": []any{"99", int64(1024)}},
			exp:  []any{int64(99), int64(1024)},
		},
		{
			test: "int_array_strict",
			path: `strict $.x.integer()`,
			json: map[string]any{"x": []any{"99", int64(1024)}},
			err:  "exec: jsonpath item method .integer() can only be applied to a string or numeric value",
		},
		{
			test: "int_obj",
			path: `$.x.integer()`,
			json: map[string]any{"x": map[string]any{"99": int64(1024)}},
			err:  "exec: jsonpath item method .integer() can only be applied to a string or numeric value",
		},
		{
			test: "int_next",
			path: "$.x.integer().abs()",
			json: map[string]any{"x": int64(-42)},
			exp:  []any{int64(42)},
		},
		{
			test: "int_null",
			path: "$.x.integer()",
			json: map[string]any{"x": nil},
			err:  "exec: jsonpath item method .integer() can only be applied to a string or numeric value",
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestExecuteStringMethod(t *testing.T) {
	t.Parallel()
	for _, tc := range []execTestCase{
		{
			test: "string_string",
			path: `$.x.string()`,
			json: map[string]any{"x": "hi"},
			exp:  []any{"hi"},
		},
		{
			test: "datetime_string",
			path: `$.x.datetime().string()`,
			json: map[string]any{"x": "2024-05-05"},
			exp:  []any{"2024-05-05"},
		},
		{
			test: "date_string",
			path: `$.x.date().string()`,
			json: map[string]any{"x": "2024-05-05"},
			exp:  []any{"2024-05-05"},
		},
		{
			test: "time_string",
			path: `$.x.time().string()`,
			json: map[string]any{"x": "12:34:56"},
			exp:  []any{"12:34:56"},
		},
		{
			test: "time_tz_string",
			path: `$.x.time_tz().string()`,
			json: map[string]any{"x": "12:34:56Z"},
			exp:  []any{"12:34:56+00:00"},
		},
		{
			test: "timestamp_string",
			path: `$.x.timestamp().string()`,
			json: map[string]any{"x": "2024-05-05 12:34:56"},
			exp:  []any{"2024-05-05T12:34:56"},
		},
		{
			test: "timestamp_tz_string",
			path: `$.x.timestamp_tz().string()`,
			json: map[string]any{"x": "2024-05-05 12:34:56Z"},
			exp:  []any{pt(context.Background(), "2024-05-05 12:34:56Z").String()},
		},
		{
			test: "json_number_string",
			path: `$.x.string()`,
			json: map[string]any{"x": json.Number("142")},
			exp:  []any{"142"},
		},
		{
			test: "int_string",
			path: `$.x.string()`,
			json: map[string]any{"x": int64(42)},
			exp:  []any{"42"},
		},
		{
			test: "float_string",
			path: `$.x.string()`,
			json: map[string]any{"x": float64(42.3)},
			exp:  []any{"42.3"},
		},
		{
			test: "true_string",
			path: `$.x.string()`,
			json: map[string]any{"x": true},
			exp:  []any{"true"},
		},
		{
			test: "false_string",
			path: `$.x.string()`,
			json: map[string]any{"x": false},
			exp:  []any{"false"},
		},
		{
			test: "null_string",
			path: `$.x.string()`,
			json: map[string]any{"x": nil},
			err:  `exec: jsonpath item method .string() can only be applied to a boolean, string, numeric, or datetime value`,
		},
		{
			test: "array_string",
			path: `$.x.string()`,
			json: map[string]any{"x": []any{int64(42), true}},
			exp:  []any{"42", "true"},
		},
		{
			test: "obj_string",
			path: `$.x.string()`,
			json: map[string]any{"x": map[string]any{"hi": 42}},
			err:  `exec: jsonpath item method .string() can only be applied to a boolean, string, numeric, or datetime value`,
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestExecuteBooleanMethod(t *testing.T) {
	t.Parallel()
	for _, tc := range []execTestCase{
		{
			test: "bool_true",
			path: "$.x.boolean()",
			json: map[string]any{"x": true},
			exp:  []any{true},
		},
		{
			test: "bool_false",
			path: "$.x.boolean()",
			json: map[string]any{"x": false},
			exp:  []any{false},
		},
		{
			test: "bool_int_1",
			path: "$.x.boolean()",
			json: map[string]any{"x": int64(1)},
			exp:  []any{true},
		},
		{
			test: "bool_int_42",
			path: "$.x.boolean()",
			json: map[string]any{"x": int64(42)},
			exp:  []any{true},
		},
		{
			test: "bool_int_0",
			path: "$.x.boolean()",
			json: map[string]any{"x": int64(0)},
			exp:  []any{false},
		},
		{
			test: "bool_float",
			path: "$.x.boolean()",
			json: map[string]any{"x": float64(0.1)},
			err:  `exec: argument "0.1" of jsonpath item method .boolean() is invalid for type boolean`,
		},
		{
			test: "bool_json_number",
			path: "$.x.boolean()",
			json: map[string]any{"x": json.Number("-42")},
			exp:  []any{true},
		},
		{
			test: "bool_json_number_float",
			path: "$.x.boolean()",
			json: map[string]any{"x": json.Number("-42.1")},
			err:  `exec: argument "-42.1" of jsonpath item method .boolean() is invalid for type boolean`,
		},
		{
			test: "bool_string_t",
			path: "$.x.boolean()",
			json: map[string]any{"x": "t"},
			exp:  []any{true},
		},
		{
			test: "bool_string_T",
			path: "$.x.boolean()",
			json: map[string]any{"x": "T"},
			exp:  []any{true},
		},
		{
			test: "bool_string_true",
			path: "$.x.boolean()",
			json: map[string]any{"x": "true"},
			exp:  []any{true},
		},
		{
			test: "bool_string_TRUE",
			path: "$.x.boolean()",
			json: map[string]any{"x": "TRUE"},
			exp:  []any{true},
		},
		{
			test: "bool_string_TrUe",
			path: "$.x.boolean()",
			json: map[string]any{"x": "TrUe"},
			exp:  []any{true},
		},
		{
			test: "bool_string_trunk",
			path: "$.x.boolean()",
			json: map[string]any{"x": "trunk"},
			err:  `exec: argument "trunk" of jsonpath item method .boolean() is invalid for type boolean`,
		},
		{
			test: "bool_string_f",
			path: "$.x.boolean()",
			json: map[string]any{"x": "f"},
			exp:  []any{false},
		},
		{
			test: "bool_string_F",
			path: "$.x.boolean()",
			json: map[string]any{"x": "F"},
			exp:  []any{false},
		},
		{
			test: "bool_string_false",
			path: "$.x.boolean()",
			json: map[string]any{"x": "false"},
			exp:  []any{false},
		},
		{
			test: "bool_string_FALSE",
			path: "$.x.boolean()",
			json: map[string]any{"x": "FALSE"},
			exp:  []any{false},
		},
		{
			test: "bool_string_FaLsE",
			path: "$.x.boolean()",
			json: map[string]any{"x": "FaLsE"},
			exp:  []any{false},
		},
		{
			test: "bool_string_flunk",
			path: "$.x.boolean()",
			json: map[string]any{"x": "flunk"},
			err:  `exec: argument "flunk" of jsonpath item method .boolean() is invalid for type boolean`,
		},
		{
			test: "bool_string_y",
			path: "$.x.boolean()",
			json: map[string]any{"x": "y"},
			exp:  []any{true},
		},
		{
			test: "bool_string_Y",
			path: "$.x.boolean()",
			json: map[string]any{"x": "Y"},
			exp:  []any{true},
		},
		{
			test: "bool_string_yes",
			path: "$.x.boolean()",
			json: map[string]any{"x": "yes"},
			exp:  []any{true},
		},
		{
			test: "bool_string_YES",
			path: "$.x.boolean()",
			json: map[string]any{"x": "YES"},
			exp:  []any{true},
		},
		{
			test: "bool_string_YeS",
			path: "$.x.boolean()",
			json: map[string]any{"x": "YeS"},
			exp:  []any{true},
		},
		{
			test: "bool_string_yet",
			path: "$.x.boolean()",
			json: map[string]any{"x": "yet"},
			err:  `exec: argument "yet" of jsonpath item method .boolean() is invalid for type boolean`,
		},
		{
			test: "bool_string_n",
			path: "$.x.boolean()",
			json: map[string]any{"x": "n"},
			exp:  []any{false},
		},
		{
			test: "bool_string_N",
			path: "$.x.boolean()",
			json: map[string]any{"x": "N"},
			exp:  []any{false},
		},
		{
			test: "bool_string_no",
			path: "$.x.boolean()",
			json: map[string]any{"x": "no"},
			exp:  []any{false},
		},
		{
			test: "bool_string_NO",
			path: "$.x.boolean()",
			json: map[string]any{"x": "NO"},
			exp:  []any{false},
		},
		{
			test: "bool_string_nO",
			path: "$.x.boolean()",
			json: map[string]any{"x": "nO"},
			exp:  []any{false},
		},
		{
			test: "bool_string_not",
			path: "$.x.boolean()",
			json: map[string]any{"x": "not"},
			err:  `exec: argument "not" of jsonpath item method .boolean() is invalid for type boolean`,
		},
		{
			test: "bool_string_on",
			path: "$.x.boolean()",
			json: map[string]any{"x": "on"},
			exp:  []any{true},
		},
		{
			test: "bool_string_ON",
			path: "$.x.boolean()",
			json: map[string]any{"x": "ON"},
			exp:  []any{true},
		},
		{
			test: "bool_string_oN",
			path: "$.x.boolean()",
			json: map[string]any{"x": "oN"},
			exp:  []any{true},
		},
		{
			test: "bool_string_o",
			path: "$.x.boolean()",
			json: map[string]any{"x": "o"},
			err:  `exec: argument "o" of jsonpath item method .boolean() is invalid for type boolean`,
		},
		{
			test: "bool_string_off",
			path: "$.x.boolean()",
			json: map[string]any{"x": "off"},
			exp:  []any{false},
		},
		{
			test: "bool_string_OFF",
			path: "$.x.boolean()",
			json: map[string]any{"x": "OFF"},
			exp:  []any{false},
		},
		{
			test: "bool_string_OfF",
			path: "$.x.boolean()",
			json: map[string]any{"x": "OfF"},
			exp:  []any{false},
		},
		{
			test: "bool_string_oft",
			path: "$.x.boolean()",
			json: map[string]any{"x": "oft"},
			err:  `exec: argument "oft" of jsonpath item method .boolean() is invalid for type boolean`,
		},
		{
			test: "bool_string_1",
			path: "$.x.boolean()",
			json: map[string]any{"x": "1"},
			exp:  []any{true},
		},
		{
			test: "bool_string_1up",
			path: "$.x.boolean()",
			json: map[string]any{"x": "1up"},
			err:  `exec: argument "1up" of jsonpath item method .boolean() is invalid for type boolean`,
		},
		{
			test: "bool_string_0",
			path: "$.x.boolean()",
			json: map[string]any{"x": "0"},
			exp:  []any{false},
		},
		{
			test: "bool_string_0n",
			path: "$.x.boolean()",
			json: map[string]any{"x": "0n"},
			err:  `exec: argument "0n" of jsonpath item method .boolean() is invalid for type boolean`,
		},
		{
			test: "bool_array",
			path: "$.x.boolean()",
			json: map[string]any{"x": []any{"0", true}},
			exp:  []any{false, true},
		},
		{
			test: "bool_array_strict",
			path: "strict $.x.boolean()",
			json: map[string]any{"x": []any{"0", true}},
			err:  `exec: jsonpath item method .boolean() can only be applied to a boolean, string, or numeric value`,
		},
		{
			test: "bool_obj",
			path: "strict $.x.boolean()",
			json: map[string]any{"x": map[string]any{"0": true}},
			err:  `exec: jsonpath item method .boolean() can only be applied to a boolean, string, or numeric value`,
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}
