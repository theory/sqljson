package exec

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/theory/sqljson/path/ast"
	"github.com/theory/sqljson/path/parser"
	"github.com/theory/sqljson/path/types"
)

func TestCompareItems(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	r := require.New(t)
	now := time.Now()
	ctx := context.Background()

	for _, tc := range []struct {
		name  string
		path  string
		left  any
		right any
		exp   predOutcome
		err   string
		isErr error
	}{
		{
			name:  "not_binary",
			path:  "$",
			exp:   predUnknown,
			err:   `exec invalid: invalid node type *ast.ConstNode passed to compareItems`,
			isErr: ErrInvalid,
		},
		{
			name:  "left_eq_null",
			path:  "$ == $",
			right: int64(6),
			exp:   predFalse,
		},
		{
			name:  "left_ne_null",
			path:  "$ != $",
			right: int64(6),
			exp:   predTrue,
		},
		{
			name: "right_eq_null",
			path: "$ == $",
			left: int64(6),
			exp:  predFalse,
		},
		{
			name: "right_ne_null",
			path: "$ != $",
			left: int64(6),
			exp:  predTrue,
		},
		{
			name: "both_null",
			path: "$ == $",
			exp:  predTrue,
		},
		{
			name:  "bool_true",
			path:  "$ == $",
			left:  true,
			right: true,
			exp:   predTrue,
		},
		{
			name:  "bool_false",
			path:  "$ == $",
			left:  true,
			right: false,
			exp:   predFalse,
		},
		{
			name:  "bool_unknown",
			path:  "$ == $",
			left:  true,
			right: "true",
			exp:   predUnknown,
		},
		{
			name:  "int_true",
			path:  "$ == $",
			left:  int64(3),
			right: int64(3),
			exp:   predTrue,
		},
		{
			name:  "int_false",
			path:  "$ == $",
			left:  int64(3),
			right: int64(4),
			exp:   predFalse,
		},
		{
			name:  "int_unknown",
			path:  "$ == $",
			left:  int64(3),
			right: "4",
			exp:   predUnknown,
		},
		{
			name:  "float_true",
			path:  "$ == $",
			left:  float64(3.0),
			right: float64(3.0),
			exp:   predTrue,
		},
		{
			name:  "float_false",
			path:  "$ == $",
			left:  float64(3.1),
			right: float64(4.2),
			exp:   predFalse,
		},
		{
			name:  "float_unknown",
			path:  "$ == $",
			left:  float64(3.0),
			right: "3.0",
			exp:   predUnknown,
		},
		{
			name:  "json_number_true",
			path:  "$ == $",
			left:  json.Number("3"),
			right: json.Number("3"),
			exp:   predTrue,
		},
		{
			name:  "json_number_false",
			path:  "$ == $",
			left:  json.Number("3"),
			right: json.Number("3.1"),
			exp:   predFalse,
		},
		{
			name:  "json_number_unknown",
			path:  "$ == $",
			left:  json.Number("3"),
			right: "3",
			exp:   predUnknown,
		},
		{
			name:  "string_true",
			path:  "$ == $",
			left:  "abc",
			right: "abc",
			exp:   predTrue,
		},
		{
			name:  "string_false",
			path:  "$ == $",
			left:  "abc",
			right: "abd",
			exp:   predFalse,
		},
		{
			name:  "string_unknown",
			path:  "$ == $",
			left:  "abc",
			right: false,
			exp:   predUnknown,
		},
		{
			name:  "string_ne_true",
			path:  "$ != $",
			left:  "abc",
			right: "abd",
			exp:   predTrue,
		},
		{
			name:  "datetime_true",
			path:  "$ == $",
			left:  types.NewDate(now),
			right: types.NewDate(now),
			exp:   predTrue,
		},
		{
			name:  "datetime_false",
			path:  "$ == $",
			left:  types.NewDate(now),
			right: types.NewTimestamp(now),
			exp:   predFalse,
		},
		{
			name:  "datetime_unknown",
			path:  "$ == $",
			left:  types.NewDate(now),
			right: types.NewTime(now),
			exp:   predUnknown,
		},
		{
			name:  "datetime_unknown_err",
			path:  "$ == $",
			left:  types.NewDate(now),
			right: "not a date",
			exp:   predUnknown,
			err:   `exec invalid: unrecognized SQL/JSON datetime type string`,
			isErr: ErrInvalid,
		},
		{
			name:  "object_unknown",
			path:  "$ == $",
			left:  map[string]any{},
			right: false,
			exp:   predUnknown,
		},
		{
			name:  "array_unknown",
			path:  "$ == $",
			left:  []any{},
			right: false,
			exp:   predUnknown,
		},
		{
			name:  "anything_else",
			path:  "$ == $",
			left:  int32(3),
			right: false,
			exp:   predUnknown,
			err:   `exec invalid: invalid json value type int32`,
			isErr: ErrInvalid,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Parse the path.
			path, err := parser.Parse(tc.path)
			r.NoError(err)

			// Execute compareItems.
			e := newTestExecutor(path, nil, true, false)
			res, err := e.compareItems(ctx, path.Root(), tc.left, tc.right)
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

func TestCompareBool(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tc := range []struct {
		name  string
		path  string
		left  bool
		right any
		exp   int
		ok    bool
	}{
		{
			name:  "true_true",
			left:  true,
			right: true,
			exp:   0,
			ok:    true,
		},
		{
			name:  "true_false",
			left:  true,
			right: false,
			exp:   1,
			ok:    true,
		},
		{
			name:  "false_true",
			left:  false,
			right: true,
			exp:   -1,
			ok:    true,
		},
		{
			name:  "false_false",
			left:  false,
			right: false,
			exp:   0,
			ok:    true,
		},
		{
			name:  "right_not_bool",
			left:  false,
			right: "false",
			exp:   0,
			ok:    false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			res, ok := compareBool(tc.left, tc.right)
			a.Equal(tc.exp, res)
			a.Equal(tc.ok, ok)
		})
	}
}

func TestApplyCompare(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	r := require.New(t)

	for _, tc := range []struct {
		name string
		op   ast.BinaryOperator
		exp  []predOutcome
		err  bool
	}{
		{
			name: "equal",
			op:   ast.BinaryEqual,
			exp:  []predOutcome{predFalse, predTrue, predFalse},
		},
		{
			name: "not_equal",
			op:   ast.BinaryNotEqual,
			exp:  []predOutcome{predTrue, predFalse, predTrue},
		},
		{
			name: "lt",
			op:   ast.BinaryLess,
			exp:  []predOutcome{predTrue, predFalse, predFalse},
		},
		{
			name: "gt",
			op:   ast.BinaryGreater,
			exp:  []predOutcome{predFalse, predFalse, predTrue},
		},
		{
			name: "le",
			op:   ast.BinaryLessOrEqual,
			exp:  []predOutcome{predTrue, predTrue, predFalse},
		},
		{
			name: "ge",
			op:   ast.BinaryGreaterOrEqual,
			exp:  []predOutcome{predFalse, predTrue, predTrue},
		},
		{
			name: "add",
			op:   ast.BinaryAdd,
			err:  true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			for i, cmp := range []int{-1, 0, 1} {
				res, err := applyCompare(tc.op, cmp)
				if tc.err {
					r.EqualError(err, "exec invalid: unrecognized jsonpath comparison operation +")
					r.ErrorIs(err, ErrInvalid)
					a.Equal(predUnknown, res)
				} else {
					r.NoError(err)
					a.Equal(tc.exp[i], res)
				}
			}
		})
	}
}

func TestCompareNumbers(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	t.Run("int_int", func(t *testing.T) {
		t.Parallel()
		a.Equal(0, compareNumbers(42, 42))
		a.Equal(-1, compareNumbers(42, 43))
		a.Equal(1, compareNumbers(42, 41))
	})

	t.Run("int_int64", func(t *testing.T) {
		t.Parallel()
		a.Equal(0, compareNumbers(42, int64(42)))
		a.Equal(-1, compareNumbers(42, int64(43)))
		a.Equal(1, compareNumbers(42, int64(41)))
	})

	t.Run("int_float64", func(t *testing.T) {
		t.Parallel()
		a.Equal(0, compareNumbers(42, float64(42.0)))
		a.Equal(-1, compareNumbers(42, float64(42.1)))
		a.Equal(1, compareNumbers(42, float64(41.9)))
	})

	t.Run("int64_int", func(t *testing.T) {
		t.Parallel()
		a.Equal(0, compareNumbers(int64(42), 42))
		a.Equal(-1, compareNumbers(int64(42), 43))
		a.Equal(1, compareNumbers(int64(42), 41))
	})

	t.Run("int64_int64", func(t *testing.T) {
		t.Parallel()
		a.Equal(0, compareNumbers(int64(42), int64(42)))
		a.Equal(-1, compareNumbers(int64(42), int64(43)))
		a.Equal(1, compareNumbers(int64(42), int64(41)))
	})

	t.Run("float64_int", func(t *testing.T) {
		t.Parallel()
		a.Equal(0, compareNumbers(float64(42.0), 42))
		a.Equal(-1, compareNumbers(float64(41.9), 42))
		a.Equal(1, compareNumbers(float64(42.1), 42))
	})

	t.Run("float64_float64", func(t *testing.T) {
		t.Parallel()
		a.Equal(0, compareNumbers(float64(42.0), float64(42.00)))
		a.Equal(-1, compareNumbers(float64(42), float64(42.1)))
		a.Equal(1, compareNumbers(float64(42.0), float64(41.9)))
	})
}

func TestCompareNumeric(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tc := range []struct {
		name  string
		left  any
		right any
		exp   int
		panic bool
	}{
		{
			name:  "int64_int64_eq",
			left:  int64(42),
			right: int64(42),
			exp:   0,
		},
		{
			name:  "int64_int64_lt",
			left:  int64(41),
			right: int64(42),
			exp:   -1,
		},
		{
			name:  "int64_int64_gt",
			left:  int64(43),
			right: int64(42),
			exp:   1,
		},
		{
			name:  "int64_float64_eq",
			left:  int64(42),
			right: float64(42.0),
			exp:   0,
		},
		{
			name:  "int64_float64_lt",
			left:  int64(42),
			right: float64(42.1),
			exp:   -1,
		},
		{
			name:  "int64_float64_gt",
			left:  int64(42),
			right: float64(41.9),
			exp:   1,
		},
		{
			name:  "int64_json_int_eq",
			left:  int64(42),
			right: json.Number("42"),
			exp:   0,
		},
		{
			name:  "int64_json_int_lt",
			left:  int64(42),
			right: json.Number("43"),
			exp:   -1,
		},
		{
			name:  "int64_json_int_gt",
			left:  int64(42),
			right: json.Number("41"),
			exp:   1,
		},
		{
			name:  "int64_json_float_eq",
			left:  int64(42),
			right: json.Number("42.0"),
			exp:   0,
		},
		{
			name:  "int64_json_float_lt",
			left:  int64(42),
			right: json.Number("42.1"),
			exp:   -1,
		},
		{
			name:  "int64_json_float_gt",
			left:  int64(42),
			right: json.Number("41.9"),
			exp:   1,
		},
		{
			name:  "int64_json_err",
			left:  int64(42),
			right: json.Number("nope"),
			panic: true,
		},
		{
			name:  "float64_float64_eq",
			left:  float64(42),
			right: float64(42.0),
			exp:   0,
		},
		{
			name:  "float64_float64_lt",
			left:  float64(42),
			right: float64(42.1),
			exp:   -1,
		},
		{
			name:  "float64_float64_gt",
			left:  float64(42),
			right: float64(41.9),
			exp:   1,
		},
		{
			name:  "float64_int64_eq",
			left:  float64(42.0),
			right: int64(42),
			exp:   0,
		},
		{
			name:  "float64_int64_lt",
			left:  float64(41.9),
			right: int64(42),
			exp:   -1,
		},
		{
			name:  "float64_int64_gt",
			left:  float64(42.1),
			right: int64(42),
			exp:   1,
		},
		{
			name:  "float64_json_eq",
			left:  float64(42.0),
			right: json.Number("42"),
			exp:   0,
		},
		{
			name:  "float64_json_lt",
			left:  float64(41.9),
			right: json.Number("42.1"),
			exp:   -1,
		},
		{
			name:  "float64_json_gt",
			left:  float64(42.1),
			right: json.Number("41.9"),
			exp:   1,
		},
		{
			name:  "float64_json_err",
			left:  float64(42.1),
			right: json.Number("nope"),
			panic: true,
		},
		{
			name:  "json_json_eq",
			left:  json.Number("42.0"),
			right: json.Number("42"),
			exp:   0,
		},
		{
			name:  "json_json_lt",
			left:  json.Number("42.0"),
			right: json.Number("42.1"),
			exp:   -1,
		},
		{
			name:  "json_json_gt",
			left:  json.Number("42.1"),
			right: json.Number("42"),
			exp:   1,
		},
		{
			name:  "json_int64_eq",
			left:  json.Number("42"),
			right: int64(42),
			exp:   0,
		},
		{
			name:  "json_int64_lt",
			left:  json.Number("42.0"),
			right: int64(43),
			exp:   -1,
		},
		{
			name:  "json_int64_gt",
			left:  json.Number("42.1"),
			right: int64(42),
			exp:   1,
		},
		{
			name:  "json_float64_eq",
			left:  json.Number("42.0"),
			right: float64(42),
			exp:   0,
		},
		{
			name:  "json_float64_lt",
			left:  json.Number("41.9"),
			right: float64(42),
			exp:   -1,
		},
		{
			name:  "json_float64_gt",
			left:  json.Number("42.1"),
			right: float64(42),
			exp:   1,
		},
		{
			name:  "json_err",
			left:  json.Number("nope"),
			panic: true,
		},
		{
			name:  "not_numeric",
			left:  "hi",
			panic: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.panic {
				a.Panics(func() { compareNumeric(tc.left, tc.right) })
			} else {
				a.Equal(tc.exp, compareNumeric(tc.left, tc.right))
			}
		})
	}
}
