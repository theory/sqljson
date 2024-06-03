//nolint:lll // Ignore long lines copied from Postgres.
package exec

// Tests from https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql
// Results from https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/expected/jsonb_jsonpath.out
// Test cases scaffolded by pasting each block of tests under __DATA__ in
// pg2go.pl and running `./internal/sqljson/path/exec/pg2go.pl | pbcopy`.

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/theory/sqljson/path/parser"
	"github.com/theory/sqljson/path/types"
)

// Convenience function to marshal a string into JSON.
func js(js string) any {
	var ret any
	if err := json.Unmarshal([]byte(js), &ret); err != nil {
		panic(err)
	}
	return ret
}

// Convenience function to marshal a string into a JSON object.
func jv(js string) map[string]any {
	var ret map[string]any
	if err := json.Unmarshal([]byte(js), &ret); err != nil {
		panic(err)
	}
	return ret
}

// Test cases for Exists().
type existsTestCase struct {
	name string
	path string
	json any
	exp  any
	err  string
	opt  []Option
}

func (tc existsTestCase) run(a *assert.Assertions, r *require.Assertions) {
	path, err := parser.Parse(tc.path)
	r.NoError(err)

	res, err := Exists(context.Background(), path, tc.json, tc.opt...)
	switch {
	case tc.err != "":
		r.EqualError(err, tc.err)
		r.ErrorIs(err, ErrExecution)
		a.False(res)
	case tc.exp == nil:
		// When Postgres returns NULL, we return false + ErrNull
		r.EqualError(err, "NULL")
		r.ErrorIs(err, NULL)
		a.False(res)
	default:
		r.NoError(err)
		a.Equal(tc.exp, res)
	}
}

// Mimic the Postgres @? operator.
func (tc existsTestCase) runAtQuestion(a *assert.Assertions, r *require.Assertions) {
	tc.opt = append(tc.opt, WithSilent())
	tc.run(a, r)
}

// Test cases for Match().
type matchTestCase existsTestCase

func (tc matchTestCase) run(a *assert.Assertions, r *require.Assertions) {
	path, err := parser.Parse(tc.path)
	r.NoError(err)

	res, err := Match(context.Background(), path, tc.json, tc.opt...)
	switch {
	case tc.err != "":
		r.EqualError(err, tc.err)
		r.ErrorIs(err, ErrExecution)
		a.False(res)
	case tc.exp == nil:
		// When Postgres returns NULL, we return false + ErrNull
		r.EqualError(err, "NULL")
		r.ErrorIs(err, NULL)
		a.False(res)
	default:
		r.NoError(err)
		a.Equal(tc.exp, res)
	}
}

// Mimic the Postgres @@ operator.
func (tc matchTestCase) runAtAt(a *assert.Assertions, r *require.Assertions) {
	tc.opt = append(tc.opt, WithSilent())
	tc.run(a, r)
}

// Test cases for Query().
type queryTestCase struct {
	name string
	path string
	json any
	exp  []any
	err  string
	opt  []Option
	rand bool
}

func (tc queryTestCase) run(a *assert.Assertions, r *require.Assertions) {
	path, err := parser.Parse(tc.path)
	r.NoError(err)
	res, err := Query(context.Background(), path, tc.json, tc.opt...)

	if tc.err != "" {
		r.EqualError(err, tc.err)
		r.ErrorIs(err, ErrExecution)
		a.Nil(res)
	} else {
		r.NoError(err)
		if tc.rand {
			a.ElementsMatch(tc.exp, res)
		} else {
			a.Equal(tc.exp, res)
		}
	}
}

// Test cases for First().
type firstTestCase struct {
	name string
	path string
	json any
	exp  any
	err  string
	opt  []Option
	rand bool
}

func (tc firstTestCase) run(a *assert.Assertions, r *require.Assertions) {
	path, err := parser.Parse(tc.path)
	r.NoError(err)
	res, err := First(context.Background(), path, tc.json, tc.opt...)

	if tc.err != "" {
		r.EqualError(err, tc.err)
		r.ErrorIs(err, ErrExecution)
		a.Nil(res)
	} else {
		r.NoError(err)
		if tc.rand {
			a.ElementsMatch(tc.exp, res)
		} else {
			a.Equal(tc.exp, res)
		}
	}
}

func TestPgAtQuestion(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L1-L40
	for _, tc := range []existsTestCase{
		{
			name: "test_1",
			json: js(`{"a": 12}`),
			path: "$",
			exp:  true,
		},
		{
			name: "test_2",
			json: js(`{"a": 12}`),
			path: "1",
			exp:  true,
		},
		{
			name: "test_3",
			json: js(`{"a": 12}`),
			path: "$.a.b",
			exp:  false,
		},
		{
			name: "test_4",
			json: js(`{"a": 12}`),
			path: "$.b",
			exp:  false,
		},
		{
			name: "test_5",
			json: js(`{"a": 12}`),
			path: "$.a + 2",
			exp:  true,
		},
		{
			name: "test_6",
			json: js(`{"a": 12}`),
			path: "$.b + 2",
			exp:  nil,
		},
		{
			name: "test_7",
			json: js(`{"a": {"a": 12}}`),
			path: "$.a.a",
			exp:  true,
		},
		{
			name: "test_8",
			json: js(`{"a": {"a": 12}}`),
			path: "$.*.a",
			exp:  true,
		},
		{
			name: "test_9",
			json: js(`{"b": {"a": 12}}`),
			path: "$.*.a",
			exp:  true,
		},
		{
			name: "test_10",
			json: js(`{"b": {"a": 12}}`),
			path: "$.*.b",
			exp:  false,
		},
		{
			name: "test_11",
			json: js(`{"b": {"a": 12}}`),
			path: "strict $.*.b",
			exp:  nil,
		},
		{
			name: "test_12",
			json: js(`{}`),
			path: "$.*",
			exp:  false,
		},
		{
			name: "test_13",
			json: js(`{"a": 1}`),
			path: "$.*",
			exp:  true,
		},
		{
			name: "test_14",
			json: js(`{"a": {"b": 1}}`),
			path: "lax $.**{1}",
			exp:  true,
		},
		{
			name: "test_15",
			json: js(`{"a": {"b": 1}}`),
			path: "lax $.**{2}",
			exp:  true,
		},
		{
			name: "test_16",
			json: js(`{"a": {"b": 1}}`),
			path: "lax $.**{3}",
			exp:  false,
		},
		{
			name: "test_17",
			json: js(`[]`),
			path: "$[*]",
			exp:  false,
		},
		{
			name: "test_18",
			json: js(`[1]`),
			path: "$[*]",
			exp:  true,
		},
		{
			name: "test_19",
			json: js(`[1]`),
			path: "$[1]",
			exp:  false,
		},
		{
			name: "test_20",
			json: js(`[1]`),
			path: "strict $[1]",
			exp:  nil,
		},
		// 21-22 in TestPgQueryCompareAtQuestion
		{
			name: "test_23",
			json: js(`[1]`),
			path: "lax $[10000000000000000]",
			exp:  nil,
		},
		{
			name: "test_24",
			json: js(`[1]`),
			path: "strict $[10000000000000000]",
			exp:  nil,
		},
		// 25-26 in TestPgQueryCompareAtQuestion
		{
			name: "test_27",
			json: js(`[1]`),
			path: "$[0]",
			exp:  true,
		},
		{
			name: "test_28",
			json: js(`[1]`),
			path: "$[0.3]",
			exp:  true,
		},
		{
			name: "test_29",
			json: js(`[1]`),
			path: "$[0.5]",
			exp:  true,
		},
		{
			name: "test_30",
			json: js(`[1]`),
			path: "$[0.9]",
			exp:  true,
		},
		{
			name: "test_31",
			json: js(`[1]`),
			path: "$[1.2]",
			exp:  false,
		},
		{
			name: "test_32",
			json: js(`[1]`),
			path: "strict $[1.2]",
			exp:  nil,
		},
		{
			name: "test_33",
			json: js(`{"a": [1,2,3], "b": [3,4,5]}`),
			path: "$ ? (@.a[*] >  @.b[*])",
			exp:  false,
		},
		{
			name: "test_34",
			json: js(`{"a": [1,2,3], "b": [3,4,5]}`),
			path: "$ ? (@.a[*] >= @.b[*])",
			exp:  true,
		},
		{
			name: "test_35",
			json: js(`{"a": [1,2,3], "b": [3,4,"5"]}`),
			path: "$ ? (@.a[*] >= @.b[*])",
			exp:  true,
		},
		{
			name: "test_36",
			json: js(`{"a": [1,2,3], "b": [3,4,"5"]}`),
			path: "strict $ ? (@.a[*] >= @.b[*])",
			exp:  false,
		},
		{
			name: "test_37",
			json: js(`{"a": [1,2,3], "b": [3,4,null]}`),
			path: "$ ? (@.a[*] >= @.b[*])",
			exp:  true,
		},
		{
			name: "test_38",
			json: js(`1`),
			path: `$ ? ((@ == "1") is unknown)`,
			exp:  true,
		},
		{
			name: "test_39",
			json: js(`1`),
			path: `$ ? ((@ == 1) is unknown)`,
			exp:  false,
		},
		{
			name: "test_40",
			json: js(`[{"a": 1}, {"a": 2}]`),
			path: `$[0 to 1] ? (@.a > 1)`,
			exp:  true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.runAtQuestion(a, r)
		})
	}
}

func TestPgQueryCompareAtQuestion(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L21-L26
	for _, tc := range []queryTestCase{
		{
			name: "test_21",
			json: js(`[1]`),
			path: "strict $[1]",
			err:  "exec: jsonpath array subscript is out of bounds",
		},
		{
			name: "test_22",
			json: js(`[1]`),
			path: "strict $[1]",
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_25",
			json: js(`[1]`),
			path: "lax $[10000000000000000]",
			err:  "exec: jsonpath array subscript is out of integer range",
		},
		{
			name: "test_26",
			json: js(`[1]`),
			path: "strict $[10000000000000000]",
			err:  "exec: jsonpath array subscript is out of integer range",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgExists(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L42-L45
	for _, tc := range []existsTestCase{
		{
			name: "test_1",
			json: js(`[{"a": 1}, {"a": 2}, 3]`),
			path: "lax $[*].a",
			exp:  true,
		},
		{
			name: "test_2",
			json: js(`[{"a": 1}, {"a": 2}, 3]`),
			path: "lax $[*].a",
			opt:  []Option{WithSilent()},
			exp:  true,
		},
		{
			name: "test_3",
			json: js(`[{"a": 1}, {"a": 2}, 3]`),
			path: "strict $[*].a",
			err:  "exec: jsonpath member accessor can only be applied to an object",
		},
		{
			name: "test_4",
			json: js(`[{"a": 1}, {"a": 2}, 3]`),
			path: "strict $[*].a",
			opt:  []Option{WithSilent()},
			exp:  nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgQueryModes(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L47-L57
	for _, tc := range []queryTestCase{
		{
			name: "test_1",
			json: js(`1`),
			path: `lax $.a`,
			exp:  []any{},
		},
		{
			name: "test_2",
			json: js(`1`),
			path: `strict $.a`,
			err:  "exec: jsonpath member accessor can only be applied to an object",
		},
		{
			name: "test_3",
			json: js(`1`),
			path: `strict $.*`,
			err:  "exec: jsonpath wildcard member accessor can only be applied to an object",
		},
		{
			name: "test_4",
			json: js(`1`),
			path: `strict $.a`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_5",
			json: js(`1`),
			path: `strict $.*`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_6",
			json: js(`[]`),
			path: `lax $.a`,
			exp:  []any{},
		},
		{
			name: "test_7",
			json: js(`[]`),
			path: `strict $.a`,
			err:  "exec: jsonpath member accessor can only be applied to an object",
		},
		{
			name: "test_8",
			json: js(`[]`),
			path: `strict $.a`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_9",
			json: js(`{}`),
			path: `lax $.a`,
			exp:  []any{},
		},
		{
			name: "test_10",
			json: js(`{}`),
			path: `strict $.a`,
			err:  `exec: JSON object does not contain key "a"`,
		},
		{
			name: "test_11",
			json: js(`{}`),
			path: `strict $.a`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgQueryStrict(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L59-L66
	for _, tc := range []queryTestCase{
		{
			name: "test_1",
			json: js(`1`),
			path: `strict $[1]`,
			err:  "exec: jsonpath array accessor can only be applied to an array",
		},
		{
			name: "test_2",
			json: js(`1`),
			path: `strict $[*]`,
			err:  "exec: jsonpath wildcard array accessor can only be applied to an array",
		},
		{
			name: "test_3",
			json: js(`[]`),
			path: `strict $[1]`,
			err:  "exec: jsonpath array subscript is out of bounds",
		},
		{
			name: "test_4",
			json: js(`[]`),
			path: `strict $["a"]`,
			err:  "exec: jsonpath array subscript is not a single numeric value",
		},
		{
			name: "test_5",
			json: js(`1`),
			path: `strict $[1]`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_6",
			json: js(`1`),
			path: `strict $[*]`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_7",
			json: js(`[]`),
			path: `strict $[1]`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_8",
			json: js(`[]`),
			path: `strict $["a"]`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgQueryBasics(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L68-L97
	for _, tc := range []queryTestCase{
		{
			name: "test_1",
			json: js(`{"a": 12, "b": {"a": 13}}`),
			path: `$.a`,
			exp:  []any{float64(12)},
		},

		{
			name: "test_2",
			json: js(`{"a": 12, "b": {"a": 13}}`),
			path: `$.b`,
			exp:  []any{js(`{"a": 13}`)},
		},
		{
			name: "test_3",
			json: js(`{"a": 12, "b": {"a": 13}}`),
			path: `$.*`,
			exp:  []any{float64(12), js(`{"a": 13}`)},
			rand: true,
		},
		{
			name: "test_4",
			json: js(`{"a": 12, "b": {"a": 13}}`),
			path: `lax $.*.a`,
			exp:  []any{float64(13)},
		},
		{
			name: "test_5",
			json: js(`[12, {"a": 13}, {"b": 14}]`),
			path: `lax $[*].a`,
			exp:  []any{float64(13)},
		},
		{
			name: "test_6",
			json: js(`[12, {"a": 13}, {"b": 14}]`),
			path: `lax $[*].*`,
			exp:  []any{float64(13), float64(14)},
			rand: true,
		},
		{
			name: "test_7",
			json: js(`[12, {"a": 13}, {"b": 14}]`),
			path: `lax $[0].a`,
			exp:  []any{},
		},
		{
			name: "test_8",
			json: js(`[12, {"a": 13}, {"b": 14}]`),
			path: `lax $[1].a`,
			exp:  []any{float64(13)},
		},
		{
			name: "test_9",
			json: js(`[12, {"a": 13}, {"b": 14}]`),
			path: `lax $[2].a`,
			exp:  []any{},
		},
		{
			name: "test_10",
			json: js(`[12, {"a": 13}, {"b": 14}]`),
			path: `lax $[0,1].a`,
			exp:  []any{float64(13)},
		},
		{
			name: "test_11",
			json: js(`[12, {"a": 13}, {"b": 14}]`),
			path: `lax $[0 to 10].a`,
			exp:  []any{float64(13)},
		},
		{
			name: "test_12",
			json: js(`[12, {"a": 13}, {"b": 14}]`),
			path: `lax $[0 to 10 / 0].a`,
			err:  "exec: division by zero",
		},
		{
			name: "test_13",
			json: js(`[12, {"a": 13}, {"b": 14}, "ccc", true]`),
			path: `$[2.5 - 1 to $.size() - 2]`,
			exp:  []any{js(`{"a": 13}`), js(`{"b": 14}`), "ccc"},
		},
		{
			name: "test_14",
			json: js(`1`),
			path: `lax $[0]`,
			exp:  []any{float64(1)},
		},
		{
			name: "test_15",
			json: js(`1`),
			path: `lax $[*]`,
			exp:  []any{float64(1)},
		},
		{
			name: "test_16",
			json: js(`[1]`),
			path: `lax $[0]`,
			exp:  []any{float64(1)},
		},
		{
			name: "test_17",
			json: js(`[1]`),
			path: `lax $[*]`,
			exp:  []any{float64(1)},
		},
		{
			name: "test_18",
			json: js(`[1,2,3]`),
			path: `lax $[*]`,
			exp:  []any{float64(1), float64(2), float64(3)},
		},
		{
			name: "test_19",
			json: js(`[1,2,3]`),
			path: `strict $[*].a`,
			err:  "exec: jsonpath member accessor can only be applied to an object",
		},
		{
			name: "test_20",
			json: js(`[1,2,3]`),
			path: `strict $[*].a`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_21",
			json: js(`[]`),
			path: `$[last]`,
			exp:  []any{},
		},
		{
			name: "test_22",
			json: js(`[]`),
			path: `$[last ? (exists(last))]`,
			exp:  []any{},
		},
		{
			name: "test_23",
			json: js(`[]`),
			path: `strict $[last]`,
			err:  "exec: jsonpath array subscript is out of bounds",
		},
		{
			name: "test_24",
			json: js(`[]`),
			path: `strict $[last]`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_25",
			json: js(`[1]`),
			path: `$[last]`,
			exp:  []any{float64(1)},
		},
		{
			name: "test_26",
			json: js(`[1,2,3]`),
			path: `$[last]`,
			exp:  []any{float64(3)},
		},
		{
			name: "test_27",
			json: js(`[1,2,3]`),
			path: `$[last - 1]`,
			exp:  []any{float64(2)},
		},
		{
			name: "test_28",
			json: js(`[1,2,3]`),
			path: `$[last ? (@.type() == "number")]`,
			exp:  []any{float64(3)},
		},
		{
			name: "test_29",
			json: js(`[1,2,3]`),
			path: `$[last ? (@.type() == "string")]`,
			err:  "exec: jsonpath array subscript is not a single numeric value",
		},
		{
			name: "test_30",
			json: js(`[1,2,3]`),
			path: `$[last ? (@.type() == "string")]`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgQueryBinaryOps(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L99-L115
	for _, tc := range []queryTestCase{
		{
			name: "test_1",
			json: js(`{"a": 10}`),
			path: `$`,
			exp:  []any{js(`{"a": 10}`)},
		},
		{
			name: "test_2",
			json: js(`{"a": 10}`),
			path: `$ ? (@.a < $value)`,
			err:  `exec: could not find jsonpath variable "value"`,
		},
		// We have more control than Postgres here, where the requirement that
		// vars be a map is enforced at compile time.
		// {
		// 	name: "test_3",
		// 	json: js(`{"a": 10}`),
		// 	path: `$ ? (@.a < $value)`,
		// 	opt:  []Option{WithVars(int64(1))},
		// 	err:  `exec: "vars" argument is not an object`,
		// },
		// {
		// 	name: "test_4",
		// 	json: js(`{"a": 10}`),
		// 	path: `$ ? (@.a < $value)`,
		// 	opt:  []Option{WithVars(jv(`[{"value" : 13}]`))},
		// 	err:  `exec: "vars" argument is not an object`,
		// },
		{
			name: "test_5",
			json: js(`{"a": 10}`),
			path: `$ ? (@.a < $value)`,
			opt:  []Option{WithVars(jv(`{"value" : 13}`))},
			exp:  []any{js(`{"a": 10}`)},
		},
		{
			name: "test_6",
			json: js(`{"a": 10}`),
			path: `$ ? (@.a < $value)`,
			opt:  []Option{WithVars(jv(`{"value" : 8}`))},
			exp:  []any{},
		},
		{
			name: "test_7",
			json: js(`{"a": 10}`),
			path: `$.a ? (@ < $value)`,
			opt:  []Option{WithVars(jv(`{"value" : 13}`))},
			exp:  []any{float64(10)},
		},
		{
			name: "test_8",
			json: js(`[10,11,12,13,14,15]`),
			path: `$[*] ? (@ < $value)`,
			opt:  []Option{WithVars(jv(`{"value" : 13}`))},
			exp:  []any{float64(10), float64(11), float64(12)},
		},
		{
			name: "test_9",
			json: js(`[10,11,12,13,14,15]`),
			path: `$[0,1] ? (@ < $x.value)`,
			opt:  []Option{WithVars(jv(`{"x": {"value" : 13}}`))},
			exp:  []any{float64(10), float64(11)},
		},
		{
			name: "test_10",
			json: js(`[10,11,12,13,14,15]`),
			path: `$[0 to 2] ? (@ < $value)`,
			opt:  []Option{WithVars(jv(`{"value" : 15}`))},
			exp:  []any{float64(10), float64(11), float64(12)},
		},
		{
			name: "test_11",
			json: js(`[1,"1",2,"2",null]`),
			path: `$[*] ? (@ == "1")`,
			exp:  []any{"1"},
		},
		{
			name: "test_12",
			json: js(`[1,"1",2,"2",null]`),
			path: `$[*] ? (@ == $value)`,
			opt:  []Option{WithVars(jv(`{"value" : "1"}`))},
			exp:  []any{"1"},
		},
		{
			name: "test_13",
			json: js(`[1,"1",2,"2",null]`),
			path: `$[*] ? (@ == $value)`,
			opt:  []Option{WithVars(jv(`{"value" : null}`))},
			exp:  []any{nil},
		},
		{
			name: "test_14",
			json: js(`[1, "2", null]`),
			path: `$[*] ? (@ != null)`,
			exp:  []any{float64(1), "2"},
		},
		{
			name: "test_15",
			json: js(`[1, "2", null]`),
			path: `$[*] ? (@ == null)`,
			exp:  []any{nil},
		},
		{
			name: "test_16",
			json: js(`{}`),
			path: `$ ? (@ == @)`,
			exp:  []any{},
		},
		{
			name: "test_17",
			json: js(`[]`),
			path: `strict $ ? (@ == @)`,
			exp:  []any{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgQueryAny(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L117-L138
	for _, tc := range []queryTestCase{
		{
			name: "test_1",
			json: js(`{"a": {"b": 1}}`),
			path: `lax $.**`,
			exp:  []any{js(`{"a": {"b": 1}}`), js(`{"b": 1}`), float64(1)},
		},
		{
			name: "test_2",
			json: js(`{"a": {"b": 1}}`),
			path: `lax $.**{0}`,
			exp:  []any{js(`{"a": {"b": 1}}`)},
		},
		{
			name: "test_3",
			json: js(`{"a": {"b": 1}}`),
			path: `lax $.**{0 to last}`,
			exp:  []any{js(`{"a": {"b": 1}}`), js(`{"b": 1}`), float64(1)},
		},
		{
			name: "test_4",
			json: js(`{"a": {"b": 1}}`),
			path: `lax $.**{1}`,
			exp:  []any{js(`{"b": 1}`)},
		},
		{
			name: "test_5",
			json: js(`{"a": {"b": 1}}`),
			path: `lax $.**{1 to last}`,
			exp:  []any{js(`{"b": 1}`), float64(1)},
		},
		{
			name: "test_6",
			json: js(`{"a": {"b": 1}}`),
			path: `lax $.**{2}`,
			exp:  []any{float64(1)},
		},
		{
			name: "test_7",
			json: js(`{"a": {"b": 1}}`),
			path: `lax $.**{2 to last}`,
			exp:  []any{float64(1)},
		},
		{
			name: "test_8",
			json: js(`{"a": {"b": 1}}`),
			path: `lax $.**{3 to last}`,
			exp:  []any{},
		},
		{
			name: "test_9",
			json: js(`{"a": {"b": 1}}`),
			path: `lax $.**{last}`,
			exp:  []any{float64(1)},
		},
		{
			name: "test_10",
			json: js(`{"a": {"b": 1}}`),
			path: `lax $.**.b ? (@ > 0)`,
			exp:  []any{float64(1)},
		},
		{
			name: "test_11",
			json: js(`{"a": {"b": 1}}`),
			path: `lax $.**{0}.b ? (@ > 0)`,
			exp:  []any{},
		},
		{
			name: "test_12",
			json: js(`{"a": {"b": 1}}`),
			path: `lax $.**{1}.b ? (@ > 0)`,
			exp:  []any{float64(1)},
		},
		{
			name: "test_13",
			json: js(`{"a": {"b": 1}}`),
			path: `lax $.**{0 to last}.b ? (@ > 0)`,
			exp:  []any{float64(1)},
		},
		{
			name: "test_14",
			json: js(`{"a": {"b": 1}}`),
			path: `lax $.**{1 to last}.b ? (@ > 0)`,
			exp:  []any{float64(1)},
		},
		{
			name: "test_15",
			json: js(`{"a": {"b": 1}}`),
			path: `lax $.**{1 to 2}.b ? (@ > 0)`,
			exp:  []any{float64(1)},
		},
		{
			name: "test_16",
			json: js(`{"a": {"c": {"b": 1}}}`),
			path: `lax $.**.b ? (@ > 0)`,
			exp:  []any{float64(1)},
		},
		{
			name: "test_17",
			json: js(`{"a": {"c": {"b": 1}}}`),
			path: `lax $.**{0}.b ? (@ > 0)`,
			exp:  []any{},
		},
		{
			name: "test_18",
			json: js(`{"a": {"c": {"b": 1}}}`),
			path: `lax $.**{1}.b ? (@ > 0)`,
			exp:  []any{},
		},
		{
			name: "test_19",
			json: js(`{"a": {"c": {"b": 1}}}`),
			path: `lax $.**{0 to last}.b ? (@ > 0)`,
			exp:  []any{float64(1)},
		},
		{
			name: "test_20",
			json: js(`{"a": {"c": {"b": 1}}}`),
			path: `lax $.**{1 to last}.b ? (@ > 0)`,
			exp:  []any{float64(1)},
		},
		{
			name: "test_21",
			json: js(`{"a": {"c": {"b": 1}}}`),
			path: `lax $.**{1 to 2}.b ? (@ > 0)`,
			exp:  []any{float64(1)},
		},
		{
			name: "test_22",
			json: js(`{"a": {"c": {"b": 1}}}`),
			path: `lax $.**{2 to 3}.b ? (@ > 0)`,
			exp:  []any{float64(1)},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgAtQuestionAny(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L140-L152
	for _, tc := range []existsTestCase{
		{
			name: "test_1",
			json: js(`{"a": {"b": 1}}`),
			path: `$.**.b ? ( @ > 0)`,
			exp:  true,
		},
		{
			name: "test_2",
			json: js(`{"a": {"b": 1}}`),
			path: `$.**{0}.b ? ( @ > 0)`,
			exp:  false,
		},
		{
			name: "test_3",
			json: js(`{"a": {"b": 1}}`),
			path: `$.**{1}.b ? ( @ > 0)`,
			exp:  true,
		},
		{
			name: "test_4",
			json: js(`{"a": {"b": 1}}`),
			path: `$.**{0 to last}.b ? ( @ > 0)`,
			exp:  true,
		},
		{
			name: "test_5",
			json: js(`{"a": {"b": 1}}`),
			path: `$.**{1 to last}.b ? ( @ > 0)`,
			exp:  true,
		},
		{
			name: "test_6",
			json: js(`{"a": {"b": 1}}`),
			path: `$.**{1 to 2}.b ? ( @ > 0)`,
			exp:  true,
		},
		{
			name: "test_7",
			json: js(`{"a": {"c": {"b": 1}}}`),
			path: `$.**.b ? ( @ > 0)`,
			exp:  true,
		},
		{
			name: "test_8",
			json: js(`{"a": {"c": {"b": 1}}}`),
			path: `$.**{0}.b ? ( @ > 0)`,
			exp:  false,
		},
		{
			name: "test_9",
			json: js(`{"a": {"c": {"b": 1}}}`),
			path: `$.**{1}.b ? ( @ > 0)`,
			exp:  false,
		},
		{
			name: "test_10",
			json: js(`{"a": {"c": {"b": 1}}}`),
			path: `$.**{0 to last}.b ? ( @ > 0)`,
			exp:  true,
		},
		{
			name: "test_11",
			json: js(`{"a": {"c": {"b": 1}}}`),
			path: `$.**{1 to last}.b ? ( @ > 0)`,
			exp:  true,
		},
		{
			name: "test_12",
			json: js(`{"a": {"c": {"b": 1}}}`),
			path: `$.**{1 to 2}.b ? ( @ > 0)`,
			exp:  true,
		},
		{
			name: "test_13",
			json: js(`{"a": {"c": {"b": 1}}}`),
			path: `$.**{2 to 3}.b ? ( @ > 0)`,
			exp:  true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.runAtQuestion(a, r)
		})
	}
}

func TestPgQueryExists(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L154-L163
	for _, tc := range []queryTestCase{
		{
			name: "test_1",
			json: js(`{"g": {"x": 2}}`),
			path: `$.g ? (exists (@.x))`,
			exp:  []any{js(`{"x": 2}`)},
		},
		{
			name: "test_2",
			json: js(`{"g": {"x": 2}}`),
			path: `$.g ? (exists (@.y))`,
			exp:  []any{},
		},
		{
			name: "test_3",
			json: js(`{"g": {"x": 2}}`),
			path: `$.g ? (exists (@.x ? (@ >= 2) ))`,
			exp:  []any{js(`{"x": 2}`)},
		},
		{
			name: "test_4",
			json: js(`{"g": [{"x": 2}, {"y": 3}]}`),
			path: `lax $.g ? (exists (@.x))`,
			exp:  []any{js(`{"x": 2}`)},
		},
		{
			name: "test_5",
			json: js(`{"g": [{"x": 2}, {"y": 3}]}`),
			path: `lax $.g ? (exists (@.x + "3"))`,
			exp:  []any{},
		},
		{
			name: "test_6",
			json: js(`{"g": [{"x": 2}, {"y": 3}]}`),
			path: `lax $.g ? ((exists (@.x + "3")) is unknown)`,
			exp:  []any{js(`{"x": 2}`), js(`{"y": 3}`)},
		},
		{
			name: "test_7",
			json: js(`{"g": [{"x": 2}, {"y": 3}]}`),
			path: `strict $.g[*] ? (exists (@.x))`,
			exp:  []any{js(`{"x": 2}`)},
		},
		{
			name: "test_8",
			json: js(`{"g": [{"x": 2}, {"y": 3}]}`),
			path: `strict $.g[*] ? ((exists (@.x)) is unknown)`,
			exp:  []any{js(`{"y": 3}`)},
		},
		{
			name: "test_9",
			json: js(`{"g": [{"x": 2}, {"y": 3}]}`),
			path: `strict $.g ? (exists (@[*].x))`,
			exp:  []any{},
		},
		{
			name: "test_10",
			json: js(`{"g": [{"x": 2}, {"y": 3}]}`),
			path: `strict $.g ? ((exists (@[*].x)) is unknown)`,
			exp:  []any{[]any{js(`{"x": 2}`), js(`{"y": 3}`)}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgQueryTernaryLogic(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L166-L190
	path1 := `$[*] ? (@ == true  &&  ($x == true && $y == true) ||
					  @ == false && !($x == true && $y == true) ||
					  @ == null  &&  ($x == true && $y == true) is unknown)`
	path2 := `$[*] ? (@ == true  &&  ($x == true || $y == true) ||
					  @ == false && !($x == true || $y == true) ||
					  @ == null  &&  ($x == true || $y == true) is unknown)`
	json := []any{true, false, nil}

	for _, tc := range []queryTestCase{
		{
			name: "test_1",
			json: json,
			path: path1,
			opt:  []Option{WithVars(jv(`{"x": true, "y": true}`))},
			exp:  []any{true},
		},
		{
			name: "test_2",
			json: json,
			path: path1,
			opt:  []Option{WithVars(jv(`{"x": true, "y": false}`))},
			exp:  []any{false},
		},
		{
			name: "test_3",
			json: json,
			path: path1,
			opt:  []Option{WithVars(jv(`{"x": true, "y": "null"}`))},
			exp:  []any{nil},
		},
		{
			name: "test_4",
			json: json,
			path: path1,
			opt:  []Option{WithVars(jv(`{"x": false, "y": true}`))},
			exp:  []any{false},
		},
		{
			name: "test_5",
			json: json,
			path: path1,
			opt:  []Option{WithVars(jv(`{"x": false, "y": false}`))},
			exp:  []any{false},
		},
		{
			name: "test_6",
			json: json,
			path: path1,
			opt:  []Option{WithVars(jv(`{"x": false, "y": "null"}`))},
			exp:  []any{false},
		},
		{
			name: "test_7",
			json: json,
			path: path1,
			opt:  []Option{WithVars(jv(`{"x": "null", "y": true}`))},
			exp:  []any{nil},
		},
		{
			name: "test_8",
			json: json,
			path: path1,
			opt:  []Option{WithVars(jv(`{"x": "null", "y": false}`))},
			exp:  []any{false},
		},
		{
			name: "test_9",
			json: json,
			path: path1,
			opt:  []Option{WithVars(jv(`{"x": "null", "y": "null"}`))},
			exp:  []any{nil},
		},
		{
			name: "test_10",
			json: json,
			path: path2,
			opt:  []Option{WithVars(jv(`{"x": true, "y": true}`))},
			exp:  []any{true},
		},
		{
			name: "test_11",
			json: json,
			path: path2,
			opt:  []Option{WithVars(jv(`{"x": true, "y": false}`))},
			exp:  []any{true},
		},
		{
			name: "test_12",
			json: json,
			path: path2,
			opt:  []Option{WithVars(jv(`{"x": true, "y": "null"}`))},
			exp:  []any{true},
		},
		{
			name: "test_13",
			json: json,
			path: path2,
			opt:  []Option{WithVars(jv(`{"x": false, "y": true}`))},
			exp:  []any{true},
		},
		{
			name: "test_14",
			json: json,
			path: path2,
			opt:  []Option{WithVars(jv(`{"x": false, "y": false}`))},
			exp:  []any{false},
		},
		{
			name: "test_15",
			json: json,
			path: path2,
			opt:  []Option{WithVars(jv(`{"x": false, "y": "null"}`))},
			exp:  []any{nil},
		},
		{
			name: "test_16",
			json: json,
			path: path2,
			opt:  []Option{WithVars(jv(`{"x": "null", "y": true}`))},
			exp:  []any{true},
		},
		{
			name: "test_17",
			json: json,
			path: path2,
			opt:  []Option{WithVars(jv(`{"x": "null", "y": false}`))},
			exp:  []any{nil},
		},
		{
			name: "test_18",
			json: json,
			path: path2,
			opt:  []Option{WithVars(jv(`{"x": "null", "y": "null"}`))},
			exp:  []any{nil},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgAtQuestionFilter(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L192-L198
	for _, tc := range []existsTestCase{
		{
			name: "test_1",
			json: js(`{"a": 1, "b":1}`),
			path: "$ ? (@.a == @.b)",
			exp:  true,
		},
		{
			name: "test_2",
			json: js(`{"c": {"a": 1, "b":1}}`),
			path: "$ ? (@.a == @.b)",
			exp:  false,
		},
		{
			name: "test_3",
			json: js(`{"c": {"a": 1, "b":1}}`),
			path: "$.c ? (@.a == @.b)",
			exp:  true,
		},
		{
			name: "test_4",
			json: js(`{"c": {"a": 1, "b":1}}`),
			path: "$.c ? ($.c.a == @.b)",
			exp:  true,
		},
		{
			name: "test_5",
			json: js(`{"c": {"a": 1, "b":1}}`),
			path: "$.* ? (@.a == @.b)",
			exp:  true,
		},
		{
			name: "test_6",
			json: js(`{"a": 1, "b":1}`),
			path: "$.** ? (@.a == @.b)",
			exp:  true,
		},
		{
			name: "test_7",
			json: js(`{"c": {"a": 1, "b":1}}`),
			path: "$.** ? (@.a == @.b)",
			exp:  true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.runAtQuestion(a, r)
		})
	}
}

func TestPgQueryAnyMath(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L200-L203
	for _, tc := range []queryTestCase{
		{
			name: "test_1",
			json: js(`{"c": {"a": 2, "b":1}}`),
			path: `$.** ? (@.a == 1 + 1)`,
			exp:  []any{js(`{"a": 2, "b": 1}`)},
		},
		{
			name: "test_2",
			json: js(`{"c": {"a": 2, "b":1}}`),
			path: `$.** ? (@.a == (1 + 1))`,
			exp:  []any{js(`{"a": 2, "b": 1}`)},
		},
		{
			name: "test_3",
			json: js(`{"c": {"a": 2, "b":1}}`),
			path: `$.** ? (@.a == @.b + 1)`,
			exp:  []any{js(`{"a": 2, "b": 1}`)},
		},
		{
			name: "test_4",
			json: js(`{"c": {"a": 2, "b":1}}`),
			path: `$.** ? (@.a == (@.b + 1))`,
			exp:  []any{js(`{"a": 2, "b": 1}`)},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgAtQuestionAnyMath(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L204-L215
	for _, tc := range []existsTestCase{
		{
			name: "test_1",
			json: js(`{"c": {"a": -1, "b":1}}`),
			path: "$.** ? (@.a == - 1)",
			exp:  true,
		},
		{
			name: "test_2",
			json: js(`{"c": {"a": -1, "b":1}}`),
			path: "$.** ? (@.a == -1)",
			exp:  true,
		},
		{
			name: "test_3",
			json: js(`{"c": {"a": -1, "b":1}}`),
			path: "$.** ? (@.a == -@.b)",
			exp:  true,
		},
		{
			name: "test_4",
			json: js(`{"c": {"a": -1, "b":1}}`),
			path: "$.** ? (@.a == - @.b)",
			exp:  true,
		},
		{
			name: "test_5",
			json: js(`{"c": {"a": 0, "b":1}}`),
			path: "$.** ? (@.a == 1 - @.b)",
			exp:  true,
		},
		{
			name: "test_6",
			json: js(`{"c": {"a": 2, "b":1}}`),
			path: "$.** ? (@.a == 1 - - @.b)",
			exp:  true,
		},
		{
			name: "test_7",
			json: js(`{"c": {"a": 0, "b":1}}`),
			path: "$.** ? (@.a == 1 - +@.b)",
			exp:  true,
		},
		{
			name: "test_8",
			json: js(`[1,2,3]`),
			path: "$ ? (+@[*] > +2)",
			exp:  true,
		},
		{
			name: "test_9",
			json: js(`[1,2,3]`),
			path: "$ ? (+@[*] > +3)",
			exp:  false,
		},
		{
			name: "test_10",
			json: js(`[1,2,3]`),
			path: "$ ? (-@[*] < -2)",
			exp:  true,
		},
		{
			name: "test_11",
			json: js(`[1,2,3]`),
			path: "$ ? (-@[*] < -3)",
			exp:  false,
		},
		{
			name: "test_12",
			json: js(`1`),
			path: "$ ? ($ > 0)",
			exp:  true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.runAtQuestion(a, r)
		})
	}
}

func TestPgQueryMathErrors(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L218-L230
	for _, tc := range []queryTestCase{
		{
			name: "test_1",
			json: js(`[1,2,0,3]`),
			path: `$[*] ? (2 / @ > 0)`,
			exp:  []any{float64(1), float64(2), float64(3)},
		},
		{
			name: "test_2",
			json: js(`[1,2,0,3]`),
			path: `$[*] ? ((2 / @ > 0) is unknown)`,
			exp:  []any{float64(0)},
		},
		{
			name: "test_3",
			json: js(`0`),
			path: `1 / $`,
			err:  "exec: division by zero",
		},
		{
			name: "test_4",
			json: js(`0`),
			path: `1 / $ + 2`,
			err:  "exec: division by zero",
		},
		{
			name: "test_5",
			json: js(`0`),
			path: `-(3 + 1 % $)`,
			err:  "exec: division by zero",
		},
		{
			name: "test_6",
			json: js(`1`),
			path: `$ + "2"`,
			err:  "exec: right operand of jsonpath operator + is not a single numeric value",
		},
		{
			name: "test_7",
			json: js(`[1, 2]`),
			path: `3 * $`,
			err:  "exec: right operand of jsonpath operator * is not a single numeric value",
		},
		{
			name: "test_8",
			json: js(`"a"`),
			path: `-$`,
			err:  "exec: operand of unary jsonpath operator - is not a numeric value",
		},
		{
			name: "test_9",
			json: js(`[1,"2",3]`),
			path: `+$`,
			err:  "exec: operand of unary jsonpath operator + is not a numeric value",
		},
		{
			name: "test_10",
			json: js(`1`),
			path: `$ + "2"`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_11",
			json: js(`[1, 2]`),
			path: `3 * $`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_12",
			json: js(`"a"`),
			path: `-$`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_13",
			json: js(`[1,"2",3]`),
			path: `+$`,
			opt:  []Option{WithSilent()},
			exp:  []any{float64(1)},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgAtQuestionMathErrors(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L231-L234
	for _, tc := range []existsTestCase{
		{
			name: "test_1",
			json: js(`["1",2,0,3]`),
			path: "-$[*]",
			exp:  true,
		},
		{
			name: "test_2",
			json: js(`[1,"2",0,3]`),
			path: "-$[*]",
			exp:  true,
		},
		{
			name: "test_3",
			json: js(`["1",2,0,3]`),
			path: "strict -$[*]",
			exp:  nil,
		},
		{
			name: "test_4",
			json: js(`[1,"2",0,3]`),
			path: "strict -$[*]",
			exp:  nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.runAtQuestion(a, r)
		})
	}
}

func TestPgQueryUnwrapping(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L236-L242
	for _, tc := range []queryTestCase{
		{
			name: "test_1",
			json: js(`{"a": [2]}`),
			path: `lax $.a * 3`,
			exp:  []any{float64(6)},
		},
		{
			name: "test_2",
			json: js(`{"a": [2]}`),
			path: `lax $.a + 3`,
			exp:  []any{float64(5)},
		},
		{
			name: "test_3",
			json: js(`{"a": [2, 3, 4]}`),
			path: `lax -$.a`,
			exp:  []any{float64(-2), float64(-3), float64(-4)},
		},
		//  should fail
		{
			name: "test_4",
			json: js(`{"a": [1, 2]}`),
			path: `lax $.a * 3`,
			err:  "exec: left operand of jsonpath operator * is not a single numeric value",
		},
		{
			name: "test_5",
			json: js(`{"a": [1, 2]}`),
			path: `lax $.a * 3`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgQueryBoolean(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L245-L247
	for _, tc := range []queryTestCase{
		{
			name: "test_1",
			json: js(`2`),
			path: `$ > 1`,
			exp:  []any{true},
		},
		{
			name: "test_2",
			json: js(`2`),
			path: `$ <= 1`,
			exp:  []any{false},
		},
		{
			name: "test_3",
			json: js(`2`),
			path: `$ == "2"`,
			exp:  []any{nil},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgAtQuestionBoolean(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L248
	for _, tc := range []existsTestCase{
		{
			name: "test_1",
			json: js(`2`),
			path: `$ == "2"`,
			exp:  true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.runAtQuestion(a, r)
		})
	}
}

func TestPgAtAtBoolean(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L250-L257
	for _, tc := range []matchTestCase{
		{
			name: "test_1",
			json: js(`2`),
			path: `$ > 1`,
			exp:  true,
		},
		{
			name: "test_2",
			json: js(`2`),
			path: `$ <= 1`,
			exp:  false,
		},
		{
			name: "test_3",
			json: js(`2`),
			path: `$ == "2"`,
			exp:  nil,
		},
		{
			name: "test_4",
			json: js(`2`),
			path: `1`,
			exp:  nil,
		},
		{
			name: "test_5",
			json: js(`{}`),
			path: `$`,
			exp:  nil,
		},
		{
			name: "test_6",
			json: js(`[]`),
			path: `$`,
			exp:  nil,
		},
		{
			name: "test_7",
			json: js(`[1,2,3]`),
			path: `$[*]`,
			exp:  nil,
		},
		{
			name: "test_8",
			json: js(`[]`),
			path: `$[*]`,
			exp:  nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.runAtAt(a, r)
		})
	}
}

func TestPgMatch(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L258-L264
	for _, tc := range []matchTestCase{
		{
			name: "test_1",
			json: js(`[[1, true], [2, false]]`),
			path: `strict $[*] ? (@[0] > $x) [1]`,
			opt:  []Option{WithVars(jv(`{"x": 1}`))},
			exp:  false,
		},
		{
			name: "test_2",
			json: js(`[[1, true], [2, false]]`),
			path: `strict $[*] ? (@[0] < $x) [1]`,
			opt:  []Option{WithVars(jv(`{"x": 2}`))},
			exp:  true,
		},
		{
			name: "test_3",
			json: js(`[{"a": 1}, {"a": 2}, 3]`),
			path: `lax exists($[*].a)`,
			exp:  true,
		},
		{
			name: "test_4",
			json: js(`[{"a": 1}, {"a": 2}, 3]`),
			path: `lax exists($[*].a)`,
			opt:  []Option{WithSilent()},
			exp:  true,
		},
		{
			name: "test_5",
			json: js(`[{"a": 1}, {"a": 2}, 3]`),
			path: `strict exists($[*].a)`,
			exp:  nil,
		},
		{
			name: "test_6",
			json: js(`[{"a": 1}, {"a": 2}, 3]`),
			opt:  []Option{WithSilent()},
			path: `strict exists($[*].a)`,
			exp:  nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgQueryTypeMethod(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L267-L273
	for _, tc := range []queryTestCase{
		{
			name: "test_1",
			json: js(`[null,1,true,"a",[],{}]`),
			path: `$.type()`,
			exp:  []any{"array"},
		},
		{
			name: "test_2",
			json: js(`[null,1,true,"a",[],{}]`),
			path: `lax $.type()`,
			exp:  []any{"array"},
		},
		{
			name: "test_3",
			json: js(`[null,1,true,"a",[],{}]`),
			path: `$[*].type()`,
			exp:  []any{"null", "number", "boolean", "string", "array", "object"},
		},
		{
			name: "test_4",
			json: js(`null`),
			path: `null.type()`,
			exp:  []any{"null"},
		},
		{
			name: "test_5",
			json: js(`null`),
			path: `true.type()`,
			exp:  []any{"boolean"},
		},
		{
			name: "test_6",
			json: js(`null`),
			path: `(123).type()`,
			exp:  []any{"number"},
		},
		{
			name: "test_7",
			json: js(`null`),
			path: `"123".type()`,
			exp:  []any{"string"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgQueryAbsFloorType(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L275-L280
	for _, tc := range []queryTestCase{
		{
			name: "test_1",
			json: js(`{"a": 2}`),
			path: `($.a - 5).abs() + 10`,
			exp:  []any{float64(13)},
		},
		{
			name: "test_2",
			json: js(`{"a": 2.5}`),
			path: `-($.a * $.a).floor() % 4.3`,
			exp:  []any{float64(-1.7000000000000002)}, // pg:1.7
		},
		{
			name: "test_3",
			json: js(`[1, 2, 3]`),
			path: `($[*] > 2) ? (@ == true)`,
			exp:  []any{true},
		},
		{
			name: "test_4",
			json: js(`[1, 2, 3]`),
			path: `($[*] > 3).type()`,
			exp:  []any{"boolean"},
		},
		{
			name: "test_5",
			json: js(`[1, 2, 3]`),
			path: `($[*].a > 3).type()`,
			exp:  []any{"boolean"},
		},
		{
			name: "test_6",
			json: js(`[1, 2, 3]`),
			path: `strict ($[*].a > 3).type()`,
			exp:  []any{"null"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgQuerySizeMethod(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L282-L284
	for _, tc := range []queryTestCase{
		{
			name: "test_1",
			json: js(`[1,null,true,"11",[],[1],[1,2,3],{},{"a":1,"b":2}]`),
			path: `strict $[*].size()`,
			err:  "exec: jsonpath item method .size() can only be applied to an array",
		},
		{
			name: "test_2",
			json: js(`[1,null,true,"11",[],[1],[1,2,3],{},{"a":1,"b":2}]`),
			path: `strict $[*].size()`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_3",
			json: js(`[1,null,true,"11",[],[1],[1,2,3],{},{"a":1,"b":2}]`),
			path: `lax $[*].size()`,
			exp: []any{
				int64(1), int64(1), int64(1), int64(1), int64(0),
				int64(1), int64(3), int64(1), int64(1),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgQueryMethodChain(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L286-L290
	for _, tc := range []queryTestCase{
		{
			name: "test_1",
			json: js(`[0, 1, -2, -3.4, 5.6]`),
			path: `$[*].abs()`,
			exp:  []any{float64(0), float64(1), float64(2), float64(3.4), float64(5.6)},
		},
		{
			name: "test_2",
			json: js(`[0, 1, -2, -3.4, 5.6]`),
			path: `$[*].floor()`,
			exp:  []any{float64(0), float64(1), float64(-2), float64(-4), float64(5)},
		},
		{
			name: "test_3",
			json: js(`[0, 1, -2, -3.4, 5.6]`),
			path: `$[*].ceiling()`,
			exp:  []any{float64(0), float64(1), float64(-2), float64(-3), float64(6)},
		},
		{
			name: "test_4",
			json: js(`[0, 1, -2, -3.4, 5.6]`),
			path: `$[*].ceiling().abs()`,
			exp:  []any{float64(0), float64(1), float64(2), float64(3), float64(6)},
		},
		{
			name: "test_5",
			json: js(`[0, 1, -2, -3.4, 5.6]`),
			path: `$[*].ceiling().abs().type()`,
			exp:  []any{"number", "number", "number", "number", "number"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func offset(a, b any) int64 {
	x := addrOf(a)
	y := addrOf(b)
	if x > y {
		return int64(x - y)
	}
	return int64(y - x)
}

func TestPgQueryKeyValue(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// Go can have different array offsets when executing stuff in parallel,
	// so create the data here so we can calculate the correct IDs in tests 5
	// and 7 below.
	array, ok := js(`[{"a": 1, "b": [1, 2]}, {"c": {"a": "bbb"}}]`).([]any)
	r.True(ok)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L292-L299
	for _, tc := range []queryTestCase{
		{
			name: "test_1",
			json: js(`[{},1]`),
			path: `$[*].keyvalue()`,
			err:  "exec: jsonpath item method .keyvalue() can only be applied to an object",
		},
		{
			name: "test_2",
			json: js(`[{},1]`),
			path: `$[*].keyvalue()`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_3",
			json: js(`{}`),
			path: `$.keyvalue()`,
			exp:  []any{},
		},
		{
			name: "test_4",
			json: js(`{"a": 1, "b": [1, 2], "c": {"a": "bbb"}}`),
			path: `$.keyvalue()`,
			exp: []any{
				map[string]any{"id": int64(0), "key": "a", "value": float64(1)},
				map[string]any{"id": int64(0), "key": "b", "value": []any{float64(1), float64(2)}},
				map[string]any{"id": int64(0), "key": "c", "value": map[string]any{"a": "bbb"}},
			},
		},
		{
			name: "test_5",
			json: array,
			path: `$[*].keyvalue()`,
			// pg: IDs vary because jsonb binary layout is more consistent than Go slices.
			exp: []any{
				map[string]any{"id": offset(array[0], array), "key": "a", "value": float64(1)},
				map[string]any{"id": offset(array[0], array), "key": "b", "value": []any{float64(1), float64(2)}},
				map[string]any{"id": offset(array[1], array), "key": "c", "value": map[string]any{"a": "bbb"}},
			},
		},
		{
			name: "test_6",
			json: array,
			path: `strict $.keyvalue()`,
			err:  "exec: jsonpath item method .keyvalue() can only be applied to an object",
		},
		{
			name: "test_7",
			json: array,
			path: `lax $.keyvalue()`,
			// pg: IDs vary because jsonb binary layout is more consistent than Go slices.
			exp: []any{
				map[string]any{"id": offset(array[0], array), "key": "a", "value": float64(1)},
				map[string]any{"id": offset(array[0], array), "key": "b", "value": []any{float64(1), float64(2)}},
				map[string]any{"id": offset(array[1], array), "key": "c", "value": map[string]any{"a": "bbb"}},
			},
		},
		{
			name: "test_8",
			json: array,
			path: `strict $.keyvalue().a`,
			err:  "exec: jsonpath item method .keyvalue() can only be applied to an object",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgAtQuestionKeyValue(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L300-L301
	for _, tc := range []existsTestCase{
		{
			name: "test_1",
			json: js(`{"a": 1, "b": [1, 2]}`),
			path: `lax $.keyvalue()`,
			exp:  true,
		},
		{
			name: "test_2",
			json: js(`{"a": 1, "b": [1, 2]}`),
			path: `lax $.keyvalue().key`,
			exp:  true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.runAtQuestion(a, r)
		})
	}
}

func TestPgQueryDoubleMethod(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L303-L321
	for _, tc := range []queryTestCase{
		{
			name: "test_1",
			json: js(`null`),
			path: `$.double()`,
			err:  "exec: jsonpath item method .double() can only be applied to a string or numeric value",
		},
		{
			name: "test_2",
			json: js(`true`),
			path: `$.double()`,
			err:  "exec: jsonpath item method .double() can only be applied to a string or numeric value",
		},
		{
			name: "test_3",
			json: js(`null`),
			path: `$.double()`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_4",
			json: js(`true`),
			path: `$.double()`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_5",
			json: js(`[]`),
			path: `$.double()`,
			exp:  []any{},
		},
		{
			name: "test_6",
			json: js(`[]`),
			path: `strict $.double()`,
			err:  "exec: jsonpath item method .double() can only be applied to a string or numeric value",
		},
		{
			name: "test_7",
			json: js(`{}`),
			path: `$.double()`,
			err:  "exec: jsonpath item method .double() can only be applied to a string or numeric value",
		},
		{
			name: "test_8",
			json: js(`[]`),
			path: `strict $.double()`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_9",
			json: js(`{}`),
			path: `$.double()`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_10",
			json: js(`1.23`),
			path: `$.double()`,
			exp:  []any{float64(1.23)},
		},
		{
			name: "test_11",
			json: js(`"1.23"`),
			path: `$.double()`,
			exp:  []any{float64(1.23)},
		},
		{
			name: "test_12",
			json: js(`"1.23aaa"`),
			path: `$.double()`,
			err:  `exec: argument "1.23aaa" of jsonpath item method .double() is invalid for type double precision`,
		},
		// Go cannot parse 1e1000 into a float because it's too big.
		// Postgres JSONB accepts arbitrary numeric sizes.
		// {
		// 	name: "test_13",
		// 	json: js(`1e1000`),
		// 	path: `$.double()`,
		// 	err:  `exec: argument "10000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000" of jsonpath item method .double() is invalid for type double precision`,
		// },
		{
			name: "test_14",
			json: js(`"nan"`),
			path: `$.double()`,
			err:  "exec: NaN or Infinity is not allowed for jsonpath item method .double()",
		},
		{
			name: "test_15",
			json: js(`"NaN"`),
			path: `$.double()`,
			err:  "exec: NaN or Infinity is not allowed for jsonpath item method .double()",
		},
		{
			name: "test_16",
			json: js(`"inf"`),
			path: `$.double()`,
			err:  "exec: NaN or Infinity is not allowed for jsonpath item method .double()",
		},
		{
			name: "test_17",
			json: js(`"-inf"`),
			path: `$.double()`,
			err:  "exec: NaN or Infinity is not allowed for jsonpath item method .double()",
		},
		{
			name: "test_18",
			json: js(`"inf"`),
			path: `$.double()`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_19",
			json: js(`"-inf"`),
			path: `$.double()`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgQueryAbsFloorCeilErr(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L323-L328
	for _, tc := range []queryTestCase{
		{
			name: "test_1",
			json: js(`{}`),
			path: `$.abs()`,
			err:  "exec: jsonpath item method .abs() can only be applied to a numeric value",
		},
		{
			name: "test_2",
			json: js(`true`),
			path: `$.floor()`,
			err:  "exec: jsonpath item method .floor() can only be applied to a numeric value",
		},
		{
			name: "test_3",
			json: js(`"1.2"`),
			path: `$.ceiling()`,
			err:  "exec: jsonpath item method .ceiling() can only be applied to a numeric value",
		},
		{
			name: "test_4",
			json: js(`{}`),
			path: `$.abs()`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_5",
			json: js(`true`),
			path: `$.floor()`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_6",
			json: js(`"1.2"`),
			path: `$.ceiling()`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgQueryStartsWith(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L330-L337
	for _, tc := range []queryTestCase{
		{
			name: "test_1",
			json: js(`["", "a", "abc", "abcabc"]`),
			path: `$[*] ? (@ starts with "abc")`,
			exp:  []any{"abc", "abcabc"},
		},
		{
			name: "test_2",
			json: js(`["", "a", "abc", "abcabc"]`),
			path: `strict $ ? (@[*] starts with "abc")`,
			exp:  []any{[]any{"", "a", "abc", "abcabc"}},
		},
		{
			name: "test_3",
			json: js(`["", "a", "abd", "abdabc"]`),
			path: `strict $ ? (@[*] starts with "abc")`,
			exp:  []any{},
		},
		{
			name: "test_4",
			json: js(`["abc", "abcabc", null, 1]`),
			path: `strict $ ? (@[*] starts with "abc")`,
			exp:  []any{},
		},
		{
			name: "test_5",
			json: js(`["abc", "abcabc", null, 1]`),
			path: `strict $ ? ((@[*] starts with "abc") is unknown)`,
			exp:  []any{[]any{"abc", "abcabc", nil, float64(1)}},
		},
		{
			name: "test_6",
			json: js(`[[null, 1, "abc", "abcabc"]]`),
			path: `lax $ ? (@[*] starts with "abc")`,
			exp:  []any{[]any{nil, float64(1), "abc", "abcabc"}},
		},
		{
			name: "test_7",
			json: js(`[[null, 1, "abd", "abdabc"]]`),
			path: `lax $ ? ((@[*] starts with "abc") is unknown)`,
			exp:  []any{[]any{nil, float64(1), "abd", "abdabc"}},
		},
		{
			name: "test_8",
			json: js(`[null, 1, "abd", "abdabc"]`),
			path: `lax $[*] ? ((@ starts with "abc") is unknown)`,
			exp:  []any{nil, float64(1)},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgQueryLikeRegex(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L339-L348
	// pg: Using \t instead of \b, because \b is word boundary only in Go, while
	// in Postgres it's bell. Using \t gets the original intent of the tests.
	for _, tc := range []queryTestCase{
		{
			name: "test_1",
			json: js(`[null, 1, "abc", "abd", "aBdC", "abdacb", "babc", "adc\nabc", "ab\nadc"]`),
			path: `lax $[*] ? (@ like_regex "^ab.*c")`,
			exp:  []any{"abc", "abdacb"},
		},
		{
			name: "test_2",
			json: js(`[null, 1, "abc", "abd", "aBdC", "abdacb", "babc", "adc\nabc", "ab\nadc"]`),
			path: `lax $[*] ? (@ like_regex "^ab.*c" flag "i")`,
			exp:  []any{"abc", "aBdC", "abdacb"},
		},
		{
			name: "test_3",
			json: js(`[null, 1, "abc", "abd", "aBdC", "abdacb", "babc", "adc\nabc", "ab\nadc"]`),
			path: `lax $[*] ? (@ like_regex "^ab.*c" flag "m")`,
			exp:  []any{"abc", "abdacb", "adc\nabc"},
		},
		{
			name: "test_4",
			json: js(`[null, 1, "abc", "abd", "aBdC", "abdacb", "babc", "adc\nabc", "ab\nadc"]`),
			path: `lax $[*] ? (@ like_regex "^ab.*c" flag "s")`,
			exp:  []any{"abc", "abdacb", "ab\nadc"},
		},
		{
			name: "test_5",
			json: js(`[null, 1, "a\t", "a\\t", "^a\\t$"]`),
			path: `lax $[*] ? (@ like_regex "a\\t" flag "q")`,
			exp:  []any{"a\\t", "^a\\t$"},
		},
		{
			name: "test_6",
			json: js(`[null, 1, "a\t", "a\\t", "^a\\t$"]`),
			path: `lax $[*] ? (@ like_regex "a\\t" flag "")`,
			exp:  []any{"a\t"},
		},
		{
			name: "test_7",
			json: js(`[null, 1, "a\t", "a\\t", "^a\\t$"]`),
			path: `lax $[*] ? (@ like_regex "^a\\t$" flag "q")`,
			exp:  []any{"^a\\t$"},
		},
		{
			name: "test_8",
			json: js(`[null, 1, "a\t", "a\\t", "^a\\t$"]`),
			path: `lax $[*] ? (@ like_regex "^a\\T$" flag "q")`,
			exp:  []any{},
		},
		{
			name: "test_9",
			json: js(`[null, 1, "a\t", "a\\t", "^a\\t$"]`),
			path: `lax $[*] ? (@ like_regex "^a\\T$" flag "iq")`,
			exp:  []any{"^a\\t$"},
		},
		{
			name: "test_10",
			json: js(`[null, 1, "a\t", "a\\t", "^a\\t$"]`),
			path: `lax $[*] ? (@ like_regex "^a\\t$" flag "")`,
			exp:  []any{"a\t"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgQueryDateTimeErr(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L350-L358
	for _, tc := range []queryTestCase{
		{
			name: "test_1",
			json: js(`null`),
			path: `$.datetime()`,
			err:  "exec: jsonpath item method .datetime() can only be applied to a string",
		},
		{
			name: "test_2",
			json: js(`true`),
			path: `$.datetime()`,
			err:  "exec: jsonpath item method .datetime() can only be applied to a string",
		},
		{
			name: "test_3",
			json: js(`1`),
			path: `$.datetime()`,
			err:  "exec: jsonpath item method .datetime() can only be applied to a string",
		},
		{
			name: "test_4",
			json: js(`[]`),
			path: `$.datetime()`,
			exp:  []any{},
		},
		{
			name: "test_5",
			json: js(`[]`),
			path: `strict $.datetime()`,
			err:  "exec: jsonpath item method .datetime() can only be applied to a string",
		},
		{
			name: "test_6",
			json: js(`{}`),
			path: `$.datetime()`,
			err:  "exec: jsonpath item method .datetime() can only be applied to a string",
		},
		{
			name: "test_7",
			json: js(`"bogus"`),
			path: `$.datetime()`,
			err:  `exec: datetime format is not recognized: "bogus"`,
		},
		{
			name: "test_8",
			json: js(`"12:34"`),
			path: `$.datetime("aaa")`,
			err:  `exec: .datetime(template) is not yet supported`,
			// err:  `exec: invalid datetime format separator: "a"`,
		},
		{
			name: "test_9",
			json: js(`"aaaa"`),
			path: `$.datetime("HH24")`,
			err:  `exec: .datetime(template) is not yet supported`,
			// err:  `exec: invalid value "aa" for "HH24"`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgQueryDateTimeAtQuestion(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L360
	for _, tc := range []existsTestCase{
		{
			name: "test_1",
			json: js(`"10-03-2017"`),
			path: `$.datetime("dd-mm-yyyy")`,
			err:  `exec: .datetime(template) is not yet supported`,
			// exp:  true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.runAtQuestion(a, r)
		})
	}
}

func pt(ts string) types.DateTime {
	val, ok := types.ParseTime(ts, -1)
	if !ok {
		panic("Failed to parse " + ts)
	}
	return val
}

func TestPgQueryDateTimeFormat(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L361-L373
	for _, tc := range []queryTestCase{
		{
			name: "test_1",
			json: js(`"10-03-2017"`),
			path: `$.datetime("dd-mm-yyyy")`,
			// exp:  []any{pt("2017-03-10")},
		},
		{
			name: "test_2",
			json: js(`"10-03-2017"`),
			path: `$.datetime("dd-mm-yyyy").type()`,
			// exp:  []any{"date"},
		},
		{
			name: "test_3",
			json: js(`"10-03-2017 12:34"`),
			path: `$.datetime("dd-mm-yyyy")`,
			// err:"exec: trailing characters remain in input string after datetime format",
		},
		{
			name: "test_4",
			json: js(`"10-03-2017 12:34"`),
			path: `$.datetime("dd-mm-yyyy").type()`,
			// err:"exec: trailing characters remain in input string after datetime format",
		},
		{
			name: "test_5",
			json: js(`"10-03-2017 12:34"`),
			path: `       $.datetime("dd-mm-yyyy HH24:MI").type()`,
			// exp:  []any{"timestamp without time zone"},
		},
		{
			name: "test_6",
			json: js(`"10-03-2017 12:34 +05:20"`),
			path: `$.datetime("dd-mm-yyyy HH24:MI TZH:TZM").type()`,
			// exp:  []any{"timestamp with time zone"},
		},
		{
			name: "test_7",
			json: js(`"12:34:56"`),
			path: `$.datetime("HH24:MI:SS").type()`,
			// exp:  []any{"time without time zone"},
		},
		{
			name: "test_8",
			json: js(`"12:34:56 +05:20"`),
			path: `$.datetime("HH24:MI:SS TZH:TZM").type()`,
			// exp:  []any{"time with time zone"},
		},
		{
			name: "test_9",
			json: js(`"10-03-2017T12:34:56"`),
			path: `$.datetime("dd-mm-yyyy\"T\"HH24:MI:SS")`,
			// exp:  []any{pt("2017-03-10T12:34:56")},
		},
		{
			name: "test_10",
			json: js(`"10-03-2017t12:34:56"`),
			path: `$.datetime("dd-mm-yyyy\"T\"HH24:MI:SS")`,
			// err:`exec: unmatched format character "T"`,
		},
		{
			name: "test_11",
			json: js(`"10-03-2017 12:34:56"`),
			path: `$.datetime("dd-mm-yyyy\"T\"HH24:MI:SS")`,
			// err:`exec: unmatched format character "T"`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.err = `exec: .datetime(template) is not yet supported`
			tc.run(a, r)
		})
	}
}

func TestPgQueryBigInt(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L375-L405
	for _, tc := range []queryTestCase{
		{
			name: "test_1",
			json: js(`null`),
			path: `$.bigint()`,
			err:  `exec: jsonpath item method .bigint() can only be applied to a string or numeric value`,
		},
		{
			name: "test_2",
			json: js(`true`),
			path: `$.bigint()`,
			err:  `exec: jsonpath item method .bigint() can only be applied to a string or numeric value`,
		},
		{
			name: "test_3",
			json: js(`null`),
			path: `$.bigint()`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_4",
			json: js(`true`),
			path: `$.bigint()`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_5",
			json: js(`[]`),
			path: `$.bigint()`,
			exp:  []any{},
		},
		{
			name: "test_6",
			json: js(`[]`),
			path: `strict $.bigint()`,
			err:  `exec: jsonpath item method .bigint() can only be applied to a string or numeric value`,
		},
		{
			name: "test_7",
			json: js(`{}`),
			path: `$.bigint()`,
			err:  `exec: jsonpath item method .bigint() can only be applied to a string or numeric value`,
		},
		{
			name: "test_8",
			json: js(`[]`),
			path: `strict $.bigint()`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_9",
			json: js(`{}`),
			path: `$.bigint()`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "`test_10`",
			json: js(`"1.23"`),
			path: `$.bigint()`,
			err:  `exec: argument "1.23" of jsonpath item method .bigint() is invalid for type bigint`,
		},
		{
			name: "test_11",
			json: js(`"1.23aaa"`),
			path: `$.bigint()`,
			err:  `exec: argument "1.23aaa" of jsonpath item method .bigint() is invalid for type bigint`,
		},
		// Go cannot parse 1e1000 into a float because it's too big.
		// Postgres JSONB accepts arbitrary numeric sizes.
		// {
		// 	name: "test_12",
		// 	json: js(`1e1000`),
		// 	path: `$.bigint()`,
		// 	err:  `exec: argument "10000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000" of jsonpath item method .bigint() is invalid for type bigint`,
		// },
		{
			name: "test_13",
			json: js(`"nan"`),
			path: `$.bigint()`,
			err:  `exec: argument "nan" of jsonpath item method .bigint() is invalid for type bigint`,
		},
		{
			name: "test_14",
			json: js(`"NaN"`),
			path: `$.bigint()`,
			err:  `exec: argument "NaN" of jsonpath item method .bigint() is invalid for type bigint`,
		},
		{
			name: "test_15",
			json: js(`"inf"`),
			path: `$.bigint()`,
			err:  `exec: argument "inf" of jsonpath item method .bigint() is invalid for type bigint`,
		},
		{
			name: "test_16",
			json: js(`"-inf"`),
			path: `$.bigint()`,
			err:  `exec: argument "-inf" of jsonpath item method .bigint() is invalid for type bigint`,
		},
		{
			name: "test_17",
			json: js(`"inf"`),
			path: `$.bigint()`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_18",
			json: js(`"-inf"`),
			path: `$.bigint()`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_19",
			json: js(`123`),
			path: `$.bigint()`,
			exp:  []any{int64(123)},
		},
		{
			name: "test_20",
			json: js(`"123"`),
			path: `$.bigint()`,
			exp:  []any{int64(123)},
		},
		{
			name: "test_21",
			json: js(`1.23`),
			path: `$.bigint()`,
			exp:  []any{int64(1)},
		},
		{
			name: "test_22",
			json: js(`1.83`),
			path: `$.bigint()`,
			exp:  []any{int64(2)},
		},
		{
			name: "test_23",
			json: js(`1234567890123`),
			path: `$.bigint()`,
			exp:  []any{int64(1234567890123)},
		},
		{
			name: "test_24",
			json: js(`"1234567890123"`),
			path: `$.bigint()`,
			exp:  []any{int64(1234567890123)},
		},
		{
			name: "test_25",
			json: js(`12345678901234567890`),
			path: `$.bigint()`,
			// pg: shows `"12345678901234567890"` in the error
			err: `exec: argument "1.2345678901234567e+19" of jsonpath item method .bigint() is invalid for type bigint`,
		},
		{
			name: "test_26",
			json: js(`"12345678901234567890"`),
			path: `$.bigint()`,
			err:  `exec: argument "12345678901234567890" of jsonpath item method .bigint() is invalid for type bigint`,
		},
		{
			name: "test_27",
			json: js(`"+123"`),
			path: `$.bigint()`,
			exp:  []any{int64(123)},
		},
		{
			name: "test_28",
			json: js(`-123`),
			path: `$.bigint()`,
			exp:  []any{int64(-123)},
		},
		{
			name: "test_29",
			json: js(`"-123"`),
			path: `$.bigint()`,
			exp:  []any{int64(-123)},
		},
		{
			name: "test_30",
			json: js(`123`),
			path: `$.bigint() * 2`,
			exp:  []any{int64(246)},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgQueryBooleanMethod(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L407-L447
	for _, tc := range []queryTestCase{
		{
			name: "test_1",
			json: js(`null`),
			path: `$.boolean()`,
			err:  `exec: jsonpath item method .boolean() can only be applied to a bool, string, or numeric value`,
		},
		{
			name: "test_2",
			json: js(`null`),
			path: `$.boolean()`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_3",
			json: js(`[]`),
			path: `$.boolean()`,
			exp:  []any{},
		},
		{
			name: "test_4",
			json: js(`[]`),
			path: `strict $.boolean()`,
			err:  `exec: jsonpath item method .boolean() can only be applied to a bool, string, or numeric value`,
		},
		{
			name: "test_5",
			json: js(`{}`),
			path: `$.boolean()`,
			err:  `exec: jsonpath item method .boolean() can only be applied to a bool, string, or numeric value`,
		},
		{
			name: "test_6",
			json: js(`[]`),
			path: `strict $.boolean()`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_7",
			json: js(`{}`),
			path: `$.boolean()`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_8",
			json: js(`1.23`),
			path: `$.boolean()`,
			err:  `exec: argument "1.23" of jsonpath item method .boolean() is invalid for type boolean`,
		},
		{
			name: "test_9",
			json: js(`"1.23"`),
			path: `$.boolean()`,
			err:  `exec: argument "1.23" of jsonpath item method .boolean() is invalid for type boolean`,
		},
		{
			name: "test_10",
			json: js(`"1.23aaa"`),
			path: `$.boolean()`,
			err:  `exec: argument "1.23aaa" of jsonpath item method .boolean() is invalid for type boolean`,
		},
		// Go cannot parse 1e1000 into a float because it's too big.
		// Postgres JSONB accepts arbitrary numeric sizes.
		// {
		// 	name: "test_11",
		// 	json: js(`1e1000`),
		// 	path: `$.boolean()`,
		// 	err:  `exec: argument "10000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000" of jsonpath item method .boolean() is invalid for type boolean`,
		// },
		{
			name: "test_12",
			json: js(`"nan"`),
			path: `$.boolean()`,
			err:  `exec: argument "nan" of jsonpath item method .boolean() is invalid for type boolean`,
		},
		{
			name: "test_13",
			json: js(`"NaN"`),
			path: `$.boolean()`,
			err:  `exec: argument "NaN" of jsonpath item method .boolean() is invalid for type boolean`,
		},
		{
			name: "test_14",
			json: js(`"inf"`),
			path: `$.boolean()`,
			err:  `exec: argument "inf" of jsonpath item method .boolean() is invalid for type boolean`,
		},
		{
			name: "test_15",
			json: js(`"-inf"`),
			path: `$.boolean()`,
			err:  `exec: argument "-inf" of jsonpath item method .boolean() is invalid for type boolean`,
		},
		{
			name: "test_16",
			json: js(`"inf"`),
			path: `$.boolean()`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_17",
			json: js(`"-inf"`),
			path: `$.boolean()`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_18",
			json: js(`"100"`),
			path: `$.boolean()`,
			err:  `exec: argument "100" of jsonpath item method .boolean() is invalid for type boolean`,
		},
		{
			name: "test_19",
			json: js(`true`),
			path: `$.boolean()`,
			exp:  []any{true},
		},
		{
			name: "test_20",
			json: js(`false`),
			path: `$.boolean()`,
			exp:  []any{false},
		},
		{
			name: "test_21",
			json: js(`1`),
			path: `$.boolean()`,
			exp:  []any{true},
		},
		{
			name: "test_22",
			json: js(`0`),
			path: `$.boolean()`,
			exp:  []any{false},
		},
		{
			name: "test_23",
			json: js(`-1`),
			path: `$.boolean()`,
			exp:  []any{true},
		},
		{
			name: "test_24",
			json: js(`100`),
			path: `$.boolean()`,
			exp:  []any{true},
		},
		{
			name: "test_25",
			json: js(`"1"`),
			path: `$.boolean()`,
			exp:  []any{true},
		},
		{
			name: "test_26",
			json: js(`"0"`),
			path: `$.boolean()`,
			exp:  []any{false},
		},
		{
			name: "test_27",
			json: js(`"true"`),
			path: `$.boolean()`,
			exp:  []any{true},
		},
		{
			name: "test_28",
			json: js(`"false"`),
			path: `$.boolean()`,
			exp:  []any{false},
		},
		{
			name: "test_29",
			json: js(`"TRUE"`),
			path: `$.boolean()`,
			exp:  []any{true},
		},
		{
			name: "test_30",
			json: js(`"FALSE"`),
			path: `$.boolean()`,
			exp:  []any{false},
		},
		{
			name: "test_31",
			json: js(`"yes"`),
			path: `$.boolean()`,
			exp:  []any{true},
		},
		{
			name: "test_32",
			json: js(`"NO"`),
			path: `$.boolean()`,
			exp:  []any{false},
		},
		{
			name: "test_33",
			json: js(`"T"`),
			path: `$.boolean()`,
			exp:  []any{true},
		},
		{
			name: "test_34",
			json: js(`"f"`),
			path: `$.boolean()`,
			exp:  []any{false},
		},
		{
			name: "test_35",
			json: js(`"y"`),
			path: `$.boolean()`,
			exp:  []any{true},
		},
		{
			name: "test_36",
			json: js(`"N"`),
			path: `$.boolean()`,
			exp:  []any{false},
		},
		{
			name: "test_37",
			json: js(`true`),
			path: `$.boolean().type()`,
			exp:  []any{"boolean"},
		},
		{
			name: "test_38",
			json: js(`123`),
			path: `$.boolean().type()`,
			exp:  []any{"boolean"},
		},
		{
			name: "test_39",
			json: js(`"Yes"`),
			path: `$.boolean().type()`,
			exp:  []any{"boolean"},
		},
		// pg: tests jsonb_path_query_array but our Query() always returns a
		// slice.
		{
			name: "test_40",
			json: js(`[1, "yes", false]`),
			path: `$[*].boolean()`,
			exp:  []any{true, true, false},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgQueryDateMethod(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L449-L466
	for _, tc := range []queryTestCase{
		{
			name: "test_1",
			json: js(`null`),
			path: `$.date()`,
			err:  `exec: jsonpath item method .date() can only be applied to a string`,
		},
		{
			name: "test_2",
			json: js(`true`),
			path: `$.date()`,
			err:  `exec: jsonpath item method .date() can only be applied to a string`,
		},
		{
			name: "test_3",
			json: js(`1`),
			path: `$.date()`,
			err:  `exec: jsonpath item method .date() can only be applied to a string`,
		},
		{
			name: "test_4",
			json: js(`[]`),
			path: `$.date()`,
			exp:  []any{},
		},
		{
			name: "test_5",
			json: js(`[]`),
			path: `strict $.date()`,
			err:  `exec: jsonpath item method .date() can only be applied to a string`,
		},
		{
			name: "test_6",
			json: js(`{}`),
			path: `$.date()`,
			err:  `exec: jsonpath item method .date() can only be applied to a string`,
		},
		{
			name: "test_7",
			json: js(`"bogus"`),
			path: `$.date()`,
			err:  `exec: date format is not recognized: "bogus"`,
		},
		// Test 8 in TestPgQueryDateAtQuestion below
		{
			name: "test_9",
			json: js(`"2023-08-15"`),
			path: `$.date()`,
			exp:  []any{pt("2023-08-15")},
		},
		{
			name: "test_10",
			json: js(`"2023-08-15"`),
			path: `$.date().type()`,
			exp:  []any{"date"},
		},
		{
			name: "test_11",
			json: js(`"12:34:56"`),
			path: `$.date()`,
			err:  `exec: date format is not recognized: "12:34:56"`,
		},
		{
			name: "test_12",
			json: js(`"12:34:56 +05:30"`),
			path: `$.date()`,
			err:  `exec: date format is not recognized: "12:34:56 +05:30"`,
		},
		{
			name: "test_13",
			json: js(`"2023-08-15 12:34:56"`),
			path: `$.date()`,
			exp:  []any{pt("2023-08-15")},
		},
		{
			name: "test_14",
			json: js(`"2023-08-15 12:34:56+05:30"`), // pg: 2023-08-15 12:34:56 +05:30
			path: `$.date()`,
			err:  `exec: cannot convert value from timestamptz to date without time zone usage.` + hint,
		},
		{
			name: "test_15",
			json: js(`"2023-08-15 12:34:56+05:30"`), // pg: 2023-08-15 12:34:56 +05:30
			path: `$.date()`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("2023-08-15")}, // should work
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgQueryDateAtQuestion(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L458
	for _, tc := range []existsTestCase{
		{
			name: "test_8",
			json: js(`"2023-08-15"`),
			path: `$.date()`,
			exp:  true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.runAtQuestion(a, r)
		})
	}
}

func TestPgQueryDateMethodSyntaxError(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L468
	t.Run("test_16", func(t *testing.T) {
		t.Parallel()
		path, err := parser.Parse("$.date(2)")
		r.EqualError(err, `parser: syntax error at 1:9`)
		r.ErrorIs(err, parser.ErrParse)
		a.Nil(path)
	})
}

//nolint:maintidx
func TestPgQueryDecimalMethod(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L470-L514
	for _, tc := range []queryTestCase{
		{
			name: "test_1",
			json: js(`null`),
			path: `$.decimal()`,
			err:  `exec: jsonpath item method .decimal() can only be applied to a string or numeric value`,
		},
		{
			name: "test_2",
			json: js(`true`),
			path: `$.decimal()`,
			err:  `exec: jsonpath item method .decimal() can only be applied to a string or numeric value`,
		},
		{
			name: "test_3",
			json: js(`null`),
			path: `$.decimal()`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_4",
			json: js(`true`),
			path: `$.decimal()`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_5",
			json: js(`[]`),
			path: `$.decimal()`,
			exp:  []any{},
		},
		{
			name: "test_6",
			json: js(`[]`),
			path: `strict $.decimal()`,
			err:  `exec: jsonpath item method .decimal() can only be applied to a string or numeric value`,
		},
		{
			name: "test_7",
			json: js(`{}`),
			path: `$.decimal()`,
			err:  `exec: jsonpath item method .decimal() can only be applied to a string or numeric value`,
		},
		{
			name: "test_8",
			json: js(`[]`),
			path: `strict $.decimal()`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_9",
			json: js(`{}`),
			path: `$.decimal()`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_10",
			json: js(`1.23`),
			path: `$.decimal()`,
			exp:  []any{float64(1.23)},
		},
		{
			name: "test_11",
			json: js(`"1.23"`),
			path: `$.decimal()`,
			exp:  []any{float64(1.23)},
		},
		{
			name: "test_12",
			json: js(`"1.23aaa"`),
			path: `$.decimal()`,
			err:  `exec: argument "1.23aaa" of jsonpath item method .decimal() is invalid for type numeric`,
		},
		// Go cannot parse 1e1000 into a float because it's too big.
		// Postgres JSONB accepts arbitrary numeric sizes.
		// {
		// 	name: "test_13",
		// 	json: js(`1e1000`),
		// 	path: `$.decimal()`,
		// 	exp:  []any{"10000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"},
		// },
		{
			name: "test_14",
			json: js(`"nan"`),
			path: `$.decimal()`,
			err:  `exec: NaN or Infinity is not allowed for jsonpath item method .decimal()`,
		},
		{
			name: "test_15",
			json: js(`"NaN"`),
			path: `$.decimal()`,
			err:  `exec: NaN or Infinity is not allowed for jsonpath item method .decimal()`,
		},
		{
			name: "test_16",
			json: js(`"inf"`),
			path: `$.decimal()`,
			err:  `exec: NaN or Infinity is not allowed for jsonpath item method .decimal()`,
		},
		{
			name: "test_17",
			json: js(`"-inf"`),
			path: `$.decimal()`,
			err:  `exec: NaN or Infinity is not allowed for jsonpath item method .decimal()`,
		},
		{
			name: "test_18",
			json: js(`"inf"`),
			path: `$.decimal()`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_19",
			json: js(`"-inf"`),
			path: `$.decimal()`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_20",
			json: js(`123`),
			path: `$.decimal()`,
			exp:  []any{float64(123)},
		},
		{
			name: "test_21",
			json: js(`"123"`),
			path: `$.decimal()`,
			exp:  []any{float64(123)},
		},
		{
			name: "test_22",
			json: js(`12345678901234567890`),
			path: `$.decimal()`,
			exp:  []any{float64(12345678901234567890)},
		},
		{
			name: "test_23",
			json: js(`"12345678901234567890"`),
			path: `$.decimal()`,
			exp:  []any{float64(12345678901234567890)},
		},
		{
			name: "test_24",
			json: js(`"+12.3"`),
			path: `$.decimal()`,
			exp:  []any{float64(12.3)},
		},
		{
			name: "test_25",
			json: js(`-12.3`),
			path: `$.decimal()`,
			exp:  []any{float64(-12.3)},
		},
		{
			name: "test_26",
			json: js(`"-12.3"`),
			path: `$.decimal()`,
			exp:  []any{float64(-12.3)},
		},
		{
			name: "test_27",
			json: js(`12.3`),
			path: `$.decimal() * 2`,
			exp:  []any{float64(24.6)},
		},
		{
			name: "test_28",
			json: js(`12345.678`),
			path: `$.decimal(6, 1)`,
			exp:  []any{float64(12345.7)},
		},
		{
			name: "test_29",
			json: js(`12345.678`),
			path: `$.decimal(6, 2)`,
			err:  `exec: argument "12345.678" of jsonpath item method .decimal() is invalid for type numeric`,
		},
		{
			name: "test_30",
			json: js(`1234.5678`),
			path: `$.decimal(6, 2)`,
			exp:  []any{float64(1234.57)},
		},
		{
			name: "test_31",
			json: js(`12345.678`),
			path: `$.decimal(4, 6)`,
			err:  `exec: argument "12345.678" of jsonpath item method .decimal() is invalid for type numeric`,
		},
		{
			name: "test_32",
			json: js(`12345.678`),
			path: `$.decimal(0, 6)`,
			err:  `exec: NUMERIC precision 0 must be between 1 and 1000`,
		},
		{
			name: "test_33",
			json: js(`12345.678`),
			path: `$.decimal(1001, 6)`,
			err:  `exec: NUMERIC precision 1001 must be between 1 and 1000`,
		},
		{
			name: "test_34",
			json: js(`1234.5678`),
			path: `$.decimal(+6, +2)`,
			exp:  []any{float64(1234.57)},
		},
		{
			name: "test_35",
			json: js(`1234.5678`),
			path: `$.decimal(+6, -2)`,
			exp:  []any{float64(1200)},
		},
		{
			name: "test_36",
			json: js(`1234.5678`),
			path: `$.decimal(-6, +2)`,
			err:  `exec: NUMERIC precision -6 must be between 1 and 1000`,
		},
		{
			name: "test_37",
			json: js(`1234.5678`),
			path: `$.decimal(6, -1001)`,
			err:  `exec: NUMERIC scale -1001 must be between -1000 and 1000`,
		},
		{
			name: "test_38",
			json: js(`1234.5678`),
			path: `$.decimal(6, 1001)`,
			err:  `exec: NUMERIC scale 1001 must be between -1000 and 1000`,
		},
		{
			name: "test_39",
			json: js(`-1234.5678`),
			path: `$.decimal(+6, -2)`,
			exp:  []any{float64(-1200)},
		},
		{
			name: "test_40",
			json: js(`0.0123456`),
			path: `$.decimal(1,2)`,
			exp:  []any{float64(0.01)},
		},
		{
			name: "test_41",
			json: js(`0.0012345`),
			path: `$.decimal(2,4)`,
			exp:  []any{float64(0.0012)},
		},
		{
			name: "test_42",
			json: js(`-0.00123456`),
			path: `$.decimal(2,-4)`,
			exp:  []any{float64(0)},
		},
		{
			name: "test_43",
			json: js(`12.3`),
			path: `$.decimal(12345678901,1)`,
			err:  `exec: precision of jsonpath item method .decimal() is out of integer range`,
		},
		{
			name: "test_44",
			json: js(`12.3`),
			path: `$.decimal(1,12345678901)`,
			err:  `exec: scale of jsonpath item method .decimal() is out of integer range`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgQueryIntegerMethod(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L516-L544
	for _, tc := range []queryTestCase{
		{
			name: "test_1",
			json: js(`null`),
			path: `$.integer()`,
			err:  `exec: jsonpath item method .integer() can only be applied to a string or numeric value`,
		},
		{
			name: "test_2",
			json: js(`true`),
			path: `$.integer()`,
			err:  `exec: jsonpath item method .integer() can only be applied to a string or numeric value`,
		},
		{
			name: "test_3",
			json: js(`null`),
			path: `$.integer()`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_4",
			json: js(`true`),
			path: `$.integer()`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_5",
			json: js(`[]`),
			path: `$.integer()`,
			exp:  []any{},
		},
		{
			name: "test_6",
			json: js(`[]`),
			path: `strict $.integer()`,
			err:  `exec: jsonpath item method .integer() can only be applied to a string or numeric value`,
		},
		{
			name: "test_7",
			json: js(`{}`),
			path: `$.integer()`,
			err:  `exec: jsonpath item method .integer() can only be applied to a string or numeric value`,
		},
		{
			name: "test_8",
			json: js(`[]`),
			path: `strict $.integer()`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_9",
			json: js(`{}`),
			path: `$.integer()`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_10",
			json: js(`"1.23"`),
			path: `$.integer()`,
			err:  `exec: argument "1.23" of jsonpath item method .integer() is invalid for type integer`,
		},
		{
			name: "test_11",
			json: js(`"1.23aaa"`),
			path: `$.integer()`,
			err:  `exec: argument "1.23aaa" of jsonpath item method .integer() is invalid for type integer`,
		},
		// Go cannot parse 1e1000 into a float because it's too big.
		// Postgres JSONB accepts arbitrary numeric sizes.
		// {
		// 	name: "test_12",
		// 	json: js(`1e1000`),
		// 	path: `$.integer()`,
		// 	err:  `exec: argument "10000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000" of jsonpath item method .integer() is invalid for type integer`,
		// },
		{
			name: "test_13",
			json: js(`"nan"`),
			path: `$.integer()`,
			err:  `exec: argument "nan" of jsonpath item method .integer() is invalid for type integer`,
		},
		{
			name: "test_14",
			json: js(`"NaN"`),
			path: `$.integer()`,
			err:  `exec: argument "NaN" of jsonpath item method .integer() is invalid for type integer`,
		},
		{
			name: "test_15",
			json: js(`"inf"`),
			path: `$.integer()`,
			err:  `exec: argument "inf" of jsonpath item method .integer() is invalid for type integer`,
		},
		{
			name: "test_16",
			json: js(`"-inf"`),
			path: `$.integer()`,
			err:  `exec: argument "-inf" of jsonpath item method .integer() is invalid for type integer`,
		},
		{
			name: "test_17",
			json: js(`"inf"`),
			path: `$.integer()`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_18",
			json: js(`"-inf"`),
			path: `$.integer()`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_19",
			json: js(`123`),
			path: `$.integer()`,
			exp:  []any{int64(123)},
		},
		{
			name: "test_20",
			json: js(`"123"`),
			path: `$.integer()`,
			exp:  []any{int64(123)},
		},
		{
			name: "test_21",
			json: js(`1.23`),
			path: `$.integer()`,
			exp:  []any{int64(1)},
		},
		{
			name: "test_22",
			json: js(`1.83`),
			path: `$.integer()`,
			exp:  []any{int64(2)},
		},
		{
			name: "test_23",
			json: js(`12345678901`),
			path: `$.integer()`,
			// pg: shows `"12345678901"` in the error
			err: `exec: argument "1.2345678901e+10" of jsonpath item method .integer() is invalid for type integer`,
		},
		{
			name: "test_24",
			json: js(`"12345678901"`),
			path: `$.integer()`,
			err:  `exec: argument "12345678901" of jsonpath item method .integer() is invalid for type integer`,
		},
		{
			name: "test_25",
			json: js(`"+123"`),
			path: `$.integer()`,
			exp:  []any{int64(123)},
		},
		{
			name: "test_26",
			json: js(`-123`),
			path: `$.integer()`,
			exp:  []any{int64(-123)},
		},
		{
			name: "test_27",
			json: js(`"-123"`),
			path: `$.integer()`,
			exp:  []any{int64(-123)},
		},
		{
			name: "test_28",
			json: js(`123`),
			path: `$.integer() * 2`,
			exp:  []any{int64(246)},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgQueryNumberMethod(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L546-L573
	for _, tc := range []queryTestCase{
		{
			name: "test_1",
			json: js(`null`),
			path: `$.number()`,
			err:  `exec: jsonpath item method .number() can only be applied to a string or numeric value`,
		},
		{
			name: "test_2",
			json: js(`true`),
			path: `$.number()`,
			err:  `exec: jsonpath item method .number() can only be applied to a string or numeric value`,
		},
		{
			name: "test_3",
			json: js(`null`),
			path: `$.number()`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_4",
			json: js(`true`),
			path: `$.number()`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_5",
			json: js(`[]`),
			path: `$.number()`,
			exp:  []any{},
		},
		{
			name: "test_6",
			json: js(`[]`),
			path: `strict $.number()`,
			err:  `exec: jsonpath item method .number() can only be applied to a string or numeric value`,
		},
		{
			name: "test_7",
			json: js(`{}`),
			path: `$.number()`,
			err:  `exec: jsonpath item method .number() can only be applied to a string or numeric value`,
		},
		{
			name: "test_8",
			json: js(`[]`),
			path: `strict $.number()`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_9",
			json: js(`{}`),
			path: `$.number()`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_10",
			json: js(`1.23`),
			path: `$.number()`,
			exp:  []any{float64(1.23)},
		},
		{
			name: "test_11",
			json: js(`"1.23"`),
			path: `$.number()`,
			exp:  []any{float64(1.23)},
		},
		{
			name: "test_12",
			json: js(`"1.23aaa"`),
			path: `$.number()`,
			err:  `exec: argument "1.23aaa" of jsonpath item method .number() is invalid for type numeric`,
		},
		// Go cannot parse 1e1000 into a float because it's too big.
		// Postgres JSONB accepts arbitrary numeric sizes.
		// {
		// 	name: "test_13",
		// 	json: js(`1e1000`),
		// 	path: `$.number()`,
		// 	exp:  []any{"10000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"},
		// },
		{
			name: "test_14",
			json: js(`"nan"`),
			path: `$.number()`,
			err:  `exec: NaN or Infinity is not allowed for jsonpath item method .number()`,
		},
		{
			name: "test_15",
			json: js(`"NaN"`),
			path: `$.number()`,
			err:  `exec: NaN or Infinity is not allowed for jsonpath item method .number()`,
		},
		{
			name: "test_16",
			json: js(`"inf"`),
			path: `$.number()`,
			err:  `exec: NaN or Infinity is not allowed for jsonpath item method .number()`,
		},
		{
			name: "test_17",
			json: js(`"-inf"`),
			path: `$.number()`,
			err:  `exec: NaN or Infinity is not allowed for jsonpath item method .number()`,
		},
		{
			name: "test_18",
			json: js(`"inf"`),
			path: `$.number()`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_19",
			json: js(`"-inf"`),
			path: `$.number()`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_20",
			json: js(`123`),
			path: `$.number()`,
			exp:  []any{float64(123)},
		},
		{
			name: "test_21",
			json: js(`"123"`),
			path: `$.number()`,
			exp:  []any{float64(123)},
		},
		{
			name: "test_22",
			json: js(`12345678901234567890`),
			path: `$.number()`,
			exp:  []any{float64(12345678901234567890)},
		},
		{
			name: "test_23",
			json: js(`"12345678901234567890"`),
			path: `$.number()`,
			exp:  []any{float64(12345678901234567890)},
		},
		{
			name: "test_24",
			json: js(`"+12.3"`),
			path: `$.number()`,
			exp:  []any{float64(12.3)},
		},
		{
			name: "test_25",
			json: js(`-12.3`),
			path: `$.number()`,
			exp:  []any{float64(-12.3)},
		},
		{
			name: "test_26",
			json: js(`"-12.3"`),
			path: `$.number()`,
			exp:  []any{float64(-12.3)},
		},
		{
			name: "test_27",
			json: js(`12.3`),
			path: `$.number() * 2`,
			exp:  []any{float64(24.6)},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgQueryStringMethod(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L575-L592
	for _, tc := range []queryTestCase{
		{
			name: "test_1",
			json: js(`null`),
			path: `$.string()`,
			err:  `exec: jsonpath item method .string() can only be applied to a bool, string, numeric, or datetime value`,
		},
		{
			name: "test_2",
			json: js(`null`),
			path: `$.string()`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_3",
			json: js(`[]`),
			path: `$.string()`,
			err:  `exec: jsonpath item method .string() can only be applied to a bool, string, numeric, or datetime value`,
		},
		{
			name: "test_4",
			json: js(`[]`),
			path: `strict $.string()`,
			err:  `exec: jsonpath item method .string() can only be applied to a bool, string, numeric, or datetime value`,
		},
		{
			name: "test_5",
			json: js(`{}`),
			path: `$.string()`,
			err:  `exec: jsonpath item method .string() can only be applied to a bool, string, numeric, or datetime value`,
		},
		{
			name: "test_6",
			json: js(`[]`),
			path: `strict $.string()`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_7",
			json: js(`{}`),
			path: `$.string()`,
			opt:  []Option{WithSilent()},
			exp:  []any{},
		},
		{
			name: "test_8",
			json: js(`1.23`),
			path: `$.string()`,
			exp:  []any{"1.23"},
		},
		{
			name: "test_9",
			json: js(`"1.23"`),
			path: `$.string()`,
			exp:  []any{"1.23"},
		},
		{
			name: "test_10",
			json: js(`"1.23aaa"`),
			path: `$.string()`,
			exp:  []any{"1.23aaa"},
		},
		{
			name: "test_11",
			json: js(`1234`),
			path: `$.string()`,
			exp:  []any{"1234"},
		},
		{
			name: "test_12",
			json: js(`true`),
			path: `$.string()`,
			exp:  []any{"true"},
		},
		{
			name: "test_13",
			json: js(`1234`),
			path: `$.string().type()`,
			exp:  []any{"string"},
		},
		{
			// pg: parses 2023-08-15 12:34:56 +5:30
			name: "test_14",
			json: js(`"2023-08-15 12:34:56+05:30"`),
			path: `$.timestamp().string()`,
			err:  `exec: cannot convert value from timestamptz to timestamp without time zone usage.` + hint,
		},
		{
			// pg: parses 2023-08-15 12:34:56 +5:30
			name: "test_15",
			json: js(`"2023-08-15 12:34:56+05:30"`),
			path: `$.timestamp().string()`,
			opt:  []Option{WithTZ()},
			exp:  []any{"2023-08-15T07:04:56"}, // should work
			// Tue Aug 15 00:04:56 2023
		},
		// pg: tests 16 & 17 use jsonb_path_query_array but our Query() always
		// returns a slice.
		{
			name: "test_16",
			json: js(`[1.23, "yes", false]`),
			path: `$[*].string()`,
			exp:  []any{"1.23", "yes", "false"},
		},
		{
			name: "test_17",
			json: js(`[1.23, "yes", false]`),
			path: `$[*].string().type()`,
			exp:  []any{"string", "string", "string"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgQueryTimeMethod(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https
	for _, tc := range []queryTestCase{
		{
			name: "test_1",
			json: js(`null`),
			path: `$.time()`,
			err:  `exec: jsonpath item method .time() can only be applied to a string`,
		},
		{
			name: "test_2",
			json: js(`true`),
			path: `$.time()`,
			err:  `exec: jsonpath item method .time() can only be applied to a string`,
		},
		{
			name: "test_3",
			json: js(`1`),
			path: `$.time()`,
			err:  `exec: jsonpath item method .time() can only be applied to a string`,
		},
		{
			name: "test_4",
			json: js(`[]`),
			path: `$.time()`,
			exp:  []any{},
		},
		{
			name: "test_5",
			json: js(`[]`),
			path: `strict $.time()`,
			err:  `exec: jsonpath item method .time() can only be applied to a string`,
		},
		{
			name: "test_6",
			json: js(`{}`),
			path: `$.time()`,
			err:  `exec: jsonpath item method .time() can only be applied to a string`,
		},
		{
			name: "test_7",
			json: js(`"bogus"`),
			path: `$.time()`,
			err:  `exec: time format is not recognized: "bogus"`,
		},
		// Test 8 in TestPgQueryTimeAtQuestion below
		{
			name: "test_9",
			json: js(`"12:34:56"`),
			path: `$.time()`,
			exp:  []any{pt("12:34:56")},
		},
		{
			name: "test_10",
			json: js(`"12:34:56"`),
			path: `$.time().type()`,
			exp:  []any{"time without time zone"},
		},
		{
			name: "test_11",
			json: js(`"2023-08-15"`),
			path: `$.time()`,
			err:  `exec: time format is not recognized: "2023-08-15"`,
		},
		{
			name: "test_12",
			json: js(`"12:34:56+05:30"`), // pg: uses 12:34:56 +05:30
			path: `$.time()`,
			err:  `exec: cannot convert value from timetz to time without time zone usage.` + hint,
		},
		{
			name: "test_13",
			json: js(`"12:34:56+05:30"`), // pg: uses 12:34:56 +05:30
			path: `$.time()`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("12:34:56")}, // should work
		},
		{
			name: "test_14",
			json: js(`"2023-08-15 12:34:56"`),
			path: `$.time()`,
			exp:  []any{pt("12:34:56")},
		},
		// Tests 15 & 16 in TestPgQueryTimeMethodSyntaxError below.
		{
			name: "test_17",
			json: js(`"12:34:56.789"`),
			path: `$.time(12345678901)`,
			err:  `exec: time precision of jsonpath item method .time() is out of integer range`,
		},
		{
			name: "test_18",
			json: js(`"12:34:56.789"`),
			path: `$.time(0)`,
			exp:  []any{pt("12:34:57")},
		},
		{
			name: "test_19",
			json: js(`"12:34:56.789"`),
			path: `$.time(2)`,
			exp:  []any{pt("12:34:56.79")},
		},
		{
			name: "test_20",
			json: js(`"12:34:56.789"`),
			path: `$.time(5)`,
			exp:  []any{pt("12:34:56.789")},
		},
		{
			name: "test_21",
			json: js(`"12:34:56.789"`),
			path: `$.time(10)`,
			exp:  []any{pt("12:34:56.789")},
		},
		{
			name: "test_22",
			json: js(`"12:34:56.789012"`),
			path: `$.time(8)`,
			exp:  []any{pt("12:34:56.789012")},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgQueryTimeAtQuestion(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L603
	for _, tc := range []existsTestCase{
		{
			name: "test_8",
			json: js(`"12:34:56"`),
			path: `$.time()`,
			exp:  true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.runAtQuestion(a, r)
		})
	}
}

func TestPgQueryTimeMethodSyntaxError(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L612-L613
	for _, tc := range []queryTestCase{
		{
			name: "test_15",
			json: js(`"12:34:56.789"`),
			path: `$.time(-1)`,
			err:  `parser: syntax error at 1:9`,
		},
		{
			name: "test_16",
			json: js(`"12:34:56.789"`),
			path: `$.time(2.0)`,
			err:  `parser: syntax error at 1:11`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			path, err := parser.Parse(tc.path)
			r.EqualError(err, tc.err)
			r.ErrorIs(err, parser.ErrParse)
			a.Nil(path)
		})
	}
}

func TestPgQueryTimeTZMethod(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L621-L644
	for _, tc := range []queryTestCase{
		{
			name: "test_1",
			json: js(`null`),
			path: `$.time_tz()`,
			err:  `exec: jsonpath item method .time_tz() can only be applied to a string`,
		},
		{
			name: "test_2",
			json: js(`true`),
			path: `$.time_tz()`,
			err:  `exec: jsonpath item method .time_tz() can only be applied to a string`,
		},
		{
			name: "test_3",
			json: js(`1`),
			path: `$.time_tz()`,
			err:  `exec: jsonpath item method .time_tz() can only be applied to a string`,
		},
		{
			name: "test_4",
			json: js(`[]`),
			path: `$.time_tz()`,
			exp:  []any{},
		},
		{
			name: "test_5",
			json: js(`[]`),
			path: `strict $.time_tz()`,
			err:  `exec: jsonpath item method .time_tz() can only be applied to a string`,
		},
		{
			name: "test_6",
			json: js(`{}`),
			path: `$.time_tz()`,
			err:  `exec: jsonpath item method .time_tz() can only be applied to a string`,
		},
		{
			name: "test_7",
			json: js(`"bogus"`),
			path: `$.time_tz()`,
			err:  `exec: time_tz format is not recognized: "bogus"`,
		},
		// Test 8 in TestPgQueryTimeTZAtQuestion below
		{
			name: "test_9",
			json: js(`"12:34:56+05:30"`), // pg: 12:34:56 +05:30
			path: `$.time_tz()`,
			exp:  []any{pt("12:34:56+05:30")},
		},
		{
			name: "test_10",
			json: js(`"12:34:56+05:30"`), // pg: 12:34:56 +05:30
			path: `$.time_tz().type()`,
			exp:  []any{"time with time zone"},
		},
		{
			name: "test_11",
			json: js(`"2023-08-15"`),
			path: `$.time_tz()`,
			err:  `exec: time_tz format is not recognized: "2023-08-15"`,
		},
		{
			name: "test_12",
			json: js(`"2023-08-15 12:34:56"`),
			path: `$.time_tz()`,
			err:  `exec: time_tz format is not recognized: "2023-08-15 12:34:56"`,
		},
		// Tests 13 & 14 in TestPgQueryTimeTZMethodSyntaxError below.
		{
			name: "test_15",
			json: js(`"12:34:56.789+05:30"`), // pg: 12:34:56.789 +05:30
			path: `$.time_tz(12345678901)`,
			err:  `exec: time precision of jsonpath item method .time_tz() is out of integer range`,
		},
		{
			name: "test_16",
			json: js(`"12:34:56.789+05:30"`), // pg: 12:34:56.789 +05:30
			path: `$.time_tz(0)`,
			exp:  []any{pt("12:34:57+05:30")},
		},
		{
			name: "test_17",
			json: js(`"12:34:56.789+05:30"`), // pg: 12:34:56.789 +05:30
			path: `$.time_tz(2)`,
			exp:  []any{pt("12:34:56.79+05:30")},
		},
		{
			name: "test_18",
			json: js(`"12:34:56.789+05:30"`), // pg: 12:34:56.789 +05:30
			path: `$.time_tz(5)`,
			exp:  []any{pt("12:34:56.789+05:30")},
		},
		{
			name: "test_19",
			json: js(`"12:34:56.789+05:30"`), // pg: 12:34:56.789 +05:30
			path: `$.time_tz(10)`,
			exp:  []any{pt("12:34:56.789+05:30")},
		},
		{
			name: "test_20",
			json: js(`"12:34:56.789012+05:30"`), // pg: 12:34:56.789012 +05:30
			path: `$.time_tz(8)`,
			exp:  []any{pt("12:34:56.789012+05:30")},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgQueryTimeTZAtQuestion(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L630
	for _, tc := range []existsTestCase{
		{
			name: "test_8",
			json: js(`"12:34:56+05:30"`), // pg: 12:34:56 +05:30
			path: `$.time_tz()`,
			exp:  true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.runAtQuestion(a, r)
		})
	}
}

func TestPgQueryTimeTZMethodSyntaxError(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L637-L638
	for _, tc := range []queryTestCase{
		{
			name: "test_13",
			json: js(`"12:34:56.789 +05:30"`),
			path: `$.time_tz(-1)`,
			err:  `parser: syntax error at 1:12`,
		},
		{
			name: "test_14",
			json: js(`"12:34:56.789 +05:30"`),
			path: `$.time_tz(2.0)`,
			err:  `parser: syntax error at 1:14`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			path, err := parser.Parse(tc.path)
			r.EqualError(err, tc.err)
			r.ErrorIs(err, parser.ErrParse)
			a.Nil(path)
		})
	}
}

func TestPgQueryTimestampMethod(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L646-L670
	for _, tc := range []queryTestCase{
		{
			name: "test_1",
			json: js(`null`),
			path: `$.timestamp()`,
			err:  `exec: jsonpath item method .timestamp() can only be applied to a string`,
		},
		{
			name: "test_2",
			json: js(`true`),
			path: `$.timestamp()`,
			err:  `exec: jsonpath item method .timestamp() can only be applied to a string`,
		},
		{
			name: "test_3",
			json: js(`1`),
			path: `$.timestamp()`,
			err:  `exec: jsonpath item method .timestamp() can only be applied to a string`,
		},
		{
			name: "test_4",
			json: js(`[]`),
			path: `$.timestamp()`,
			exp:  []any{},
		},
		{
			name: "test_5",
			json: js(`[]`),
			path: `strict $.timestamp()`,
			err:  `exec: jsonpath item method .timestamp() can only be applied to a string`,
		},
		{
			name: "test_6",
			json: js(`{}`),
			path: `$.timestamp()`,
			err:  `exec: jsonpath item method .timestamp() can only be applied to a string`,
		},
		{
			name: "test_7",
			json: js(`"bogus"`),
			path: `$.timestamp()`,
			err:  `exec: timestamp format is not recognized: "bogus"`,
		},
		// Test 8 in TestPgQueryTimestampAtQuestion below
		{
			name: "test_9",
			json: js(`"2023-08-15 12:34:56"`),
			path: `$.timestamp()`,
			exp:  []any{pt("2023-08-15T12:34:56")},
		},
		{
			name: "test_10",
			json: js(`"2023-08-15 12:34:56"`),
			path: `$.timestamp().type()`,
			exp:  []any{"timestamp without time zone"},
		},
		{
			name: "test_11",
			json: js(`"2023-08-15"`),
			path: `$.timestamp()`,
			exp:  []any{pt("2023-08-15T00:00:00")},
		},
		{
			name: "test_12",
			json: js(`"12:34:56"`),
			path: `$.timestamp()`,
			err:  `exec: timestamp format is not recognized: "12:34:56"`,
		},
		{
			name: "test_13",
			json: js(`"12:34:56+05:30"`), // pg: 12:34:56 +05:30
			path: `$.timestamp()`,
			err:  `exec: timestamp format is not recognized: "12:34:56+05:30"`,
		},
		// Tests 14 & 15 in TestPgQueryTimestampMethodSyntaxError below.
		{
			name: "test_16",
			json: js(`"2023-08-15 12:34:56.789"`),
			path: `$.timestamp(12345678901)`,
			err:  `exec: time precision of jsonpath item method .timestamp() is out of integer range`,
		},
		{
			name: "test_17",
			json: js(`"2023-08-15 12:34:56.789"`),
			path: `$.timestamp(0)`,
			exp:  []any{pt("2023-08-15T12:34:57")},
		},
		{
			name: "test_18",
			json: js(`"2023-08-15 12:34:56.789"`),
			path: `$.timestamp(2)`,
			exp:  []any{pt("2023-08-15T12:34:56.79")},
		},
		{
			name: "test_19",
			json: js(`"2023-08-15 12:34:56.789"`),
			path: `$.timestamp(5)`,
			exp:  []any{pt("2023-08-15T12:34:56.789")},
		},
		{
			name: "test_20",
			json: js(`"2023-08-15 12:34:56.789"`),
			path: `$.timestamp(10)`,
			exp:  []any{pt("2023-08-15T12:34:56.789")},
		},
		{
			name: "test_21",
			json: js(`"2023-08-15 12:34:56.789012"`),
			path: `$.timestamp(8)`,
			exp:  []any{pt("2023-08-15T12:34:56.789012")},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgQueryTimestampAtQuestion(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L655
	for _, tc := range []existsTestCase{
		{
			name: "test_8",
			json: js(`"2023-08-15 12:34:56"`),
			path: `$.timestamp()`,
			exp:  true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.runAtQuestion(a, r)
		})
	}
}

func TestPgQueryTimestampMethodSyntaxError(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L663-L664
	for _, tc := range []queryTestCase{
		{
			name: "test_14",
			json: js(`"2023-08-15 12:34:56.789"`),
			path: `$.timestamp(-1)`,
			err:  `parser: syntax error at 1:14`,
		},
		{
			name: "test_15",
			json: js(`"2023-08-15 12:34:56.789"`),
			path: `$.timestamp(2.0)`,
			err:  `parser: syntax error at 1:16`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			path, err := parser.Parse(tc.path)
			r.EqualError(err, tc.err)
			r.ErrorIs(err, parser.ErrParse)
			a.Nil(path)
		})
	}
}

func TestPgQueryTimestampTZMethod(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L672-L697
	for _, tc := range []queryTestCase{
		{
			name: "test_1",
			json: js(`null`),
			path: `$.timestamp_tz()`,
			err:  `exec: jsonpath item method .timestamp_tz() can only be applied to a string`,
		},
		{
			name: "test_2",
			json: js(`true`),
			path: `$.timestamp_tz()`,
			err:  `exec: jsonpath item method .timestamp_tz() can only be applied to a string`,
		},
		{
			name: "test_3",
			json: js(`1`),
			path: `$.timestamp_tz()`,
			err:  `exec: jsonpath item method .timestamp_tz() can only be applied to a string`,
		},
		{
			name: "test_4",
			json: js(`[]`),
			path: `$.timestamp_tz()`,
			exp:  []any{},
		},
		{
			name: "test_5",
			json: js(`[]`),
			path: `strict $.timestamp_tz()`,
			err:  `exec: jsonpath item method .timestamp_tz() can only be applied to a string`,
		},
		{
			name: "test_6",
			json: js(`{}`),
			path: `$.timestamp_tz()`,
			err:  `exec: jsonpath item method .timestamp_tz() can only be applied to a string`,
		},
		{
			name: "test_7",
			json: js(`"bogus"`),
			path: `$.timestamp_tz()`,
			err:  `exec: timestamp_tz format is not recognized: "bogus"`,
		},
		// Test 8 in TestPgQueryTimestampTZAtQuestion below
		{
			name: "test_9",
			json: js(`"2023-08-15 12:34:56+05:30"`), // pg: 2023-08-15 12:34:56 +05:30
			path: `$.timestamp_tz()`,
			exp:  []any{pt("2023-08-15T12:34:56+05:30")},
		},
		{
			name: "test_10",
			json: js(`"2023-08-15 12:34:56+05:30"`), // pg: 2023-08-15 12:34:56 +05:30
			path: `$.timestamp_tz().type()`,
			exp:  []any{"timestamp with time zone"},
		},
		{
			name: "test_11",
			json: js(`"2023-08-15"`),
			path: `$.timestamp_tz()`,
			err:  `exec: cannot convert value from date to timestamptz without time zone usage.` + hint,
		},
		{
			name: "test_12",
			json: js(`"2023-08-15"`),
			path: `$.timestamp_tz()`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("2023-08-15T00:00:00Z")}, // should work // pg: 2023-08-15T07:00:00+00:00
		},
		{
			name: "test_13",
			json: js(`"12:34:56"`),
			path: `$.timestamp_tz()`,
			err:  `exec: timestamp_tz format is not recognized: "12:34:56"`,
		},
		{
			name: "test_14",
			json: js(`"12:34:56+05:30"`), // pg: 12:34:56 +05:30
			path: `$.timestamp_tz()`,
			err:  `exec: timestamp_tz format is not recognized: "12:34:56+05:30"`,
		},
		// Tests 15 & 16 in TestPgQueryTimestampTZMethodSyntaxError below.
		{
			name: "test_17",
			json: js(`"2023-08-15 12:34:56.789 +05:30"`), // pg: 2023-08-15 12:34:56.789 +05:30
			path: `$.timestamp_tz(12345678901)`,
			err:  `exec: time precision of jsonpath item method .timestamp_tz() is out of integer range`,
		},
		{
			name: "test_18",
			json: js(`"2023-08-15 12:34:56.789+05:30"`), // pg: 2023-08-15 12:34:56.789 +05:30
			path: `$.timestamp_tz(0)`,
			exp:  []any{pt("2023-08-15T12:34:57+05:30")},
		},
		{
			name: "test_19",
			json: js(`"2023-08-15 12:34:56.789+05:30"`), // pg: 2023-08-15 12:34:56.789 +05:30
			path: `$.timestamp_tz(2)`,
			exp:  []any{pt("2023-08-15T12:34:56.79+05:30")},
		},
		{
			name: "test_20",
			json: js(`"2023-08-15 12:34:56.789+05:30"`), // pg: 2023-08-15 12:34:56.789 +05:30
			path: `$.timestamp_tz(5)`,
			exp:  []any{pt("2023-08-15T12:34:56.789+05:30")},
		},
		{
			name: "test_21",
			json: js(`"2023-08-15 12:34:56.789+05:30"`), // pg: 2023-08-15 12:34:56.789 +05:30
			path: `$.timestamp_tz(10)`,
			exp:  []any{pt("2023-08-15T12:34:56.789+05:30")},
		},
		{
			name: "test_22",
			json: js(`"2023-08-15 12:34:56.789012+05:30"`), // pg: 2023-08-15 12:34:56.789012 +05:30
			path: `$.timestamp_tz(8)`,
			exp:  []any{pt("2023-08-15T12:34:56.789012+05:30")},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgQueryTimestampTZAtQuestion(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L681
	for _, tc := range []existsTestCase{
		{
			name: "test_8",
			json: js(`"2023-08-15 12:34:56+05:30"`), // pg: 2023-08-15 12:34:56 +05:30
			path: `$.timestamp_tz()`,
			exp:  true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.runAtQuestion(a, r)
		})
	}
}

func TestPgQueryTimestampTZMethodSyntaxError(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L690-L691
	for _, tc := range []queryTestCase{
		{
			name: "test_15",
			json: js(`"2023-08-15 12:34:56.789+05:30"`), // pg: "2023-08-15 12:34:56.789 +05:30"
			path: `$.timestamp_tz(-1)`,
			err:  `parser: syntax error at 1:17`,
		},
		{
			name: "test_16",
			json: js(`"2023-08-15 12:34:56.789+05:30"`), // pg: "2023-08-15 12:34:56.789 +05:30"
			path: `$.timestamp_tz(2.0)`,
			err:  `parser: syntax error at 1:19`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			path, err := parser.Parse(tc.path)
			r.EqualError(err, tc.err)
			r.ErrorIs(err, parser.ErrParse)
			a.Nil(path)
		})
	}
}

func TestPgQueryDateTimeMethodsUTC(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// For these tests, Postgres uses
	//   set time zone '+00';
	// UTC is the default for us, so not much to adjust here.

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L700-L723
	for _, tc := range []queryTestCase{
		{
			name: "test_1",
			json: js(`"2023-08-15 12:34:56+05:30"`), // pg: 2023-08-15 12:34:56 +05:30
			path: `$.time()`,
			err:  `exec: cannot convert value from timestamptz to time without time zone usage.` + hint,
		},
		{
			name: "test_2",
			json: js(`"2023-08-15 12:34:56+05:30"`), // pg: 2023-08-15 12:34:56 +05:30
			path: `$.time()`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("07:04:56")}, // should work
		},
		{
			name: "test_3",
			json: js(`"2023-08-15 12:34:56+05:30"`), // pg: 2023-08-15 12:34:56 +05:30
			path: `$.time_tz()`,
			// Postgres displays the output as UTC thanks to `set time zone '+00'`.
			// The Go Time object stringifies for the parsed offset.
			exp: []any{pt("12:34:56+05:30")}, // Retains TZ (pg: 07:04:56+00:00)
		},
		{
			name: "test_4",
			json: js(`"12:34:56"`),
			path: `$.time_tz()`,
			err:  `exec: cannot convert value from time to timetz without time zone usage.` + hint,
		},
		{
			name: "test_5",
			json: js(`"12:34:56"`),
			path: `$.time_tz()`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("12:34:56Z")}, // should work
		},
		{
			name: "test_6",
			json: js(`"2023-08-15 12:34:56+05:30"`), // pg: 2023-08-15 12:34:56 +05:30
			path: `$.timestamp()`,
			err:  `exec: cannot convert value from timestamptz to timestamp without time zone usage.` + hint,
		},
		{
			name: "test_7",
			json: js(`"2023-08-15 12:34:56+05:30"`), // pg: 2023-08-15 12:34:56 +05:30
			path: `$.timestamp()`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("2023-08-15T07:04:56")}, // should work
		},
		{
			name: "test_8",
			json: js(`"2023-08-15 12:34:56"`),
			path: `$.timestamp_tz()`,
			err:  `exec: cannot convert value from timestamp to timestamptz without time zone usage.` + hint,
		},
		{
			name: "test_9",
			json: js(`"2023-08-15 12:34:56"`),
			path: `$.timestamp_tz()`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("2023-08-15T12:34:56Z")}, // should work
		},
		// Remove err field from remaining tests once .datetime(template) implemented
		{
			name: "test_10",
			json: js(`"10-03-2017 12:34"`),
			path: `$.datetime("dd-mm-yyyy HH24:MI")`,
			exp:  []any{pt("2017-03-10T12:34:00")},
			err:  `exec: .datetime(template) is not yet supported`,
		},
		{
			name: "test_11",
			json: js(`"10-03-2017 12:34"`),
			path: `$.datetime("dd-mm-yyyy HH24:MI TZH")`,
			err:  `exec: .datetime(template) is not yet supported`,
			// err: `exec: input string is too short for datetime format`,
		},
		{
			name: "test_12",
			json: js(`"10-03-2017 12:34 +05"`),
			path: `$.datetime("dd-mm-yyyy HH24:MI TZH")`,
			exp:  []any{pt("2017-03-10T12:34:00+05:00")},
			err:  `exec: .datetime(template) is not yet supported`,
		},
		{
			name: "test_13",
			json: js(`"10-03-2017 12:34 -05"`),
			path: `$.datetime("dd-mm-yyyy HH24:MI TZH")`,
			exp:  []any{pt("2017-03-10T12:34:00-05:00")},
			err:  `exec: .datetime(template) is not yet supported`,
		},
		{
			name: "test_14",
			json: js(`"10-03-2017 12:34 +05:20"`),
			path: `$.datetime("dd-mm-yyyy HH24:MI TZH:TZM")`,
			exp:  []any{pt("2017-03-10T12:34:00+05:20")},
			err:  `exec: .datetime(template) is not yet supported`,
		},
		{
			name: "test_15",
			json: js(`"10-03-2017 12:34 -05:20"`),
			path: `$.datetime("dd-mm-yyyy HH24:MI TZH:TZM")`,
			exp:  []any{pt("2017-03-10T12:34:00-05:20")},
			err:  `exec: .datetime(template) is not yet supported`,
		},
		{
			name: "test_16",
			json: js(`"12:34"`),
			path: `$.datetime("HH24:MI")`,
			exp:  []any{pt("12:34:00")},
			err:  `exec: .datetime(template) is not yet supported`,
		},
		{
			name: "test_17",
			json: js(`"12:34"`),
			path: `$.datetime("HH24:MI TZH")`,
			err:  `exec: .datetime(template) is not yet supported`,
			// err:  `exec: input string is too short for datetime format`,
		},
		{
			name: "test_18",
			json: js(`"12:34 +05"`),
			path: `$.datetime("HH24:MI TZH")`,
			exp:  []any{pt("12:34:00+05:00")},
			err:  `exec: .datetime(template) is not yet supported`,
		},
		{
			name: "test_19",
			json: js(`"12:34 -05"`),
			path: `$.datetime("HH24:MI TZH")`,
			err:  `exec: .datetime(template) is not yet supported`,
			exp:  []any{pt("12:34:00-05:00")},
		},
		{
			name: "test_20",
			json: js(`"12:34 +05:20"`),
			path: `$.datetime("HH24:MI TZH:TZM")`,
			exp:  []any{pt("12:34:00+05:20")},
			err:  `exec: .datetime(template) is not yet supported`,
		},
		{
			name: "test_21",
			json: js(`"12:34 -05:20"`),
			path: `$.datetime("HH24:MI TZH:TZM")`,
			exp:  []any{pt("12:34:00-05:20")},
			err:  `exec: .datetime(template) is not yet supported`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgQueryDateTimeMethodsPlus10(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// For these tests, Postgres uses
	//   set time zone '+10';
	// UTC is the default for us, so the results here differ by +10, but the
	// precise instant should be the same.

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L725-L747
	for _, tc := range []queryTestCase{
		{
			name: "test_1",
			json: js(`"2023-08-15 12:34:56+05:30"`), // pg: 2023-08-15 12:34:56 +05:30
			path: `$.time()`,
			err:  `exec: cannot convert value from timestamptz to time without time zone usage.` + hint,
		},
		{
			name: "test_2",
			json: js(`"2023-08-15 12:34:56+05:30"`), // pg: 2023-08-15 12:34:56 +05:30
			path: `$.time()`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("07:04:56")}, // should work // pg: 17:04:56
		},
		{
			name: "test_3",
			json: js(`"2023-08-15 12:34:56+05:30"`), // pg: 2023-08-15 12:34:56 +05:30
			path: `$.time_tz()`,
			exp:  []any{pt("12:34:56+05:30")}, // retains tz. pg: "17:04:56+10:00"
		},
		{
			name: "test_4",
			json: js(`"2023-08-15 12:34:56+05:30"`), // pg: 2023-08-15 12:34:56 +05:30
			path: `$.timestamp()`,
			err:  `exec: cannot convert value from timestamptz to timestamp without time zone usage.` + hint,
		},
		{
			name: "test_5",
			json: js(`"2023-08-15 12:34:56+05:30"`), // pg: 2023-08-15 12:34:56 +05:30
			path: `$.timestamp()`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("2023-08-15T07:04:56")}, // should work // pg: 2023-08-15T17:04:56
		},
		{
			name: "test_6",
			json: js(`"2023-08-15 12:34:56"`),
			path: `$.timestamp_tz()`,
			err:  `exec: cannot convert value from timestamp to timestamptz without time zone usage.` + hint,
		},
		{
			name: "test_7",
			json: js(`"2023-08-15 12:34:56"`),
			path: `$.timestamp_tz()`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("2023-08-15T12:34:56Z")}, // should work // pg: 2023-08-15T02:34:56+00:00
		},
		{
			name: "test_8",
			json: js(`"2023-08-15 12:34:56+05:30"`), // pg: 2023-08-15 12:34:56 +05:30
			path: `$.timestamp_tz()`,
			exp:  []any{pt("2023-08-15T12:34:56+05:30")},
		},
		{
			name: "test_9",
			json: js(`"10-03-2017 12:34"`),
			path: `$.datetime("dd-mm-yyyy HH24:MI")`,
			exp:  []any{pt("2017-03-10T12:34:00")},
			err:  `exec: .datetime(template) is not yet supported`,
		},
		{
			name: "test_10",
			json: js(`"10-03-2017 12:34"`),
			path: `$.datetime("dd-mm-yyyy HH24:MI TZH")`,
			err:  `exec: .datetime(template) is not yet supported`,
			// err:  `exec: input string is too short for datetime format`,
		},
		{
			name: "test_11",
			json: js(`"10-03-2017 12:34 +05"`),
			path: `$.datetime("dd-mm-yyyy HH24:MI TZH")`,
			exp:  []any{pt("2017-03-10T12:34:00+05:00")},
			err:  `exec: .datetime(template) is not yet supported`,
		},
		{
			name: "test_12",
			json: js(`"10-03-2017 12:34 -05"`),
			path: `$.datetime("dd-mm-yyyy HH24:MI TZH")`,
			exp:  []any{pt("2017-03-10T12:34:00-05:00")},
			err:  `exec: .datetime(template) is not yet supported`,
		},
		{
			name: "test_13",
			json: js(`"10-03-2017 12:34 +05:20"`),
			path: `$.datetime("dd-mm-yyyy HH24:MI TZH:TZM")`,
			exp:  []any{pt("2017-03-10T12:34:00+05:20")},
			err:  `exec: .datetime(template) is not yet supported`,
		},
		{
			name: "test_14",
			json: js(`"10-03-2017 12:34 -05:20"`),
			path: `$.datetime("dd-mm-yyyy HH24:MI TZH:TZM")`,
			exp:  []any{pt("2017-03-10T12:34:00-05:20")},
			err:  `exec: .datetime(template) is not yet supported`,
		},
		{
			name: "test_15",
			json: js(`"12:34"`),
			path: `$.datetime("HH24:MI")`,
			exp:  []any{pt("12:34:00")},
			err:  `exec: .datetime(template) is not yet supported`,
		},
		{
			name: "test_16",
			json: js(`"12:34"`),
			path: `$.datetime("HH24:MI TZH")`,
			err:  `exec: .datetime(template) is not yet supported`,
			// err:  `exec: input string is too short for datetime format`,
		},
		{
			name: "test_17",
			json: js(`"12:34 +05"`),
			path: `$.datetime("HH24:MI TZH")`,
			exp:  []any{pt("12:34:00+05:00")},
			err:  `exec: .datetime(template) is not yet supported`,
		},
		{
			name: "test_18",
			json: js(`"12:34 -05"`),
			path: `$.datetime("HH24:MI TZH")`,
			exp:  []any{pt("12:34:00-05:00")},
			err:  `exec: .datetime(template) is not yet supported`,
		},
		{
			name: "test_19",
			json: js(`"12:34 +05:20"`),
			path: `$.datetime("HH24:MI TZH:TZM")`,
			exp:  []any{pt("12:34:00+05:20")},
			err:  `exec: .datetime(template) is not yet supported`,
		},
		{
			name: "test_20",
			json: js(`"12:34 -05:20"`),
			path: `$.datetime("HH24:MI TZH:TZM")`,
			exp:  []any{pt("12:34:00-05:20")},
			err:  `exec: .datetime(template) is not yet supported`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgQueryDateTimeMethodsDefaultTZ(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// For these tests, Postgres uses
	//   set time zone default;
	// Which seems to be -07. UTC is the default for us, so the results here
	// differ by -07, but the precise instant should be the same.

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L749-L778
	for _, tc := range []queryTestCase{
		{
			name: "test_1",
			json: js(`"2023-08-15 12:34:56+05:30"`), // pg: 2023-08-15 12:34:56 +05:30
			path: `$.time()`,
			err:  `exec: cannot convert value from timestamptz to time without time zone usage.` + hint,
		},
		{
			name: "test_2",
			json: js(`"2023-08-15 12:34:56+05:30"`), // pg: 2023-08-15 12:34:56 +05:30
			path: `$.time()`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("07:04:56")}, // should work, converted to UTC // pg: "00:04:56"
		},
		{
			name: "test_3",
			json: js(`"2023-08-15 12:34:56+05:30"`), // pg: 2023-08-15 12:34:56 +05:30
			path: `$.time_tz()`,
			exp:  []any{pt("12:34:56+05:30")}, // retains tz. pg: "00:04:56-07:00"
		},
		{
			name: "test_4",
			json: js(`"2023-08-15 12:34:56+05:30"`), // pg: 2023-08-15 12:34:56 +05:30
			path: `$.timestamp()`,
			err:  `exec: cannot convert value from timestamptz to timestamp without time zone usage.` + hint,
		},
		{
			name: "test_5",
			json: js(`"2023-08-15 12:34:56+05:30"`), // pg: 2023-08-15 12:34:56 +05:30
			path: `$.timestamp()`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("2023-08-15T07:04:56")}, // should work // pg: 2023-08-15T00:04:56
		},
		{
			name: "test_6",
			json: js(`"2023-08-15 12:34:56+05:30"`), // pg: 2023-08-15 12:34:56 +05:30
			path: `$.timestamp_tz()`,
			exp:  []any{pt("2023-08-15T12:34:56+05:30")},
		},
		{
			name: "test_7",
			json: js(`"2017-03-10"`),
			path: `$.datetime().type()`,
			exp:  []any{"date"},
		},
		{
			name: "test_8",
			json: js(`"2017-03-10"`),
			path: `$.datetime()`,
			exp:  []any{pt("2017-03-10")},
		},
		{
			name: "test_9",
			json: js(`"2017-03-10 12:34:56"`),
			path: `$.datetime().type()`,
			exp:  []any{"timestamp without time zone"},
		},
		{
			name: "test_10",
			json: js(`"2017-03-10 12:34:56"`),
			path: `$.datetime()`,
			exp:  []any{pt("2017-03-10T12:34:56")},
		},
		{
			name: "test_11",
			json: js(`"2017-03-10 12:34:56+03"`), // pg: 2017-03-10 12:34:56+3
			path: `$.datetime().type()`,
			exp:  []any{"timestamp with time zone"},
		},
		{
			name: "test_12",
			json: js(`"2017-03-10 12:34:56+03"`), // pg: 2017-03-10 12:34:56+3
			path: `$.datetime()`,
			exp:  []any{pt("2017-03-10T12:34:56+03:00")},
		},
		{
			name: "test_13",
			json: js(`"2017-03-10 12:34:56+03:10"`), // pg: 2017-03-10 12:34:56+3:10
			path: `$.datetime().type()`,
			exp:  []any{"timestamp with time zone"},
		},
		{
			name: "test_14",
			json: js(`"2017-03-10 12:34:56+03:10"`), // pg: 2017-03-10 12:34:56+3:10
			path: `$.datetime()`,
			exp:  []any{pt("2017-03-10T12:34:56+03:10")},
		},
		{
			name: "test_15",
			json: js(`"2017-03-10T12:34:56+03:10"`), // pg: 2017-03-10T12:34:56+3:10
			path: `$.datetime()`,
			exp:  []any{pt("2017-03-10T12:34:56+03:10")},
		},
		{
			name: "test_16",
			json: js(`"2017-03-10t12:34:56+03:10"`), // pg: 2017-03-10t12:34:56+3:10
			path: `$.datetime()`,
			err:  `exec: datetime format is not recognized: "2017-03-10t12:34:56+03:10"`,
		},
		{
			name: "test_17",
			json: js(`"2017-03-10 12:34:56.789+03:10"`), // pg: 2017-03-10 12:34:56.789+3:10
			path: `$.datetime()`,
			exp:  []any{pt("2017-03-10T12:34:56.789+03:10")},
		},
		{
			name: "test_18",
			json: js(`"2017-03-10T12:34:56.789+03:10"`), // pg: 2017-03-10T12:34:56.789+3:10
			path: `$.datetime()`,
			exp:  []any{pt("2017-03-10T12:34:56.789+03:10")},
		},
		{
			name: "test_19",
			json: js(`"2017-03-10t12:34:56.789+03:10"`), // pg: 2017-03-10t12:34:56.789+3:10
			path: `$.datetime()`,
			err:  `exec: datetime format is not recognized: "2017-03-10t12:34:56.789+03:10"`,
		},
		{
			name: "test_20",
			json: js(`"2017-03-10T12:34:56.789-05:00"`), // pg: 2017-03-10T12:34:56.789EST
			path: `$.datetime()`,
			exp:  []any{pt("2017-03-10T12:34:56.789-05:00")},
		},
		{
			name: "test_21",
			json: js(`"2017-03-10T12:34:56.789Z"`),
			path: `$.datetime()`,
			exp:  []any{pt("2017-03-10T12:34:56.789Z")}, // pg 2017-03-10T12:34:56.789+00:00
		},
		{
			name: "test_22",
			json: js(`"12:34:56"`),
			path: `$.datetime().type()`,
			exp:  []any{"time without time zone"},
		},
		{
			name: "test_23",
			json: js(`"12:34:56"`),
			path: `$.datetime()`,
			exp:  []any{pt("12:34:56")},
		},
		{
			name: "test_24",
			json: js(`"12:34:56+03"`), // pg: 12:34:56+3
			path: `$.datetime().type()`,
			exp:  []any{"time with time zone"},
		},
		{
			name: "test_25",
			json: js(`"12:34:56+03"`), // pg: 12:34:56+3
			path: `$.datetime()`,
			exp:  []any{pt("12:34:56+03:00")},
		},
		{
			name: "test_26",
			json: js(`"12:34:56+03:10"`), // pg: 12:34:56+3:10
			path: `$.datetime().type()`,
			exp:  []any{"time with time zone"},
		},
		{
			name: "test_27",
			json: js(`"12:34:56+03:10"`), // pg: 12:34:56+3:10
			path: `$.datetime()`,
			exp:  []any{pt("12:34:56+03:10")},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgQueryDateComparison(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L782-L828
	for _, tc := range []queryTestCase{
		{
			name: "test_1",
			json: js(`["2017-03-10", "2017-03-11", "2017-03-09", "12:34:56", "01:02:03+04", "2017-03-10 00:00:00", "2017-03-10 12:34:56", "2017-03-10 01:02:03+04", "2017-03-10 03:00:00+03"]`),
			path: `$[*].datetime() ? (@ == "10.03.2017".datetime("dd.mm.yyyy"))`,
			err:  `exec: .datetime(template) is not yet supported`,
			// err:  `exec: cannot convert value from date to timestamptz without time zone usage.`+hint,
		},
		{
			name: "test_2",
			json: js(`["2017-03-10", "2017-03-11", "2017-03-09", "12:34:56", "01:02:03+04", "2017-03-10 00:00:00", "2017-03-10 12:34:56", "2017-03-10 01:02:03+04", "2017-03-10 03:00:00+03"]`),
			path: `$[*].datetime() ? (@ >= "10.03.2017".datetime("dd.mm.yyyy"))`,
			err:  `exec: .datetime(template) is not yet supported`,
			// err:  `exec: cannot convert value from date to timestamptz without time zone usage.`+hint,
		},
		{
			name: "test_3",
			json: js(`["2017-03-10", "2017-03-11", "2017-03-09", "12:34:56", "01:02:03+04", "2017-03-10 00:00:00", "2017-03-10 12:34:56", "2017-03-10 01:02:03+04", "2017-03-10 03:00:00+03"]`),
			path: `$[*].datetime() ? (@ <  "10.03.2017".datetime("dd.mm.yyyy"))`,
			err:  `exec: .datetime(template) is not yet supported`,
			// err:  `exec: cannot convert value from date to timestamptz without time zone usage.`+hint,
		},
		{
			name: "test_4",
			json: js(`["2017-03-10", "2017-03-11", "2017-03-09", "12:34:56", "01:02:03+04", "2017-03-10 00:00:00", "2017-03-10 12:34:56", "2017-03-10 01:02:03+04", "2017-03-10 03:00:00+03"]`),
			path: `$[*].datetime() ? (@ == "10.03.2017".datetime("dd.mm.yyyy"))`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("2017-03-10"), pt("2017-03-10T00:00:00"), pt("2017-03-10T03:00:00+03:00")},
			err:  `exec: .datetime(template) is not yet supported`,
		},
		{
			name: "test_5",
			json: js(`["2017-03-10", "2017-03-11", "2017-03-09", "12:34:56", "01:02:03+04", "2017-03-10 00:00:00", "2017-03-10 12:34:56", "2017-03-10 01:02:03+04", "2017-03-10 03:00:00+03"]`),
			path: `$[*].datetime() ? (@ >= "10.03.2017".datetime("dd.mm.yyyy"))`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("2017-03-10"), pt("2017-03-11"), pt("2017-03-10T00:00:00"), pt("2017-03-10T12:34:56"), pt("2017-03-10T03:00:00+03:00")},
			err:  `exec: .datetime(template) is not yet supported`,
		},
		{
			name: "test_6",
			json: js(`["2017-03-10", "2017-03-11", "2017-03-09", "12:34:56", "01:02:03+04", "2017-03-10 00:00:00", "2017-03-10 12:34:56", "2017-03-10 01:02:03+04", "2017-03-10 03:00:00+03"]`),
			path: `$[*].datetime() ? (@ <  "10.03.2017".datetime("dd.mm.yyyy"))`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("2017-03-09"), pt("2017-03-10T01:02:03+04:00")},
			err:  `exec: .datetime(template) is not yet supported`,
		},
		{
			name: "test_7",
			json: js(`["2017-03-10", "2017-03-11", "2017-03-09", "2017-03-10 00:00:00", "2017-03-10 12:34:56", "2017-03-10 01:02:03+04", "2017-03-10 03:00:00+03"]`),
			path: `$[*].datetime() ? (@ == "2017-03-10".date())`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("2017-03-10"), pt("2017-03-10T00:00:00"), pt("2017-03-10T03:00:00+03:00")},
		},
		{
			name: "test_8",
			json: js(`["2017-03-10", "2017-03-11", "2017-03-09", "2017-03-10 00:00:00", "2017-03-10 12:34:56", "2017-03-10 01:02:03+04", "2017-03-10 03:00:00+03"]`),
			path: `$[*].datetime() ? (@ >= "2017-03-10".date())`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("2017-03-10"), pt("2017-03-11"), pt("2017-03-10T00:00:00"), pt("2017-03-10T12:34:56"), pt("2017-03-10T03:00:00+03:00")},
		},
		{
			name: "test_9",
			json: js(`["2017-03-10", "2017-03-11", "2017-03-09", "2017-03-10 00:00:00", "2017-03-10 12:34:56", "2017-03-10 01:02:03+04", "2017-03-10 03:00:00+03"]`),
			path: `$[*].datetime() ? (@ <  "2017-03-10".date())`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("2017-03-09"), pt("2017-03-10T01:02:03+04:00")},
		},
		{
			name: "test_10",
			json: js(`["2017-03-10", "2017-03-11", "2017-03-09", "2017-03-10 00:00:00", "2017-03-10 12:34:56", "2017-03-10 01:02:03+04", "2017-03-10 03:00:00+03"]`),
			path: `$[*].date() ? (@ == "2017-03-10".date())`,
			err:  `exec: cannot convert value from timestamptz to date without time zone usage.` + hint,
		},
		{
			name: "test_11",
			json: js(`["2017-03-10", "2017-03-11", "2017-03-09", "2017-03-10 00:00:00", "2017-03-10 12:34:56", "2017-03-10 01:02:03+04", "2017-03-10 03:00:00+03"]`),
			path: `$[*].date() ? (@ >= "2017-03-10".date())`,
			err:  `exec: cannot convert value from timestamptz to date without time zone usage.` + hint,
		},
		{
			name: "test_12",
			json: js(`["2017-03-10", "2017-03-11", "2017-03-09", "2017-03-10 00:00:00", "2017-03-10 12:34:56", "2017-03-10 01:02:03+04", "2017-03-10 03:00:00+03"]`),
			path: `$[*].date() ? (@ <  "2017-03-10".date())`,
			err:  `exec: cannot convert value from timestamptz to date without time zone usage.` + hint,
		},
		{
			name: "test_13",
			json: js(`["2017-03-10", "2017-03-11", "2017-03-09", "2017-03-10 00:00:00", "2017-03-10 12:34:56", "2017-03-10 01:02:03+04", "2017-03-10 03:00:00+03"]`),
			path: `$[*].date() ? (@ == "2017-03-10".date())`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("2017-03-10"), pt("2017-03-10"), pt("2017-03-10"), pt("2017-03-10")},
		},
		{
			name: "test_14",
			json: js(`["2017-03-10", "2017-03-11", "2017-03-09", "2017-03-10 00:00:00", "2017-03-10 12:34:56", "2017-03-10 01:02:03+04", "2017-03-10 03:00:00+03"]`),
			path: `$[*].date() ? (@ >= "2017-03-10".date())`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("2017-03-10"), pt("2017-03-11"), pt("2017-03-10"), pt("2017-03-10"), pt("2017-03-10")},
		},
		{
			name: "test_15",
			json: js(`["2017-03-10", "2017-03-11", "2017-03-09", "2017-03-10 00:00:00", "2017-03-10 12:34:56", "2017-03-10 01:02:03+04", "2017-03-10 03:00:00+03"]`),
			path: `$[*].date() ? (@ <  "2017-03-10".date())`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("2017-03-09"), pt("2017-03-09")},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgQueryTimeComparison(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L830-L882
	for _, tc := range []queryTestCase{
		{
			name: "test_1",
			json: js(`["12:34:00", "12:35:00", "12:36:00", "12:35:00+00", "12:35:00+01", "13:35:00+01", "2017-03-10", "2017-03-10 12:35:00", "2017-03-10 12:35:00+01"]`),
			path: `$[*].datetime() ? (@ == "12:35".datetime("HH24:MI"))`,
			err:  `exec: .datetime(template) is not yet supported`,
			// err:  `exec: cannot convert value from time to timetz without time zone usage.` + hint,
		},
		{
			name: "test_2",
			json: js(`["12:34:00", "12:35:00", "12:36:00", "12:35:00+00", "12:35:00+01", "13:35:00+01", "2017-03-10", "2017-03-10 12:35:00", "2017-03-10 12:35:00+01"]`),
			path: `$[*].datetime() ? (@ >= "12:35".datetime("HH24:MI"))`,
			err:  `exec: .datetime(template) is not yet supported`,
			// err:  `exec: cannot convert value from time to timetz without time zone usage.` + hint,
		},
		{
			name: "test_3",
			json: js(`["12:34:00", "12:35:00", "12:36:00", "12:35:00+00", "12:35:00+01", "13:35:00+01", "2017-03-10", "2017-03-10 12:35:00", "2017-03-10 12:35:00+01"]`),
			path: `$[*].datetime() ? (@ <  "12:35".datetime("HH24:MI"))`,
			err:  `exec: .datetime(template) is not yet supported`,
			// err:  `exec: cannot convert value from time to timetz without time zone usage.` + hint,
		},
		{
			name: "test_4",
			json: js(`["12:34:00", "12:35:00", "12:36:00", "12:35:00+00", "12:35:00+01", "13:35:00+01", "2017-03-10", "2017-03-10 12:35:00", "2017-03-10 12:35:00+01"]`),
			path: `$[*].datetime() ? (@ == "12:35".datetime("HH24:MI"))`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("12:35:00"), pt("12:35:00+00:00")},
			err:  `exec: .datetime(template) is not yet supported`,
		},
		{
			name: "test_5",
			json: js(`["12:34:00", "12:35:00", "12:36:00", "12:35:00+00", "12:35:00+01", "13:35:00+01", "2017-03-10", "2017-03-10 12:35:00", "2017-03-10 12:35:00+01"]`),
			path: `$[*].datetime() ? (@ >= "12:35".datetime("HH24:MI"))`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("12:35:00"), pt("12:36:00"), pt("12:35:00+00:00")},
			err:  `exec: .datetime(template) is not yet supported`,
		},
		{
			name: "test_6",
			json: js(`["12:34:00", "12:35:00", "12:36:00", "12:35:00+00", "12:35:00+01", "13:35:00+01", "2017-03-10", "2017-03-10 12:35:00", "2017-03-10 12:35:00+01"]`),
			path: `$[*].datetime() ? (@ <  "12:35".datetime("HH24:MI"))`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("12:34:00"), pt("12:35:00+01:00"), pt("13:35:00+01:00")},
			err:  `exec: .datetime(template) is not yet supported`,
		},
		{
			name: "test_7",
			json: js(`["12:34:00", "12:35:00", "12:36:00", "12:35:00+00", "12:35:00+01", "13:35:00+01", "2017-03-10 12:35:00", "2017-03-10 12:35:00+01"]`),
			path: `$[*].datetime() ? (@ == "12:35:00".time())`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("12:35:00"), pt("12:35:00+00:00")},
		},
		{
			name: "test_8",
			json: js(`["12:34:00", "12:35:00", "12:36:00", "12:35:00+00", "12:35:00+01", "13:35:00+01", "2017-03-10 12:35:00", "2017-03-10 12:35:00+01"]`),
			path: `$[*].datetime() ? (@ >= "12:35:00".time())`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("12:35:00"), pt("12:36:00"), pt("12:35:00+00:00"), pt("13:35:00+01")}, // pg excludes 13:35:00+01
		},
		{
			name: "test_9",
			json: js(`["12:34:00", "12:35:00", "12:36:00", "12:35:00+00", "12:35:00+01", "13:35:00+01", "2017-03-10 12:35:00", "2017-03-10 12:35:00+01"]`),
			path: `$[*].datetime() ? (@ <  "12:35:00".time())`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("12:34:00"), pt("12:35:00+01:00")}, // pg: also includes 13:35:00+01:00
		},
		{
			name: "test_10",
			json: js(`["12:34:00", "12:35:00", "12:36:00", "12:35:00+00", "12:35:00+01", "13:35:00+01", "2017-03-10 12:35:00", "2017-03-10 12:35:00+01"]`),
			path: `$[*].time() ? (@ == "12:35:00".time())`,
			err:  `exec: cannot convert value from timetz to time without time zone usage.` + hint,
		},
		{
			name: "test_11",
			json: js(`["12:34:00", "12:35:00", "12:36:00", "12:35:00+00", "12:35:00+01", "13:35:00+01", "2017-03-10 12:35:00", "2017-03-10 12:35:00+01"]`),
			path: `$[*].time() ? (@ >= "12:35:00".time())`,
			err:  `exec: cannot convert value from timetz to time without time zone usage.` + hint,
		},
		{
			name: "test_12",
			json: js(`["12:34:00", "12:35:00", "12:36:00", "12:35:00+00", "12:35:00+01", "13:35:00+01", "2017-03-10 12:35:00", "2017-03-10 12:35:00+01"]`),
			path: `$[*].time() ? (@ <  "12:35:00".time())`,
			err:  `exec: cannot convert value from timetz to time without time zone usage.` + hint,
		},
		{
			name: "test_13",
			json: js(`["12:34:00.123", "12:35:00.123", "12:36:00.1123", "12:35:00.1123+00", "12:35:00.123+01", "13:35:00.123+01", "2017-03-10 12:35:00.1", "2017-03-10 12:35:00.123+01"]`),
			path: `$[*].time(2) ? (@ >= "12:35:00.123".time(2))`,
			err:  `exec: cannot convert value from timetz to time without time zone usage.` + hint,
		},
		{
			name: "test_14",
			json: js(`["12:34:00", "12:35:00", "12:36:00", "12:35:00+00", "12:35:00+01", "13:35:00+01", "2017-03-10 12:35:00", "2017-03-10 12:35:00+01"]`),
			path: `$[*].time() ? (@ == "12:35:00".time())`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("12:35:00"), pt("12:35:00"), pt("12:35:00"), pt("12:35:00")},
		},
		{
			name: "test_15",
			json: js(`["12:34:00", "12:35:00", "12:36:00", "12:35:00+00", "12:35:00+01", "13:35:00+01", "2017-03-10 12:35:00", "2017-03-10 12:35:00+01"]`),
			path: `$[*].time() ? (@ >= "12:35:00".time())`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("12:35:00"), pt("12:36:00"), pt("12:35:00"), pt("12:35:00"), pt("13:35:00"), pt("12:35:00")},
		},
		{
			name: "test_16",
			json: js(`["12:34:00", "12:35:00", "12:36:00", "12:35:00+00", "12:35:00+01", "13:35:00+01", "2017-03-10 12:35:00", "2017-03-10 12:35:00+01"]`),
			path: `$[*].time() ? (@ <  "12:35:00".time())`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("12:34:00"), pt("11:35:00")},
		},
		{
			name: "test_17",
			json: js(`["12:34:00.123", "12:35:00.123", "12:36:00.1123", "12:35:00.1123+00", "12:35:00.123+01", "13:35:00.123+01", "2017-03-10 12:35:00.1", "2017-03-10 12:35:00.123+01"]`),
			path: `$[*].time(2) ? (@ >= "12:35:00.123".time(2))`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("12:35:00.12"), pt("12:36:00.11"), pt("12:35:00.12"), pt("13:35:00.12")},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgQueryTimeTZComparison(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L885-L937
	// All ` +1`s replaced with `+01`.
	for _, tc := range []queryTestCase{
		{
			name: "test_1",
			json: js(`["12:34:00+01", "12:35:00+01", "12:36:00+01", "12:35:00+02", "12:35:00-02", "10:35:00", "11:35:00", "12:35:00", "2017-03-10", "2017-03-10 12:35:00", "2017-03-10 12:35:00+01"]`),
			path: `$[*].datetime() ? (@ == "12:35 +1".datetime("HH24:MI TZH"))`,
			err:  `exec: .datetime(template) is not yet supported`,
			// err:  `exec: cannot convert value from time to timetz without time zone usage.` + hint,
		},
		{
			name: "test_2",
			json: js(`["12:34:00+01", "12:35:00+01", "12:36:00+01", "12:35:00+02", "12:35:00-02", "10:35:00", "11:35:00", "12:35:00", "2017-03-10", "2017-03-10 12:35:00", "2017-03-10 12:35:00+01"]`),
			path: `$[*].datetime() ? (@ >= "12:35 +1".datetime("HH24:MI TZH"))`,
			err:  `exec: .datetime(template) is not yet supported`,
			// err:  `exec: cannot convert value from time to timetz without time zone usage.` + hint,
		},
		{
			name: "test_3",
			json: js(`["12:34:00+01", "12:35:00+01", "12:36:00+01", "12:35:00+02", "12:35:00-02", "10:35:00", "11:35:00", "12:35:00", "2017-03-10", "2017-03-10 12:35:00", "2017-03-10 12:35:00+01"]`),
			path: `$[*].datetime() ? (@ <  "12:35 +1".datetime("HH24:MI TZH"))`,
			err:  `exec: .datetime(template) is not yet supported`,
			// err:  `exec: cannot convert value from time to timetz without time zone usage.` + hint,
		},
		{
			name: "test_4",
			json: js(`["12:34:00+01", "12:35:00+01", "12:36:00+01", "12:35:00+02", "12:35:00-02", "10:35:00", "11:35:00", "12:35:00", "2017-03-10", "2017-03-10 12:35:00", "2017-03-10 12:35:00+01"]`),
			path: `$[*].datetime() ? (@ == "12:35 +1".datetime("HH24:MI TZH"))`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("12:35:00+01:00")},
			err:  `exec: .datetime(template) is not yet supported`,
		},
		{
			name: "test_5",
			json: js(`["12:34:00+01", "12:35:00+01", "12:36:00+01", "12:35:00+02", "12:35:00-02", "10:35:00", "11:35:00", "12:35:00", "2017-03-10", "2017-03-10 12:35:00", "2017-03-10 12:35:00+01"]`),
			path: `$[*].datetime() ? (@ >= "12:35 +1".datetime("HH24:MI TZH"))`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("12:35:00+01:00"), pt("12:36:00+01:00"), pt("12:35:00-02:00"), pt("11:35:00"), pt("12:35:00")},
			err:  `exec: .datetime(template) is not yet supported`,
		},
		{
			name: "test_6",
			json: js(`["12:34:00+01", "12:35:00+01", "12:36:00+01", "12:35:00+02", "12:35:00-02", "10:35:00", "11:35:00", "12:35:00", "2017-03-10", "2017-03-10 12:35:00", "2017-03-10 12:35:00+01"]`),
			path: `$[*].datetime() ? (@ <  "12:35 +1".datetime("HH24:MI TZH"))`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("12:34:00+01:00"), pt("12:35:00+02:00"), pt("10:35:00")},
			err:  `exec: .datetime(template) is not yet supported`,
		},
		{
			name: "test_7",
			json: js(`["12:34:00+01", "12:35:00+01", "12:36:00+01", "12:35:00+02", "12:35:00-02", "10:35:00", "11:35:00", "12:35:00", "2017-03-10 12:35:00+01"]`),
			path: `$[*].datetime() ? (@ == "12:35:00+01".time_tz())`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("12:35:00+01:00")},
		},
		{
			name: "test_8",
			json: js(`["12:34:00+01", "12:35:00+01", "12:36:00+01", "12:35:00+02", "12:35:00-02", "10:35:00", "11:35:00", "12:35:00", "2017-03-10 12:35:00+01"]`),
			path: `$[*].datetime() ? (@ >= "12:35:00+01".time_tz())`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("12:35:00+01:00"), pt("12:36:00+01:00"), pt("12:35:00-02:00"), pt("12:35:00")},
			// pg: has 11:35:00 for fourth item because UTC < +1.
			// pg: Has fifth item, 12:35:00, because the "2017-03" timestamptz does not preserve the offset.
		},
		{
			name: "test_9",
			json: js(`["12:34:00+01", "12:35:00+01", "12:36:00+01", "12:35:00+02", "12:35:00-02", "10:35:00", "11:35:00", "12:35:00", "2017-03-10 12:35:00+01"]`),
			path: `$[*].datetime() ? (@ <  "12:35:00+01".time_tz())`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("12:34:00+01:00"), pt("12:35:00+02:00"), pt("10:35:00"), pt("11:35:00")},
			// pg: does not have 11:35:00
		},
		{
			name: "test_10",
			json: js(`["12:34:00+01", "12:35:00+01", "12:36:00+01", "12:35:00+02", "12:35:00-02", "10:35:00", "11:35:00", "12:35:00", "2017-03-10 12:35:00+01"]`),
			path: `$[*].time_tz() ? (@ == "12:35:00+01".time_tz())`,
			err:  `exec: cannot convert value from time to timetz without time zone usage.` + hint,
		},
		{
			name: "test_11",
			json: js(`["12:34:00+01", "12:35:00+01", "12:36:00+01", "12:35:00+02", "12:35:00-02", "10:35:00", "11:35:00", "12:35:00", "2017-03-10 12:35:00+01"]`),
			path: `$[*].time_tz() ? (@ >= "12:35:00+01".time_tz())`,
			err:  `exec: cannot convert value from time to timetz without time zone usage.` + hint,
		},
		{
			name: "test_12",
			json: js(`["12:34:00+01", "12:35:00+01", "12:36:00+01", "12:35:00+02", "12:35:00-02", "10:35:00", "11:35:00", "12:35:00", "2017-03-10 12:35:00+01"]`),
			path: `$[*].time_tz() ? (@ <  "12:35:00+01".time_tz())`,
			err:  `exec: cannot convert value from time to timetz without time zone usage.` + hint,
		},
		{
			name: "test_13",
			json: js(`["12:34:00.123+01", "12:35:00.123+01", "12:36:00.1123+01", "12:35:00.1123+02", "12:35:00.123-02", "10:35:00.123", "11:35:00.1", "12:35:00.123", "2017-03-10 12:35:00.123 +1"]`),
			path: `$[*].time_tz(2) ? (@ >= "12:35:00.123 +1".time_tz(2))`,
			err:  `exec: cannot convert value from time to timetz without time zone usage.` + hint,
		},
		{
			name: "test_14",
			json: js(`["12:34:00+01", "12:35:00+01", "12:36:00+01", "12:35:00+02", "12:35:00-02", "10:35:00", "11:35:00", "12:35:00", "2017-03-10 12:35:00+01"]`),
			path: `$[*].time_tz() ? (@ == "12:35:00+01".time_tz())`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("12:35:00+01:00"), pt("12:35:00+01:00")},
			// pg: does not include second entry, because `12:35:00+01".time_tz()` preserves the offset and timestamptz (the 2017 value) does not.
		},
		{
			name: "test_15",
			json: js(`["12:34:00+01", "12:35:00+01", "12:36:00+01", "12:35:00+02", "12:35:00-02", "10:35:00", "11:35:00", "12:35:00", "2017-03-10 12:35:00+01"]`),
			path: `$[*].time_tz() ? (@ >= "12:35:00+01".time_tz())`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("12:35:00+01:00"), pt("12:36:00+01:00"), pt("12:35:00-02:00"), pt("12:35:00Z"), pt("12:35:00+01:00")},
			// pg: fourth item is 11:35:00+00:00, removed here because 2:35:00+01".time_tz() preserves the offset so they are not equal
			// pg: fifth item is "12:35:00+00:00" but we have 12:35:00Z.
			// pg: sixth item is 11:35:00+00:00 because the does not preserve the offset.
		},
		{
			name: "test_16",
			json: js(`["12:34:00+01", "12:35:00+01", "12:36:00+01", "12:35:00+02", "12:35:00-02", "10:35:00", "11:35:00", "12:35:00", "2017-03-10 12:35:00+01"]`),
			path: `$[*].time_tz() ? (@ <  "12:35:00+01".time_tz())`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("12:34:00+01:00"), pt("12:35:00+02:00"), pt("10:35:00Z"), pt("11:35:00Z")},
			// pg: does not have third and fourth because "10:35:00" and "11:35:00" are UTC and 12:35:00+01 is not.
		},
		{
			name: "test_17",
			json: js(`["12:34:00.123+01", "12:35:00.123+01", "12:36:00.1123+01", "12:35:00.1123+02", "12:35:00.123-02", "10:35:00.123", "11:35:00.1", "12:35:00.123", "2017-03-10 12:35:00.123+01"]`),
			path: `$[*].time_tz(2) ? (@ >= "12:35:00.123+01".time_tz(2))`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("12:35:00.12+01:00"), pt("12:36:00.11+01:00"), pt("12:35:00.12-02:00"), pt("12:35:00.12Z"), pt("12:35:00.12+01:00")},
			// pg: has 12:35:00.12+00:00 for the fourth item but our display favors Z.
			// pg: has 11:35:00.12+00:00 for the fifth item because timestamptz does not preserve the time zone but we do.
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgQueryTimestampComparison(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L939-L991
	for _, tc := range []queryTestCase{
		{
			name: "test_1",
			json: js(`["2017-03-10 12:34:00", "2017-03-10 12:35:00", "2017-03-10 12:36:00", "2017-03-10 12:35:00+01", "2017-03-10 13:35:00+01", "2017-03-10 12:35:00-01", "2017-03-10", "2017-03-11", "12:34:56", "12:34:56+01"]`),
			path: `$[*].datetime() ? (@ == "10.03.2017 12:35".datetime("dd.mm.yyyy HH24:MI"))`,
			err:  `exec: .datetime(template) is not yet supported`,
			// err:  `exec: cannot convert value from timestamp to timestamptz without time zone usage.` + hint,
		},
		{
			name: "test_2",
			json: js(`["2017-03-10 12:34:00", "2017-03-10 12:35:00", "2017-03-10 12:36:00", "2017-03-10 12:35:00+01", "2017-03-10 13:35:00+01", "2017-03-10 12:35:00-01", "2017-03-10", "2017-03-11", "12:34:56", "12:34:56+01"]`),
			path: `$[*].datetime() ? (@ >= "10.03.2017 12:35".datetime("dd.mm.yyyy HH24:MI"))`,
			err:  `exec: .datetime(template) is not yet supported`,
			// err:  `exec: cannot convert value from timestamp to timestamptz without time zone usage.` + hint,
		},
		{
			name: "test_3",
			json: js(`["2017-03-10 12:34:00", "2017-03-10 12:35:00", "2017-03-10 12:36:00", "2017-03-10 12:35:00+01", "2017-03-10 13:35:00+01", "2017-03-10 12:35:00-01", "2017-03-10", "2017-03-11", "12:34:56", "12:34:56+01"]`),
			path: `$[*].datetime() ? (@ < "10.03.2017 12:35".datetime("dd.mm.yyyy HH24:MI"))`,
			err:  `exec: .datetime(template) is not yet supported`,
			// err:  `exec: cannot convert value from timestamp to timestamptz without time zone usage.` + hint,
		},
		{
			name: "test_4",
			json: js(`["2017-03-10 12:34:00", "2017-03-10 12:35:00", "2017-03-10 12:36:00", "2017-03-10 12:35:00+01", "2017-03-10 13:35:00+01", "2017-03-10 12:35:00-01", "2017-03-10", "2017-03-11", "12:34:56", "12:34:56+01"]`),
			path: `$[*].datetime() ? (@ == "10.03.2017 12:35".datetime("dd.mm.yyyy HH24:MI"))`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("2017-03-10T12:35:00"), pt("2017-03-10T13:35:00+01:00")},
			err:  `exec: .datetime(template) is not yet supported`,
		},
		{
			name: "test_5",
			json: js(`["2017-03-10 12:34:00", "2017-03-10 12:35:00", "2017-03-10 12:36:00", "2017-03-10 12:35:00+01", "2017-03-10 13:35:00+01", "2017-03-10 12:35:00-01", "2017-03-10", "2017-03-11", "12:34:56", "12:34:56+01"]`),
			path: `$[*].datetime() ? (@ >= "10.03.2017 12:35".datetime("dd.mm.yyyy HH24:MI"))`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("2017-03-10T12:35:00"), pt("2017-03-10T12:36:00"), pt("2017-03-10T13:35:00+01:00"), pt("2017-03-10T12:35:00-01:00"), pt("2017-03-11")},
			err:  `exec: .datetime(template) is not yet supported`,
		},
		{
			name: "test_6",
			json: js(`["2017-03-10 12:34:00", "2017-03-10 12:35:00", "2017-03-10 12:36:00", "2017-03-10 12:35:00+01", "2017-03-10 13:35:00+01", "2017-03-10 12:35:00-01", "2017-03-10", "2017-03-11", "12:34:56", "12:34:56+01"]`),
			path: `$[*].datetime() ? (@ < "10.03.2017 12:35".datetime("dd.mm.yyyy HH24:MI"))`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("2017-03-10T12:34:00"), pt("2017-03-10T12:35:00+01:00"), pt("2017-03-10")},
			err:  `exec: .datetime(template) is not yet supported`,
		},
		{
			name: "test_7",
			json: js(`["2017-03-10 12:34:00", "2017-03-10 12:35:00", "2017-03-10 12:36:00", "2017-03-10 12:35:00+01", "2017-03-10 13:35:00+01", "2017-03-10 12:35:00-01", "2017-03-10", "2017-03-11"]`),
			path: `$[*].datetime() ? (@ == "2017-03-10 12:35:00".timestamp())`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("2017-03-10T12:35:00"), pt("2017-03-10T13:35:00+01:00")},
		},
		{
			name: "test_8",
			json: js(`["2017-03-10 12:34:00", "2017-03-10 12:35:00", "2017-03-10 12:36:00", "2017-03-10 12:35:00+01", "2017-03-10 13:35:00+01", "2017-03-10 12:35:00-01", "2017-03-10", "2017-03-11"]`),
			path: `$[*].datetime() ? (@ >= "2017-03-10 12:35:00".timestamp())`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("2017-03-10T12:35:00"), pt("2017-03-10T12:36:00"), pt("2017-03-10T13:35:00+01:00"), pt("2017-03-10T12:35:00-01:00"), pt("2017-03-11")},
		},
		{
			name: "test_9",
			json: js(`["2017-03-10 12:34:00", "2017-03-10 12:35:00", "2017-03-10 12:36:00", "2017-03-10 12:35:00+01", "2017-03-10 13:35:00+01", "2017-03-10 12:35:00-01", "2017-03-10", "2017-03-11"]`),
			path: `$[*].datetime() ? (@ < "2017-03-10 12:35:00".timestamp())`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("2017-03-10T12:34:00"), pt("2017-03-10T12:35:00+01:00"), pt("2017-03-10")},
		},
		{
			name: "test_10",
			json: js(`["2017-03-10 12:34:00", "2017-03-10 12:35:00", "2017-03-10 12:36:00", "2017-03-10 12:35:00+01", "2017-03-10 13:35:00+01", "2017-03-10 12:35:00-01", "2017-03-10", "2017-03-11"]`),
			path: `$[*].timestamp() ? (@ == "2017-03-10 12:35:00".timestamp())`,
			err:  `exec: cannot convert value from timestamptz to timestamp without time zone usage.` + hint,
		},
		{
			name: "test_11",
			json: js(`["2017-03-10 12:34:00", "2017-03-10 12:35:00", "2017-03-10 12:36:00", "2017-03-10 12:35:00+01", "2017-03-10 13:35:00+01", "2017-03-10 12:35:00-01", "2017-03-10", "2017-03-11"]`),
			path: `$[*].timestamp() ? (@ >= "2017-03-10 12:35:00".timestamp())`,
			err:  `exec: cannot convert value from timestamptz to timestamp without time zone usage.` + hint,
		},
		{
			name: "test_12",
			json: js(`["2017-03-10 12:34:00", "2017-03-10 12:35:00", "2017-03-10 12:36:00", "2017-03-10 12:35:00+01", "2017-03-10 13:35:00+01", "2017-03-10 12:35:00-01", "2017-03-10", "2017-03-11"]`),
			path: `$[*].timestamp() ? (@ < "2017-03-10 12:35:00".timestamp())`,
			err:  `exec: cannot convert value from timestamptz to timestamp without time zone usage.` + hint,
		},
		{
			name: "test_13",
			json: js(`["2017-03-10 12:34:00.123", "2017-03-10 12:35:00.123", "2017-03-10 12:36:00.1123", "2017-03-10 12:35:00.1123+01", "2017-03-10 13:35:00.123+01", "2017-03-10 12:35:00.1-01", "2017-03-10", "2017-03-11"]`),
			path: `$[*].timestamp(2) ? (@ >= "2017-03-10 12:35:00.123".timestamp(2))`,
			err:  `exec: cannot convert value from timestamptz to timestamp without time zone usage.` + hint,
		},
		{
			name: "test_14",
			json: js(`["2017-03-10 12:34:00", "2017-03-10 12:35:00", "2017-03-10 12:36:00", "2017-03-10 12:35:00+01", "2017-03-10 13:35:00+01", "2017-03-10 12:35:00-01", "2017-03-10", "2017-03-11"]`),
			path: `$[*].timestamp() ? (@ == "2017-03-10 12:35:00".timestamp())`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("2017-03-10T12:35:00"), pt("2017-03-10T12:35:00")},
		},
		{
			name: "test_15",
			json: js(`["2017-03-10 12:34:00", "2017-03-10 12:35:00", "2017-03-10 12:36:00", "2017-03-10 12:35:00+01", "2017-03-10 13:35:00+01", "2017-03-10 12:35:00-01", "2017-03-10", "2017-03-11"]`),
			path: `$[*].timestamp() ? (@ >= "2017-03-10 12:35:00".timestamp())`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("2017-03-10T12:35:00"), pt("2017-03-10T12:36:00"), pt("2017-03-10T12:35:00"), pt("2017-03-10T13:35:00"), pt("2017-03-11T00:00:00")},
		},
		{
			name: "test_16",
			json: js(`["2017-03-10 12:34:00", "2017-03-10 12:35:00", "2017-03-10 12:36:00", "2017-03-10 12:35:00+01", "2017-03-10 13:35:00+01", "2017-03-10 12:35:00-01", "2017-03-10", "2017-03-11"]`),
			path: `$[*].timestamp() ? (@ < "2017-03-10 12:35:00".timestamp())`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("2017-03-10T12:34:00"), pt("2017-03-10T11:35:00"), pt("2017-03-10T00:00:00")},
		},
		{
			name: "test_17",
			json: js(`["2017-03-10 12:34:00.123", "2017-03-10 12:35:00.123", "2017-03-10 12:36:00.1123", "2017-03-10 12:35:00.1123+01", "2017-03-10 13:35:00.123+01", "2017-03-10 12:35:00.1-01", "2017-03-10", "2017-03-11"]`),
			path: `$[*].timestamp(2) ? (@ >= "2017-03-10 12:35:00.123".timestamp(2))`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("2017-03-10T12:35:00.12"), pt("2017-03-10T12:36:00.11"), pt("2017-03-10T12:35:00.12"), pt("2017-03-10T13:35:00.1"), pt("2017-03-11T00:00:00")},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgQueryTimestampTZComparison(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L993-L1045
	for _, tc := range []queryTestCase{
		{
			name: "test_1",
			json: js(`["2017-03-10 12:34:00+01", "2017-03-10 12:35:00+01", "2017-03-10 12:36:00+01", "2017-03-10 12:35:00+02", "2017-03-10 12:35:00-02", "2017-03-10 10:35:00", "2017-03-10 11:35:00", "2017-03-10 12:35:00", "2017-03-10", "2017-03-11", "12:34:56", "12:34:56+01"]`),
			path: `$[*].datetime() ? (@ == "10.03.2017 12:35 +1".datetime("dd.mm.yyyy HH24:MI TZH"))`,
			err:  `exec: .datetime(template) is not yet supported`,
			// err:  `exec: cannot convert value from timestamp to timestamptz without time zone usage.` + hint,
		},
		{
			name: "test_2",
			json: js(`["2017-03-10 12:34:00+01", "2017-03-10 12:35:00+01", "2017-03-10 12:36:00+01", "2017-03-10 12:35:00+02", "2017-03-10 12:35:00-02", "2017-03-10 10:35:00", "2017-03-10 11:35:00", "2017-03-10 12:35:00", "2017-03-10", "2017-03-11", "12:34:56", "12:34:56+01"]`),
			path: `$[*].datetime() ? (@ >= "10.03.2017 12:35 +1".datetime("dd.mm.yyyy HH24:MI TZH"))`,
			err:  `exec: .datetime(template) is not yet supported`,
			// err:  `exec: cannot convert value from timestamp to timestamptz without time zone usage.` + hint,
		},
		{
			name: "test_3",
			json: js(`["2017-03-10 12:34:00+01", "2017-03-10 12:35:00+01", "2017-03-10 12:36:00+01", "2017-03-10 12:35:00+02", "2017-03-10 12:35:00-02", "2017-03-10 10:35:00", "2017-03-10 11:35:00", "2017-03-10 12:35:00", "2017-03-10", "2017-03-11", "12:34:56", "12:34:56+01"]`),
			path: `$[*].datetime() ? (@ < "10.03.2017 12:35 +1".datetime("dd.mm.yyyy HH24:MI TZH"))`,
			err:  `exec: .datetime(template) is not yet supported`,
			// err:  `exec: cannot convert value from timestamp to timestamptz without time zone usage.` + hint,
		},
		{
			name: "test_4",
			json: js(`["2017-03-10 12:34:00+01", "2017-03-10 12:35:00+01", "2017-03-10 12:36:00+01", "2017-03-10 12:35:00+02", "2017-03-10 12:35:00-02", "2017-03-10 10:35:00", "2017-03-10 11:35:00", "2017-03-10 12:35:00", "2017-03-10", "2017-03-11", "12:34:56", "12:34:56+01"]`),
			path: `$[*].datetime() ? (@ == "10.03.2017 12:35 +1".datetime("dd.mm.yyyy HH24:MI TZH"))`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("2017-03-10T12:35:00+01:00"), pt("2017-03-10T11:35:00")},
			err:  `exec: .datetime(template) is not yet supported`,
		},
		{
			name: "test_5",
			json: js(`["2017-03-10 12:34:00+01", "2017-03-10 12:35:00+01", "2017-03-10 12:36:00+01", "2017-03-10 12:35:00+02", "2017-03-10 12:35:00-02", "2017-03-10 10:35:00", "2017-03-10 11:35:00", "2017-03-10 12:35:00", "2017-03-10", "2017-03-11", "12:34:56", "12:34:56+01"]`),
			path: `$[*].datetime() ? (@ >= "10.03.2017 12:35 +1".datetime("dd.mm.yyyy HH24:MI TZH"))`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("2017-03-10T12:35:00+01:00"), pt("2017-03-10T12:36:00+01:00"), pt("2017-03-10T12:35:00-02:00"), pt("2017-03-10T11:35:00"), pt("2017-03-10T12:35:00"), pt("2017-03-11")},
			err:  `exec: .datetime(template) is not yet supported`,
		},
		{
			name: "test_6",
			json: js(`["2017-03-10 12:34:00+01", "2017-03-10 12:35:00+01", "2017-03-10 12:36:00+01", "2017-03-10 12:35:00+02", "2017-03-10 12:35:00-02", "2017-03-10 10:35:00", "2017-03-10 11:35:00", "2017-03-10 12:35:00", "2017-03-10", "2017-03-11", "12:34:56", "12:34:56+01"]`),
			path: `$[*].datetime() ? (@ < "10.03.2017 12:35 +1".datetime("dd.mm.yyyy HH24:MI TZH"))`,
			opt:  []Option{WithTZ()},
			exp:  []any{pt("2017-03-10T12:34:00+01:00"), pt("2017-03-10T12:35:00+02:00"), pt("2017-03-10T10:35:00"), pt("2017-03-10")},
			err:  `exec: .datetime(template) is not yet supported`,
		},
		{
			name: "test_7",
			json: js(`["2017-03-10 12:34:00+01", "2017-03-10 12:35:00+01", "2017-03-10 12:36:00+01", "2017-03-10 12:35:00+02", "2017-03-10 12:35:00-02", "2017-03-10 10:35:00", "2017-03-10 11:35:00", "2017-03-10 12:35:00", "2017-03-10", "2017-03-11"]`),
			path: `$[*].datetime() ? (@ == "2017-03-10 12:35:00+01".timestamp_tz())`, // pg: 2017-03-10 12:35:00 +1
			opt:  []Option{WithTZ()},
			exp:  []any{pt("2017-03-10T12:35:00+01:00"), pt("2017-03-10T11:35:00")},
		},
		{
			name: "test_8",
			json: js(`["2017-03-10 12:34:00+01", "2017-03-10 12:35:00+01", "2017-03-10 12:36:00+01", "2017-03-10 12:35:00+02", "2017-03-10 12:35:00-02", "2017-03-10 10:35:00", "2017-03-10 11:35:00", "2017-03-10 12:35:00", "2017-03-10", "2017-03-11"]`),
			path: `$[*].datetime() ? (@ >= "2017-03-10 12:35:00+01".timestamp_tz())`, // pg: 2017-03-10 12:35:00 +1
			opt:  []Option{WithTZ()},
			exp:  []any{pt("2017-03-10T12:35:00+01:00"), pt("2017-03-10T12:36:00+01:00"), pt("2017-03-10T12:35:00-02:00"), pt("2017-03-10T11:35:00"), pt("2017-03-10T12:35:00"), pt("2017-03-11")},
		},
		{
			name: "test_9",
			json: js(`["2017-03-10 12:34:00+01", "2017-03-10 12:35:00+01", "2017-03-10 12:36:00+01", "2017-03-10 12:35:00+02", "2017-03-10 12:35:00-02", "2017-03-10 10:35:00", "2017-03-10 11:35:00", "2017-03-10 12:35:00", "2017-03-10", "2017-03-11"]`),
			path: `$[*].datetime() ? (@ < "2017-03-10 12:35:00+01".timestamp_tz())`, // pg: 2017-03-10 12:35:00 +1
			opt:  []Option{WithTZ()},
			exp:  []any{pt("2017-03-10T12:34:00+01:00"), pt("2017-03-10T12:35:00+02:00"), pt("2017-03-10T10:35:00"), pt("2017-03-10")},
		},
		{
			name: "test_10",
			json: js(`["2017-03-10 12:34:00+01", "2017-03-10 12:35:00+01", "2017-03-10 12:36:00+01", "2017-03-10 12:35:00+02", "2017-03-10 12:35:00-02", "2017-03-10 10:35:00", "2017-03-10 11:35:00", "2017-03-10 12:35:00", "2017-03-10", "2017-03-11"]`),
			path: `$[*].timestamp_tz() ? (@ == "2017-03-10 12:35:00+01".timestamp_tz())`, // pg: 2017-03-10 12:35:00 +1
			err:  `exec: cannot convert value from timestamp to timestamptz without time zone usage.` + hint,
		},
		{
			name: "test_11",
			json: js(`["2017-03-10 12:34:00+01", "2017-03-10 12:35:00+01", "2017-03-10 12:36:00+01", "2017-03-10 12:35:00+02", "2017-03-10 12:35:00-02", "2017-03-10 10:35:00", "2017-03-10 11:35:00", "2017-03-10 12:35:00", "2017-03-10", "2017-03-11"]`),
			path: `$[*].timestamp_tz() ? (@ >= "2017-03-10 12:35:00+01".timestamp_tz())`, // pg: 2017-03-10 12:35:00 +1
			err:  `exec: cannot convert value from timestamp to timestamptz without time zone usage.` + hint,
		},
		{
			name: "test_12",
			json: js(`["2017-03-10 12:34:00+01", "2017-03-10 12:35:00+01", "2017-03-10 12:36:00+01", "2017-03-10 12:35:00+02", "2017-03-10 12:35:00-02", "2017-03-10 10:35:00", "2017-03-10 11:35:00", "2017-03-10 12:35:00", "2017-03-10", "2017-03-11"]`),
			path: `$[*].timestamp_tz() ? (@ < "2017-03-10 12:35:00+01".timestamp_tz())`, // pg: 2017-03-10 12:35:00 +1
			err:  `exec: cannot convert value from timestamp to timestamptz without time zone usage.` + hint,
		},
		{
			name: "test_13",
			json: js(`["2017-03-10 12:34:00.123+01", "2017-03-10 12:35:00.123+01", "2017-03-10 12:36:00.1123+01", "2017-03-10 12:35:00.1123+02", "2017-03-10 12:35:00.123-02", "2017-03-10 10:35:00.123", "2017-03-10 11:35:00.1", "2017-03-10 12:35:00.123", "2017-03-10", "2017-03-11"]`),
			path: `$[*].timestamp_tz(2) ? (@ >= "2017-03-10 12:35:00.123+01".timestamp_tz(2))`, // pg" 2017-03-10 12:35:00.123 +1
			err:  `exec: cannot convert value from timestamp to timestamptz without time zone usage.` + hint,
		},
		{
			name: "test_14",
			json: js(`["2017-03-10 12:34:00+01", "2017-03-10 12:35:00+01", "2017-03-10 12:36:00+01", "2017-03-10 12:35:00+02", "2017-03-10 12:35:00-02", "2017-03-10 10:35:00", "2017-03-10 11:35:00", "2017-03-10 12:35:00", "2017-03-10", "2017-03-11"]`),
			path: `$[*].timestamp_tz() ? (@ == "2017-03-10 12:35:00+01".timestamp_tz())`, // pg: 2017-03-10 12:35:00 +1
			opt:  []Option{WithTZ()},
			exp:  []any{pt("2017-03-10T12:35:00+01:00"), pt("2017-03-10T11:35:00Z")}, // pg: +00:00 instead of Z
		},
		{
			name: "test_15",
			json: js(`["2017-03-10 12:34:00+01", "2017-03-10 12:35:00+01", "2017-03-10 12:36:00+01", "2017-03-10 12:35:00+02", "2017-03-10 12:35:00-02", "2017-03-10 10:35:00", "2017-03-10 11:35:00", "2017-03-10 12:35:00", "2017-03-10", "2017-03-11"]`),
			path: `$[*].timestamp_tz() ? (@ >= "2017-03-10 12:35:00+01".timestamp_tz())`, // pg: 2017-03-10 12:35:00 +1
			opt:  []Option{WithTZ()},
			exp:  []any{pt("2017-03-10T12:35:00+01:00"), pt("2017-03-10T12:36:00+01:00"), pt("2017-03-10T12:35:00-02:00"), pt("2017-03-10T11:35:00Z"), pt("2017-03-10T12:35:00Z"), pt("2017-03-11T00:00:00Z")}, // pg: +00:00 instead of Z
		},
		{
			name: "test_16",
			json: js(`["2017-03-10 12:34:00+01", "2017-03-10 12:35:00+01", "2017-03-10 12:36:00+01", "2017-03-10 12:35:00+02", "2017-03-10 12:35:00-02", "2017-03-10 10:35:00", "2017-03-10 11:35:00", "2017-03-10 12:35:00", "2017-03-10", "2017-03-11"]`),
			path: `$[*].timestamp_tz() ? (@ < "2017-03-10 12:35:00+01".timestamp_tz())`, // pg: 2017-03-10 12:35:00 +1
			opt:  []Option{WithTZ()},
			exp:  []any{pt("2017-03-10T12:34:00+01:00"), pt("2017-03-10T12:35:00+02:00"), pt("2017-03-10T10:35:00Z"), pt("2017-03-10T00:00:00Z")}, // pg: +00:00 instead of Z
		},
		{
			name: "test_17",
			json: js(`["2017-03-10 12:34:00.123+01", "2017-03-10 12:35:00.123+01", "2017-03-10 12:36:00.1123+01", "2017-03-10 12:35:00.1123+02", "2017-03-10 12:35:00.123-02", "2017-03-10 10:35:00.123", "2017-03-10 11:35:00.1", "2017-03-10 12:35:00.123", "2017-03-10", "2017-03-11"]`),
			path: `$[*].timestamp_tz(2) ? (@ >= "2017-03-10 12:35:00.123+01".timestamp_tz(2))`, // pg: 2017-03-10 12:35:00.123 +1
			opt:  []Option{WithTZ()},
			exp:  []any{pt("2017-03-10T12:35:00.12+01:00"), pt("2017-03-10T12:36:00.11+01:00"), pt("2017-03-10T12:35:00.12-02:00"), pt("2017-03-10T12:35:00.12Z"), pt("2017-03-11T00:00:00Z")}, // pg: +00:00 instead of Z
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgQueryComparisonOverflow(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L1048-L1049
	for _, tc := range []queryTestCase{
		{
			name: "test_1",
			json: js(`"1000000-01-01"`),
			path: `$.datetime() > "2020-01-01 12:00:00".datetime()`,
			exp:  []any{nil}, // pg: returns true, because it handles years 9999 but Go does not
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgQueryOperators(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L1053-L1065
	for _, tc := range []queryTestCase{
		{
			name: "test_1",
			json: js(`[{"a": 1}, {"a": 2}]`),
			path: `$[*]`,
			exp:  []any{js(`{"a": 1}`), js(`{"a": 2}`)},
		},
		{
			name: "test_2",
			json: js(`[{"a": 1}, {"a": 2}]`),
			path: `$[*] ? (@.a > 10)`,
			exp:  []any{},
		},
		{
			name: "test_3",
			json: js(`[{"a": 1}]`),
			path: `$undefined_var`,
			err:  `exec: could not find jsonpath variable "undefined_var"`,
		},
		// pg: tests 4-10 use jsonb_path_query_array but our Query() always
		// returns a slice.
		{
			name: "test_4",
			json: js(`[{"a": 1}]`),
			path: `false`,
			exp:  []any{false},
		},
		{
			name: "test_5",
			json: js(`[{"a": 1}, {"a": 2}, {}]`),
			path: `strict $[*].a`,
			err:  `exec: JSON object does not contain key "a"`,
		},
		{
			name: "test_6",
			json: js(`[{"a": 1}, {"a": 2}]`),
			path: `$[*].a`,
			exp:  []any{float64(1), float64(2)},
		},
		{
			name: "test_7",
			json: js(`[{"a": 1}, {"a": 2}]`),
			path: `$[*].a ? (@ == 1)`,
			exp:  []any{float64(1)},
		},
		{
			name: "test_8",
			json: js(`[{"a": 1}, {"a": 2}]`),
			path: `$[*].a ? (@ > 10)`,
			exp:  []any{},
		},
		{
			name: "test_9",
			json: js(`[{"a": 1}, {"a": 2}, {"a": 3}, {"a": 5}]`),
			path: `$[*].a ? (@ > $min && @ < $max)`,
			opt:  []Option{WithVars(jv(`{"min": 1, "max": 4}`))},
			exp:  []any{float64(2), float64(3)},
		},
		{
			name: "test_10",
			json: js(`[{"a": 1}, {"a": 2}, {"a": 3}, {"a": 5}]`),
			path: `$[*].a ? (@ > $min && @ < $max)`,
			opt:  []Option{WithVars(jv(`{"min": 3, "max": 4}`))},
			exp:  []any{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgFirst(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L1067-L1075
	for _, tc := range []firstTestCase{
		{
			name: "test_1",
			json: js(`[{"a": 1}, {"a": 2}, {}]`),
			path: `strict $[*].a`,
			err:  `exec: JSON object does not contain key "a"`,
		},
		{
			name: "test_2",
			json: js(`[{"a": 1}, {"a": 2}, {}]`),
			path: `strict $[*].a`,
			opt:  []Option{WithSilent()},
			exp:  float64(1),
		},
		{
			name: "test_3",
			json: js(`[{"a": 1}, {"a": 2}]`),
			path: `$[*].a`,
			exp:  float64(1),
		},
		{
			name: "test_4",
			json: js(`[{"a": 1}, {"a": 2}]`),
			path: `$[*].a ? (@ == 1)`,
			exp:  float64(1),
		},
		{
			name: "test_5",
			json: js(`[{"a": 1}, {"a": 2}]`),
			path: `$[*].a ? (@ > 10)`,
			exp:  nil,
		},
		{
			name: "test_6",
			json: js(`[{"a": 1}, {"a": 2}, {"a": 3}, {"a": 5}]`),
			path: `$[*].a ? (@ > $min && @ < $max)`,
			opt:  []Option{WithVars(jv(`{"min": 1, "max": 4}`))},
			exp:  float64(2),
		},
		{
			name: "test_7",
			json: js(`[{"a": 1}, {"a": 2}, {"a": 3}, {"a": 5}]`),
			path: `$[*].a ? (@ > $min && @ < $max)`,
			opt:  []Option{WithVars(jv(`{"min": 3, "max": 4}`))},
			exp:  nil,
		},
		{
			name: "test_8",
			json: js(`[{"a": 1}]`),
			path: `$undefined_var`,
			err:  `exec: could not find jsonpath variable "undefined_var"`,
		},
		{
			name: "test_9",
			json: js(`[{"a": 1}]`),
			path: `false`,
			exp:  false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgAtQuestionOperators(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L1077-L1078
	for _, tc := range []existsTestCase{
		{
			name: "test_1",
			json: js(`[{"a": 1}, {"a": 2}]`),
			path: "$[*].a ? (@ > 1)",
			exp:  true,
		},
		{
			name: "test_2",
			json: js(`[{"a": 1}, {"a": 2}]`),
			path: "$[*] ? (@.a > 2)",
			exp:  false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.runAtQuestion(a, r)
		})
	}
}

func TestPgExistsOperators(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L1079-L1083
	for _, tc := range []existsTestCase{
		{
			name: "test_1",
			json: js(`[{"a": 1}, {"a": 2}]`),
			path: "$[*].a ? (@ > 1)",
			exp:  true,
		},
		{
			name: "test_2",
			json: js(`[{"a": 1}, {"a": 2}, {"a": 3}, {"a": 5}]`),
			path: "$[*] ? (@.a > $min && @.a < $max)",
			opt:  []Option{WithVars(jv(`{"min": 1, "max": 4}`))},
			exp:  true,
		},
		{
			name: "test_3",
			json: js(`[{"a": 1}, {"a": 2}, {"a": 3}, {"a": 5}]`),
			path: "$[*] ? (@.a > $min && @.a < $max)",
			opt:  []Option{WithVars(jv(`{"min": 3, "max": 4}`))},
			exp:  false,
		},
		{
			name: "test_4",
			json: js(`[{"a": 1}]`),
			path: "$undefined_var",
			err:  `exec: could not find jsonpath variable "undefined_var"`,
		},
		{
			name: "test_5",
			json: js(`[{"a": 1}]`),
			path: "false",
			exp:  true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgMatchOperators(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L1085-L1101
	for _, tc := range []matchTestCase{
		{
			name: "test_1",
			json: js(`true`),
			path: `$`,
			opt:  []Option{},
			exp:  true,
		},
		{
			name: "test_2",
			json: js(`false`),
			path: `$`,
			opt:  []Option{},
			exp:  false,
		},
		{
			name: "test_3",
			json: js(`null`),
			path: `$`,
			opt:  []Option{},
			exp:  nil,
		},
		{
			name: "test_4",
			json: js(`1`),
			path: `$`,
			opt:  []Option{WithSilent()},
			exp:  nil,
		},
		{
			name: "test_5",
			json: js(`1`),
			path: `$`,
			opt:  []Option{},
			err:  `exec: single boolean result is expected`,
		},
		{
			name: "test_6",
			json: js(`"a"`),
			path: `$`,
			opt:  []Option{},
			err:  `exec: single boolean result is expected`,
		},
		{
			name: "test_7",
			json: js(`{}`),
			path: `$`,
			opt:  []Option{},
			err:  `exec: single boolean result is expected`,
		},
		{
			name: "test_8",
			json: js(`[true]`),
			path: `$`,
			opt:  []Option{},
			err:  `exec: single boolean result is expected`,
		},
		{
			name: "test_9",
			json: js(`{}`),
			path: `lax $.a`,
			opt:  []Option{},
			err:  `exec: single boolean result is expected`,
		},
		{
			name: "test_10",
			json: js(`{}`),
			path: `strict $.a`,
			opt:  []Option{},
			err:  `exec: JSON object does not contain key "a"`,
		},
		{
			name: "test_11",
			json: js(`{}`),
			path: `strict $.a`,
			opt:  []Option{WithSilent()},
			exp:  nil,
		},
		{
			name: "test_12",
			json: js(`[true, true]`),
			path: `$[*]`,
			opt:  []Option{},
			err:  `exec: single boolean result is expected`,
		},
		// Tests 13 & 14 in TestPgAtAtOperators below.
		{
			name: "test_15",
			json: js(`[{"a": 1}, {"a": 2}]`),
			path: `$[*].a > 1`,
			exp:  true,
		},
		{
			name: "test_16",
			json: js(`[{"a": 1}]`),
			path: `$undefined_var`,
			err:  `exec: could not find jsonpath variable "undefined_var"`,
		},
		{
			name: "test_17",
			json: js(`[{"a": 1}]`),
			path: `false`,
			exp:  false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(a, r)
		})
	}
}

func TestPgAtAtOperators(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L1097-L1098
	for _, tc := range []matchTestCase{
		{
			name: "test_1",
			json: js(`[{"a": 1}, {"a": 2}]`),
			path: `$[*].a > 1`,
			exp:  true,
		},
		{
			name: "test_2",
			json: js(`[{"a": 1}, {"a": 2}]`),
			path: `$[*].a > 2`,
			exp:  false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.runAtAt(a, r)
		})
	}
}

func TestPgFirstStringComparison(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)

	// https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/sql/jsonb_jsonpath.sql#L1103-L1117
	i := 0
	for _, tc := range []struct {
		obj1 map[string]any
		obj2 map[string]any
		lt   bool
		le   bool
		eq   bool
		ge   bool
		gt   bool
	}{
		// Table copied from https://github.com/postgres/postgres/blob/REL_17_BETA1/src/test/regress/expected/jsonb_jsonpath.out#L4241-L4384
		{jv(`{"s": ""}`), jv(`{"s": ""}`), false, true, true, true, false},
		{jv(`{"s": ""}`), jv(`{"s": "a"}`), true, true, false, false, false},
		{jv(`{"s": ""}`), jv(`{"s": "ab"}`), true, true, false, false, false},
		{jv(`{"s": ""}`), jv(`{"s": "abc"}`), true, true, false, false, false},
		{jv(`{"s": ""}`), jv(`{"s": "abcd"}`), true, true, false, false, false},
		{jv(`{"s": ""}`), jv(`{"s": "b"}`), true, true, false, false, false},
		{jv(`{"s": ""}`), jv(`{"s": "A"}`), true, true, false, false, false},
		{jv(`{"s": ""}`), jv(`{"s": "AB"}`), true, true, false, false, false},
		{jv(`{"s": ""}`), jv(`{"s": "ABC"}`), true, true, false, false, false},
		{jv(`{"s": ""}`), jv(`{"s": "ABc"}`), true, true, false, false, false},
		{jv(`{"s": ""}`), jv(`{"s": "ABcD"}`), true, true, false, false, false},
		{jv(`{"s": ""}`), jv(`{"s": "B"}`), true, true, false, false, false},
		{jv(`{"s": "a"}`), jv(`{"s": ""}`), false, false, false, true, true},
		{jv(`{"s": "a"}`), jv(`{"s": "a"}`), false, true, true, true, false},
		{jv(`{"s": "a"}`), jv(`{"s": "ab"}`), true, true, false, false, false},
		{jv(`{"s": "a"}`), jv(`{"s": "abc"}`), true, true, false, false, false},
		{jv(`{"s": "a"}`), jv(`{"s": "abcd"}`), true, true, false, false, false},
		{jv(`{"s": "a"}`), jv(`{"s": "b"}`), true, true, false, false, false},
		{jv(`{"s": "a"}`), jv(`{"s": "A"}`), false, false, false, true, true},
		{jv(`{"s": "a"}`), jv(`{"s": "AB"}`), false, false, false, true, true},
		{jv(`{"s": "a"}`), jv(`{"s": "ABC"}`), false, false, false, true, true},
		{jv(`{"s": "a"}`), jv(`{"s": "ABc"}`), false, false, false, true, true},
		{jv(`{"s": "a"}`), jv(`{"s": "ABcD"}`), false, false, false, true, true},
		{jv(`{"s": "a"}`), jv(`{"s": "B"}`), false, false, false, true, true},
		{jv(`{"s": "ab"}`), jv(`{"s": ""}`), false, false, false, true, true},
		{jv(`{"s": "ab"}`), jv(`{"s": "a"}`), false, false, false, true, true},
		{jv(`{"s": "ab"}`), jv(`{"s": "ab"}`), false, true, true, true, false},
		{jv(`{"s": "ab"}`), jv(`{"s": "abc"}`), true, true, false, false, false},
		{jv(`{"s": "ab"}`), jv(`{"s": "abcd"}`), true, true, false, false, false},
		{jv(`{"s": "ab"}`), jv(`{"s": "b"}`), true, true, false, false, false},
		{jv(`{"s": "ab"}`), jv(`{"s": "A"}`), false, false, false, true, true},
		{jv(`{"s": "ab"}`), jv(`{"s": "AB"}`), false, false, false, true, true},
		{jv(`{"s": "ab"}`), jv(`{"s": "ABC"}`), false, false, false, true, true},
		{jv(`{"s": "ab"}`), jv(`{"s": "ABc"}`), false, false, false, true, true},
		{jv(`{"s": "ab"}`), jv(`{"s": "ABcD"}`), false, false, false, true, true},
		{jv(`{"s": "ab"}`), jv(`{"s": "B"}`), false, false, false, true, true},
		{jv(`{"s": "abc"}`), jv(`{"s": ""}`), false, false, false, true, true},
		{jv(`{"s": "abc"}`), jv(`{"s": "a"}`), false, false, false, true, true},
		{jv(`{"s": "abc"}`), jv(`{"s": "ab"}`), false, false, false, true, true},
		{jv(`{"s": "abc"}`), jv(`{"s": "abc"}`), false, true, true, true, false},
		{jv(`{"s": "abc"}`), jv(`{"s": "abcd"}`), true, true, false, false, false},
		{jv(`{"s": "abc"}`), jv(`{"s": "b"}`), true, true, false, false, false},
		{jv(`{"s": "abc"}`), jv(`{"s": "A"}`), false, false, false, true, true},
		{jv(`{"s": "abc"}`), jv(`{"s": "AB"}`), false, false, false, true, true},
		{jv(`{"s": "abc"}`), jv(`{"s": "ABC"}`), false, false, false, true, true},
		{jv(`{"s": "abc"}`), jv(`{"s": "ABc"}`), false, false, false, true, true},
		{jv(`{"s": "abc"}`), jv(`{"s": "ABcD"}`), false, false, false, true, true},
		{jv(`{"s": "abc"}`), jv(`{"s": "B"}`), false, false, false, true, true},
		{jv(`{"s": "abcd"}`), jv(`{"s": ""}`), false, false, false, true, true},
		{jv(`{"s": "abcd"}`), jv(`{"s": "a"}`), false, false, false, true, true},
		{jv(`{"s": "abcd"}`), jv(`{"s": "ab"}`), false, false, false, true, true},
		{jv(`{"s": "abcd"}`), jv(`{"s": "abc"}`), false, false, false, true, true},
		{jv(`{"s": "abcd"}`), jv(`{"s": "abcd"}`), false, true, true, true, false},
		{jv(`{"s": "abcd"}`), jv(`{"s": "b"}`), true, true, false, false, false},
		{jv(`{"s": "abcd"}`), jv(`{"s": "A"}`), false, false, false, true, true},
		{jv(`{"s": "abcd"}`), jv(`{"s": "AB"}`), false, false, false, true, true},
		{jv(`{"s": "abcd"}`), jv(`{"s": "ABC"}`), false, false, false, true, true},
		{jv(`{"s": "abcd"}`), jv(`{"s": "ABc"}`), false, false, false, true, true},
		{jv(`{"s": "abcd"}`), jv(`{"s": "ABcD"}`), false, false, false, true, true},
		{jv(`{"s": "abcd"}`), jv(`{"s": "B"}`), false, false, false, true, true},
		{jv(`{"s": "b"}`), jv(`{"s": ""}`), false, false, false, true, true},
		{jv(`{"s": "b"}`), jv(`{"s": "a"}`), false, false, false, true, true},
		{jv(`{"s": "b"}`), jv(`{"s": "ab"}`), false, false, false, true, true},
		{jv(`{"s": "b"}`), jv(`{"s": "abc"}`), false, false, false, true, true},
		{jv(`{"s": "b"}`), jv(`{"s": "abcd"}`), false, false, false, true, true},
		{jv(`{"s": "b"}`), jv(`{"s": "b"}`), false, true, true, true, false},
		{jv(`{"s": "b"}`), jv(`{"s": "A"}`), false, false, false, true, true},
		{jv(`{"s": "b"}`), jv(`{"s": "AB"}`), false, false, false, true, true},
		{jv(`{"s": "b"}`), jv(`{"s": "ABC"}`), false, false, false, true, true},
		{jv(`{"s": "b"}`), jv(`{"s": "ABc"}`), false, false, false, true, true},
		{jv(`{"s": "b"}`), jv(`{"s": "ABcD"}`), false, false, false, true, true},
		{jv(`{"s": "b"}`), jv(`{"s": "B"}`), false, false, false, true, true},
		{jv(`{"s": "A"}`), jv(`{"s": ""}`), false, false, false, true, true},
		{jv(`{"s": "A"}`), jv(`{"s": "a"}`), true, true, false, false, false},
		{jv(`{"s": "A"}`), jv(`{"s": "ab"}`), true, true, false, false, false},
		{jv(`{"s": "A"}`), jv(`{"s": "abc"}`), true, true, false, false, false},
		{jv(`{"s": "A"}`), jv(`{"s": "abcd"}`), true, true, false, false, false},
		{jv(`{"s": "A"}`), jv(`{"s": "b"}`), true, true, false, false, false},
		{jv(`{"s": "A"}`), jv(`{"s": "A"}`), false, true, true, true, false},
		{jv(`{"s": "A"}`), jv(`{"s": "AB"}`), true, true, false, false, false},
		{jv(`{"s": "A"}`), jv(`{"s": "ABC"}`), true, true, false, false, false},
		{jv(`{"s": "A"}`), jv(`{"s": "ABc"}`), true, true, false, false, false},
		{jv(`{"s": "A"}`), jv(`{"s": "ABcD"}`), true, true, false, false, false},
		{jv(`{"s": "A"}`), jv(`{"s": "B"}`), true, true, false, false, false},
		{jv(`{"s": "AB"}`), jv(`{"s": ""}`), false, false, false, true, true},
		{jv(`{"s": "AB"}`), jv(`{"s": "a"}`), true, true, false, false, false},
		{jv(`{"s": "AB"}`), jv(`{"s": "ab"}`), true, true, false, false, false},
		{jv(`{"s": "AB"}`), jv(`{"s": "abc"}`), true, true, false, false, false},
		{jv(`{"s": "AB"}`), jv(`{"s": "abcd"}`), true, true, false, false, false},
		{jv(`{"s": "AB"}`), jv(`{"s": "b"}`), true, true, false, false, false},
		{jv(`{"s": "AB"}`), jv(`{"s": "A"}`), false, false, false, true, true},
		{jv(`{"s": "AB"}`), jv(`{"s": "AB"}`), false, true, true, true, false},
		{jv(`{"s": "AB"}`), jv(`{"s": "ABC"}`), true, true, false, false, false},
		{jv(`{"s": "AB"}`), jv(`{"s": "ABc"}`), true, true, false, false, false},
		{jv(`{"s": "AB"}`), jv(`{"s": "ABcD"}`), true, true, false, false, false},
		{jv(`{"s": "AB"}`), jv(`{"s": "B"}`), true, true, false, false, false},
		{jv(`{"s": "ABC"}`), jv(`{"s": ""}`), false, false, false, true, true},
		{jv(`{"s": "ABC"}`), jv(`{"s": "a"}`), true, true, false, false, false},
		{jv(`{"s": "ABC"}`), jv(`{"s": "ab"}`), true, true, false, false, false},
		{jv(`{"s": "ABC"}`), jv(`{"s": "abc"}`), true, true, false, false, false},
		{jv(`{"s": "ABC"}`), jv(`{"s": "abcd"}`), true, true, false, false, false},
		{jv(`{"s": "ABC"}`), jv(`{"s": "b"}`), true, true, false, false, false},
		{jv(`{"s": "ABC"}`), jv(`{"s": "A"}`), false, false, false, true, true},
		{jv(`{"s": "ABC"}`), jv(`{"s": "AB"}`), false, false, false, true, true},
		{jv(`{"s": "ABC"}`), jv(`{"s": "ABC"}`), false, true, true, true, false},
		{jv(`{"s": "ABC"}`), jv(`{"s": "ABc"}`), true, true, false, false, false},
		{jv(`{"s": "ABC"}`), jv(`{"s": "ABcD"}`), true, true, false, false, false},
		{jv(`{"s": "ABC"}`), jv(`{"s": "B"}`), true, true, false, false, false},
		{jv(`{"s": "ABc"}`), jv(`{"s": ""}`), false, false, false, true, true},
		{jv(`{"s": "ABc"}`), jv(`{"s": "a"}`), true, true, false, false, false},
		{jv(`{"s": "ABc"}`), jv(`{"s": "ab"}`), true, true, false, false, false},
		{jv(`{"s": "ABc"}`), jv(`{"s": "abc"}`), true, true, false, false, false},
		{jv(`{"s": "ABc"}`), jv(`{"s": "abcd"}`), true, true, false, false, false},
		{jv(`{"s": "ABc"}`), jv(`{"s": "b"}`), true, true, false, false, false},
		{jv(`{"s": "ABc"}`), jv(`{"s": "A"}`), false, false, false, true, true},
		{jv(`{"s": "ABc"}`), jv(`{"s": "AB"}`), false, false, false, true, true},
		{jv(`{"s": "ABc"}`), jv(`{"s": "ABC"}`), false, false, false, true, true},
		{jv(`{"s": "ABc"}`), jv(`{"s": "ABc"}`), false, true, true, true, false},
		{jv(`{"s": "ABc"}`), jv(`{"s": "ABcD"}`), true, true, false, false, false},
		{jv(`{"s": "ABc"}`), jv(`{"s": "B"}`), true, true, false, false, false},
		{jv(`{"s": "ABcD"}`), jv(`{"s": ""}`), false, false, false, true, true},
		{jv(`{"s": "ABcD"}`), jv(`{"s": "a"}`), true, true, false, false, false},
		{jv(`{"s": "ABcD"}`), jv(`{"s": "ab"}`), true, true, false, false, false},
		{jv(`{"s": "ABcD"}`), jv(`{"s": "abc"}`), true, true, false, false, false},
		{jv(`{"s": "ABcD"}`), jv(`{"s": "abcd"}`), true, true, false, false, false},
		{jv(`{"s": "ABcD"}`), jv(`{"s": "b"}`), true, true, false, false, false},
		{jv(`{"s": "ABcD"}`), jv(`{"s": "A"}`), false, false, false, true, true},
		{jv(`{"s": "ABcD"}`), jv(`{"s": "AB"}`), false, false, false, true, true},
		{jv(`{"s": "ABcD"}`), jv(`{"s": "ABC"}`), false, false, false, true, true},
		{jv(`{"s": "ABcD"}`), jv(`{"s": "ABc"}`), false, false, false, true, true},
		{jv(`{"s": "ABcD"}`), jv(`{"s": "ABcD"}`), false, true, true, true, false},
		{jv(`{"s": "ABcD"}`), jv(`{"s": "B"}`), true, true, false, false, false},
		{jv(`{"s": "B"}`), jv(`{"s": ""}`), false, false, false, true, true},
		{jv(`{"s": "B"}`), jv(`{"s": "a"}`), true, true, false, false, false},
		{jv(`{"s": "B"}`), jv(`{"s": "ab"}`), true, true, false, false, false},
		{jv(`{"s": "B"}`), jv(`{"s": "abc"}`), true, true, false, false, false},
		{jv(`{"s": "B"}`), jv(`{"s": "abcd"}`), true, true, false, false, false},
		{jv(`{"s": "B"}`), jv(`{"s": "b"}`), true, true, false, false, false},
		{jv(`{"s": "B"}`), jv(`{"s": "A"}`), false, false, false, true, true},
		{jv(`{"s": "B"}`), jv(`{"s": "AB"}`), false, false, false, true, true},
		{jv(`{"s": "B"}`), jv(`{"s": "ABC"}`), false, false, false, true, true},
		{jv(`{"s": "B"}`), jv(`{"s": "ABc"}`), false, false, false, true, true},
		{jv(`{"s": "B"}`), jv(`{"s": "ABcD"}`), false, false, false, true, true},
		{jv(`{"s": "B"}`), jv(`{"s": "B"}`), false, true, true, true, false},
	} {
		for _, opCase := range []struct {
			op  string
			exp bool
		}{
			{"<", tc.lt},
			{"<=", tc.le},
			{"==", tc.eq},
			{">", tc.gt},
			{">=", tc.ge},
		} {
			i++
			t.Run(fmt.Sprintf("test_%v", i), func(t *testing.T) {
				t.Parallel()
				firstTestCase{
					json: tc.obj1,
					path: "$.s" + opCase.op + " $s",
					opt:  []Option{WithVars(Vars(tc.obj2))},
					exp:  opCase.exp,
				}.run(a, r)
			})
		}
	}
}
