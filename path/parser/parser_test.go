package parser

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/theory/sqljson/path/ast"
)

func mkAST(t *testing.T, lax, pred bool, node ast.Node) *ast.AST {
	t.Helper()
	ast, err := ast.New(lax, pred, node)
	require.NoError(t, err)
	return ast
}

func TestParser(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		path string
		ast  *ast.AST
		err  string
	}{
		{
			name: "root",
			path: "$",
			ast:  mkAST(t, true, false, ast.LinkNodes([]ast.Node{ast.NewConst(ast.ConstRoot)})),
		},
		{
			name: "strict_root",
			path: "strict $",
			ast:  mkAST(t, false, false, ast.LinkNodes([]ast.Node{ast.NewConst(ast.ConstRoot)})),
		},
		{
			name: "predicate",
			path: "$ == 1",
			ast:  mkAST(t, true, true, ast.NewBinary(ast.BinaryEqual, ast.NewConst(ast.ConstRoot), ast.NewInteger("1"))),
		},
		{
			name: "error",
			path: "$()",
			err:  "parser: syntax error at 1:3",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)
			r := require.New(t)

			ast, err := Parse(tc.path)
			if tc.err == "" {
				r.NoError(err)
				a.Equal(tc.ast, ast)
			} else {
				r.EqualError(err, tc.err)
				r.ErrorIs(err, ErrParse)
				a.Nil(ast)
			}
		})
	}
}

type testCase struct {
	name string
	path string
	exp  string
	err  string
}

func (tc testCase) run(t *testing.T) {
	t.Parallel()
	ast, err := Parse(tc.path)
	if tc.err == "" {
		require.NoError(t, err)
		assert.Equal(t, tc.exp, ast.String())
	} else {
		require.EqualError(t, err, tc.err)
		require.ErrorIs(t, err, ErrParse)
		assert.Nil(t, ast)
	}
}

func TestJSONPathString(t *testing.T) {
	// https://github.com/postgres/postgres/blob/REL_18_BETA2/src/test/regress/sql/jsonpath.sql#L3-L30
	t.Parallel()

	//nolint:paralleltest
	for _, tc := range []testCase{
		{
			name: "empty",
			err:  `parser: syntax error at 1:1`,
		},
		{
			name: "root",
			path: "$",
			exp:  "$",
		},
		{
			name: "strict",
			path: "strict $",
			exp:  "strict $",
		},
		{
			name: "lax",
			path: "lax $",
			exp:  "$",
		},
		{
			name: "a",
			path: "$.a",
			exp:  `$."a"`,
		},
		{
			name: "a_v",
			path: "$.a.v",
			exp:  `$."a"."v"`,
		},
		{
			name: "a_star",
			path: "$.a.*",
			exp:  `$."a".*`,
		},
		{
			name: "star_any_array",
			path: "$.*[*]",
			exp:  "$.*[*]",
		},
		{
			name: "a_any_array",
			path: "$.a[*]",
			exp:  `$."a"[*]`,
		},
		{
			name: "a_any_array_x2",
			path: "$.a[*][*]",
			exp:  `$."a"[*][*]`,
		},
		{
			name: "root_any_array",
			path: "$[*]",
			exp:  "$[*]",
		},
		{
			name: "root_array_index",
			path: "$[0]",
			exp:  "$[0]",
		},
		{
			name: "root_any_array_index",
			path: "$[*][0]",
			exp:  "$[*][0]",
		},
		{
			name: "any_array_a",
			path: "$[*].a",
			exp:  `$[*]."a"`,
		},
		{
			name: "any_array_index_a_b",
			path: "$[*][0].a.b",
			exp:  `$[*][0]."a"."b"`,
		},
		{
			name: "a_any_b",
			path: "$.a.**.b",
			exp:  `$."a".**."b"`,
		},
		{
			name: "a_any2_b",
			path: "$.a.**{2}.b",
			exp:  `$."a".**{2}."b"`,
		},
		{
			name: "a_any2_2_b",
			path: "$.a.**{2 to 2}.b",
			exp:  `$."a".**{2}."b"`,
		},
		{
			name: "a_any2_5_b",
			path: "$.a.**{2 to 5}.b",
			exp:  `$."a".**{2 to 5}."b"`,
		},
		{
			name: "a_any0_5_b",
			path: "$.a.**{0 to 5}.b",
			exp:  `$."a".**{0 to 5}."b"`,
		},
		{
			name: "a_any5_last_b",
			path: "$.a.**{5 to last}.b",
			exp:  `$."a".**{5 to last}."b"`,
		},
		{
			name: "a_any_last_b",
			path: "$.a.**{last}.b",
			exp:  `$."a".**{last}."b"`,
		},
		{
			name: "a_any_last_5_b",
			path: "$.a.**{last to 5}.b",
			exp:  `$."a".**{last to 5}."b"`,
		},
		{
			name: "plus_one",
			path: "$+1",
			exp:  "($ + 1)",
		},
		{
			name: "minus_one",
			path: "$-1",
			exp:  "($ - 1)",
		},
		{
			name: "minus_plus_one",
			path: "$--+1",
			exp:  "($ - -1)",
		},
		{
			name: "a_div_plus_minus_one",
			path: "$.a/+-1",
			exp:  `($."a" / -1)`,
		},
		{
			name: "math",
			path: "1 * 2 + 4 % -3 != false",
			exp:  "(1 * 2 + 4 % -3 != false)",
		},
	} {
		t.Run(tc.name, tc.run)
	}
}

func TestJSONPathEscapesString(t *testing.T) {
	// https://github.com/postgres/postgres/blob/REL_18_BETA2/src/test/regress/sql/jsonpath.sql#L32-L35
	t.Parallel()

	//nolint:paralleltest
	for _, tc := range []testCase{
		{
			name: "js_escapes",
			path: `"\b\f\r\n\t\v\"\'\\"`,
			exp:  `"\b\f\r\n\t\v\"'\\"`,
		},
		{
			name: "hex_and_unicode_escapes",
			path: `"\x50\u0067\u{53}\u{051}\u{00004C}"`,
			exp:  `"PgSQL"`,
		},
		{
			name: "more_unicode",
			path: `$.foo\x50\u0067\u{53}\u{051}\u{00004C}\t\"bar`,
			exp:  `$."fooPgSQL\t\"bar"`,
		},
		{
			name: "literal",
			path: `"\z"`, // unrecognized escape is just the literal char
			exp:  `"z"`,
		},
	} {
		t.Run(tc.name, tc.run)
	}
}

func TestJSONPathFilterString(t *testing.T) {
	// https://github.com/postgres/postgres/blob/REL_18_BETA2/src/test/regress/sql/jsonpath.sql#L37-L50
	t.Parallel()

	//nolint:paralleltest
	for _, tc := range []testCase{
		{
			name: "g_a_1",
			path: `$.g ? ($.a == 1)`,
			exp:  `$."g"?($."a" == 1)`,
		},
		{
			name: "g_current_1",
			path: `$.g ? (@ == 1)`,
			exp:  `$."g"?(@ == 1)`,
		},
		{
			name: "g_a_current_1",
			path: `$.g ? (@.a == 1)`,
			exp:  `$."g"?(@."a" == 1)`,
		},
		{
			name: "g_a_or_current",
			path: `$.g ? (@.a == 1 || @.a == 4)`,
			exp:  `$."g"?(@."a" == 1 || @."a" == 4)`,
		},
		{
			name: "g_a_or_current_4",
			path: `$.g ? (@.a == 1 && @.a == 4)`,
			exp:  `$."g"?(@."a" == 1 && @."a" == 4)`,
		},
		{
			name: "g_a_4_7",
			path: `$.g ? (@.a == 1 || @.a == 4 && @.b == 7)`,
			exp:  `$."g"?(@."a" == 1 || @."a" == 4 && @."b" == 7)`,
		},
		{
			name: "g_a_4_b_7",
			path: `$.g ? (@.a == 1 || !(@.a == 4) && @.b == 7)`,
			exp:  `$."g"?(@."a" == 1 || !(@."a" == 4) && @."b" == 7)`,
		},
		{
			name: "g_a_x_a_b",
			path: `$.g ? (@.a == 1 || !(@.x >= 123 || @.a == 4) && @.b == 7)`,
			exp:  `$."g"?(@."a" == 1 || !(@."x" >= 123 || @."a" == 4) && @."b" == 7)`,
		},
		{
			name: "g_a_gt_abc",
			path: `$.g ? (@.x >= @[*]?(@.a > "abc"))`,
			exp:  `$."g"?(@."x" >= @[*]?(@."a" > "abc"))`,
		},
		{
			name: "g_x_a_is_unknown",
			path: `$.g ? ((@.x >= 123 || @.a == 4) is unknown)`,
			exp:  `$."g"?((@."x" >= 123 || @."a" == 4) is unknown)`,
		},
		{
			name: "g_exists_x",
			path: `$.g ? (exists (@.x))`,
			exp:  `$."g"?(exists (@."x"))`,
		},
		{
			name: "g_exists_x_or_14",
			path: `$.g ? (exists (@.x ? (@ == 14)))`,
			exp:  `$."g"?(exists (@."x"?(@ == 14)))`,
		},
		{
			name: "g_x_124_or_exists",
			path: `$.g ? ((@.x >= 123 || @.a == 4) && exists (@.x ? (@ == 14)))`,
			exp:  `$."g"?((@."x" >= 123 || @."a" == 4) && exists (@."x"?(@ == 14)))`,
		},
		{
			name: "g_x_gt_a",
			path: `$.g ? (+@.x >= +-(+@.a + 2))`,
			exp:  `$."g"?(+@."x" >= +(-(+@."a" + 2)))`,
		},
	} {
		t.Run(tc.name, tc.run)
	}
}

func TestJSONPathArrayStuffString(t *testing.T) {
	// https://github.com/postgres/postgres/blob/REL_18_BETA2/src/test/regress/sql/jsonpath.sql#L52-L64
	t.Parallel()

	//nolint:paralleltest
	for _, tc := range []testCase{
		{
			name: "a",
			path: `$a`,
			exp:  `$"a"`,
		},
		{
			name: "a_b",
			path: `$a.b`,
			exp:  `$"a"."b"`,
		},
		{
			name: "a_array",
			path: `$a[*]`,
			exp:  `$"a"[*]`,
		},
		{
			name: "g_filter",
			path: `$.g ? (@.zip == $zip)`,
			exp:  `$."g"?(@."zip" == $"zip")`,
		},
		{
			name: "a_array_multi",
			path: `$.a[1,2, 3 to 16]`,
			exp:  `$."a"[1,2,3 to 16]`,
		},
		{
			name: "a_array_math",
			path: `$.a[$a + 1, ($b[*]) to -($[0] * 2)]`,
			exp:  `$."a"[$"a" + 1,$"b"[*] to -($[0] * 2)]`,
		},
		{
			name: "a_array_method",
			path: `$.a[$.a.size() - 3]`,
			exp:  `$."a"[$."a".size() - 3]`,
		},
		{
			name: "last",
			path: `last`,
			err:  "parser: LAST is allowed only in array subscripts",
		},
		{
			name: "last_string",
			path: `"last"`,
			exp:  `"last"`,
		},
		{
			name: "last_ident",
			path: `$.last`,
			exp:  `$."last"`,
		},
		{
			name: "last_operand",
			path: `$ ? (last > 0)`,
			err:  "parser: LAST is allowed only in array subscripts",
		},
		{
			name: "array_last",
			path: `$[last]`,
			exp:  `$[last]`,
		},
		{
			name: "filter_array_last",
			path: `$[$[0] ? (last > 0)]`,
			exp:  `$[$[0]?(last > 0)]`,
		},
	} {
		t.Run(tc.name, tc.run)
	}
}

func TestJSONPathMethodString(t *testing.T) {
	// https://github.com/postgres/postgres/blob/REL_18_BETA2/src/test/regress/sql/jsonpath.sql#L66-L88
	t.Parallel()

	//nolint:paralleltest
	for _, tc := range []testCase{
		{
			name: "null_type",
			path: `null.type()`,
			exp:  `null.type()`,
		},
		{
			name: "one_type",
			path: `1.type()`,
			err:  `parser: trailing junk after numeric literal at 1:3`,
		},
		{
			name: "parentheses_one_type",
			path: `(1).type()`,
			exp:  `(1).type()`,
		},
		{
			name: "numeric_type",
			path: `1.2.type()`,
			exp:  `(1.2).type()`,
		},
		{
			name: "string_type",
			path: `"aaa".type()`,
			exp:  `"aaa".type()`,
		},
		{
			name: "bool_typ",
			path: `true.type()`,
			exp:  `true.type()`,
		},
		{
			name: "four_meths",
			path: `$.double().floor().ceiling().abs()`,
			exp:  `$.double().floor().ceiling().abs()`,
		},
		{
			name: "keyvalue_key",
			path: `$.keyvalue().key`,
			exp:  `$.keyvalue()."key"`,
		},
		{
			name: "datetime",
			path: `$.datetime()`,
			exp:  `$.datetime()`,
		},
		{
			name: "datetime_template",
			path: `$.datetime("datetime template")`,
			exp:  `$.datetime("datetime template")`,
		},
		{
			name: "four_numeric_meths",
			path: `$.bigint().integer().number().decimal()`,
			exp:  `$.bigint().integer().number().decimal()`,
		},
		{
			name: "boolean",
			path: `$.boolean()`,
			exp:  `$.boolean()`,
		},
		{
			name: "date",
			path: `$.date()`,
			exp:  `$.date()`,
		},
		{
			name: "decimal",
			path: `$.decimal(4,2)`,
			exp:  `$.decimal(4,2)`,
		},
		{
			name: "string",
			path: `$.string()`,
			exp:  `$.string()`,
		},
		{
			name: "time",
			path: `$.time()`,
			exp:  `$.time()`,
		},
		{
			name: "time_arg",
			path: `$.time(6)`,
			exp:  `$.time(6)`,
		},
		{
			name: "time_tz",
			path: `$.time_tz()`,
			exp:  `$.time_tz()`,
		},
		{
			name: "time_tz_arg",
			path: `$.time_tz(4)`,
			exp:  `$.time_tz(4)`,
		},
		{
			name: "timestamp",
			path: `$.timestamp()`,
			exp:  `$.timestamp()`,
		},
		{
			name: "timestamp_arg",
			path: `$.timestamp(2)`,
			exp:  `$.timestamp(2)`,
		},
		{
			name: "timestamp_tz",
			path: `$.timestamp_tz()`,
			exp:  `$.timestamp_tz()`,
		},
		{
			name: "timestamp_tz_arg",
			path: `$.timestamp_tz(0)`,
			exp:  `$.timestamp_tz(0)`,
		},
	} {
		t.Run(tc.name, tc.run)
	}
}

func TestJSONPathDecimal(t *testing.T) {
	t.Parallel()

	//nolint:paralleltest
	for _, tc := range []testCase{
		{
			name: "decimal",
			path: `$.decimal()`,
			exp:  `$.decimal()`,
		},
		{
			name: "decimal_p",
			path: `$.decimal(4)`,
			exp:  `$.decimal(4)`,
		},
		{
			name: "decimal_plus_p",
			path: `$.decimal(+4)`,
			exp:  `$.decimal(4)`,
		},
		{
			name: "decimal_minus_p",
			path: `$.decimal(-4)`,
			exp:  `$.decimal(-4)`,
		},
		{
			name: "decimal_p_s",
			path: `$.decimal(4,2)`,
			exp:  `$.decimal(4,2)`,
		},
		{
			name: "decimal_p_s_err",
			path: `$.decimal(4,2,1)`,
			err:  "parser: invalid input syntax: .decimal() can only have an optional precision[,scale] at 1:17",
		},
	} {
		t.Run(tc.name, tc.run)
	}
}

func TestJSONPathStartsWithString(t *testing.T) {
	// https://github.com/postgres/postgres/blob/REL_18_BETA2/src/test/regress/sql/jsonpath.sql#L90-L91
	t.Parallel()

	//nolint:paralleltest
	for _, tc := range []testCase{
		{
			name: "starts_with_string",
			path: `$ ? (@ starts with "abc")`,
			exp:  `$?(@ starts with "abc")`,
		},
		{
			name: "starts_with_variable",
			path: `$ ? (@ starts with $var)`,
			exp:  `$?(@ starts with $"var")`,
		},
	} {
		t.Run(tc.name, tc.run)
	}
}

func TestJSONPathRegexString(t *testing.T) {
	// https://github.com/postgres/postgres/blob/REL_18_BETA2/src/test/regress/sql/jsonpath.sql#L93-L103
	t.Parallel()

	//nolint:paralleltest
	for _, tc := range []testCase{
		{
			name: "invalid_pattern",
			path: `$ ? (@ like_regex "(invalid pattern")`,
			err:  "parser: error parsing regexp: missing closing ): `(invalid pattern` at 1:38",
		},
		{
			name: "valid_pattern",
			path: `$ ? (@ like_regex "pattern")`,
			exp:  `$?(@ like_regex "pattern")`,
		},
		{
			name: "empty_flag",
			path: `$ ? (@ like_regex "pattern" flag "")`,
			exp:  `$?(@ like_regex "pattern")`,
		},
		{
			name: "flag_i",
			path: `$ ? (@ like_regex "pattern" flag "i")`,
			exp:  `$?(@ like_regex "pattern" flag "i")`,
		},
		{
			name: "flag_is",
			path: `$ ? (@ like_regex "pattern" flag "is")`,
			exp:  `$?(@ like_regex "pattern" flag "is")`,
		},
		{
			name: "flag_isim",
			path: `$ ? (@ like_regex "pattern" flag "isim")`,
			exp:  `$?(@ like_regex "pattern" flag "ism")`,
		},
		{
			name: "flag_xsms",
			path: `$ ? (@ like_regex "pattern" flag "xsms")`,
			err:  `parser: XQuery "x" flag (expanded regular expressions) is not implemented at 1:40`,
		},
		{
			name: "flag_q",
			path: `$ ? (@ like_regex "pattern" flag "q")`,
			exp:  `$?(@ like_regex "pattern" flag "q")`,
		},
		{
			name: "flag_iq",
			path: `$ ? (@ like_regex "pattern" flag "iq")`,
			exp:  `$?(@ like_regex "pattern" flag "iq")`,
		},
		{
			name: "flag_smixq",
			path: `$ ? (@ like_regex "pattern" flag "smixq")`,
			exp:  `$?(@ like_regex "pattern" flag "ismxq")`,
		},
		{
			name: "flag_a",
			path: `$ ? (@ like_regex "pattern" flag "a")`,
			err:  `parser: Unrecognized flag character "a" in LIKE_REGEX predicate at 1:37`,
		},
	} {
		t.Run(tc.name, tc.run)
	}
}

func TestJSONPathMathsString(t *testing.T) {
	// https://github.com/postgres/postgres/blob/REL_18_BETA2/src/test/regress/sql/jsonpath.sql#L105-107
	t.Parallel()

	//nolint:paralleltest
	for _, tc := range []testCase{
		{
			name: "lt",
			path: `$ < 1`,
			exp:  `($ < 1)`,
		},
		{
			name: "lt_or_le",
			path: `($ < 1) || $.a.b <= $x`,
			exp:  `($ < 1 || $."a"."b" <= $"x")`,
		},
		{
			name: "plus",
			path: `@ + 1`,
			err:  `parser: @ is not allowed in root expressions`,
		},
	} {
		t.Run(tc.name, tc.run)
	}
}

func TestJSONPathNumericString(t *testing.T) {
	// https://github.com/postgres/postgres/blob/REL_18_BETA2/src/test/regress/sql/jsonpath.sql#L37-L50
	t.Parallel()

	//nolint:paralleltest
	for _, tc := range []testCase{
		{
			name: "root_a_b",
			path: `($).a.b`,
			exp:  `$."a"."b"`,
		},
		{
			name: "root_a_b_c_d",
			path: `($.a.b).c.d`,
			exp:  `$."a"."b"."c"."d"`,
		},
		{
			name: "ab_xy_cd",
			path: `($.a.b + -$.x.y).c.d`,
			exp:  `($."a"."b" + -$."x"."y")."c"."d"`,
		},
		{
			name: "ab_cd",
			path: `(-+$.a.b).c.d`,
			exp:  `(-(+$."a"."b"))."c"."d"`,
		},
		{
			name: "1_ab_plus_cd",
			path: `1 + ($.a.b + 2).c.d`,
			exp:  `(1 + ($."a"."b" + 2)."c"."d")`,
		},
		{
			name: "1_ab_gt_cd",
			path: `1 + ($.a.b > 2).c.d`,
			exp:  `(1 + ($."a"."b" > 2)."c"."d")`,
		},
		{
			name: "parentheses_root",
			path: `($)`,
			exp:  `$`,
		},
		{
			name: "2parentheses_root",
			path: `(($))`,
			exp:  `$`,
		},
		{
			name: "extreme_parentheses",
			path: `((($ + 1)).a + ((2)).b ? ((((@ > 1)) || (exists(@.c)))))`,
			exp:  `(($ + 1)."a" + (2)."b"?(@ > 1 || exists (@."c")))`,
		},
	} {
		t.Run(tc.name, tc.run)
	}
}

func TestJSONPathCompareNumbersString(t *testing.T) {
	// https://github.com/postgres/postgres/blob/REL_18_BETA2/src/test/regress/sql/jsonpath.sql#L37-L50
	t.Parallel()

	//nolint:paralleltest
	for _, tc := range []testCase{
		{
			name: "a_lt_1",
			path: `$ ? (@.a < 1)`,
			exp:  `$?(@."a" < 1)`,
		},
		{
			name: "a_lt_neg_1",
			path: `$ ? (@.a < -1)`,
			exp:  `$?(@."a" < -1)`,
		},
		{
			name: "a_lt_pos_1",
			path: `$ ? (@.a < +1)`,
			exp:  `$?(@."a" < 1)`,
		},
		{
			name: "a_lt_dot_1",
			path: `$ ? (@.a < .1)`,
			exp:  `$?(@."a" < 0.1)`,
		},
		{
			name: "a_lt_neg_dot_1",
			path: `$ ? (@.a < -.1)`,
			exp:  `$?(@."a" < -0.1)`,
		},
		{
			name: "a_lt_pos_dot_1",
			path: `$ ? (@.a < +.1)`,
			exp:  `$?(@."a" < 0.1)`,
		},
		{
			name: "a_lt_0_dot_1",
			path: `$ ? (@.a < 0.1)`,
			exp:  `$?(@."a" < 0.1)`,
		},
		{
			name: "a_lt_neg_0_dot_1",
			path: `$ ? (@.a < -0.1)`,
			exp:  `$?(@."a" < -0.1)`,
		},
		{
			name: "a_lt_pos_0_dot_1",
			path: `$ ? (@.a < +0.1)`,
			exp:  `$?(@."a" < 0.1)`,
		},
		{
			name: "a_lt_10_dot_1",
			path: `$ ? (@.a < 10.1)`,
			exp:  `$?(@."a" < 10.1)`,
		},
		{
			name: "a_lt_neg_10_dot_1",
			path: `$ ? (@.a < -10.1)`,
			exp:  `$?(@."a" < -10.1)`,
		},
		{
			name: "a_lt_pos_10_dot_1",
			path: `$ ? (@.a < +10.1)`,
			exp:  `$?(@."a" < 10.1)`,
		},
		{
			name: "a_lt_expo",
			path: `$ ? (@.a < 1e1)`,
			exp:  `$?(@."a" < 10)`,
		},
		{
			name: "a_lt_neg_expo",
			path: `$ ? (@.a < -1e1)`,
			exp:  `$?(@."a" < -10)`,
		},
		{
			name: "a_lt_pos_expo",
			path: `$ ? (@.a < +1e1)`,
			exp:  `$?(@."a" < 10)`,
		},
		{
			name: "a_lt_dot_expo",
			path: `$ ? (@.a < .1e1)`,
			exp:  `$?(@."a" < 1)`,
		},
		{
			name: "a_lt_neg_dot_expo",
			path: `$ ? (@.a < -.1e1)`,
			exp:  `$?(@."a" < -1)`,
		},
		{
			name: "a_lt_pos_dot_expo",
			path: `$ ? (@.a < +.1e1)`,
			exp:  `$?(@."a" < 1)`,
		},
		{
			name: "a_lt_0_dot_expo",
			path: `$ ? (@.a < 0.1e1)`,
			exp:  `$?(@."a" < 1)`,
		},
		{
			name: "a_lt_neg_0_dot_expo",
			path: `$ ? (@.a < -0.1e1)`,
			exp:  `$?(@."a" < -1)`,
		},
		{
			name: "a_lt_0_pos_expo",
			path: `$ ? (@.a < +0.1e1)`,
			exp:  `$?(@."a" < 1)`,
		},
		{
			name: "a_lt_10_dot_expo",
			path: `$ ? (@.a < 10.1e1)`,
			exp:  `$?(@."a" < 101)`,
		},
		{
			name: "a_lt_neg_10_dot_expo",
			path: `$ ? (@.a < -10.1e1)`,
			exp:  `$?(@."a" < -101)`,
		},
		{
			name: "a_lt_pos_10_dot_expo",
			path: `$ ? (@.a < +10.1e1)`,
			exp:  `$?(@."a" < 101)`,
		},
		{
			name: "a_lt_1_neg_expo",
			path: `$ ? (@.a < 1e-1)`,
			exp:  `$?(@."a" < 0.1)`,
		},
		{
			name: "a_lt_neg_1_neg_expo",
			path: `$ ? (@.a < -1e-1)`,
			exp:  `$?(@."a" < -0.1)`,
		},
		{
			name: "a_lt_pos_1_neg_expo",
			path: `$ ? (@.a < +1e-1)`,
			exp:  `$?(@."a" < 0.1)`,
		},
		{
			name: "a_lt_dot_1_expo",
			path: `$ ? (@.a < .1e-1)`,
			exp:  `$?(@."a" < 0.01)`,
		},
		{
			name: "a_lt_neg_dot_1_expo",
			path: `$ ? (@.a < -.1e-1)`,
			exp:  `$?(@."a" < -0.01)`,
		},
		{
			name: "a_lt_pos_dot_1_expo",
			path: `$ ? (@.a < +.1e-1)`,
			exp:  `$?(@."a" < 0.01)`,
		},
		{
			name: "a_lt_0_dot_1_neg_expo",
			path: `$ ? (@.a < 0.1e-1)`,
			exp:  `$?(@."a" < 0.01)`,
		},
		{
			name: "a_lt_neg_0_dot_1_neg_expo",
			path: `$ ? (@.a < -0.1e-1)`,
			exp:  `$?(@."a" < -0.01)`,
		},
		{
			name: "a_lt_pos_0_dot_1_neg_expo",
			path: `$ ? (@.a < +0.1e-1)`,
			exp:  `$?(@."a" < 0.01)`,
		},
		{
			name: "a_lt_10_dot_1_neg_expo",
			path: `$ ? (@.a < 10.1e-1)`,
			exp:  `$?(@."a" < 1.01)`,
		},
		{
			name: "a_lt_neg_10_dot_1_neg_expo",
			path: `$ ? (@.a < -10.1e-1)`,
			exp:  `$?(@."a" < -1.01)`,
		},
		{
			name: "a_lt_pos_10_dot_1_neg_expo",
			path: `$ ? (@.a < +10.1e-1)`,
			exp:  `$?(@."a" < 1.01)`,
		},
		{
			name: "a_lt_1_pos_expo",
			path: `$ ? (@.a < 1e+1)`,
			exp:  `$?(@."a" < 10)`,
		},
		{
			name: "a_lt_neg_1_pos_expo",
			path: `$ ? (@.a < -1e+1)`,
			exp:  `$?(@."a" < -10)`,
		},
		{
			name: "a_lt_pos_1_pos_expo",
			path: `$ ? (@.a < +1e+1)`,
			exp:  `$?(@."a" < 10)`,
		},
		{
			name: "a_lt_dot_1_pos_expo",
			path: `$ ? (@.a < .1e+1)`,
			exp:  `$?(@."a" < 1)`,
		},
		{
			name: "a_lt_neg_dot_1_pos_expo",
			path: `$ ? (@.a < -.1e+1)`,
			exp:  `$?(@."a" < -1)`,
		},
		{
			name: "a_lt_pos_dot_1_pos_expo",
			path: `$ ? (@.a < +.1e+1)`,
			exp:  `$?(@."a" < 1)`,
		},
		{
			name: "a_lt_0_dot_1_pos_expo",
			path: `$ ? (@.a < 0.1e+1)`,
			exp:  `$?(@."a" < 1)`,
		},
		{
			name: "a_lt_neg_0_dot_1_pos_expo",
			path: `$ ? (@.a < -0.1e+1)`,
			exp:  `$?(@."a" < -1)`,
		},
		{
			name: "a_lt_pos_0_dot_1_pos_expo",
			path: `$ ? (@.a < +0.1e+1)`,
			exp:  `$?(@."a" < 1)`,
		},
		{
			name: "a_lt_10_dot_1_pos_expo",
			path: `$ ? (@.a < 10.1e+1)`,
			exp:  `$?(@."a" < 101)`,
		},
		{
			name: "a_lt_neg_10_dot_1_pos_expo",
			path: `$ ? (@.a < -10.1e+1)`,
			exp:  `$?(@."a" < -101)`,
		},
		{
			name: "a_lt_pos_10_dot_1_pos_expo",
			path: `$ ? (@.a < +10.1e+1)`,
			exp:  `$?(@."a" < 101)`,
		},
	} {
		t.Run(tc.name, tc.run)
	}
}

func TestJSONPathNumericLiteralsString(t *testing.T) {
	// https://github.com/postgres/postgres/blob/REL_18_BETA2/src/test/regress/sql/jsonpath.sql#L170-205
	t.Parallel()

	//nolint:paralleltest
	for _, tc := range []testCase{
		{
			name: "zero",
			path: `0`,
			exp:  `0`,
		},
		{
			name: "zero_zero",
			path: `00`,
			err:  `parser: trailing junk after numeric literal at 1:2`,
		},
		{
			name: "leading_zero",
			path: `0755`,
			err:  `parser: trailing junk after numeric literal at 1:2`,
		},
		{
			name: "zero_dot_zero",
			path: `0.0`,
			exp:  `0`, // postgres: 0.00
		},
		{
			name: "zero_dot_000",
			path: `0.000`,
			exp:  `0`, // postgres: 0.00
		},
		{
			name: "float_expo_1",
			path: `0.000e1`,
			exp:  `0`, // postgres: 0.00
		},
		{
			name: "float_expo_2",
			path: `0.000e2`,
			exp:  `0`, // postgres: 0.00
		},
		{
			name: "float_expo_3",
			path: `0.000e3`,
			exp:  `0`,
		},
		{
			name: "0_dot_0010",
			path: `0.0010`,
			exp:  `0.001`, // postgres: 0.0010
		},
		{
			name: "float_neg_expo_1",
			path: `0.0010e-1`,
			exp:  `0.0001`, // postgres: 0.00010
		},
		{
			name: "float_pos_expo_1",
			path: `0.0010e+1`,
			exp:  `0.01`, // postgres: 0.010
		},
		{
			name: "float_pos_expo_2",
			path: `0.0010e+2`,
			exp:  `0.1`, // postgres: 0.10
		},
		{
			name: "dot_001",
			path: `.001`,
			exp:  `0.001`,
		},
		{
			name: "dot_001e1",
			path: `.001e1`,
			exp:  `0.01`,
		},
		{
			name: "one_dot",
			path: `1.`,
			exp:  `1`,
		},
		{
			name: "done_dot_expo_1",
			path: `1.e1`,
			exp:  `10`,
		},
		{
			name: "1a",
			path: `1a`,
			err:  `parser: trailing junk after numeric literal at 1:2`,
		},
		{
			name: "1e",
			path: `1e`,
			err:  `parser: exponent has no digits at 1:3`,
		},
		{
			name: "1_dot_e",
			path: `1.e`,
			err:  `parser: exponent has no digits at 1:4`,
		},
		{
			name: "1_dot_2a",
			path: `1.2a`,
			err:  `parser: trailing junk after numeric literal at 1:4`,
		},
		{
			name: "one_dot_2e",
			path: `1.2e`,
			err:  `parser: exponent has no digits at 1:5`,
		},
		{
			name: "one_dot_2_dot_e",
			path: `1.2.e`,
			exp:  `(1.2)."e"`,
		},
		{
			name: "parens_one_dot_two_then_e",
			path: `(1.2).e`,
			exp:  `(1.2)."e"`,
		},
		{
			name: "1e3",
			path: `1e3`,
			exp:  `1000`,
		},
		{
			name: "1_dot_e3",
			path: `1.e3`,
			exp:  `1000`,
		},
		{
			name: "1_dot_e3_dot_e",
			path: `1.e3.e`,
			exp:  `(1000)."e"`,
		},
		{
			name: "1_dot_e3_dot_e4",
			path: `1.e3.e4`,
			exp:  `(1000)."e4"`,
		},
		{
			name: "1_dot_2e3",
			path: `1.2e3`,
			exp:  `1200`,
		},
		{
			name: "1_dot_2e3a",
			path: `1.2e3a`,
			err:  `parser: trailing junk after numeric literal at 1:6`,
		},
		{
			name: "1_dot_2_dot_e3",
			path: `1.2.e3`,
			exp:  `(1.2)."e3"`,
		},
		{
			name: "parens_1_dot_2_then_dot_e3",
			path: `(1.2).e3`,
			exp:  `(1.2)."e3"`,
		},
		{
			name: "1_2dot_3",
			path: `1..e`,
			exp:  `(1)."e"`,
		},
		{
			name: "1_2dot_e3",
			path: `1..e3`,
			exp:  `(1)."e3"`,
		},
		{
			name: "parens_1_dot_then_dot_3",
			path: `(1.).e`,
			exp:  `(1)."e"`,
		},
		{
			name: "parens_1_dot_then_dot_e3",
			path: `(1.).e3`,
			exp:  `(1)."e3"`,
		},
		{
			name: "1_filter_2_gt_3",
			path: `1?(2>3)`,
			exp:  `(1)?(2 > 3)`,
		},
	} {
		t.Run(tc.name, tc.run)
	}
}

func TestJSONPathNonDecimalString(t *testing.T) {
	// https://github.com/postgres/postgres/blob/REL_18_BETA2/src/test/regress/sql/jsonpath.sql#L207-L223
	t.Parallel()

	//nolint:paralleltest
	for _, tc := range []testCase{
		{
			name: "binary",
			path: `0b100101`,
			exp:  `37`,
		},
		{
			name: "octal",
			path: `0o273`,
			exp:  `187`,
		},
		{
			name: "hex",
			path: `0x42F`,
			exp:  `1071`,
		},
		// error cases
		{
			name: "empty_binary",
			path: `0b`,
			err:  `parser: trailing junk after numeric literal at 1:3`,
		},
		{
			name: "1b",
			path: `1b`,
			err:  `parser: trailing junk after numeric literal at 1:2`,
		},
		{
			name: "0b0x",
			path: `0b0x`,
			err:  `parser: trailing junk after numeric literal at 1:4`,
		},

		{
			name: "empty_octal",
			path: `0o`,
			err:  `parser: trailing junk after numeric literal at 1:3`,
		},
		{
			name: "1o",
			path: `1o`,
			err:  `parser: trailing junk after numeric literal at 1:2`,
		},
		{
			name: "0o0x",
			path: `0o0x`,
			err:  `parser: trailing junk after numeric literal at 1:4`,
		},

		{
			name: "empty_hex",
			path: `0x`,
			err:  `parser: trailing junk after numeric literal at 1:3`,
		},
		{
			name: "1x",
			path: `1x`,
			err:  `parser: trailing junk after numeric literal at 1:2`,
		},
		{
			name: "0x0y",
			path: `0x0y`,
			err:  `parser: trailing junk after numeric literal at 1:4`,
		},
	} {
		t.Run(tc.name, tc.run)
	}
}

func TestJSONPathUnderscoreNumberString(t *testing.T) {
	// https://github.com/postgres/postgres/blob/REL_18_BETA2/src/test/regress/sql/jsonpath.sql#L225-L251
	t.Parallel()

	//nolint:paralleltest
	for _, tc := range []testCase{
		{
			name: "1_000_000",
			path: `1_000_000`,
			exp:  `1000000`,
		},
		{
			name: "1_2_3",
			path: `1_2_3`,
			exp:  `123`,
		},
		{
			name: "0x1EEE_FFFF",
			path: `0x1EEE_FFFF`,
			exp:  `518979583`,
		},
		{
			name: "0o2_73",
			path: `0o2_73`,
			exp:  `187`,
		},
		{
			name: "0b10_0101",
			path: `0b10_0101`,
			exp:  `37`,
		},

		{
			name: "1_000_dot_000_005",
			path: `1_000.000_005`,
			exp:  `1000.000005`,
		},
		{
			name: "1_000_dot",
			path: `1_000.`,
			exp:  `1000`,
		},
		{
			name: "dot_000_005",
			path: `.000_005`,
			exp:  `0.000005`,
		},
		{
			name: "1_000_dot_5e0_1",
			path: `1_000.5e0_1`,
			exp:  `10005`,
		},
		// error cases
		{
			name: "_100",
			path: `_100`,
			err:  `parser: syntax error at 1:5`,
		},
		{
			name: "100_",
			path: `100_`,
			err:  `parser: '_' must separate successive digits at 1:5`,
		},
		{
			name: "100__000",
			path: `100__000`,
			err:  `parser: '_' must separate successive digits at 1:9`,
		},

		{
			name: "_1_000dot5",
			path: `_1_000.5`,
			err:  `parser: syntax error at 1:7`,
		},
		{
			name: "1_000_dot_5",
			path: `1_000_.5`,
			err:  `parser: '_' must separate successive digits at 1:9`,
		},
		{
			name: "1_000dot__5",
			path: `1_000._5`,
			err:  `parser: '_' must separate successive digits at 1:9`,
		},
		{
			name: "1_000dot5_",
			path: `1_000.5_`,
			err:  `parser: '_' must separate successive digits at 1:9`,
		},
		{
			name: "1_000dot5e_1",
			path: `1_000.5e_1`,
			err:  `parser: '_' must separate successive digits at 1:11`,
		},

		// underscore after prefix not allowed in JavaScript (but allowed in SQL)
		{
			name: "0b_10_0101",
			path: `0b_10_0101`,
			err:  `parser: underscore disallowed at start of numeric literal at 1:3`,
		},
		{
			name: "0o_273",
			path: `0o_273`,
			err:  `parser: underscore disallowed at start of numeric literal at 1:3`,
		},
		{
			name: "0x_42F",
			path: `0x_42F`,
			err:  `parser: underscore disallowed at start of numeric literal at 1:3`,
		},
	} {
		t.Run(tc.name, tc.run)
	}
}

func TestJSONPathEncodingString(t *testing.T) {
	// https://github.com/postgres/postgres/blob/REL_18_BETA2/src/test/regress/sql/jsonpath_encoding.sql
	t.Parallel()

	//nolint:paralleltest
	for _, tc := range []testCase{
		// checks for double-quoted values
		// basic unicode input
		{
			name: "empty_unicode",
			path: `"\u"`, // ERROR, incomplete escape
			err:  `parser: invalid Unicode escape sequence at 1:4`,
		},
		{
			name: "unicode_00",
			path: `"\u00"`, // ERROR, incomplete escape
			err:  `parser: invalid Unicode escape sequence at 1:6`,
		},
		{
			name: "unicode_invalid_hex",
			path: `"\u000g"`, // ERROR, g is not a hex digit
			err:  `parser: invalid Unicode escape sequence at 1:7`,
		},
		{
			name: "unicode_0000",
			path: `"\u0000"`, // OK, legal escape [but Postgres doesn't support null bytes in strings]
			err:  `parser: \u0000 cannot be converted to text at 1:7`,
		},
		{
			name: "unicode_aBcD",
			path: `"\uaBcD"`, // OK, uppercase and lower case both OK
			exp:  `"ꯍ"`,
		},

		// handling of unicode surrogate pairs
		{
			name: "smiley_dog",
			path: `"\ud83d\ude04\ud83d\udc36"`, // correct in utf8
			exp:  `"😄🐶"`,
		},
		{
			name: "two_highs",
			path: `"\ud83d\ud83d"`, // 2 high surrogates in a row
			err:  `parser: Unicode low surrogate must follow a high surrogate at 1:13`,
		},
		{
			name: "wrong_order",
			path: `"\ude04\ud83d"`, // surrogates in wrong order
			err:  `parser: Unicode low surrogate must follow a high surrogate at 1:13`,
		},
		{
			name: "orphan_high",
			path: `"\ud83dX"`, // orphan high surrogate
			err:  `parser: Unicode low surrogate must follow a high surrogate at 1:8`,
		},
		{
			name: "orphan_low",
			path: `"\ude04X"`, // orphan low surrogate
			err:  `parser: Unicode low surrogate must follow a high surrogate at 1:8`,
		},

		// handling of simple unicode escapes
		{
			name: "copyright_sign",
			path: `"the Copyright \u00a9 sign"`, // correct in utf8
			exp:  `"the Copyright © sign"`,
		},
		{
			name: "dollar_character",
			path: `"dollar \u0024 character"`, // correct everywhere
			exp:  `"dollar $ character"`,
		},
		{
			name: "not_escape",
			path: `"dollar \\u0024 character"`, // not an escape
			exp:  `"dollar \\u0024 character"`,
		},
		{
			name: "unescaped_null",
			path: `"null \u0000 escape"`, // not escaped
			err:  `parser: \u0000 cannot be converted to text at 1:12`,
		},
		{
			name: "escaped_null",
			path: `"null \\u0000 escape"`, // not an escape
			exp:  `"null \\u0000 escape"`,
		},

		//  checks for quoted key names
		//  basic unicode input
		{
			name: "incomplete_escape",
			path: `$."\u"`, // ERROR, incomplete escape
			err:  `parser: invalid Unicode escape sequence at 1:6`,
		},
		{
			name: "incomplete_escape_null",
			path: `$."\u00"`, // ERROR, incomplete escape
			err:  `parser: invalid Unicode escape sequence at 1:8`,
		},
		{
			name: "invalid_hex_digit",
			path: `$."\u000g"`, // ERROR, g is not a hex digit
			err:  `parser: invalid Unicode escape sequence at 1:9`,
		},
		{
			name: "null_byte_in_string",
			path: `$."\u0000"`, // OK, legal escape  [but Postgres doesn't support null bytes in strings]
			err:  `parser: \u0000 cannot be converted to text at 1:9`,
		},
		{
			name: "mixed_case_ok",
			path: `$."\uaBcD"`, // OK, uppercase and lower case both OK
			exp:  `$."ꯍ"`,
		},

		//  handling of unicode surrogate pairs
		{
			name: "smiley_dog_key",
			path: `$."\ud83d\ude04\ud83d\udc36"`, // correct in utf8
			exp:  `$."😄🐶"`,
		},
		{
			name: "two_highs_key",
			path: `$."\ud83d\ud83d"`, // 2 high surrogates in a row
			err:  `parser: Unicode low surrogate must follow a high surrogate at 1:15`,
		},
		{
			name: "wrong_order_key",
			path: `$."\ude04\ud83d"`, // surrogates in wrong order
			err:  `parser: Unicode low surrogate must follow a high surrogate at 1:15`,
		},
		{
			name: "orphan_high_key",
			path: `$."\ud83dX"`, // orphan high surrogate
			err:  `parser: Unicode low surrogate must follow a high surrogate at 1:10`,
		},
		{
			name: "orphan_low_key",
			path: `$."\ude04X"`, // orphan low surrogate
			err:  `parser: Unicode low surrogate must follow a high surrogate at 1:10`,
		},

		// handling of simple unicode escapes
		{
			name: "copyright_sign_key",
			path: `$."the Copyright \u00a9 sign"`, // correct in utf8
			exp:  `$."the Copyright © sign"`,
		},
		{
			name: "dollar_sign_key",
			path: `$."dollar \u0024 character"`, // correct everywhere
			exp:  `$."dollar $ character"`,
		},
		{
			name: "not_escape_key",
			path: `$."dollar \\u0024 character"`, // not an escape
			exp:  `$."dollar \\u0024 character"`,
		},
		{
			name: "unescaped_null_key",
			path: `$."null \u0000 escape"`, // not unescaped
			err:  `parser: \u0000 cannot be converted to text at 1:14`,
		},
		{
			name: "escaped_null_key",
			path: `$."null \\u0000 escape"`, // not an escape
			exp:  `$."null \\u0000 escape"`,
		},
	} {
		t.Run(tc.name, tc.run)
	}
}

func TestNumericEdgeCases(t *testing.T) {
	t.Parallel()

	//nolint:paralleltest
	for _, tc := range []testCase{
		// https://www.postgresql.org/message-id/flat/2F757EB8-AEB9-49E8-A2C6-613E06BA05D4%40justatheory.com
		{
			name: "hex_then_path_key",
			path: `0x2.p10`,
			exp:  `(2)."p10"`,
		},
		{
			name: "float_then_path_key",
			path: `3.14.p10`,
			exp:  `(3.14)."p10"`,
		},
		{
			name: "whitespace_disambiguation",
			path: `2 .p10`,
			exp:  `(2)."p10"`,
		},
		{
			name: "go_float_example_12",
			path: "0x2.p10",
			exp:  `(2)."p10"`,
		},
		{
			name: "go_float_example_13",
			path: "0x1.Fp+0",
			exp:  `((1)."Fp" + 0)`,
		},
		{
			name: "go_float_example_16",
			path: "0x15e-2",
			exp:  "(350 - 2)",
		},
		{
			name: "go_float_example_19",
			path: "0x1.5e-2",
			err:  "parser: syntax error at 1:9",
		},
		{
			name: "hex_dot_path_utf8",
			path: `0x2."😀"`,
			exp:  `(2)."😀"`,
		},
	} {
		t.Run(tc.name, tc.run)
	}
}

func TestDebugOutput(t *testing.T) {
	t.Parallel()
	node, _ := Parse("$.x + 2")
	buf := new(bytes.Buffer)
	printNode(buf, node.Root(), 0, "")
	assert.Equal(t, `BinaryNode(
  $
    "x"
  +
  2
)
`, buf.String())
}

// Placeholder function to generate output to describe an AST. Move to ast
// package?
func printNode(w io.Writer, node ast.Node, depth int, prefix string) {
	indent := strings.Repeat(" ", depth*2)
	switch node := node.(type) {
	case nil:
		return
	case *ast.ConstNode, *ast.MethodNode, *ast.StringNode, *ast.VariableNode,
		*ast.KeyNode, *ast.NumericNode, *ast.IntegerNode, *ast.AnyNode:
		fmt.Fprintf(w, "%v%v%v\n", indent, prefix, node.String())
	case *ast.BinaryNode:
		fmt.Fprintf(w, "%v%vBinaryNode(\n", indent, prefix)
		printNode(w, node.Left(), depth+1, "")
		fmt.Fprintf(w, "%v  %v\n", indent, node.Operator())
		printNode(w, node.Right(), depth+1, "")
		fmt.Fprintf(w, "%v)\n", indent)
	case *ast.UnaryNode:
		fmt.Fprintf(w, "%v%vUnaryNode(\n%v%v\n", indent, prefix, indent, node.Operator())
		printNode(w, node.Operand(), depth+1, "")
		fmt.Fprintf(w, "%v)\n", indent)
	case *ast.RegexNode:
		fmt.Fprintf(w, "%v%vRegexNode(\n", indent, prefix)
		printNode(w, node.Operand(), depth+1, "")
		fmt.Fprintf(w, "%v%v\n", indent, node.String())
		fmt.Fprintf(w, "%v)\n", indent)
	case *ast.ArrayIndexNode:
		fmt.Fprintf(w, "%v%vArrayIndexNode(\n", indent, prefix)
		for _, n := range node.Subscripts() {
			printNode(w, n, depth+1, "•  ")
		}
		fmt.Fprintf(w, "%v)\n", indent)
	}

	if next := node.Next(); next != nil {
		printNode(w, next, depth+1, "")
	}
}
