package exec

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/theory/sqljson/path/ast"
	"github.com/theory/sqljson/path/parser"
	"github.com/theory/sqljson/path/types"
)

func TestTZRequiredCast(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	for _, tc := range []struct {
		name string
		t1   string
		t2   string
	}{
		{
			name: "date_timestamptz",
			t1:   "date",
			t2:   "timestamptz",
		},
		{
			name: "time_timetz",
			t1:   "time",
			t2:   "timetz",
		},
		{
			name: "timestamp_timestamptz",
			t1:   "timestamp",
			t2:   "timestamptz",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tzRequiredCast(tc.t1, tc.t2)
			r.EqualError(err, fmt.Sprintf(
				"exec: cannot convert value from %v to %v without time zone usage."+tzHint,
				tc.t1, tc.t2,
			))
			r.ErrorIs(err, ErrExecution)
		})
	}
}

func TestUnknownDateTime(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	r := require.New(t)

	for _, tc := range []struct {
		name string
		val  any
	}{
		{
			name: "string",
			val:  "foo",
		},
		{
			name: "array",
			val:  []any{},
		},
		{
			name: "object",
			val:  map[string]any{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			res, err := unknownDateTime(tc.val)
			a.Equal(0, res)
			r.EqualError(
				err,
				fmt.Sprintf("exec invalid: unrecognized SQL/JSON datetime type %T", tc.val),
			)
			r.ErrorIs(err, ErrInvalid)
		})
	}
}

type testDatetimeCompare struct {
	name  string
	val1  any
	val2  any
	useTZ bool
	exp   int
	err   error
}

func (tc testDatetimeCompare) checkCompare(t *testing.T, res int, err error) {
	t.Helper()
	a := assert.New(t)
	r := require.New(t)
	a.Equal(tc.exp, res)
	if tc.err == nil {
		a.NoError(err)
	} else {
		r.EqualError(err, tc.err.Error())
		if errors.Is(tc.err, ErrExecution) {
			r.ErrorIs(err, ErrExecution)
		} else {
			r.ErrorIs(err, ErrInvalid)
		}
	}
}

func stableTime() time.Time {
	return time.Date(2024, time.June, 6, 1, 48, 22, 939932000, time.FixedZone("", 0))
}

func TestCompareDatetime(t *testing.T) {
	t.Parallel()
	moment := stableTime()
	ctx := context.Background()

	for _, tc := range []testDatetimeCompare{
		{
			name: "date_date",
			val1: types.NewDate(moment),
			val2: types.NewDate(moment),
		},
		{
			name: "date_timestamp",
			val1: types.NewDate(moment),
			val2: types.NewTimestamp(moment),
			exp:  -1,
		},
		{
			name: "date_timestamptz",
			val1: types.NewDate(moment),
			val2: types.NewTimestampTZ(ctx, moment),
			err:  tzRequiredCast("date", "timestamptz"),
		},
		{
			name:  "date_timestamptz_cast",
			val1:  types.NewDate(moment),
			val2:  types.NewTimestampTZ(ctx, moment),
			useTZ: true,
			exp:   -1,
		},
		{
			name: "time_time",
			val1: types.NewTime(moment),
			val2: types.NewTime(moment),
		},
		{
			name: "time_timetz",
			val1: types.NewTime(moment),
			val2: types.NewTimeTZ(moment),
			err:  tzRequiredCast("time", "timetz"),
		},
		{
			name:  "time_timetz_cast",
			val1:  types.NewTime(moment),
			val2:  types.NewTimeTZ(moment),
			useTZ: true,
			exp:   0,
		},
		{
			name: "timetz_timetz",
			val1: types.NewTimeTZ(moment),
			val2: types.NewTimeTZ(moment),
		},
		{
			name: "timetz_time",
			val1: types.NewTimeTZ(moment),
			val2: types.NewTime(moment),
			err:  tzRequiredCast("time", "timetz"),
		},
		{
			name:  "timetz_time_cast",
			val1:  types.NewTimeTZ(moment),
			val2:  types.NewTime(moment),
			useTZ: true,
			exp:   0,
		},
		{
			name: "timestamp_timestamp",
			val1: types.NewTimestamp(moment),
			val2: types.NewTimestamp(moment),
		},
		{
			name: "timestamp_date",
			val1: types.NewTimestamp(moment),
			val2: types.NewDate(moment),
			exp:  1,
		},
		{
			name: "timestamp_timestamptz",
			val1: types.NewTimestamp(moment),
			val2: types.NewTimestampTZ(ctx, moment),
			err:  tzRequiredCast("timestamp", "timestamptz"),
		},
		{
			name:  "timestamp_timestamptz_cast",
			val1:  types.NewTimestamp(moment),
			val2:  types.NewTimestampTZ(ctx, moment),
			useTZ: true,
		},
		{
			name: "timestamptz_timestamptz",
			val1: types.NewTimestampTZ(ctx, moment),
			val2: types.NewTimestampTZ(ctx, moment),
		},
		{
			name: "timestamptz_time",
			val1: types.NewTimestampTZ(ctx, moment),
			val2: types.NewTime(moment),
			exp:  -2,
		},
		{
			name: "timestamptz_timestamp",
			val1: types.NewTimestampTZ(ctx, moment),
			val2: types.NewTimestamp(moment),
			err:  tzRequiredCast("timestamp", "timestamptz"),
		},
		{
			name:  "timestamptz_timestamp_cast",
			val1:  types.NewTimestampTZ(ctx, moment),
			val2:  types.NewTimestamp(moment),
			useTZ: true,
		},
		{
			name: "unknown_type",
			val1: "not a timestamp",
			err:  errors.New("exec invalid: unrecognized SQL/JSON datetime type string"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			res, err := compareDatetime(ctx, tc.val1, tc.val2, tc.useTZ)
			tc.checkCompare(t, res, err)
		})
	}
}

func TestCompareDate(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	moment := stableTime()
	ctx := context.Background()

	for _, tc := range []testDatetimeCompare{
		{
			name: "date_date",
			val1: types.NewDate(moment),
			val2: types.NewDate(moment),
		},
		{
			name: "date_timestamp",
			val1: types.NewDate(moment),
			val2: types.NewTimestamp(moment),
			exp:  -1,
		},
		{
			name: "date_timestamptz",
			val1: types.NewDate(moment),
			val2: types.NewTimestampTZ(ctx, moment),
			err:  tzRequiredCast("date", "timestamptz"),
		},
		{
			name:  "date_timestamptz_cast",
			val1:  types.NewDate(moment),
			val2:  types.NewTimestampTZ(ctx, moment),
			useTZ: true,
			exp:   -1,
		},
		{
			name: "date_time",
			val1: types.NewDate(moment),
			val2: types.NewTime(moment),
			exp:  -2,
		},
		{
			name: "date_timetz",
			val1: types.NewDate(moment),
			val2: types.NewTime(moment),
			exp:  -2,
		},
		{
			name: "unknown_type",
			val1: types.NewDate(moment),
			val2: "not a timestamp",
			err:  errors.New("exec invalid: unrecognized SQL/JSON datetime type string"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			val1, ok := tc.val1.(*types.Date)
			a.True(ok)
			res, err := compareDate(ctx, val1, tc.val2, tc.useTZ)
			tc.checkCompare(t, res, err)
		})
	}
}

func TestCompareTime(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	r := require.New(t)

	moment := stableTime()
	loc, err := time.LoadLocation("PST8PDT")
	r.NoError(err)
	ctx := types.ContextWithTZ(context.Background(), loc)

	for _, tc := range []testDatetimeCompare{
		{
			name: "time_time",
			val1: types.NewTime(moment),
			val2: types.NewTime(moment),
		},
		{
			name: "time_timetz",
			val1: types.NewTime(moment),
			val2: types.NewTimeTZ(moment),
			err:  tzRequiredCast("time", "timetz"),
		},
		{
			name:  "time_timetz_cast",
			val1:  types.NewTime(moment),
			val2:  types.NewTimeTZ(moment),
			useTZ: true,
			exp:   1,
		},
		{
			name: "time_date",
			val1: types.NewTime(moment),
			val2: types.NewDate(moment),
			exp:  -2,
		},
		{
			name: "time_timestamp",
			val1: types.NewTime(moment),
			val2: types.NewTimestamp(moment),
			exp:  -2,
		},
		{
			name: "time_timestamptz",
			val1: types.NewTime(moment),
			val2: types.NewTimestampTZ(ctx, moment),
			exp:  -2,
		},
		{
			name: "unknown_type",
			val1: types.NewTime(moment),
			val2: "not a timestamp",
			err:  errors.New("exec invalid: unrecognized SQL/JSON datetime type string"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			val1, ok := tc.val1.(*types.Time)
			a.True(ok)
			res, err := compareTime(ctx, val1, tc.val2, tc.useTZ)
			tc.checkCompare(t, res, err)
		})
	}
}

func TestCompareTimeTZ(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	r := require.New(t)

	moment := stableTime()
	loc, err := time.LoadLocation("PST8PDT")
	r.NoError(err)
	ctx := types.ContextWithTZ(context.Background(), loc)

	for _, tc := range []testDatetimeCompare{
		{
			name: "timetz_timetz",
			val1: types.NewTimeTZ(moment),
			val2: types.NewTimeTZ(moment),
		},
		{
			name: "timetz_time",
			val1: types.NewTimeTZ(moment),
			val2: types.NewTime(moment),
			err:  tzRequiredCast("time", "timetz"),
		},
		{
			name:  "timetz_time_cast",
			val1:  types.NewTimeTZ(moment),
			val2:  types.NewTime(moment),
			useTZ: true,
			exp:   -1,
		},
		{
			name: "timetz_date",
			val1: types.NewTimeTZ(moment),
			val2: types.NewDate(moment),
			exp:  -2,
		},
		{
			name: "timetz_timestamp",
			val1: types.NewTimeTZ(moment),
			val2: types.NewTimestamp(moment),
			exp:  -2,
		},
		{
			name: "timetz_timestamptz",
			val1: types.NewTimeTZ(moment),
			val2: types.NewTimestampTZ(ctx, moment),
			exp:  -2,
		},
		{
			name: "unknown_type",
			val1: types.NewTimeTZ(moment),
			val2: "not a timestamp",
			err:  errors.New("exec invalid: unrecognized SQL/JSON datetime type string"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			val1, ok := tc.val1.(*types.TimeTZ)
			a.True(ok)
			res, err := compareTimeTZ(ctx, val1, tc.val2, tc.useTZ)
			tc.checkCompare(t, res, err)
		})
	}
}

func TestCompareTimestamp(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	moment := stableTime()
	ctx := context.Background()

	for _, tc := range []testDatetimeCompare{
		{
			name: "timestamp_timestamp",
			val1: types.NewTimestamp(moment),
			val2: types.NewTimestamp(moment),
		},
		{
			name: "timestamp_date",
			val1: types.NewTimestamp(moment),
			val2: types.NewDate(moment),
			exp:  1,
		},
		{
			name: "timestamp_timestamptz",
			val1: types.NewTimestamp(moment),
			val2: types.NewTimestampTZ(ctx, moment),
			err:  tzRequiredCast("timestamp", "timestamptz"),
		},
		{
			name:  "timestamp_timestamptz_cast",
			val1:  types.NewTimestamp(moment),
			val2:  types.NewTimestampTZ(ctx, moment),
			useTZ: true,
		},
		{
			name: "timestamp_time",
			val1: types.NewTimestamp(moment),
			val2: types.NewTime(moment),
			exp:  -2,
		},
		{
			name: "timestamp_timetz",
			val1: types.NewTimestamp(moment),
			val2: types.NewTimeTZ(moment),
			exp:  -2,
		},
		{
			name: "unknown_type",
			val1: types.NewTimestamp(moment),
			val2: "not a timestamp",
			err:  errors.New("exec invalid: unrecognized SQL/JSON datetime type string"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			val1, ok := tc.val1.(*types.Timestamp)
			a.True(ok)
			res, err := compareTimestamp(ctx, val1, tc.val2, tc.useTZ)
			tc.checkCompare(t, res, err)
		})
	}
}

func TestCompareTimestampTZ(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	moment := stableTime()
	ctx := context.Background()

	for _, tc := range []testDatetimeCompare{
		{
			name: "timestamptz_timestamptz",
			val1: types.NewTimestampTZ(ctx, moment),
			val2: types.NewTimestampTZ(ctx, moment),
		},
		{
			name: "timestamptz_timestamp",
			val1: types.NewTimestampTZ(ctx, moment),
			val2: types.NewTimestamp(moment),
			err:  tzRequiredCast("timestamp", "timestamptz"),
		},
		{
			name:  "timestamptz_timestamp_cast",
			val1:  types.NewTimestampTZ(ctx, moment),
			val2:  types.NewTimestamp(moment),
			useTZ: true,
		},
		{
			name: "timestamptz_date",
			val1: types.NewTimestampTZ(ctx, moment),
			val2: types.NewDate(moment),
			err:  tzRequiredCast("date", "timestamptz"),
		},
		{
			name:  "timestamptz_date_cast",
			val1:  types.NewTimestampTZ(ctx, moment),
			val2:  types.NewDate(moment),
			useTZ: true,
			exp:   1,
		},
		{
			name: "timestamptz_time",
			val1: types.NewTimestampTZ(ctx, moment),
			val2: types.NewTime(moment),
			exp:  -2,
		},
		{
			name: "timestamptz_timetz",
			val1: types.NewTimestampTZ(ctx, moment),
			val2: types.NewTime(moment),
			exp:  -2,
		},
		{
			name: "unknown_type",
			val1: types.NewTimestampTZ(ctx, moment),
			val2: "not a timestamp",
			err:  errors.New("exec invalid: unrecognized SQL/JSON datetime type string"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			val1, ok := tc.val1.(*types.TimestampTZ)
			a.True(ok)
			res, err := compareTimestampTZ(ctx, val1, tc.val2, tc.useTZ)
			tc.checkCompare(t, res, err)
		})
	}
}

func TestExecuteDateTimeMethod(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	r := require.New(t)
	ctx := context.Background()
	path, _ := parser.Parse("$")

	for _, tc := range []struct {
		name   string
		node   ast.Node
		value  any
		silent bool
		find   []any
		exp    resultStatus
		err    string
		isErr  error
	}{
		{
			name:  "not_string",
			node:  ast.NewUnary(ast.UnaryDateTime, nil),
			value: true,
			exp:   statusFailed,
			err:   `exec: jsonpath item method .datetime() can only be applied to a string`,
			isErr: ErrVerbose,
		},
		{
			name:  "datetime_format_unsupported",
			node:  ast.NewUnary(ast.UnaryDateTime, ast.NewString("YYYY")),
			value: "2024-06-05",
			exp:   statusFailed,
			err:   `exec: .datetime(template) is not yet supported`,
			isErr: ErrExecution,
		},
		{
			name:  "datetime_parse_failure",
			node:  ast.NewUnary(ast.UnaryDateTime, nil),
			value: "nope",
			exp:   statusFailed,
			err:   `exec: datetime format is not recognized: "nope"`,
			isErr: ErrExecution,
		},
		{
			name:   "datetime_parse_failure_silent",
			node:   ast.NewUnary(ast.UnaryDateTime, nil),
			value:  "nope",
			exp:    statusFailed,
			silent: true,
		},
		{
			name:  "datetime_parse_success",
			node:  ast.NewUnary(ast.UnaryDateTime, nil),
			value: "2024-06-05",
			exp:   statusOK,
			find:  []any{types.NewDate(time.Date(2024, 6, 5, 0, 0, 0, 0, time.UTC))},
		},
		{
			name:  "date_parse_success",
			node:  ast.NewUnary(ast.UnaryDate, nil),
			value: "2024-06-05",
			exp:   statusOK,
			find:  []any{types.NewDate(time.Date(2024, 6, 5, 0, 0, 0, 0, time.UTC))},
		},
		{
			name:  "date_parse_fail",
			node:  ast.NewUnary(ast.UnaryDate, nil),
			value: "nope",
			exp:   statusFailed,
			err:   `exec: date format is not recognized: "nope"`,
			isErr: ErrExecution,
		},
		{
			name:   "date_parse_fail_silent",
			node:   ast.NewUnary(ast.UnaryDate, nil),
			value:  "nope",
			exp:    statusFailed,
			silent: true,
		},
		{
			name:  "date_parse_cast",
			node:  ast.NewUnary(ast.UnaryDate, nil),
			value: "2024-06-05T12:32:42",
			exp:   statusOK,
			find:  []any{types.NewDate(time.Date(2024, 6, 5, 0, 0, 0, 0, time.UTC))},
		},
		{
			name:  "time_parse_success",
			node:  ast.NewUnary(ast.UnaryTime, nil),
			value: "12:32:43",
			exp:   statusOK,
			find:  []any{types.NewTime(time.Date(0, 1, 1, 12, 32, 43, 0, time.UTC))},
		},
		{
			name:  "time_parse_fail",
			node:  ast.NewUnary(ast.UnaryTime, nil),
			value: "nope",
			exp:   statusFailed,
			err:   `exec: time format is not recognized: "nope"`,
			isErr: ErrExecution,
		},
		{
			name:   "time_parse_fail_silent",
			node:   ast.NewUnary(ast.UnaryTime, nil),
			value:  "nope",
			exp:    statusFailed,
			silent: true,
		},
		{
			name:  "time_parse_cast",
			node:  ast.NewUnary(ast.UnaryTime, nil),
			value: "2024-06-05T12:32:42",
			exp:   statusOK,
			find:  []any{types.NewTime(time.Date(0, 1, 1, 12, 32, 42, 0, time.UTC))},
		},
		{
			name:  "timetz_parse_success",
			node:  ast.NewUnary(ast.UnaryTimeTZ, nil),
			value: "12:32:43+01",
			exp:   statusOK,
			find:  []any{types.NewTimeTZ(time.Date(0, 1, 1, 12, 32, 43, 0, time.FixedZone("", 60*60)))},
		},
		{
			name:  "timetz_parse_fail",
			node:  ast.NewUnary(ast.UnaryTimeTZ, nil),
			value: "nope",
			exp:   statusFailed,
			err:   `exec: time_tz format is not recognized: "nope"`,
			isErr: ErrExecution,
		},
		{
			name:   "timetz_parse_fail_silent",
			node:   ast.NewUnary(ast.UnaryTimeTZ, nil),
			value:  "nope",
			exp:    statusFailed,
			silent: true,
		},
		{
			name:  "timetz_parse_cast",
			node:  ast.NewUnary(ast.UnaryTimeTZ, nil),
			value: "2024-06-05T12:32:42Z",
			exp:   statusOK,
			find: []any{
				types.NewTimestampTZ(
					ctx, time.Date(2024, 6, 5, 12, 32, 42, 0, time.FixedZone("", 0)),
				).ToTimeTZ(ctx),
			},
		},
		{
			name:  "timestamp_parse_success",
			node:  ast.NewUnary(ast.UnaryTimestamp, nil),
			value: "2024-06-05T12:32:43",
			exp:   statusOK,
			find:  []any{types.NewTimestamp(time.Date(2024, 6, 5, 12, 32, 43, 0, time.FixedZone("", 0)))},
		},
		{
			name:  "timestamp_parse_fail",
			node:  ast.NewUnary(ast.UnaryTimestamp, nil),
			value: "nope",
			exp:   statusFailed,
			err:   `exec: timestamp format is not recognized: "nope"`,
			isErr: ErrExecution,
		},
		{
			name:   "timestamp_parse_fail_silent",
			node:   ast.NewUnary(ast.UnaryTimestamp, nil),
			value:  "nope",
			exp:    statusFailed,
			silent: true,
		},
		{
			name:  "timestamp_parse_cast",
			node:  ast.NewUnary(ast.UnaryTimestamp, nil),
			value: "2024-06-05",
			exp:   statusOK,
			find:  []any{types.NewTimestamp(time.Date(2024, 6, 5, 0, 0, 0, 0, time.UTC))},
		},
		{
			name:  "timestamptz_parse_success",
			node:  ast.NewUnary(ast.UnaryTimestampTZ, nil),
			value: "2024-06-05T12:32:43+01",
			exp:   statusOK,
			find:  []any{types.NewTimestampTZ(ctx, time.Date(2024, 6, 5, 12, 32, 43, 0, time.FixedZone("", 60*60)))},
		},
		{
			name:  "timestamptz_parse_fail",
			node:  ast.NewUnary(ast.UnaryTimestampTZ, nil),
			value: "nope",
			exp:   statusFailed,
			err:   `exec: timestamp_tz format is not recognized: "nope"`,
			isErr: ErrExecution,
		},
		{
			name:   "timestamptz_parse_fail_silent",
			node:   ast.NewUnary(ast.UnaryTimestampTZ, nil),
			value:  "nope",
			exp:    statusFailed,
			silent: true,
		},
		{
			name:  "timestamptz_parse_cast_fail",
			node:  ast.NewUnary(ast.UnaryTimestampTZ, nil),
			value: "2024-06-05T12:32:43",
			exp:   statusFailed,
			find:  []any{},
			err:   "exec: cannot convert value from timestamp to timestamptz without time zone usage." + tzHint,
			isErr: ErrExecution,
		},
		{
			name:  "date_no_found",
			node:  ast.NewUnary(ast.UnaryDate, nil),
			value: "2024-06-05",
			exp:   statusOK,
		},
		{
			name:  "date_parse_with_next",
			node:  ast.LinkNodes([]ast.Node{ast.NewUnary(ast.UnaryDate, nil), ast.NewMethod(ast.MethodString)}),
			value: "2024-06-05",
			exp:   statusOK,
			find:  []any{"2024-06-05"},
		},
		{
			name:  "unary_not_datetime",
			node:  ast.NewUnary(ast.UnaryNot, nil),
			value: "2024-06-05",
			exp:   statusFailed,
			err:   `exec invalid: unrecognized jsonpath datetime method: !`,
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

			// Should have UnaryNode
			node, ok := tc.node.(*ast.UnaryNode)
			a.True(ok)

			// Test executeDateTimeMethod with the root node set to tc.value.
			e := newTestExecutor(path, nil, true, false)
			e.root = tc.value
			if tc.silent {
				e.verbose = false
			}
			res, err := e.executeDateTimeMethod(ctx, node, tc.value, found)
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

func TestParseDateTimeFormat(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	e := &Executor{}
	err := e.parseDateTimeFormat("", nil)
	r.EqualError(err, "exec: .datetime(template) is not yet supported")
	r.ErrorIs(err, ErrExecution)
}

func TestParseDateTime(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	r := require.New(t)
	ctx := context.Background()
	path, _ := parser.Parse("$")

	for _, tc := range []struct {
		name  string
		op    ast.UnaryOperator
		value string
		arg   ast.Node
		exp   types.DateTime
		err   string
		isErr error
	}{
		{
			name:  "invalid_precision",
			op:    ast.UnaryTime,
			arg:   ast.NewString("hi"),
			err:   "exec: invalid jsonpath item type for .time() time precision",
			isErr: ErrExecution,
		},
		{
			name:  "negative_precision",
			op:    ast.UnaryTime,
			arg:   ast.NewInteger("-1"),
			err:   "exec: time precision of jsonpath item method .time() is invalid",
			isErr: ErrExecution,
		},
		{
			name:  "max_precision_six",
			op:    ast.UnaryTime,
			arg:   ast.NewInteger("9"),
			value: "14:15:31.78599685301",
			exp:   types.NewTime(time.Date(0, 1, 1, 14, 15, 31, 785997000, time.UTC)),
		},
		{
			name:  "precision_three",
			op:    ast.UnaryTime,
			arg:   ast.NewInteger("3"),
			value: "14:15:31.78599685301",
			exp:   types.NewTime(time.Date(0, 1, 1, 14, 15, 31, 786000000, time.UTC)),
		},
		{
			name:  "format_not_recognized",
			op:    ast.UnaryTime,
			value: "nope",
			err:   `exec: time format is not recognized: "nope"`,
			isErr: ErrVerbose,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Test parseDateTime.
			e := newTestExecutor(path, nil, true, false)
			res, err := e.parseDateTime(ctx, tc.op, tc.value, tc.arg)
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

func TestNotRecognized(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	for _, tc := range []struct {
		name string
		op   ast.UnaryOperator
		typ  string
		val  string
	}{
		{
			name: "date_nope",
			op:   ast.UnaryDate,
			typ:  "date",
			val:  "nope",
		},
		{
			name: "timestamp_time",
			op:   ast.UnaryTimestamp,
			typ:  "timestamp",
			val:  "12:34:21",
		},
		{
			name: "timestamptz_time",
			op:   ast.UnaryTimestampTZ,
			typ:  "timestamp_tz",
			val:  "12:34:21",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := notRecognized(tc.op, tc.val)
			r.EqualError(
				err,
				fmt.Sprintf(`exec: %v format is not recognized: "%v"`, tc.typ, tc.val),
			)
			r.ErrorIs(err, ErrVerbose)
		})
	}
}

type testDatetimeCast struct {
	name  string
	val   types.DateTime
	str   string
	useTZ bool
	exp   types.DateTime
	err   string
	isErr error
}

func (tc testDatetimeCast) run(t *testing.T, cast func(*Executor) (types.DateTime, error)) {
	t.Helper()
	a := assert.New(t)
	r := require.New(t)

	// Test castDate.
	e := &Executor{}
	e.useTZ = tc.useTZ
	res, err := cast(e)
	a.Equal(tc.exp, res)

	// Check the error.
	if tc.isErr == nil {
		r.NoError(err)
	} else {
		r.EqualError(err, tc.err)
		r.ErrorIs(err, tc.isErr)
	}
}

// To test the handling of unknown types.DateTime types.
type mockDateTime struct{}

func (mockDateTime) GoTime() time.Time               { return time.Now() }
func (mockDateTime) ToString(context.Context) string { return "" }
func TestCastDate(t *testing.T) {
	t.Parallel()
	moment := stableTime()
	var nilDate *types.Date
	ctx := context.Background()

	for _, tc := range []testDatetimeCast{
		{
			name: "date",
			val:  types.NewDate(moment),
			exp:  types.NewDate(moment),
		},
		{
			name:  "time",
			val:   types.NewTime(moment),
			str:   "a datetime string",
			exp:   nilDate,
			err:   `exec: date format is not recognized: "a datetime string"`,
			isErr: ErrVerbose,
		},
		{
			name:  "timetz",
			val:   types.NewTimeTZ(moment),
			str:   "a datetime string",
			exp:   nilDate,
			err:   `exec: date format is not recognized: "a datetime string"`,
			isErr: ErrVerbose,
		},
		{
			name: "timestamp",
			val:  types.NewTimestamp(moment),
			exp:  types.NewDate(moment),
		},
		{
			name:  "timestamptz",
			val:   types.NewTimestampTZ(ctx, moment),
			exp:   nilDate,
			err:   "exec: cannot convert value from timestamptz to date without time zone usage." + tzHint,
			isErr: ErrExecution,
		},
		{
			name:  "timestamptz_cast",
			val:   types.NewTimestampTZ(ctx, moment),
			exp:   types.NewDate(moment.UTC()),
			useTZ: true,
		},
		{
			name:  "unknown_datetime_type",
			val:   mockDateTime{},
			exp:   nilDate,
			err:   "exec invalid: type exec.mockDateTime not supported",
			isErr: ErrInvalid,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t, func(e *Executor) (types.DateTime, error) {
				return e.castDate(ctx, tc.val, tc.str)
			})
		})
	}
}

func TestCastTime(t *testing.T) {
	t.Parallel()
	moment := stableTime()
	var nilTime *types.Time
	ctx := context.Background()

	for _, tc := range []testDatetimeCast{
		{
			name: "time",
			val:  types.NewTime(moment),
			exp:  types.NewTime(moment),
		},
		{
			name:  "date",
			val:   types.NewDate(moment),
			str:   "hi",
			exp:   nilTime,
			err:   `exec: time format is not recognized: "hi"`,
			isErr: ErrVerbose,
		},
		{
			name:  "timetz",
			val:   types.NewTimeTZ(moment),
			exp:   nilTime,
			err:   "exec: cannot convert value from timetz to time without time zone usage." + tzHint,
			isErr: ErrExecution,
		},
		{
			name:  "timetz_cast",
			val:   types.NewTimeTZ(moment),
			exp:   types.NewTime(moment),
			useTZ: true,
		},
		{
			name: "timestamp",
			val:  types.NewTimestamp(moment),
			exp:  types.NewTime(moment),
		},
		{
			name:  "timestamptz",
			val:   types.NewTimestampTZ(ctx, moment),
			exp:   nilTime,
			err:   "exec: cannot convert value from timestamptz to time without time zone usage." + tzHint,
			isErr: ErrExecution,
		},
		{
			name:  "timestamptz_cast",
			val:   types.NewTimestampTZ(ctx, moment),
			exp:   types.NewTime(moment.UTC()),
			useTZ: true,
		},
		{
			name:  "unknown_datetime_type",
			val:   mockDateTime{},
			exp:   nilTime,
			err:   "exec invalid: type exec.mockDateTime not supported",
			isErr: ErrInvalid,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t, func(e *Executor) (types.DateTime, error) {
				return e.castTime(ctx, tc.val, tc.str)
			})
		})
	}
}

func TestCastTimeTZ(t *testing.T) {
	t.Parallel()
	moment := stableTime()
	var nilTimeTZ *types.TimeTZ
	ctx := context.Background()

	for _, tc := range []testDatetimeCast{
		{
			name: "timetz",
			val:  types.NewTimeTZ(moment),
			exp:  types.NewTimeTZ(moment),
		},
		{
			name:  "date",
			val:   types.NewDate(moment),
			str:   "hi",
			exp:   nilTimeTZ,
			err:   `exec: time_tz format is not recognized: "hi"`,
			isErr: ErrVerbose,
		},
		{
			name:  "time",
			val:   types.NewTime(moment),
			exp:   nilTimeTZ,
			err:   "exec: cannot convert value from time to timetz without time zone usage." + tzHint,
			isErr: ErrExecution,
		},
		{
			name: "time_cast",
			val:  types.NewTime(moment),
			exp: types.NewTimeTZ(time.Date(
				0, 1, 1,
				moment.Hour(), moment.Minute(), moment.Second(), moment.Nanosecond(),
				time.UTC,
			)),
			useTZ: true,
		},
		{
			name:  "timestamp",
			val:   types.NewTimestamp(moment),
			str:   "hi",
			exp:   nilTimeTZ,
			err:   `exec: time_tz format is not recognized: "hi"`,
			isErr: ErrVerbose,
		},
		{
			name: "timestamptz",
			val:  types.NewTimestampTZ(ctx, moment),
			exp:  types.NewTimestampTZ(ctx, moment).ToTimeTZ(ctx),
		},
		{
			name:  "unknown_datetime_type",
			val:   mockDateTime{},
			exp:   nilTimeTZ,
			err:   "exec invalid: type exec.mockDateTime not supported",
			isErr: ErrInvalid,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t, func(e *Executor) (types.DateTime, error) {
				return e.castTimeTZ(ctx, tc.val, tc.str)
			})
		})
	}
}

func TestCastTimestamp(t *testing.T) {
	t.Parallel()
	moment := stableTime()
	var nilTimestamp *types.Timestamp
	ctx := context.Background()

	for _, tc := range []testDatetimeCast{
		{
			name: "timestamp",
			val:  types.NewTimestamp(moment),
			exp:  types.NewTimestamp(moment),
		},
		{
			name: "date",
			val:  types.NewDate(moment),
			exp:  types.NewTimestamp(types.NewDate(moment).GoTime()),
		},
		{
			name:  "time",
			val:   types.NewTime(moment),
			exp:   nilTimestamp,
			str:   "foo",
			err:   `exec: timestamp format is not recognized: "foo"`,
			isErr: ErrVerbose,
		},
		{
			name:  "timetz",
			val:   types.NewTimeTZ(moment),
			exp:   nilTimestamp,
			str:   "bar",
			err:   `exec: timestamp format is not recognized: "bar"`,
			isErr: ErrVerbose,
		},
		{
			name:  "timestamptz",
			val:   types.NewTimestampTZ(ctx, moment),
			exp:   nilTimestamp,
			err:   "exec: cannot convert value from timestamptz to timestamp without time zone usage." + tzHint,
			isErr: ErrExecution,
		},
		{
			name:  "timestamptz_cast",
			val:   types.NewTimestampTZ(ctx, moment),
			exp:   types.NewTimestamp(moment.UTC()),
			useTZ: true,
		},
		{
			name:  "unknown_datetime_type",
			val:   mockDateTime{},
			exp:   nilTimestamp,
			err:   "exec invalid: type exec.mockDateTime not supported",
			isErr: ErrInvalid,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t, func(e *Executor) (types.DateTime, error) {
				return e.castTimestamp(ctx, tc.val, tc.str)
			})
		})
	}
}

func TestCastTimestampTZ(t *testing.T) {
	t.Parallel()
	moment := stableTime()
	var nilTimestampTZ *types.TimestampTZ
	ctx := context.Background()

	for _, tc := range []testDatetimeCast{
		{
			name: "timestamptz",
			val:  types.NewTimestampTZ(ctx, moment),
			exp:  types.NewTimestampTZ(ctx, moment),
		},
		{
			name:  "date",
			val:   types.NewDate(moment),
			exp:   nilTimestampTZ,
			err:   "exec: cannot convert value from date to timestamptz without time zone usage." + tzHint,
			isErr: ErrExecution,
		},
		{
			name:  "date_cast",
			val:   types.NewDate(moment),
			exp:   types.NewDate(moment).ToTimestampTZ(ctx),
			useTZ: true,
		},
		{
			name:  "time",
			val:   types.NewTime(moment),
			exp:   nilTimestampTZ,
			str:   "foo",
			err:   `exec: timestamp_tz format is not recognized: "foo"`,
			isErr: ErrVerbose,
		},
		{
			name:  "timetz",
			val:   types.NewTimeTZ(moment),
			exp:   nilTimestampTZ,
			str:   "bar",
			err:   `exec: timestamp_tz format is not recognized: "bar"`,
			isErr: ErrVerbose,
		},
		{
			name:  "timestamp",
			val:   types.NewTimestamp(moment),
			exp:   nilTimestampTZ,
			err:   "exec: cannot convert value from timestamp to timestamptz without time zone usage." + tzHint,
			isErr: ErrExecution,
		},
		{
			name:  "timestamp_cast",
			val:   types.NewTimestamp(moment),
			exp:   types.NewTimestampTZ(ctx, moment.UTC()),
			useTZ: true,
		},
		{
			name:  "unknown_datetime_type",
			val:   mockDateTime{},
			exp:   nilTimestampTZ,
			err:   "exec invalid: type exec.mockDateTime not supported",
			isErr: ErrInvalid,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t, func(e *Executor) (types.DateTime, error) {
				return e.castTimestampTZ(ctx, tc.val, tc.str)
			})
		})
	}
}
