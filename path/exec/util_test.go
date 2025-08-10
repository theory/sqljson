package exec

import (
	"encoding/json"
	"math"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/theory/sqljson/path/ast"
)

func TestCastJSONNumber(t *testing.T) {
	t.Parallel()

	doubleInt := func(i int64) int64 { return i * 2 }
	doubleFloat := func(i float64) float64 { return i * 2 }

	for _, tc := range []struct {
		test string
		num  json.Number
		exp  any
		ok   bool
	}{
		{
			test: "int",
			num:  json.Number("42"),
			exp:  doubleInt(42),
			ok:   true,
		},
		{
			test: "float",
			num:  json.Number("98.6"),
			exp:  doubleFloat(98.6),
			ok:   true,
		},
		{
			test: "nan",
			num:  json.Number("foo"),
			ok:   false,
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)

			val, ok := castJSONNumber(tc.num, doubleInt, doubleFloat)
			a.Equal(tc.exp, val)
			a.Equal(ok, tc.ok)
		})
	}
}

func TestGetNodeInt32(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		test  string
		node  ast.Node
		meth  string
		field string
		exp   int
		err   string
		isErr error
	}{
		{
			test: "int",
			node: ast.NewInteger("42"),
			exp:  42,
		},
		{
			test:  "numeric",
			node:  ast.NewNumeric("98.6"),
			meth:  ".hi()",
			field: "xxx",
			err:   `exec: invalid jsonpath item type for .hi() xxx`,
			isErr: ErrExecution,
		},
		{
			test:  "string",
			node:  ast.NewString("foo"),
			meth:  ".hi()",
			field: "xxx",
			err:   `exec: invalid jsonpath item type for .hi() xxx`,
			isErr: ErrExecution,
		},
		{
			test:  "too_big",
			node:  ast.NewInteger(strconv.FormatInt(int64(math.MaxInt32+1), 10)),
			meth:  ".go()",
			field: "aaa",
			err:   `exec: aaa of jsonpath item method .go() is out of integer range`,
			isErr: ErrExecution,
		},
		{
			test:  "too_small",
			node:  ast.NewInteger(strconv.FormatInt(int64(math.MinInt32-1), 10)),
			meth:  ".go()",
			field: "aaa",
			err:   `exec: aaa of jsonpath item method .go() is out of integer range`,
			isErr: ErrExecution,
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)
			r := require.New(t)

			val, err := getNodeInt32(tc.node, tc.meth, tc.field)
			a.Equal(tc.exp, val)
			if tc.isErr == nil {
				r.NoError(err)
			} else {
				r.EqualError(err, tc.err)
				r.ErrorIs(err, tc.isErr)
			}
		})
	}
}

func TestGetJSONInt32(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		test  string
		val   any
		op    string
		exp   int
		err   string
		isErr error
	}{
		{
			test: "int",
			val:  int64(42),
			exp:  42,
		},
		{
			test: "float",
			val:  float64(42),
			exp:  42,
		},
		{
			test: "float_trunc_2",
			val:  float64(42.2),
			exp:  42,
		},
		{
			test: "float_trunc_5",
			val:  float64(42.5),
			exp:  42,
		},
		{
			test: "float_trunc_9",
			val:  float64(42.9),
			exp:  42,
		},
		{
			test: "json_num_int",
			val:  json.Number("99"),
			exp:  99,
		},
		{
			test: "json_num_float",
			val:  json.Number("99.0"),
			exp:  99,
		},
		{
			test: "json_num_float_trunc_2",
			val:  json.Number("99.2"),
			exp:  99,
		},
		{
			test: "json_num_float_trunc_5",
			val:  json.Number("99.5"),
			exp:  99,
		},
		{
			test: "json_num_float_trunc_9",
			val:  json.Number("99.999"),
			exp:  99,
		},
		{
			test:  "float_nan",
			val:   math.NaN(),
			op:    "myThing",
			err:   `exec: NaN or Infinity is not allowed for jsonpath myThing`,
			isErr: ErrVerbose,
		},
		{
			test:  "float_inf",
			val:   math.Inf(1),
			op:    "myThing",
			err:   `exec: NaN or Infinity is not allowed for jsonpath myThing`,
			isErr: ErrVerbose,
		},
		{
			test:  "json_invalid",
			val:   json.Number("oof"),
			op:    "oof",
			err:   `exec invalid: jsonpath oof is not a single numeric value`,
			isErr: ErrInvalid,
		},
		{
			test:  "json_nan",
			val:   json.Number("nan"),
			op:    "xyz",
			err:   `exec: NaN or Infinity is not allowed for jsonpath xyz`,
			isErr: ErrVerbose,
		},
		{
			test:  "json_inf",
			val:   json.Number("-inf"),
			op:    "xyz",
			err:   `exec: NaN or Infinity is not allowed for jsonpath xyz`,
			isErr: ErrVerbose,
		},
		{
			test:  "string",
			val:   "hi",
			op:    "xxx",
			err:   `exec: jsonpath xxx is not a single numeric value`,
			isErr: ErrVerbose,
		},
		{
			test:  "too_big",
			val:   int64(math.MaxInt32 + 1),
			op:    "max",
			err:   `exec: jsonpath max is out of integer range`,
			isErr: ErrVerbose,
		},
		{
			test:  "too_small",
			val:   int64(math.MinInt32 - 1),
			op:    "max",
			err:   `exec: jsonpath max is out of integer range`,
			isErr: ErrVerbose,
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)
			r := require.New(t)

			val, err := getJSONInt32(tc.val, tc.op)
			a.Equal(tc.exp, val)
			if tc.isErr == nil {
				r.NoError(err)
			} else {
				r.EqualError(err, tc.err)
				r.ErrorIs(err, tc.isErr)
			}
		})
	}
}
