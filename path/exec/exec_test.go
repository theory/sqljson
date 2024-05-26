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

type execTestCase struct {
	name   string
	path   string
	vars   vars
	useTZ  bool
	silent bool
	result resultStatus
	json   any
	exp    []any
	err    string
	rand   bool
}

func newExecutor(path *ast.AST, vars vars, throwErrors, useTZ bool) *Executor {
	return &Executor{
		path:                   path,
		vars:                   vars,
		innermostArraySize:     -1,
		useTZ:                  useTZ,
		ignoreStructuralErrors: path.IsLax(),
		throwErrors:            throwErrors,
		lastGeneratedObjectID:  1,
	}
}

func (tc execTestCase) run(t *testing.T) {
	t.Helper()
	r := require.New(t)
	a := assert.New(t)
	path, err := parser.Parse(tc.path)
	r.NoError(err)
	exec := newExecutor(path, tc.vars, !tc.silent, tc.useTZ)
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
			name: "root_obj",
			path: "$",
			json: map[string]any{"x": 42},
			exp:  []any{map[string]any{"x": 42}},
		},
		{
			name: "root_num",
			path: "$",
			json: 42.0,
			exp:  []any{42.0},
		},
		{
			name: "root_bool",
			path: "$",
			json: true,
			exp:  []any{true},
		},
		{
			name: "root_array",
			path: "$",
			json: []any{42, true, "hi"},
			exp:  []any{[]any{42, true, "hi"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestExecuteLiteral(t *testing.T) {
	t.Parallel()
	for _, tc := range []execTestCase{
		{
			name: "null_only",
			path: "null",
			json: `""`,
			exp:  []any{nil},
		},
		{
			name: "true_only",
			path: "true",
			json: `""`,
			exp:  []any{true},
		},
		{
			name: "false_only",
			path: "false",
			json: `""`,
			exp:  []any{false},
		},
		{
			name: "string",
			path: `"yes"`,
			json: []any{1, 2, 3},
			exp:  []any{"yes"},
		},
		{
			name: "int",
			path: `42`,
			json: nil,
			exp:  []any{int64(42)},
		},
		{
			name: "float",
			path: `42.0`,
			json: nil,
			exp:  []any{float64(42.0)},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestExecutePathKeys(t *testing.T) {
	t.Parallel()
	for _, tc := range []execTestCase{
		{
			name: "path_x",
			path: "$.x",
			json: map[string]any{"x": 42},
			exp:  []any{42},
		},
		{
			name: "path_xy",
			path: "$.x.y",
			json: map[string]any{"x": map[string]any{"y": "hi"}},
			exp:  []any{"hi"},
		},
		{
			name: "path_xyz",
			path: "$.x.y.z",
			json: map[string]any{"x": map[string]any{"y": map[string]any{"z": "yep"}}},
			exp:  []any{"yep"},
		},
		{
			name: "path_xy_obj",
			path: "$.x.y",
			json: map[string]any{"x": map[string]any{"y": map[string]any{"z": "yep"}}},
			exp:  []any{map[string]any{"z": "yep"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestExecuteAny(t *testing.T) {
	t.Parallel()
	for _, tc := range []execTestCase{
		{
			name: "any_key",
			path: "$.*",
			json: map[string]any{"x": "hi", "y": 42},
			exp:  []any{"hi", 42},
			rand: true, // Results can be in any order
		},
		{
			name: "any_key_mixed",
			path: "$.*",
			json: map[string]any{"x": map[string]any{"y": 42}, "z": false},
			exp:  []any{map[string]any{"y": 42}, false},
			rand: true, // Results can be in any order
		},
		{
			name: "any_array",
			path: "$[*]",
			json: []any{"hi", 42},
			exp:  []any{"hi", 42},
		},
		{
			name: "any_array_mixed",
			path: "$[*]",
			json: []any{"hi", 42, true, map[string]any{"x": 1}, nil},
			exp:  []any{"hi", 42, true, map[string]any{"x": 1}, nil},
		},
		{
			name: "path_x_any_array",
			path: "$.x[*]",
			json: map[string]any{"x": []any{"hi", 42}},
			exp:  []any{"hi", 42},
		},
		{
			name: "path_xy_any_array",
			path: "$.x.y[*]",
			json: map[string]any{"x": map[string]any{"y": []any{"hi", 42}}},
			exp:  []any{"hi", 42},
		},
		{
			name: "any",
			path: "$.**",
			json: map[string]any{"x": "hi", "y": 42},
			exp:  []any{map[string]any{"x": "hi", "y": 42}, "hi", 42},
			rand: true, // Results can be in any order
		},
		{
			name: "any_nested",
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
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestExecuteMath(t *testing.T) {
	t.Parallel()
	for _, tc := range []execTestCase{
		{
			name: "add_ints",
			path: "$ + 1",
			json: int64(2),
			exp:  []any{int64(3)},
		},
		{
			name: "add_floats",
			path: "$ + 3.2",
			json: float64(98.6),
			exp:  []any{float64(101.8)},
		},
		{
			name: "add_int_flat",
			path: "$ + 3",
			json: float64(98.6),
			exp:  []any{float64(101.6)},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestExecuteAndOr(t *testing.T) {
	t.Parallel()
	for _, tc := range []execTestCase{
		{
			name: "binary_or_ints",
			path: "$.x == 3 || $.x == 4",
			json: map[string]any{"x": int64(4)},
			exp:  []any{true},
		},
		{
			name: "binary_or_int_float",
			path: "$.x == 3 || $.y == 4.0",
			json: map[string]any{"x": int64(4), "y": float64(4.0)},
			exp:  []any{true},
		},
		{
			name: "binary_and_strings",
			path: `$.x == "hi" && $.y starts with "good"`,
			json: map[string]any{"x": "hi", "y": "good bye"},
			exp:  []any{true},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestExecuteNumberMethods(t *testing.T) {
	t.Parallel()
	for _, tc := range []execTestCase{
		{
			name: "number_method",
			path: `$.x.number()`,
			json: map[string]any{"x": int64(3)},
			exp:  []any{float64(3)},
		},
		{
			name: "number_method_string",
			path: `$.x.number()`,
			json: map[string]any{"x": "3.4"},
			exp:  []any{float64(3.4)},
		},
		{
			name: "number_method_json_number",
			path: `$.x.number()`,
			json: map[string]any{"x": json.Number("3.4")},
			exp:  []any{float64(3.4)},
		},
		{
			name: "number_method_json_number_int",
			path: `$.x.number()`,
			json: map[string]any{"x": json.Number("1714004682")},
			exp:  []any{float64(1714004682)},
		},
		{
			name: "decimal_method",
			path: `$.x.decimal()`,
			json: map[string]any{"x": "12.2"},
			exp:  []any{float64(12.2)},
		},
		{
			name: "decimal_method_precision",
			path: `$.x.decimal(4)`,
			json: map[string]any{"x": "12.2"},
			exp:  []any{float64(12)},
		},
		{
			name: "decimal_method_precision_short",
			path: `$.x.decimal(1)`,
			json: map[string]any{"x": "12.233"},
			// exp:  []any{float64(12)},
			err: `exec: argument "12.233" of jsonpath item method .decimal() is invalid for type numeric`,
		},
		{
			name: "decimal_method_precision_ok",
			path: `$.x.decimal(5,3)`,
			json: map[string]any{"x": "12.233"},
			exp:  []any{float64(12.233)},
		},
		{
			name: "decimal_method_precision_scale",
			path: `$.x.decimal(4, 2)`,
			json: map[string]any{"x": "12.233"},
			exp:  []any{float64(12.23)},
		},
		{
			name: "decimal_method_precision_scale_short",
			path: `$.x.decimal(3, 2)`,
			json: map[string]any{"x": "12.233"},
			err:  `exec: argument "12.233" of jsonpath item method .decimal() is invalid for type numeric`,
		},
		{
			name: "abs_int",
			path: `$.x.abs()`,
			json: map[string]any{"x": int64(-42)},
			exp:  []any{int64(42)},
		},
		{
			name: "abs_float",
			path: `$.x.abs()`,
			json: map[string]any{"x": float64(-42.22)},
			exp:  []any{float64(42.22)},
		},
		{
			name: "abs_json_number_int",
			path: `$.x.abs()`,
			json: map[string]any{"x": json.Number("-99")},
			exp:  []any{int64(99)},
		},
		{
			name: "abs_json_number_float",
			path: `$.x.abs()`,
			json: map[string]any{"x": json.Number("-42.22")},
			exp:  []any{float64(42.22)},
		},
		{
			name: "floor_int",
			path: `$.x.floor()`,
			json: map[string]any{"x": int64(42)},
			exp:  []any{int64(42)},
		},
		{
			name: "floor_float",
			path: `$.x.floor()`,
			json: map[string]any{"x": float64(42.22)},
			exp:  []any{float64(42)},
		},
		{
			name: "floor_json_number_int",
			path: `$.x.floor()`,
			json: map[string]any{"x": json.Number("99")},
			exp:  []any{int64(99)},
		},
		{
			name: "floor_json_number_float",
			path: `$.x.floor()`,
			json: map[string]any{"x": json.Number("88.88")},
			exp:  []any{float64(88)},
		},
		{
			name: "ceiling_int",
			path: `$.x.ceiling()`,
			json: map[string]any{"x": int64(42)},
			exp:  []any{int64(42)},
		},
		{
			name: "ceiling_float",
			path: `$.x.ceiling()`,
			json: map[string]any{"x": float64(42.22)},
			exp:  []any{float64(43)},
		},
		{
			name: "ceiling_json_number_int",
			path: `$.x.ceiling()`,
			json: map[string]any{"x": json.Number("99")},
			exp:  []any{int64(99)},
		},
		{
			name: "ceiling_json_number_float",
			path: `$.x.ceiling()`,
			json: map[string]any{"x": json.Number("88.88")},
			exp:  []any{float64(89)},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestExecuteArraySubscripts(t *testing.T) {
	t.Parallel()
	for _, tc := range []execTestCase{
		{
			name: "array_subscript_0",
			path: `$.x[0]`,
			json: map[string]any{"x": []any{"hi"}},
			exp:  []any{"hi"},
		},
		{
			name: "array_subscript_2",
			path: `$.x[2]`,
			json: map[string]any{"x": []any{"hi", "", true}},
			exp:  []any{true},
		},
		{
			name: "array_subscript_from_to",
			path: `$.x[1 to 2]`,
			json: map[string]any{"x": []any{"xx", "hi", true}},
			exp:  []any{"hi", true},
		},
		{
			name: "array_subscript_last",
			path: `$.x[last]`,
			json: map[string]any{"x": []any{"hi", "", true}},
			exp:  []any{true},
		},
		{
			name: "array_subscript_to_last",
			path: `$.x[1 to last]`,
			json: map[string]any{"x": []any{"hi", "", true}},
			exp:  []any{"", true},
		},
		{
			name: "array_subscript_multi",
			path: `$.x[0, 3 to 4]`,
			json: map[string]any{"x": []any{"hi", "", true, "x", "y"}},
			exp:  []any{"hi", "x", "y"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestExecuteLikeRegex(t *testing.T) {
	t.Parallel()
	for _, tc := range []execTestCase{
		{
			name: "like_regex",
			path: `$.x like_regex "."`,
			json: map[string]any{"x": "x"},
			exp:  []any{true},
		},
		{
			name: "like_regex_prefix",
			path: `$.x like_regex "^hi"`,
			json: map[string]any{"x": "hi there"},
			exp:  []any{true},
		},
		{
			name: "like_regex_false",
			path: `$.x like_regex "^hi"`,
			json: map[string]any{"x": "say hi there"},
			exp:  []any{false},
		},
		{
			name: "like_regex_flag",
			path: `$.x like_regex "^hi" flag "i"`,
			json: map[string]any{"x": "HIGH"},
			exp:  []any{true},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestExecuteFilter(t *testing.T) {
	t.Parallel()
	for _, tc := range []execTestCase{
		{
			name: "filter_true",
			path: `$.x ?(@ == "hi")`,
			json: map[string]any{"x": "hi"},
			exp:  []any{"hi"},
		},
		{
			name: "filter_false",
			path: `$.x ?(@ != "hi")`,
			json: map[string]any{"x": "hi"},
			exp:  []any{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestExecuteTypeSizeMethods(t *testing.T) {
	t.Parallel()
	for _, tc := range []execTestCase{
		{
			name: "type_method_string",
			path: `$.x.type()`,
			json: map[string]any{"x": "hi"},
			exp:  []any{"string"},
		},
		{
			name: "type_method_multi",
			path: `$[*].type()`,
			json: []any{int64(1), "2", map[string]any{}},
			exp:  []any{"number", "string", "object"},
		},
		{
			name: "size_method_array",
			path: `$.x.size()`,
			json: map[string]any{"x": []any{1, 2, 3}},
			exp:  []any{int64(3)},
		},
		{
			name: "size_method_other",
			path: `$.x.size()`,
			json: map[string]any{"x": true},
			exp:  []any{int64(1)},
		},
		{
			name: "size_method_error",
			path: `strict $.x.size()`,
			json: map[string]any{"x": true},
			err:  `exec: jsonpath item method .size() can only be applied to an array`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestExecuteUnaryPlusMinus(t *testing.T) {
	t.Parallel()
	for _, tc := range []execTestCase{
		{
			name: "unary_plus",
			path: `+$.x`,
			json: map[string]any{"x": int64(42)},
			exp:  []any{int64(42)},
		},
		{
			name: "unary_minus_pos",
			path: `-$.x`,
			json: map[string]any{"x": int64(42)},
			exp:  []any{int64(-42)},
		},
		{
			name: "unary_minus_neg",
			path: `-$.x`,
			json: map[string]any{"x": int64(-42)},
			exp:  []any{int64(42)},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestExecuteDateTime(t *testing.T) {
	t.Parallel()
	for _, tc := range []execTestCase{
		{
			name: "date",
			path: `$.x.date()`,
			json: map[string]any{"x": "2009-10-03"},
			exp: []any{&types.Date{
				Time: time.Date(2009, 10, 3, 0, 0, 0, 0, time.UTC),
			}},
		},
		{
			name: "time",
			path: `$.x.time()`,
			json: map[string]any{"x": "20:59:19.79142"},
			exp: []any{&types.Time{
				Time: time.Date(0, 1, 1, 20, 59, 19, 791420000, time.UTC),
			}},
		},
		{
			name: "time_tz",
			path: `$.x.time_tz()`,
			json: map[string]any{"x": "20:59:19.79142-04"},
			exp: []any{&types.TimeTZ{
				Time: time.Date(0, 1, 1, 20, 59, 19, 791420000, time.FixedZone("", -4*60*60)),
			}},
		},
		{
			name: "timestamp_T",
			path: `$.x.timestamp()`,
			json: map[string]any{"x": "2024-05-05T20:59:19.79142"},
			exp: []any{&types.Timestamp{
				Time: time.Date(2024, 5, 5, 20, 59, 19, 791420000, time.UTC),
			}},
		},
		{
			name: "timestamp_space",
			path: `$.x.timestamp()`,
			json: map[string]any{"x": "2024-05-05 20:59:19.79142"},
			exp: []any{&types.Timestamp{
				Time: time.Date(2024, 5, 5, 20, 59, 19, 791420000, time.UTC),
			}},
		},
		{
			name: "timestamp_T_tz",
			path: `$.x.timestamp_tz()`,
			json: map[string]any{"x": "2024-05-05T20:59:19.79142-05"},
			exp: []any{&types.TimestampTZ{
				Time: time.Date(2024, 5, 5, 20, 59, 19, 791420000, time.FixedZone("", -5*60*60)),
			}},
		},
		{
			name: "timestamp_space_tz",
			path: `$.x.timestamp_tz()`,
			json: map[string]any{"x": "2024-05-05 20:59:19.79142-05"},
			exp: []any{&types.TimestampTZ{
				Time: time.Date(2024, 5, 5, 20, 59, 19, 791420000, time.FixedZone("", -5*60*60)),
			}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
			// .datetime() should also work
			tc.name += "_datetime"
			tc.path = `$.x.datetime()`
			tc.run(t)
		})
	}
}

func TestExecuteDateTimeErrors(t *testing.T) {
	t.Parallel()
	for _, tc := range []execTestCase{
		{
			name: "not_a_string",
			path: `$.x.timestamp_tz()`,
			json: map[string]any{"x": int64(42)},
			err:  "exec: jsonpath item method .timestamp_tz() can only be applied to a string",
		},
		{
			name: "datetime_template_not_supported",
			path: `$.x.datetime("HH24:MI")`,
			json: map[string]any{"x": "2024-05-05 20:59:19.79142-05"},
			err:  "exec: .datetime(template) is not yet supported",
		},
		{
			name: "invalid_precision",
			path: fmt.Sprintf(`$.x.time(%v)`, math.MaxInt32+1),
			json: map[string]any{"x": "2024-05-05 20:59:19.79142-05"},
			err:  `exec: time precision of jsonpath item method .time() is out of integer range`,
		},
		{
			name: "not_a_timestamp",
			path: `$.x.time()`,
			json: map[string]any{"x": "NOT A TIMESTAMP"},
			err:  `exec: time format is not recognized: "NOT A TIMESTAMP"`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

const hint = " HINT: Use WithTZ() option for time zone support"

func TestExecuteDateTimeCast(t *testing.T) {
	t.Parallel()
	for _, tc := range []execTestCase{
		// Cast to date
		{
			name: "date_to_date",
			path: `$.x.date()`,
			json: map[string]any{"x": "2009-10-03"},
			exp: []any{&types.Date{
				Time: time.Date(2009, 10, 3, 0, 0, 0, 0, time.UTC),
			}},
		},
		{
			name: "timestamp_to_date",
			path: `$.x.date()`,
			json: map[string]any{"x": "2009-10-03 20:59:19.79142"},
			exp: []any{&types.Date{
				Time: time.Date(2009, 10, 3, 0, 0, 0, 0, time.UTC),
			}},
		},
		{
			name: "timestamp_tz_to_date",
			path: `$.x.date()`,
			json: map[string]any{"x": "2009-10-03 20:59:19.79142-01"},
			err:  "exec: cannot convert value from timestamptz to date without time zone usage." + hint,
		},
		{
			name:  "timestamp_with_tz_to_date",
			path:  `$.x.date()`,
			useTZ: true,
			json:  map[string]any{"x": "2009-10-03 20:59:19.79142-01"},
			exp: []any{&types.Date{
				Time: time.Date(2009, 10, 3, 0, 0, 0, 0, time.UTC),
			}},
		},
		{
			name: "time_to_date",
			path: `$.x.date()`,
			json: map[string]any{"x": "20:59:19.79142"},
			err:  `exec: date format is not recognized: "20:59:19.79142"`,
		},
		{
			name: "time_tz_to_date",
			path: `$.x.date()`,
			json: map[string]any{"x": "20:59:19.79142-01"},
			err:  `exec: date format is not recognized: "20:59:19.79142-01"`,
		},
		// Cast to time
		{
			name: "date_to_time",
			path: `$.x.time()`,
			json: map[string]any{"x": "2009-10-03"},
			err:  `exec: time format is not recognized: "2009-10-03"`,
		},
		{
			name: "time_to_time",
			path: `$.x.time()`,
			json: map[string]any{"x": "20:59:19.79142"},
			exp: []any{&types.Time{
				Time: time.Date(0, 1, 1, 20, 59, 19, 791420000, time.UTC),
			}},
		},
		{
			name: "time_tz_to_time",
			path: `$.x.time()`,
			json: map[string]any{"x": "20:59:19.79142-01"},
			err:  "exec: cannot convert value from timetz to time without time zone usage." + hint,
		},
		{
			name:  "time_with_tz_to_time",
			path:  `$.x.time()`,
			useTZ: true,
			json:  map[string]any{"x": "20:59:19.79142-01"},
			exp: []any{&types.Time{
				Time: time.Date(0, 1, 1, 20, 59, 19, 791420000, time.UTC),
			}},
		},
		{
			name: "timestamp_to_time",
			path: `$.x.time()`,
			json: map[string]any{"x": "2009-10-03 20:59:19.79142"},
			exp: []any{&types.Time{
				Time: time.Date(0, 1, 1, 20, 59, 19, 791420000, time.UTC),
			}},
		},
		{
			name: "timestamp_tz_to_time",
			path: `$.x.time()`,
			json: map[string]any{"x": "2009-10-03 20:59:19.79142+01"},
			err:  "exec: cannot convert value from timestamptz to time without time zone usage." + hint,
		},
		{
			name:  "timestamp_with_tz_to_time",
			path:  `$.x.time()`,
			useTZ: true,
			json:  map[string]any{"x": "2009-10-03 20:59:19.79142+01"},
			exp: []any{&types.Time{
				Time: time.Date(0, 1, 1, 19, 59, 19, 791420000, time.UTC),
			}},
		},
		// Cast to timetz
		{
			name: "date_to_timetz",
			path: `$.x.time_tz()`,
			json: map[string]any{"x": "2009-10-03"},
			err:  `exec: time_tz format is not recognized: "2009-10-03"`,
		},
		{
			name: "time_to_timetz",
			path: `$.x.time_tz()`,
			json: map[string]any{"x": "20:59:19.79142"},
			err:  "exec: cannot convert value from time to timetz without time zone usage." + hint,
		},
		{
			name:  "time_to_time_with_tz",
			path:  `$.x.time_tz()`,
			useTZ: true,
			json:  map[string]any{"x": "20:59:19.79142"},
			exp: []any{&types.TimeTZ{
				Time: time.Date(0, 1, 1, 20, 59, 19, 791420000, time.UTC),
			}},
		},
		{
			name: "timetz_to_timetz",
			path: `$.x.time_tz()`,
			json: map[string]any{"x": "20:59:19.79142Z"},
			exp: []any{&types.TimeTZ{
				Time: time.Date(0, 1, 1, 20, 59, 19, 791420000, time.UTC),
			}},
		},
		{
			name: "timestamp_to_timetz",
			path: `$.x.time_tz()`,
			json: map[string]any{"x": "2009-10-03 20:59:19.79142"},
			err:  `exec: time_tz format is not recognized: "2009-10-03 20:59:19.79142"`,
		},
		{
			name: "timestamp_tz_to_timetz",
			path: `$.x.time_tz()`,
			json: map[string]any{"x": "2009-10-03 20:59:19.79142Z"},
			exp: []any{&types.TimeTZ{
				Time: time.Date(0, 1, 1, 20, 59, 19, 791420000, time.UTC),
			}},
		},
		// Cast to timestamp
		{
			name: "date_to_timestamp",
			path: `$.x.timestamp()`,
			json: map[string]any{"x": "2009-10-03"},
			exp: []any{&types.Timestamp{
				Time: time.Date(2009, 10, 3, 0, 0, 0, 0, time.UTC),
			}},
		},
		{
			name: "time_to_timestamp",
			path: `$.x.timestamp()`,
			json: map[string]any{"x": "20:59:19.79142"},
			err:  `exec: timestamp format is not recognized: "20:59:19.79142"`,
		},
		{
			name: "time_tz_to_timestamp",
			path: `$.x.timestamp()`,
			json: map[string]any{"x": "20:59:19.79142-01"},
			err:  `exec: timestamp format is not recognized: "20:59:19.79142-01"`,
		},
		{
			name: "timestamp_to_timestamp",
			path: `$.x.timestamp()`,
			json: map[string]any{"x": "2009-10-03 20:59:19.79142"},
			exp: []any{&types.Timestamp{
				Time: time.Date(2009, 10, 3, 20, 59, 19, 791420000, time.UTC),
			}},
		},
		{
			name: "timestamp_tz_to_timestamp",
			path: `$.x.timestamp()`,
			json: map[string]any{"x": "2009-10-03 20:59:19.79142Z"},
			err:  "exec: cannot convert value from timestamptz to timestamp without time zone usage." + hint,
		},
		{
			name:  "timestamp_with_tz_to_timestamp",
			path:  `$.x.timestamp()`,
			useTZ: true,
			json:  map[string]any{"x": "2009-10-03 20:59:19.79142Z"},
			exp: []any{&types.Timestamp{
				Time: time.Date(2009, 10, 3, 20, 59, 19, 791420000, time.UTC),
			}},
		},
		// Cast to timestamptz
		{
			name: "date_to_timestamptz",
			path: `$.x.timestamp_tz()`,
			json: map[string]any{"x": "2009-10-03"},
			err:  "exec: cannot convert value from date to timestamptz without time zone usage." + hint,
		},
		{
			name:  "date_to_timestamp_with_tz",
			path:  `$.x.timestamp_tz()`,
			useTZ: true,
			json:  map[string]any{"x": "2009-10-03"},
			exp: []any{&types.TimestampTZ{
				Time: time.Date(2009, 10, 3, 0, 0, 0, 0, time.UTC),
			}},
		},
		{
			name: "time_to_timestamptz",
			path: `$.x.timestamp_tz()`,
			json: map[string]any{"x": "20:59:19.79142"},
			err:  `exec: timestamp_tz format is not recognized: "20:59:19.79142"`,
		},
		{
			name: "time_tz_to_timestamptz",
			path: `$.x.timestamp_tz()`,
			json: map[string]any{"x": "20:59:19.79142-01"},
			err:  `exec: timestamp_tz format is not recognized: "20:59:19.79142-01"`,
		},
		{
			name: "timestamp_to_timestamptz",
			path: `$.x.timestamp_tz()`,
			json: map[string]any{"x": "2009-10-03 20:59:19.79142"},
			err:  "exec: cannot convert value from timestamp to timestamptz without time zone usage." + hint,
		},
		{
			name:  "timestamp_to_timestamp_with_tz",
			path:  `$.x.timestamp_tz()`,
			useTZ: true,
			json:  map[string]any{"x": "2009-10-03 20:59:19.79142"},
			exp: []any{&types.TimestampTZ{
				Time: time.Date(2009, 10, 3, 20, 59, 19, 791420000, time.UTC),
			}},
		},
		{
			name: "timestamp_tz_to_timestamptz",
			path: `$.x.timestamp_tz()`,
			json: map[string]any{"x": "2009-10-03 20:59:19.79142Z"},
			exp: []any{&types.TimestampTZ{
				Time: time.Date(2009, 10, 3, 20, 59, 19, 791420000, time.UTC),
			}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestExecuteTimePrecision(t *testing.T) {
	t.Parallel()
	for _, tc := range []execTestCase{
		{
			name: "time_precision",
			path: `$.x.time(3)`,
			json: map[string]any{"x": "20:59:19.79142"},
			exp: []any{&types.Time{
				Time: time.Date(0, 1, 1, 20, 59, 19, 791000000, time.UTC),
			}},
		},
		{
			name: "time_tz_precision",
			path: `$.x.time_tz(4)`,
			json: map[string]any{"x": "20:59:19.79142+01"},
			exp: []any{&types.TimeTZ{
				Time: time.Date(0, 1, 1, 20, 59, 19, 791400000, time.FixedZone("", 1*60*60)),
			}},
		},
		{
			name: "timestamp_precision",
			path: `$.x.timestamp(2)`,
			json: map[string]any{"x": "2024-05-05T20:59:19.791423"},
			exp: []any{&types.Timestamp{
				Time: time.Date(2024, 5, 5, 20, 59, 19, 790000000, time.UTC),
			}},
		},
		{
			name: "timestamp_tz_precision",
			path: `$.x.timestamp_tz(5)`,
			json: map[string]any{"x": "2024-05-05T20:59:19.791423+02:30"},
			exp: []any{&types.TimestampTZ{
				Time: time.Date(2024, 5, 5, 20, 59, 19, 791420000, time.FixedZone("", 2*60*60+30*60)),
			}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestExecuteDateComparison(t *testing.T) {
	t.Parallel()
	for _, tc := range []execTestCase{
		{
			name: "date_eq_date",
			path: `$.date() == $.date()`,
			json: "2024-05-03",
			exp:  []any{true},
		},
		{
			name: "date_ne_date",
			path: `$.date() != $.date()`,
			json: "2024-05-03",
			exp:  []any{false},
		},
		{
			name: "unequal_dates",
			path: `$.x.date() == $.y.date()`,
			json: map[string]any{"x": "2024-05-03", "y": "2024-05-04"},
			exp:  []any{false},
		},
		{
			name: "gt_date",
			path: `$.y.date() >= $.x.date()`,
			json: map[string]any{"x": "2024-05-03", "y": "2024-05-04"},
			exp:  []any{true},
		},
		{
			name: "same_date",
			path: `$.date() == $.date()`,
			json: "2024-05-03",
			exp:  []any{true},
		},
		{
			name: "date_eq_timestamp",
			path: `$.x.date() == $.y.timestamp()`,
			json: map[string]any{"x": "2024-05-03", "y": "2024-05-03 23:53:42.232"},
			exp:  []any{false},
		},
		{
			name: "date_lt_timestamp",
			path: `$.x.date() < $.y.timestamp()`,
			json: map[string]any{"x": "2024-05-03", "y": "2024-05-03 23:53:42.232"},
			exp:  []any{true},
		},
		{
			name: "date_eq_timestamp_midnight",
			path: `$.x.date() == $.y.timestamp()`,
			json: map[string]any{"x": "2024-05-03", "y": "2024-05-03 00:00:00"},
			exp:  []any{true},
		},
		{
			name: "date_eq_timestamp_tz",
			path: `$.x.date() == $.y.timestamp_tz()`,
			json: map[string]any{"x": "2024-05-03", "y": "2024-05-03 23:53:42.232Z"},
			err:  "exec: cannot convert value from date to timestamptz without time zone usage." + hint,
		},
		{
			name:  "date_eq_timestamp_with_tz",
			path:  `$.x.date() == $.y.timestamp_tz()`,
			useTZ: true,
			json:  map[string]any{"x": "2024-05-03", "y": "2024-05-03 23:53:42.232Z"},
			exp:   []any{false},
		},
		{
			name:  "date_le_timestamp_with_tz",
			path:  `$.x.date() <= $.y.timestamp_tz()`,
			useTZ: true,
			json:  map[string]any{"x": "2024-05-03", "y": "2024-05-03 23:53:42.232Z"},
			exp:   []any{true},
		},
		{
			name: "date_eq_time",
			path: `$.x.date() == $.y.time()`,
			json: map[string]any{"x": "2024-05-03", "y": "23:53:42.232"},
			exp:  []any{nil},
		},
		{
			name: "date_eq_time_tz",
			path: `$.x.date() == $.y.time_tz()`,
			json: map[string]any{"x": "2024-05-03", "y": "23:53:42.232Z"},
			exp:  []any{nil},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestExecuteTimeComparison(t *testing.T) {
	t.Parallel()
	for _, tc := range []execTestCase{
		{
			name: "time_eq_time",
			path: `$.time() == $.time()`,
			json: "14:32:43.123345",
			exp:  []any{true},
		},
		{
			name: "time_ne_time",
			path: `$.time() != $.time()`,
			json: "14:32:43.123345",
			exp:  []any{false},
		},
		{
			name: "time_ne_time_true",
			path: `$.time(3) != $.time(4)`,
			json: "14:32:43.123345",
			exp:  []any{true},
		},
		{
			name: "time_eq_time_tz",
			path: `$.x.time() == $.y.time_tz()`,
			json: map[string]any{"x": "14:32:43.123345", "y": "14:32:43.123345Z"},
			err:  "exec: cannot convert value from time to timetz without time zone usage." + hint,
		},
		{
			name:  "time_eq_time_with_tz",
			path:  `$.x.time() == $.y.time_tz()`,
			useTZ: true,
			json:  map[string]any{"x": "14:32:43.123345", "y": "14:32:43.123345Z"},
			exp:   []any{true},
		},
		{
			name:  "time_eq_time_with_tz_conv",
			path:  `$.x.time() != $.y.time_tz()`,
			useTZ: true,
			json:  map[string]any{"x": "14:32:43.123345", "y": "14:32:43.123345-01"},
			exp:   []any{true},
		},
		{
			name: "time_eq_date",
			path: `$.x.time() == $.y.date()`,
			json: map[string]any{"x": "14:32:43", "y": "2024-05-05"},
			exp:  []any{nil},
		},
		{
			name: "time_eq_timestamp",
			path: `$.x.time() == $.y.timestamp()`,
			json: map[string]any{"x": "14:32:43", "y": "2024-05-05 14:32:43"},
			exp:  []any{nil},
		},
		{
			name: "time_eq_timestamp_tz",
			path: `$.x.time() == $.y.timestamp_tz()`,
			json: map[string]any{"x": "14:32:43", "y": "2024-05-05 14:32:43Z"},
			exp:  []any{nil},
		},
		{
			name: "timetz_eq_timetz",
			path: `$.time_tz() == $.time_tz()`,
			json: "14:32:43.123345Z",
			exp:  []any{true},
		},
		{
			name: "timetz_ne_timetz",
			path: `$.time_tz() != $.time_tz()`,
			json: "14:32:43.123345Z",
			exp:  []any{false},
		},
		{
			name: "timetz_ne_timetz_true",
			path: `$.time_tz(3) != $.time_tz(4)`,
			json: "14:32:43.123345Z",
			exp:  []any{true},
		},
		{
			name: "timetz_eq_time",
			path: `$.y.time_tz() == $.x.time()`,
			json: map[string]any{"x": "14:32:43.123345", "y": "14:32:43.123345Z"},
			err:  "exec: cannot convert value from time to timetz without time zone usage." + hint,
		},
		{
			name:  "time_with_tz_eq_time",
			path:  `$.y.time_tz() == $.x.time()`,
			useTZ: true,
			json:  map[string]any{"x": "14:32:43.123345", "y": "14:32:43.123345Z"},
			exp:   []any{true},
		},
		{
			name:  "time_with_tz_conv_eq_time",
			path:  `$.y.time_tz() != $.x.time()`,
			useTZ: true,
			json:  map[string]any{"x": "14:32:43.123345", "y": "14:32:43.123345-01"},
			exp:   []any{true},
		},
		{
			name: "timetz_eq_date",
			path: `$.x.time_tz() == $.y.date()`,
			json: map[string]any{"x": "14:32:43Z", "y": "2024-05-05"},
			exp:  []any{nil},
		},
		{
			name: "timetz_eq_timestamp",
			path: `$.x.time_tz() == $.y.timestamp()`,
			json: map[string]any{"x": "14:32:43Z", "y": "2024-05-05 14:32:43"},
			exp:  []any{nil},
		},
		{
			name: "timetz_eq_timestamp_tz",
			path: `$.x.time_tz() == $.y.timestamp_tz()`,
			json: map[string]any{"x": "14:32:43Z", "y": "2024-05-05 14:32:43Z"},
			exp:  []any{nil},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestExecuteTimestampComparison(t *testing.T) {
	t.Parallel()
	for _, tc := range []execTestCase{
		{
			name: "ts_eq_ts",
			path: `$.timestamp() == $.timestamp()`,
			json: "2024-05-05 14:32:43.123345",
			exp:  []any{true},
		},
		{
			name: "ts_ne_ts",
			path: `$.timestamp() != $.timestamp()`,
			json: "2024-05-05 14:32:43.123345",
			exp:  []any{false},
		},
		{
			name: "ts_eq_ts_precision",
			path: `$.timestamp(3) == $.timestamp(4)`,
			json: "2024-05-05 14:32:43.123345",
			exp:  []any{false},
		},
		{
			name: "ts_ne_date",
			path: `$[0].timestamp() != $[1].date()`,
			json: []any{"2024-05-05 14:32:43.123345", "2024-05-05"},
			exp:  []any{true},
		},
		{
			name: "ts_eq_date",
			path: `$[0].timestamp() == $[1].date()`,
			json: []any{"2024-05-05 00:00:00", "2024-05-05"},
			exp:  []any{true},
		},
		{
			name: "ts_eq_ts_tz",
			path: `$[0].timestamp() == $[1].timestamp_tz()`,
			json: []any{"2024-05-05 00:00:00", "2024-05-05 00:00:00Z"},
			err:  "exec: cannot convert value from timestamp to timestamptz without time zone usage." + hint,
		},
		{
			name:  "ts_eq_ts_with_tz",
			path:  `$[0].timestamp() == $[1].timestamp_tz()`,
			useTZ: true,
			json:  []any{"2024-05-05 00:00:00", "2024-05-05 00:00:00Z"},
			exp:   []any{true},
		},
		{
			name: "ts_eq_time",
			path: `$[0].timestamp() == $[1].time()`,
			json: []any{"2024-05-05 00:00:00", "00:00:00"},
			exp:  []any{nil},
		},
		{
			name: "ts_eq_time",
			path: `$[0].timestamp() == $[1].time_tz()`,
			json: []any{"2024-05-05 00:00:00", "00:00:00Z"},
			exp:  []any{nil},
		},
		{
			name: "ts_tz_eq_ts_tz",
			path: `$.timestamp_tz() == $.timestamp_tz()`,
			json: "2024-05-05 14:32:43.123345Z",
			exp:  []any{true},
		},
		{
			name: "ts_tz_ne_ts_tz",
			path: `$.timestamp_tz() != $.timestamp_tz()`,
			json: "2024-05-05 14:32:43.123345Z",
			exp:  []any{false},
		},
		{
			name: "ts_tz_eq_ts_tz_precision",
			path: `$.timestamp_tz(2) == $.timestamp_tz(2)`,
			json: "2024-05-05 14:32:43.123345Z",
			exp:  []any{true},
		},
		{
			name: "ts_tz_ne_ts_tz_precision",
			path: `$.timestamp_tz(2) != $.timestamp_tz(3)`,
			json: "2024-05-05 14:32:43.123345Z",
			exp:  []any{true},
		},
		{
			name: "ts_tz_eq_date",
			path: `$[0].timestamp_tz() == $[1].date()`,
			json: []any{"2024-05-05 14:32:43.123345Z", "2024-05-05"},
			err:  "exec: cannot convert value from date to timestamptz without time zone usage." + hint,
		},
		{
			name:  "ts_with_tz_ne_date",
			path:  `$[0].timestamp_tz() != $[1].date()`,
			useTZ: true,
			json:  []any{"2024-05-05 14:32:43.123345Z", "2024-05-05"},
			exp:   []any{true},
		},
		{
			name:  "ts_with_tz_eq_date",
			path:  `$[0].timestamp_tz() == $[1].date()`,
			useTZ: true,
			json:  []any{"2024-05-05 00:00:00Z", "2024-05-05"},
			exp:   []any{true},
		},
		{
			name: "ts_tz_eq_timestamp",
			path: `$[0].timestamp_tz() == $[1].timestamp()`,
			json: []any{"2024-05-05 14:32:43.123345Z", "2024-05-05 14:32:43.123345"},
			err:  "exec: cannot convert value from timestamp to timestamptz without time zone usage." + hint,
		},
		{
			name:  "ts_with_tz_eq_timestamp",
			path:  `$[0].timestamp_tz() == $[1].timestamp()`,
			useTZ: true,
			json:  []any{"2024-05-05 14:32:43.123345Z", "2024-05-05 14:32:43.123345"},
			exp:   []any{true},
		},
		{
			name: "ts_tz_eq_time",
			path: `$[0].timestamp_tz() == $[1].time()`,
			json: []any{"2024-05-05 00:00:00Z", "00:00:00"},
			exp:  []any{nil},
		},
		{
			name: "ts_tz_eq_time",
			path: `$[0].timestamp_tz() == $[1].time_tz()`,
			json: []any{"2024-05-05 00:00:00Z", "00:00:00Z"},
			exp:  []any{nil},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestExecuteDoubleMethod(t *testing.T) {
	t.Parallel()
	for _, tc := range []execTestCase{
		{
			name: "double_int",
			path: `$.x.double()`,
			json: map[string]any{"x": int64(42)},
			exp:  []any{float64(42)},
		},
		{
			name: "double_float",
			path: `$.x.double()`,
			json: map[string]any{"x": float64(98.6)},
			exp:  []any{float64(98.6)},
		},
		{
			name: "double_json_number",
			path: `$.x.double()`,
			json: map[string]any{"x": json.Number("1024.3")},
			exp:  []any{float64(1024.3)},
		},
		{
			name: "double_invalid_json_number",
			path: `$.x.double()`,
			json: map[string]any{"x": json.Number("hi")},
			err:  `exec: argument "hi" of jsonpath item method .double() is invalid for type double precision`,
		},
		{
			name: "double_string",
			path: `$.x.double()`,
			json: map[string]any{"x": "1024.3"},
			exp:  []any{float64(1024.3)},
		},
		{
			name: "double_invalid_string",
			path: `$.x.double()`,
			json: map[string]any{"x": "lol"},
			err:  `exec: argument "lol" of jsonpath item method .double() is invalid for type double precision`,
		},
		{
			name: "double_array",
			path: `$.x.double()`,
			json: map[string]any{"x": []any{"1024.3", int64(42)}},
			exp:  []any{float64(1024.3), float64(42)},
		},
		{
			name: "strict_double_array",
			path: `strict $.x.double()`,
			json: map[string]any{"x": []any{"1024.3", int64(42)}},
			err:  "exec: jsonpath item method .double() can only be applied to a string or numeric value",
		},
		{
			name: "double_bool",
			path: `strict $.x.double()`,
			json: map[string]any{"x": true},
			err:  "exec: jsonpath item method .double() can only be applied to a string or numeric value",
		},
		{
			name: "double_infinity",
			path: `strict $.x.double()`,
			json: map[string]any{"x": "infinity"},
			err:  "exec: NaN or Infinity is not allowed for jsonpath item method .double()",
		},
		{
			name: "double_nan",
			path: `strict $.x.double()`,
			json: map[string]any{"x": "NaN"},
			err:  "exec: NaN or Infinity is not allowed for jsonpath item method .double()",
		},
		{
			name: "double_null",
			path: `strict $.x.double()`,
			json: map[string]any{"x": nil},
			err:  "exec: jsonpath item method .double() can only be applied to a string or numeric value",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestExecuteBigintMethod(t *testing.T) {
	t.Parallel()
	for _, tc := range []execTestCase{
		{
			name: "int_int",
			path: `$.x.bigint()`,
			json: map[string]any{"x": int64(9876543219)},
			exp:  []any{int64(9876543219)},
		},
		{
			name: "int_float",
			path: `$.x.bigint()`,
			json: map[string]any{"x": float64(42.3)},
			exp:  []any{int64(42)},
		},
		{
			name: "int_json_number",
			path: `$.x.bigint()`,
			json: map[string]any{"x": json.Number("9876543219")},
			exp:  []any{int64(9876543219)},
		},
		{
			name: "int_json_number_float",
			path: `$.x.bigint()`,
			json: map[string]any{"x": json.Number("9876543219.0")},
			err:  `exec: argument "9876543219.0" of jsonpath item method .bigint() is invalid for type bigint`,
		},
		{
			name: "int_string",
			path: `$.x.bigint()`,
			json: map[string]any{"x": "99"},
			exp:  []any{int64(99)},
		},
		{
			name: "int_string_float",
			path: `$.x.bigint()`,
			json: map[string]any{"x": "98.6"},
			err:  `exec: argument "98.6" of jsonpath item method .bigint() is invalid for type bigint`,
		},
		{
			name: "int_array",
			path: `$.x.bigint()`,
			json: map[string]any{"x": []any{"99", int64(1024)}},
			exp:  []any{int64(99), int64(1024)},
		},
		{
			name: "int_array_strict",
			path: `strict $.x.bigint()`,
			json: map[string]any{"x": []any{"99", int64(1024)}},
			err:  "exec: jsonpath item method .bigint() can only be applied to a string or numeric value",
		},
		{
			name: "int_obj",
			path: `$.x.bigint()`,
			json: map[string]any{"x": map[string]any{"99": int64(1024)}},
			err:  "exec: jsonpath item method .bigint() can only be applied to a string or numeric value",
		},
		{
			name: "int_next",
			path: "$.x.bigint().abs()",
			json: map[string]any{"x": int64(-9876543219)},
			exp:  []any{int64(9876543219)},
		},
		{
			name: "int_null",
			path: "$.x.bigint()",
			json: map[string]any{"x": nil},
			err:  "exec: jsonpath item method .bigint() can only be applied to a string or numeric value",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestExecuteIntegerMethod(t *testing.T) {
	t.Parallel()
	for _, tc := range []execTestCase{
		{
			name: "int_int",
			path: `$.x.integer()`,
			json: map[string]any{"x": int64(42)},
			exp:  []any{int64(42)},
		},
		{
			name: "int_bigint",
			path: `$.x.integer()`,
			json: map[string]any{"x": int64(9876543219)},
			err:  `exec: argument "9876543219" of jsonpath item method .integer() is invalid for type integer`,
		},
		{
			name: "int_bigint_neg",
			path: `$.x.integer()`,
			json: map[string]any{"x": int64(-3147483648)},
			err:  `exec: argument "-3147483648" of jsonpath item method .integer() is invalid for type integer`,
		},
		{
			name: "int_float",
			path: `$.x.integer()`,
			json: map[string]any{"x": float64(42.3)},
			exp:  []any{int64(42)},
		},
		{
			name: "int_json_number",
			path: `$.x.integer()`,
			json: map[string]any{"x": json.Number("42")},
			exp:  []any{int64(42)},
		},
		{
			name: "int_json_number_float",
			path: `$.x.integer()`,
			json: map[string]any{"x": json.Number("42.0")},
			err:  `exec: argument "42.0" of jsonpath item method .integer() is invalid for type integer`,
		},
		{
			name: "int_json_number_big",
			path: `$.x.integer()`,
			json: map[string]any{"x": json.Number("9876543219")},
			err:  `exec: argument "9876543219" of jsonpath item method .integer() is invalid for type integer`,
		},
		{
			name: "int_json_number_big_neg",
			path: `$.x.integer()`,
			json: map[string]any{"x": json.Number("-3147483648")},
			err:  `exec: argument "-3147483648" of jsonpath item method .integer() is invalid for type integer`,
		},
		{
			name: "int_string",
			path: `$.x.integer()`,
			json: map[string]any{"x": "99"},
			exp:  []any{int64(99)},
		},
		{
			name: "int_string_float",
			path: `$.x.integer()`,
			json: map[string]any{"x": "98.6"},
			err:  `exec: argument "98.6" of jsonpath item method .integer() is invalid for type integer`,
		},
		{
			name: "int_string_big",
			path: `$.x.integer()`,
			json: map[string]any{"x": "9876543219"},
			err:  `exec: argument "9876543219" of jsonpath item method .integer() is invalid for type integer`,
		},
		{
			name: "int_string_big_neg",
			path: `$.x.integer()`,
			json: map[string]any{"x": "-3147483648"},
			err:  `exec: argument "-3147483648" of jsonpath item method .integer() is invalid for type integer`,
		},
		{
			name: "int_array",
			path: `$.x.integer()`,
			json: map[string]any{"x": []any{"99", int64(1024)}},
			exp:  []any{int64(99), int64(1024)},
		},
		{
			name: "int_array_strict",
			path: `strict $.x.integer()`,
			json: map[string]any{"x": []any{"99", int64(1024)}},
			err:  "exec: jsonpath item method .integer() can only be applied to a string or numeric value",
		},
		{
			name: "int_obj",
			path: `$.x.integer()`,
			json: map[string]any{"x": map[string]any{"99": int64(1024)}},
			err:  "exec: jsonpath item method .integer() can only be applied to a string or numeric value",
		},
		{
			name: "int_next",
			path: "$.x.integer().abs()",
			json: map[string]any{"x": int64(-42)},
			exp:  []any{int64(42)},
		},
		{
			name: "int_null",
			path: "$.x.integer()",
			json: map[string]any{"x": nil},
			err:  "exec: jsonpath item method .integer() can only be applied to a string or numeric value",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestExecuteStringMethod(t *testing.T) {
	t.Parallel()
	for _, tc := range []execTestCase{
		{
			name: "string_string",
			path: `$.x.string()`,
			json: map[string]any{"x": "hi"},
			exp:  []any{"hi"},
		},
		{
			name: "datetime_string",
			path: `$.x.datetime().string()`,
			json: map[string]any{"x": "2024-05-05"},
			exp:  []any{"2024-05-05"},
		},
		{
			name: "date_string",
			path: `$.x.date().string()`,
			json: map[string]any{"x": "2024-05-05"},
			exp:  []any{"2024-05-05"},
		},
		{
			name: "time_string",
			path: `$.x.time().string()`,
			json: map[string]any{"x": "12:34:56"},
			exp:  []any{"12:34:56"},
		},
		{
			name: "time_tz_string",
			path: `$.x.time_tz().string()`,
			json: map[string]any{"x": "12:34:56Z"},
			exp:  []any{"12:34:56Z"},
		},
		{
			name: "timestamp_string",
			path: `$.x.timestamp().string()`,
			json: map[string]any{"x": "2024-05-05 12:34:56"},
			exp:  []any{"2024-05-05T12:34:56"},
		},
		//nolint:forcetypeassert
		{
			name: "timestamp_tz_string",
			path: `$.x.timestamp_tz().string()`,
			json: map[string]any{"x": "2024-05-05 12:34:56Z"},
			exp:  []any{pt("2024-05-05 12:34:56Z").(fmt.Stringer).String()},
		},
		{
			name: "json_number_string",
			path: `$.x.string()`,
			json: map[string]any{"x": json.Number("142")},
			exp:  []any{"142"},
		},
		{
			name: "int_string",
			path: `$.x.string()`,
			json: map[string]any{"x": int64(42)},
			exp:  []any{"42"},
		},
		{
			name: "float_string",
			path: `$.x.string()`,
			json: map[string]any{"x": float64(42.3)},
			exp:  []any{"42.3"},
		},
		{
			name: "true_string",
			path: `$.x.string()`,
			json: map[string]any{"x": true},
			exp:  []any{"true"},
		},
		{
			name: "false_string",
			path: `$.x.string()`,
			json: map[string]any{"x": false},
			exp:  []any{"false"},
		},
		{
			name: "null_string",
			path: `$.x.string()`,
			json: map[string]any{"x": nil},
			err:  `exec: jsonpath item method .string() can only be applied to a bool, string, numeric, or datetime value`,
		},
		{
			name: "array_string",
			path: `$.x.string()`,
			json: map[string]any{"x": []any{"hi"}},
			err:  `exec: jsonpath item method .string() can only be applied to a bool, string, numeric, or datetime value`,
		},
		{
			name: "obj_string",
			path: `$.x.string()`,
			json: map[string]any{"x": map[string]any{"hi": 42}},
			err:  `exec: jsonpath item method .string() can only be applied to a bool, string, numeric, or datetime value`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

//nolint:maintidx
func TestExecuteBooleanMethod(t *testing.T) {
	t.Parallel()
	for _, tc := range []execTestCase{
		{
			name: "bool_true",
			path: "$.x.boolean()",
			json: map[string]any{"x": true},
			exp:  []any{true},
		},
		{
			name: "bool_false",
			path: "$.x.boolean()",
			json: map[string]any{"x": false},
			exp:  []any{false},
		},
		{
			name: "bool_int_1",
			path: "$.x.boolean()",
			json: map[string]any{"x": int64(1)},
			exp:  []any{true},
		},
		{
			name: "bool_int_42",
			path: "$.x.boolean()",
			json: map[string]any{"x": int64(42)},
			exp:  []any{true},
		},
		{
			name: "bool_int_0",
			path: "$.x.boolean()",
			json: map[string]any{"x": int64(0)},
			exp:  []any{false},
		},
		{
			name: "bool_float",
			path: "$.x.boolean()",
			json: map[string]any{"x": float64(0.1)},
			err:  `exec: argument "0.1" of jsonpath item method .boolean() is invalid for type boolean`,
		},
		{
			name: "bool_json_number",
			path: "$.x.boolean()",
			json: map[string]any{"x": json.Number("-42")},
			exp:  []any{true},
		},
		{
			name: "bool_json_number_float",
			path: "$.x.boolean()",
			json: map[string]any{"x": json.Number("-42.0")},
			err:  `exec: argument "-42.0" of jsonpath item method .boolean() is invalid for type boolean`,
		},
		{
			name: "bool_string_t",
			path: "$.x.boolean()",
			json: map[string]any{"x": "t"},
			exp:  []any{true},
		},
		{
			name: "bool_string_T",
			path: "$.x.boolean()",
			json: map[string]any{"x": "T"},
			exp:  []any{true},
		},
		{
			name: "bool_string_true",
			path: "$.x.boolean()",
			json: map[string]any{"x": "true"},
			exp:  []any{true},
		},
		{
			name: "bool_string_TRUE",
			path: "$.x.boolean()",
			json: map[string]any{"x": "TRUE"},
			exp:  []any{true},
		},
		{
			name: "bool_string_TrUe",
			path: "$.x.boolean()",
			json: map[string]any{"x": "TrUe"},
			exp:  []any{true},
		},
		{
			name: "bool_string_trunk",
			path: "$.x.boolean()",
			json: map[string]any{"x": "trunk"},
			err:  `exec: argument "trunk" of jsonpath item method .boolean() is invalid for type boolean`,
		},
		{
			name: "bool_string_f",
			path: "$.x.boolean()",
			json: map[string]any{"x": "f"},
			exp:  []any{false},
		},
		{
			name: "bool_string_F",
			path: "$.x.boolean()",
			json: map[string]any{"x": "F"},
			exp:  []any{false},
		},
		{
			name: "bool_string_false",
			path: "$.x.boolean()",
			json: map[string]any{"x": "false"},
			exp:  []any{false},
		},
		{
			name: "bool_string_FALSE",
			path: "$.x.boolean()",
			json: map[string]any{"x": "FALSE"},
			exp:  []any{false},
		},
		{
			name: "bool_string_FaLsE",
			path: "$.x.boolean()",
			json: map[string]any{"x": "FaLsE"},
			exp:  []any{false},
		},
		{
			name: "bool_string_flunk",
			path: "$.x.boolean()",
			json: map[string]any{"x": "flunk"},
			err:  `exec: argument "flunk" of jsonpath item method .boolean() is invalid for type boolean`,
		},
		{
			name: "bool_string_y",
			path: "$.x.boolean()",
			json: map[string]any{"x": "y"},
			exp:  []any{true},
		},
		{
			name: "bool_string_Y",
			path: "$.x.boolean()",
			json: map[string]any{"x": "Y"},
			exp:  []any{true},
		},
		{
			name: "bool_string_yes",
			path: "$.x.boolean()",
			json: map[string]any{"x": "yes"},
			exp:  []any{true},
		},
		{
			name: "bool_string_YES",
			path: "$.x.boolean()",
			json: map[string]any{"x": "YES"},
			exp:  []any{true},
		},
		{
			name: "bool_string_YeS",
			path: "$.x.boolean()",
			json: map[string]any{"x": "YeS"},
			exp:  []any{true},
		},
		{
			name: "bool_string_yet",
			path: "$.x.boolean()",
			json: map[string]any{"x": "yet"},
			err:  `exec: argument "yet" of jsonpath item method .boolean() is invalid for type boolean`,
		},
		{
			name: "bool_string_n",
			path: "$.x.boolean()",
			json: map[string]any{"x": "n"},
			exp:  []any{false},
		},
		{
			name: "bool_string_N",
			path: "$.x.boolean()",
			json: map[string]any{"x": "N"},
			exp:  []any{false},
		},
		{
			name: "bool_string_no",
			path: "$.x.boolean()",
			json: map[string]any{"x": "no"},
			exp:  []any{false},
		},
		{
			name: "bool_string_NO",
			path: "$.x.boolean()",
			json: map[string]any{"x": "NO"},
			exp:  []any{false},
		},
		{
			name: "bool_string_nO",
			path: "$.x.boolean()",
			json: map[string]any{"x": "nO"},
			exp:  []any{false},
		},
		{
			name: "bool_string_not",
			path: "$.x.boolean()",
			json: map[string]any{"x": "not"},
			err:  `exec: argument "not" of jsonpath item method .boolean() is invalid for type boolean`,
		},
		{
			name: "bool_string_on",
			path: "$.x.boolean()",
			json: map[string]any{"x": "on"},
			exp:  []any{true},
		},
		{
			name: "bool_string_ON",
			path: "$.x.boolean()",
			json: map[string]any{"x": "ON"},
			exp:  []any{true},
		},
		{
			name: "bool_string_oN",
			path: "$.x.boolean()",
			json: map[string]any{"x": "oN"},
			exp:  []any{true},
		},
		{
			name: "bool_string_o",
			path: "$.x.boolean()",
			json: map[string]any{"x": "o"},
			err:  `exec: argument "o" of jsonpath item method .boolean() is invalid for type boolean`,
		},
		{
			name: "bool_string_off",
			path: "$.x.boolean()",
			json: map[string]any{"x": "off"},
			exp:  []any{false},
		},
		{
			name: "bool_string_OFF",
			path: "$.x.boolean()",
			json: map[string]any{"x": "OFF"},
			exp:  []any{false},
		},
		{
			name: "bool_string_OfF",
			path: "$.x.boolean()",
			json: map[string]any{"x": "OfF"},
			exp:  []any{false},
		},
		{
			name: "bool_string_oft",
			path: "$.x.boolean()",
			json: map[string]any{"x": "oft"},
			err:  `exec: argument "oft" of jsonpath item method .boolean() is invalid for type boolean`,
		},
		{
			name: "bool_string_1",
			path: "$.x.boolean()",
			json: map[string]any{"x": "1"},
			exp:  []any{true},
		},
		{
			name: "bool_string_1up",
			path: "$.x.boolean()",
			json: map[string]any{"x": "1up"},
			err:  `exec: argument "1up" of jsonpath item method .boolean() is invalid for type boolean`,
		},
		{
			name: "bool_string_0",
			path: "$.x.boolean()",
			json: map[string]any{"x": "0"},
			exp:  []any{false},
		},
		{
			name: "bool_string_0n",
			path: "$.x.boolean()",
			json: map[string]any{"x": "0n"},
			err:  `exec: argument "0n" of jsonpath item method .boolean() is invalid for type boolean`,
		},
		{
			name: "bool_array",
			path: "$.x.boolean()",
			json: map[string]any{"x": []any{"0", true}},
			exp:  []any{false, true},
		},
		{
			name: "bool_array_strict",
			path: "strict $.x.boolean()",
			json: map[string]any{"x": []any{"0", true}},
			err:  `exec: jsonpath item method .boolean() can only be applied to a bool, string, or numeric value`,
		},
		{
			name: "bool_obj",
			path: "strict $.x.boolean()",
			json: map[string]any{"x": map[string]any{"0": true}},
			err:  `exec: jsonpath item method .boolean() can only be applied to a bool, string, or numeric value`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestExecuteKeyValueMethod(t *testing.T) {
	t.Parallel()
	for _, tc := range []execTestCase{
		{
			name: "kv_single",
			path: "$.keyvalue()",
			json: map[string]any{"x": true},
			exp:  []any{map[string]any{"key": "x", "value": true, "id": int64(0)}},
		},
		{
			name: "kv_double",
			path: "$.keyvalue()",
			json: map[string]any{"x": true, "y": "hi"},
			exp: []any{
				map[string]any{"key": "x", "value": true, "id": int64(0)},
				map[string]any{"key": "y", "value": "hi", "id": int64(0)},
			},
			rand: true, // Results can be in any order
		},
		{
			name: "kv_sequence",
			path: "$.keyvalue().keyvalue()",
			json: map[string]any{"x": true, "y": "hi"},
			exp: []any{
				map[string]any{"id": int64(20000000000), "key": "key", "value": "x"},
				map[string]any{"id": int64(20000000000), "key": "value", "value": true},
				map[string]any{"id": int64(20000000000), "key": "id", "value": int64(0)},
				map[string]any{"id": int64(60000000000), "key": "id", "value": int64(0)},
				map[string]any{"id": int64(60000000000), "key": "key", "value": "y"},
				map[string]any{"id": int64(60000000000), "key": "value", "value": "hi"},
			},
			rand: true, // Results can be in any order
		},
		{
			name: "kv_nested",
			path: "$.keyvalue()",
			json: map[string]any{"foo": map[string]any{"x": true, "y": "hi"}},
			exp: []any{
				map[string]any{"id": int64(0), "key": "foo", "value": map[string]any{"x": true, "y": "hi"}},
			},
			rand: true, // Results can be in any order
		},
		{
			name: "kv_nested_sequence",
			path: "$.keyvalue().keyvalue()",
			json: map[string]any{"foo": map[string]any{"x": true, "y": "hi"}},
			exp: []any{
				map[string]any{"id": int64((20000000000)), "key": "id", "value": int64(0)},
				map[string]any{"id": int64(20000000000), "key": "key", "value": "foo"},
				map[string]any{"id": int64(20000000000), "key": "value", "value": map[string]any{"x": true, "y": "hi"}},
			},
			rand: true, // Results can be in any order
		},
		{
			name: "kv_multi_nested_sequence",
			path: "$.keyvalue().keyvalue()",
			json: map[string]any{"foo": map[string]any{"x": true, "y": "hi"}, "bar": 2, "baz": 1},
			exp: []any{
				map[string]any{"id": int64(20000000000), "key": "id", "value": int64(0)},
				map[string]any{"id": int64(20000000000), "key": "key", "value": "bar"},
				map[string]any{"id": int64(20000000000), "key": "value", "value": 2},
				map[string]any{"id": int64(60000000000), "key": "id", "value": int64(0)},
				map[string]any{"id": int64(60000000000), "key": "key", "value": "baz"},
				map[string]any{"id": int64(60000000000), "key": "value", "value": 1},
				map[string]any{"id": int64(100000000000), "key": "id", "value": int64(0)},
				map[string]any{"id": int64(100000000000), "key": "key", "value": "foo"},
				map[string]any{"id": int64(100000000000), "key": "value", "value": map[string]any{"x": true, "y": "hi"}},
			},
			rand: true, // Results can be in any order
		},
		{
			name: "kv_variable",
			path: "$foo.keyvalue()",
			vars: vars{"foo": map[string]any{"x": true, "y": 1}},
			json: `""`,
			exp: []any{
				map[string]any{"key": "x", "value": true, "id": int64(10000000048)},
				map[string]any{"key": "y", "value": 1, "id": int64(10000000048)},
			},
			rand: true, // Results can be in any order
		},
		{
			name: "kv_empty",
			path: "$.keyvalue()",
			json: map[string]any{},
			exp:  []any{},
		},
		{
			name: "kv_null",
			path: "$.keyvalue()",
			json: nil,
			err:  "exec: jsonpath item method .keyvalue() can only be applied to an object",
			exp:  []any{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}
