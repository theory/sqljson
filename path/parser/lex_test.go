package parser

import (
	"reflect"
	"strings"
	"testing"
	"text/scanner"

	"github.com/stretchr/testify/assert"
)

func TestNewLexer(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	path := "$.foo?(@==1)"

	l := newLexer(path)
	a.NotNil(l)

	// Examine the scanner.
	s := l.scanner
	a.NotNil(s)
	a.Equal("path", s.Filename)
	a.Equal(uint(scanner.ScanInts|scanner.ScanFloats), s.Mode)
	a.Equal(
		reflect.ValueOf(isIdentRune).Pointer(),
		reflect.ValueOf(s.IsIdentRune).Pointer(),
	)
	a.Equal(
		reflect.ValueOf(l.scanError).Pointer(),
		reflect.ValueOf(s.Error).Pointer(),
	)

	// Make sure path was loaded into the scanner.
	buf := new(strings.Builder)

	for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
		buf.WriteString(s.TokenText())
	}

	a.Equal(path, buf.String())
}

func TestIsIdentRune(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tc := range []struct {
		name string
		val  rune
		char int
		exp  bool
	}{
		{"null_first", 0, 0, false},
		{"null_second", 0, 1, false},
		{"underscore_first", '_', 0, true},
		{"underscore_second", '_', 1, true},
		{"dollar_first", '$', 0, false},
		{"dollar_second", '$', 1, false},
		{"char_first", 'a', 0, true},
		{"char_second", 'a', 1, true},
		{"alpha_first", 'a', 0, true},
		{"alpha_second", 'a', 1, true},
		{"letter_first", 'ઓ', 0, true},
		{"letter_second", 'ઓ', 1, true},
		{"digit_first", '9', 0, false},
		{"digit_second", '9', 1, true},
		{"emoji_first", '🎉', 0, false},
		{"emoji_second", '🎉', 1, false},
		{"backslash_first", '\\', 0, true},
		{"backslash_second", '\\', 1, true},
		{"slash_first", '/', 0, false},
		{"slash_second", '/', 1, false},
		{"space_first", ' ', 0, false},
		{"space_second", ' ', 1, false},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a.Equal(tc.exp, isIdentRune(tc.val, tc.char))
		})
	}
}

func TestScanError(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	l := newLexer("$.x == $y")
	a.NotNil(l)
	a.Equal([]string{}, l.errors)

	l.scanError(l.scanner, "oops")
	a.Equal([]string{"oops at path:1:1"}, l.errors)

	a.Equal('$', l.scanner.Scan())
	l.scanError(l.scanner, "yikes")
	a.Equal([]string{"oops at path:1:1", "yikes at path:1:2"}, l.errors)

	l.Error("hello")
	a.Equal(
		[]string{"oops at path:1:1", "yikes at path:1:2", "hello"},
		l.errors,
	)
}

func TestScanIdent(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tc := range []struct {
		name string
		word string
		exp  string
		tok  int
		err  string
	}{
		{"xxx", "xxx", "xxx", IDENT_P, ""},
		// Case-sensitive identifiers.
		{"null", "null", "null", NULL_P, ""},
		{"NULL", "NULL", "NULL", IDENT_P, ""},
		{"true", "true", "true", TRUE_P, ""},
		{"True", "True", "True", IDENT_P, ""},
		{"TRUE", "TRUE", "TRUE", IDENT_P, ""},
		{"false", "false", "false", FALSE_P, ""},
		{"False", "False", "False", IDENT_P, ""},
		{"FALSE", "FALSE", "FALSE", IDENT_P, ""},
		{"TRUE", "TRUE", "TRUE", IDENT_P, ""},

		// Case-insensitive identifiers.
		{"is", "is", "is", IS_P, ""},
		{"Is", "Is", "Is", IS_P, ""},
		{"IS", "IS", "IS", IS_P, ""},
		{"to", "to", "to", TO_P, ""},
		{"To", "To", "To", TO_P, ""},
		{"TO", "TO", "TO", TO_P, ""},
		{"abs", "abs", "abs", ABS_P, ""},
		{"Abs", "Abs", "Abs", ABS_P, ""},
		{"ABS", "ABS", "ABS", ABS_P, ""},
		{"lax", "lax", "lax", LAX_P, ""},
		{"Lax", "Lax", "Lax", LAX_P, ""},
		{"LAX", "LAX", "LAX", LAX_P, ""},
		{"date", "date", "date", DATE_P, ""},
		{"Date", "Date", "Date", DATE_P, ""},
		{"DATE", "DATE", "DATE", DATE_P, ""},
		{"flag", "flag", "flag", FLAG_P, ""},
		{"Flag", "Flag", "Flag", FLAG_P, ""},
		{"FLAG", "FLAG", "FLAG", FLAG_P, ""},
		{"last", "last", "last", LAST_P, ""},
		{"Last", "Last", "Last", LAST_P, ""},
		{"LAST", "LAST", "LAST", LAST_P, ""},
		{"size", "size", "size", SIZE_P, ""},
		{"Size", "Size", "Size", SIZE_P, ""},
		{"SIZE", "SIZE", "SIZE", SIZE_P, ""},
		{"time", "time", "time", TIME_P, ""},
		{"Time", "Time", "Time", TIME_P, ""},
		{"TIME", "TIME", "TIME", TIME_P, ""},
		{"type", "type", "type", TYPE_P, ""},
		{"Type", "Type", "Type", TYPE_P, ""},
		{"TYPE", "TYPE", "TYPE", TYPE_P, ""},
		{"with", "with", "with", WITH_P, ""},
		{"With", "With", "With", WITH_P, ""},
		{"WITH", "WITH", "WITH", WITH_P, ""},
		{"floor", "floor", "floor", FLOOR_P, ""},
		{"Floor", "Floor", "Floor", FLOOR_P, ""},
		{"FLOOR", "FLOOR", "FLOOR", FLOOR_P, ""},
		{"bigint", "bigint", "bigint", BIGINT_P, ""},
		{"Bigint", "Bigint", "Bigint", BIGINT_P, ""},
		{"BIGINT", "BIGINT", "BIGINT", BIGINT_P, ""},
		{"double", "double", "double", DOUBLE_P, ""},
		{"Double", "Double", "Double", DOUBLE_P, ""},
		{"DOUBLE", "DOUBLE", "DOUBLE", DOUBLE_P, ""},
		{"exists", "exists", "exists", EXISTS_P, ""},
		{"Exists", "Exists", "Exists", EXISTS_P, ""},
		{"EXISTS", "EXISTS", "EXISTS", EXISTS_P, ""},
		{"number", "number", "number", NUMBER_P, ""},
		{"Number", "Number", "Number", NUMBER_P, ""},
		{"NUMBER", "NUMBER", "NUMBER", NUMBER_P, ""},
		{"starts", "starts", "starts", STARTS_P, ""},
		{"Starts", "Starts", "Starts", STARTS_P, ""},
		{"STARTS", "STARTS", "STARTS", STARTS_P, ""},
		{"strict", "strict", "strict", STRICT_P, ""},
		{"Strict", "Strict", "Strict", STRICT_P, ""},
		{"STRICT", "STRICT", "STRICT", STRICT_P, ""},
		{"string", "string", "string", STRINGFUNC_P, ""},
		{"String", "String", "String", STRINGFUNC_P, ""},
		{"STRING", "STRING", "STRING", STRINGFUNC_P, ""},
		{"boolean", "boolean", "boolean", BOOLEAN_P, ""},
		{"Boolean", "Boolean", "Boolean", BOOLEAN_P, ""},
		{"BOOLEAN", "BOOLEAN", "BOOLEAN", BOOLEAN_P, ""},
		{"ceiling", "ceiling", "ceiling", CEILING_P, ""},
		{"Ceiling", "Ceiling", "Ceiling", CEILING_P, ""},
		{"CEILING", "CEILING", "CEILING", CEILING_P, ""},
		{"decimal", "decimal", "decimal", DECIMAL_P, ""},
		{"Decimal", "Decimal", "Decimal", DECIMAL_P, ""},
		{"DECIMAL", "DECIMAL", "DECIMAL", DECIMAL_P, ""},
		{"integer", "integer", "integer", INTEGER_P, ""},
		{"Integer", "Integer", "Integer", INTEGER_P, ""},
		{"INTEGER", "INTEGER", "INTEGER", INTEGER_P, ""},
		{"time_tz", "time_tz", "time_tz", TIME_TZ_P, ""},
		{"Time_tz", "Time_tz", "Time_tz", TIME_TZ_P, ""},
		{"TIME_TZ", "TIME_TZ", "TIME_TZ", TIME_TZ_P, ""},
		{"unknown", "unknown", "unknown", UNKNOWN_P, ""},
		{"Unknown", "Unknown", "Unknown", UNKNOWN_P, ""},
		{"UNKNOWN", "UNKNOWN", "UNKNOWN", UNKNOWN_P, ""},
		{"datetime", "datetime", "datetime", DATETIME_P, ""},
		{"Datetime", "Datetime", "Datetime", DATETIME_P, ""},
		{"DATETIME", "DATETIME", "DATETIME", DATETIME_P, ""},
		{"keyvalue", "keyvalue", "keyvalue", KEYVALUE_P, ""},
		{"Keyvalue", "Keyvalue", "Keyvalue", KEYVALUE_P, ""},
		{"KEYVALUE", "KEYVALUE", "KEYVALUE", KEYVALUE_P, ""},
		{"timestamp", "timestamp", "timestamp", TIMESTAMP_P, ""},
		{"Timestamp", "Timestamp", "Timestamp", TIMESTAMP_P, ""},
		{"TIMESTAMP", "TIMESTAMP", "TIMESTAMP", TIMESTAMP_P, ""},
		{"like_regex", "like_regex", "like_regex", LIKE_REGEX_P, ""},
		{"Like_regex", "Like_regex", "Like_regex", LIKE_REGEX_P, ""},
		{"LIKE_REGEX", "LIKE_REGEX", "LIKE_REGEX", LIKE_REGEX_P, ""},
		{"timestamp_tz", "timestamp_tz", "timestamp_tz", TIMESTAMP_TZ_P, ""},
		{"Timestamp_tz", "Timestamp_tz", "Timestamp_tz", TIMESTAMP_TZ_P, ""},
		{"TIMESTAMP_TZ", "TIMESTAMP_TZ", "TIMESTAMP_TZ", TIMESTAMP_TZ_P, ""},

		// Basic identifiers.
		{"underscore", "x_y_z", "x_y_z", IDENT_P, ""},
		{"mixed_case", "XoX", "XoX", IDENT_P, ""},
		{"unicode", "XöX", "XöX", IDENT_P, ""},

		// Identifiers with escapes.
		{"escaped_dot", `X\.X`, "X.X", IDENT_P, ""},
		{"hex", `\x22hi\x22`, `"hi"`, IDENT_P, ""},
		{"hex", `\x22hi\x22`, `"hi"`, IDENT_P, ""},
		{"bell", `x\by`, "x\by", IDENT_P, ""},
		{"form_feed", `x\fy`, "x\fy", IDENT_P, ""},
		{"new_line", `x\ny`, "x\ny", IDENT_P, ""},
		{"return", `x\ry`, "x\ry", IDENT_P, ""},
		{"return_form_feed", `x\r\ny`, "x\r\ny", IDENT_P, ""},
		{"tab", `x\ty`, "x\ty", IDENT_P, ""},
		{"vertical_tab", `x\vy`, "x\vy", IDENT_P, ""},
		{"quote", `x\"y`, `x"y`, IDENT_P, ""},
		{"slash", `x\/y`, `x/y`, IDENT_P, ""},
		{"backslash", `x\\y`, `x\y`, IDENT_P, ""},
		{"unknown_escape", `x\zy`, `xzy`, IDENT_P, ""},
		{"unicode", `fo\u00f8`, "foø", IDENT_P, ""},
		{"brace_unicode_two", `p\u{67}`, "pg", IDENT_P, ""},
		{"brace_unicode_four", `fo\u{00f8}`, "foø", IDENT_P, ""},
		{"brace_unicode_six", `LO\u{00004C}`, "LOL", IDENT_P, ""},
		{
			"ridiculous",
			`foo\x50\u0067\u{53}\u{051}\u{00004C}\t\"bar`,
			"fooPgSQL\t\"bar",
			IDENT_P,
			"",
		},

		// Errors.
		{
			"invalid_hex",
			`LO\xzz`,
			"",
			IDENT_P,
			"invalid hexadecimal character sequence at path:1:6",
		},
		{
			"brace_unicode_eight",
			`LO\u{00004C00}`,
			"",
			IDENT_P,
			"invalid Unicode escape sequence at path:1:13",
		},
		{
			"missing_brace",
			`LO\u{0067`,
			"",
			IDENT_P,
			"invalid hexadecimal character sequence at path:1:10",
		},
		{
			"bad_unicode_brace_hex",
			`LO\u{zzzz}`,
			"",
			IDENT_P,
			"invalid hexadecimal character sequence at path:1:7",
		},
		{
			"bad_unicode_hex",
			`LO\uzzzz`,
			"",
			IDENT_P,
			"invalid hexadecimal character sequence at path:1:6",
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sym := &pathSymType{str: ""}
			l := newLexer(tc.word)
			tok := l.scanner.Scan()
			a.Equal(l.scanIdent(sym, tok), tc.tok)
			a.Equal(tc.exp, sym.str)

			if tc.err == "" {
				a.Empty(l.errors)
			} else {
				a.Equal([]string{tc.err}, l.errors)
			}
		})
	}
}

func TestScanString(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tc := range []struct {
		name string
		str  string
		exp  string
		tok  int
		err  string
	}{
		{"xxx", `"xxx"`, "xxx", STRING_P, ""},
		{"with_spaces", `"hi there"`, "hi there", STRING_P, ""},
		{"with_unicode", `"Go on 🎉"`, "Go on 🎉", STRING_P, ""},
		{"surrogate_pair", `"\uD834\uDD1E"`, "\U0001D11E", STRING_P, ""},

		// Identifiers with escapes.
		{"hex", `"\x22hi\x22"`, `"hi"`, STRING_P, ""},
		{"hex", `"\x22hi\x22"`, `"hi"`, STRING_P, ""},
		{"bell", `"x\by"`, "x\by", STRING_P, ""},
		{"form_feed", `"x\fy"`, "x\fy", STRING_P, ""},
		{"new_line", `"x\ny"`, "x\ny", STRING_P, ""},
		{"return", `"x\ry"`, "x\ry", STRING_P, ""},
		{"return_form_feed", `"x\r\ny"`, "x\r\ny", STRING_P, ""},
		{"tab", `"x\ty"`, "x\ty", STRING_P, ""},
		{"vertical_tab", `"x\vy"`, "x\vy", STRING_P, ""},
		{"quote", `"x\"y"`, `x"y`, STRING_P, ""},
		{"slash", `"x\/y"`, `x/y`, STRING_P, ""},
		{"backslash", `"x\\y"`, `x\y`, STRING_P, ""},
		{"unknown_escape", `"x\zy"`, `xzy`, STRING_P, ""},
		{"unicode", `"fo\u00f8"`, "foø", STRING_P, ""},
		{"brace_unicode_two", `"p\u{67}"`, "pg", STRING_P, ""},
		{"brace_unicode_four", `"fo\u{00f8}"`, "foø", STRING_P, ""},
		{"brace_unicode_six", `"LO\u{00004C}"`, "LOL", STRING_P, ""},
		{
			"ridiculous",
			`"foo\x50\u0067\u{53}\u{051}\u{00004C}\t\"bar"`,
			"fooPgSQL\t\"bar",
			STRING_P,
			"",
		},

		// Errors.
		{
			"invalid_surrogate_pair",
			`"\uD834\ufffd"`,
			"", STRING_P,
			"Unicode low surrogate must follow a high surrogate at path:1:14",
		},
		{
			"missing_surrogate_pair",
			`"\uD834lol"`,
			"", STRING_P,
			"Unicode low surrogate must follow a high surrogate at path:1:8",
		},
		{
			"bad_surrogate_pair",
			`"\uD834\uzzzz`,
			"", STRING_P,
			"invalid hexadecimal character sequence at path:1:11",
		},
		{
			"wrong_surrogate_pair",
			`"\uD834\x34"`,
			"",
			STRING_P,
			"Unicode low surrogate must follow a high surrogate at path:1:9",
		},
		{
			"null_byte",
			`"go \x00"`,
			"",
			STRING_P,
			"invalid hexadecimal character sequence at path:1:9",
		},
		{
			"invalid_hex",
			`"LO\xzz"`,
			"",
			STRING_P,
			"invalid hexadecimal character sequence at path:1:7",
		},
		{
			"null_hex",
			`"LO\x00"`,
			"",
			STRING_P,
			"invalid hexadecimal character sequence at path:1:8",
		},
		{
			"null_unicode",
			`"LO\u0000"`,
			"",
			STRING_P,
			"invalid Unicode escape sequence at path:1:10",
		},
		{
			"null_unicode_brace",
			`"LO\u{000000}"`,
			"",
			STRING_P,
			"invalid Unicode escape sequence at path:1:14",
		},
		{
			"brace_unicode_eight",
			`"LO\u{00004C00}"`,
			"",
			STRING_P,
			"invalid Unicode escape sequence at path:1:14",
		},
		{
			"missing_brace",
			`"LO\u{0067"`,
			"",
			STRING_P,
			"invalid hexadecimal character sequence at path:1:12",
		},
		{
			"bad_unicode_brace_hex",
			`"LO\u{zzzz}"`,
			"",
			STRING_P,
			"invalid hexadecimal character sequence at path:1:8",
		},
		{
			"bad_unicode_hex",
			`"LO\uzzzz"`,
			"",
			STRING_P,
			"invalid hexadecimal character sequence at path:1:7",
		},
		{
			"unclosed_string",
			`"go`,
			"",
			STRING_P,
			"literal not terminated at path:1:4",
		},
		{
			"string_with_newline",
			"\"go\nhome\"",
			"",
			STRING_P,
			"literal not terminated at path:2:1",
		},
		{
			"unterminated_backslash",
			`"go \`,
			"",
			STRING_P,
			"unexpected end after backslash at path:1:6",
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sym := &pathSymType{}
			l := newLexer(tc.str)
			a.Equal('"', l.scanner.Scan()) // Remove opening "
			a.Equal(l.scanString(sym, tc.tok), tc.tok)
			a.Equal(tc.exp, sym.str)

			if tc.err == "" {
				a.Empty(l.errors)
			} else {
				a.Equal([]string{tc.err}, l.errors)
			}
		})
	}
}

func TestScanNumbers(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tc := range []struct {
		name string
		num  string
		exp  string
		tok  int
		err  string
	}{
		{"one", "1", "1", INT_P, ""},
		{"max_int", "9223372036854775807", "9223372036854775807", INT_P, ""},
		{"min_int", "9223372036854775808", "9223372036854775808", INT_P, ""}, // without -
		{"max_uint", "18446744073709551615", "18446744073709551615", INT_P, ""},
		{"underscores", "1_000_000", "1_000_000", INT_P, ""},
		{"hex", "0x1EEE_FFFF", "0x1EEE_FFFF", INT_P, ""},
		{"HEX", "0X1EEE_FFFF", "0X1EEE_FFFF", INT_P, ""},
		{"octal", "0o273", "0o273", INT_P, ""},
		{"underscore_octal", "0o27_3", "0o27_3", INT_P, ""},
		{"OCTAL", "0O273", "0O273", INT_P, ""},
		{"zero_octal", "0273", "0273", INT_P, "trailing junk after numeric literal at path:1:5"}, // Not supported by Postgres
		{"binary", "0b100101", "0b100101", INT_P, ""},
		{"underscore_binary", "0b10_0101", "0b10_0101", INT_P, ""},
		{"BINARY", "0B100101", "0B100101", INT_P, ""},
		{"float", "0.42", "0.42", NUMERIC_P, ""},
		{
			"max_float",
			"1.79769313486231570814527423731704356798070e+308",
			"1.79769313486231570814527423731704356798070e+308",
			NUMERIC_P,
			"",
		},
		{
			"min_float", // without -
			"4.9406564584124654417656879286822137236505980e-324",
			"4.9406564584124654417656879286822137236505980e-324",
			NUMERIC_P,
			"",
		},

		// https://go.dev/ref/spec#Integer_literals
		{"go_int_example_01", "42", "42", INT_P, ""},
		{"go_int_example_02", "4_2", "4_2", INT_P, ""},
		{
			"go_int_example_03",
			"0600", // Not supported by Postgres
			"0600",
			INT_P,
			"trailing junk after numeric literal at path:1:5",
		},
		{
			"go_int_example_04",
			"0_600", // Not supported by Postgres
			"0_600",
			INT_P,
			"trailing junk after numeric literal at path:1:6",
		},
		{"go_int_example_05", "0o600", "0o600", INT_P, ""},
		{"go_int_example_06", "0o600", "0o600", INT_P, ""},
		{"go_int_example_07", "0O600", "0O600", INT_P, ""},
		{"go_int_example_08", "0xBadFace", "0xBadFace", INT_P, ""},
		{"go_int_example_09", "0xBad_Face", "0xBad_Face", INT_P, ""},
		{
			"go_int_example_10",
			"0x_67_7a_2f_cc_40_c6", // Not supported by Postgres
			"0x_67_7a_2f_cc_40_c6",
			INT_P,
			"underscore disallowed at start of numeric literal at path:1:21",
		},
		{
			"go_int_example_11",
			"170141183460469231731687303715884105727",
			"170141183460469231731687303715884105727",
			INT_P,
			"",
		},
		{
			"go_int_example_12",
			"170_141183_460469_231731_687303_715884_105727",
			"170_141183_460469_231731_687303_715884_105727",
			INT_P,
			"",
		},
		{"go_int_example_13", "_42", "_42", IDENT_P, ""},
		{"go_int_example_14", "42_", "42_", INT_P, "'_' must separate successive digits at path:1:4"},
		{"go_int_example_15", "42_", "42_", INT_P, "'_' must separate successive digits at path:1:4"},
		{"go_int_example_16", "4__2", "4__2", INT_P, "'_' must separate successive digits at path:1:5"},
		{"go_int_example_17", "0_xBadFace", "0_", INT_P, "'_' must separate successive digits at path:1:3"},

		// https://go.dev/ref/spec#Floating-point_literals
		{"go_float_example_01", "0.", "0.", NUMERIC_P, ""},
		{"go_float_example_02", "72.40", "72.40", NUMERIC_P, ""},
		{
			"go_float_example_03",
			"072.40", // Not supported by Postgres
			"072.40",
			NUMERIC_P,
			"trailing junk after numeric literal at path:1:7",
		},
		{"go_float_example_04", "2.71828", "2.71828", NUMERIC_P, ""},
		{"go_float_example_05", "1.e+0", "1.e+0", NUMERIC_P, ""},
		{"go_float_example_06", "6.67428e-11", "6.67428e-11", NUMERIC_P, ""},
		{"go_float_example_06", "1E6", "1E6", NUMERIC_P, ""},
		{"go_float_example_07", ".25", ".25", NUMERIC_P, ""},
		{"go_float_example_08", ".12345E+5", ".12345E+5", NUMERIC_P, ""},
		{"go_float_example_09", "1_5.", "1_5.", NUMERIC_P, ""},
		{"go_float_example_10", "0.15e+0_2", "0.15e+0_2", NUMERIC_P, ""},
		{
			"go_float_example_11",
			"0x1p-2", // Not supported by Postgres
			"0x1p-2",
			NUMERIC_P,
			"trailing junk after numeric literal at path:1:7",
		},
		{
			"go_float_example_12",
			"0x2.p10", // Not supported by Postgres
			"0x2.p10",
			NUMERIC_P,
			"trailing junk after numeric literal at path:1:8",
		},
		{
			"go_float_example_13",
			"0x1.Fp+0", // Not supported by Postgres
			"0x1.Fp+0",
			NUMERIC_P,
			"trailing junk after numeric literal at path:1:9",
		},
		{
			"go_float_example_14",
			"0X.8p-0", // Not supported by Postgres
			"0X.8p-0",
			NUMERIC_P,
			"trailing junk after numeric literal at path:1:8",
		},
		{
			"go_float_example_15",
			"0X_1FFFP-16", // Not supported by Postgres
			"0X_1FFFP-16",
			NUMERIC_P,
			"underscore disallowed at start of numeric literal at path:1:12",
		},
		{"go_float_example_16", "0x15e-2", "0x15e", INT_P, ""}, // (integer subtraction)
		{"go_float_example_17", "0x.p1", "0x.p1", NUMERIC_P, "hexadecimal literal has no digits at path:1:4"},
		{"go_float_example_18", "1p-2", "1p-2", NUMERIC_P, "'p' exponent requires hexadecimal mantissa at path:1:2"},
		{"go_float_example_19", "0x1.5e-2", "0x1.5e", NUMERIC_P, "hexadecimal mantissa requires a 'p' exponent at path:1:7"},
		{"go_float_example_20", "1_.5", "1_.5", NUMERIC_P, "'_' must separate successive digits at path:1:5"},
		{"go_float_example_21", "1._5", "1._5", NUMERIC_P, "'_' must separate successive digits at path:1:5"},
		{"go_float_example_22", "1.5_e1", "1.5_e1", NUMERIC_P, "'_' must separate successive digits at path:1:7"},
		{"go_float_example_23", "1.5e_1", "1.5e_1", NUMERIC_P, "'_' must separate successive digits at path:1:7"},
		{"go_float_example_24", "1.5e1_", "1.5e1_", NUMERIC_P, "'_' must separate successive digits at path:1:7"},

		// Errors
		{
			"underscore_hex_early",
			"0x_1EEEFFFF",
			"0x_1EEEFFFF",
			INT_P,
			"underscore disallowed at start of numeric literal at path:1:12",
		},
		{
			"underscore_octal_early",
			"0o_273",
			"0o_273",
			INT_P,
			"underscore disallowed at start of numeric literal at path:1:7",
		},
		{
			"underscore_binary_early",
			"0b_100101",
			"0b_100101",
			INT_P,
			"underscore disallowed at start of numeric literal at path:1:10",
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sym := &pathSymType{}
			l := newLexer(tc.num)
			a.Equal(l.Lex(sym), tc.tok)
			a.Equal(tc.exp, sym.str)

			if tc.err == "" {
				a.Empty(l.errors)
			} else {
				a.Equal([]string{tc.err}, l.errors)
			}
		})
	}
}

func TestScanVariable(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tc := range []struct {
		name     string
		variable string
		exp      string
		tok      int
		err      string
	}{
		{"xxx", "$xxx", "xxx", VARIABLE_P, ""},
		{"num_prefix", "$42x", "42x", VARIABLE_P, ""},
		{"numeric", "$999", "999", VARIABLE_P, ""},
		{"mixed_case", "$XoX", "XoX", VARIABLE_P, ""},
		{"underscore", "$x_y_z", "x_y_z", VARIABLE_P, ""},
		{"mixed_case", "$XoX", "XoX", VARIABLE_P, ""},
		{"unicode", "$XöX", "XöX", VARIABLE_P, ""},
		{"emoji", "$🤘🏻🤘🏻", "", '$', ""},
		{"quoted", `$"xxx"`, "xxx", VARIABLE_P, ""},
		{"with_spaces", `$"hi there"`, "hi there", VARIABLE_P, ""},
		{"with_unicode", `$"Go on 🎉"`, "Go on 🎉", VARIABLE_P, ""},
		{"surrogate_pair", `$"\uD834\uDD1E"`, "\U0001D11E", VARIABLE_P, ""},
		{"root", "$", "", '$', ""},
		{"root_path", "$.x.y", "", '$', ""},
		{"root_path", "$.x.y", "", '$', ""},
		{
			"null_byte",
			`$"go \x00"`,
			"",
			VARIABLE_P,
			"invalid hexadecimal character sequence at path:1:10",
		},
		{
			"invalid_hex",
			`$"LO\xzz"`,
			"",
			VARIABLE_P,
			"invalid hexadecimal character sequence at path:1:8",
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sym := &pathSymType{}
			l := newLexer(tc.variable)
			a.Equal('$', l.scanner.Scan())
			a.Equal(l.scanVariable(sym), tc.tok)
			a.Equal(tc.exp, sym.str)

			if tc.err == "" {
				a.Empty(l.errors)
			} else {
				a.Equal([]string{tc.err}, l.errors)
			}
		})
	}
}

func TestScanComment(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tc := range []struct {
		name string
		path string
		tok  int
		err  string
	}{
		{"simple", "/* foo */", scanner.Comment, ""},
		{"stars", "/*foo****/", scanner.Comment, ""},
		{"escape_star", "/*foo \\**/", scanner.Comment, ""},
		{"escape_other", "/*foo \\! */", scanner.Comment, ""},
		{"multi_word", "/* foo bar baz */", scanner.Comment, ""},
		{"multi_line", "/* foo bar\nbaz */", scanner.Comment, ""},
		{"multi_line_prefix", "/* foo bar\n * baz */", scanner.Comment, ""},
		{"EOF", "/* foo ", 0, "unexpected end of comment at path:1:8"},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			l := newLexer(tc.path)
			a.Equal('/', l.scanner.Scan())
			a.Equal(l.scanComment(), tc.tok)
			a.Equal(len(tc.path), l.scanner.Pos().Offset)

			if tc.err == "" {
				a.Empty(l.errors)
			} else {
				a.Equal([]string{tc.err}, l.errors)
			}
		})
	}
}

func TestScanOperator(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tc := range []struct {
		name string
		op   string
		tok  int
		exp  string
	}{
		{"equal_to", "==", EQUAL_P, "=="},
		{"equal_sign_eof", "=", '=', "="},
		{"equal_sign_stop", "=[xyz]", '=', "="},
		{"ge", ">=", GREATEREQUAL_P, ">="},
		{"ge_stop", ">=x", GREATEREQUAL_P, ">="},
		{"gt", ">", GREATER_P, ">"},
		{"gt_stop", ">{x}", GREATER_P, ">"},
		{"le", "<=", LESSEQUAL_P, "<="},
		{"le_stop", "<=x", LESSEQUAL_P, "<="},
		{"le_ne", "<>", NOTEQUAL_P, "!="},
		{"le_ne_stop", "<>x", NOTEQUAL_P, "!="},
		{"lt", "<", LESS_P, "<"},
		{"lt_stop", "<{x}", LESS_P, "<"},
		{"not", "!", NOT_P, "!"},
		{"not_stop", "!x", NOT_P, "!"},
		{"not_equal", "!=", NOTEQUAL_P, "!="},
		{"not_equal_stop", "!=!", NOTEQUAL_P, "!="},
		{"and", "&&", AND_P, "&&"},
		{"and_stop", "&&.", AND_P, "&&"},
		{"ampersand", "&", '&', "&"},
		{"ampersand_stop", "&=", '&', "&"},
		{"or", "||", OR_P, "||"},
		{"or_stop", "||.", OR_P, "||"},
		{"pipe", "|", '|', "|"},
		{"pipe_stop", "|=", '|', "|"},
		{"any", "**", ANY_P, "**"},
		{"any_stop", "**.", ANY_P, "**"},
		{"star", "*", '*', "*"},
		{"star_stop", "*=", '*', "*"},
		{"something_else", "^^", '^', "^"},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			l := newLexer(tc.op)
			tok := l.scanner.Scan()
			a.Equal(rune(tc.op[0]), tok)
			sym := &pathSymType{str: string(tok)}
			a.Equal(l.scanOperator(sym, tok), tc.tok)
			a.Equal(tc.exp, sym.str)
			a.Empty(l.errors)
		})
	}
}

func TestLexer(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tc := range []struct {
		name string
		path string
		exp  string
		tok  int
		err  string
	}{
		{"root", "$", "$", '$', ""},
		{"plus", "+", "+", '+', ""},
		{"percent", "%", "%", '%', ""},
		{"ident", "hello", "hello", IDENT_P, ""},
		{"boolean", "true", "true", TRUE_P, ""},
		{"keyword", "is", "is", IS_P, ""},
		{"integer", "42", "42", INT_P, ""},
		{"float", "42.0", "42.0", NUMERIC_P, ""},
		{"string", `"xxx"`, "xxx", STRING_P, ""},
		{"string_with_spaces", `"hi there"`, "hi there", STRING_P, ""},
		{"string_with_unicode", `"Go on 🎉"`, "Go on 🎉", STRING_P, ""},
		{"variable", `$xxx`, "xxx", VARIABLE_P, ""},
		{"quoted_variable", `$"xxx"`, "xxx", VARIABLE_P, ""},
		{"variable_with_spaces", `$"hi there"`, "hi there", VARIABLE_P, ""},
		{"variable_with_unicode", `$"Go on 🎉"`, "Go on 🎉", VARIABLE_P, ""},
		{"comment", "/* foo */", "", scanner.EOF, ""},
		{"comment_token", "/* foo */ $", "$", '$', ""},
		{"comment", "/* foo */", "", scanner.EOF, ""},
		{"not_comment", "/ foo", "/", '/', ""},
		{"op", "==foo", "==", EQUAL_P, ""},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sym := &pathSymType{}
			l := newLexer(tc.path)
			a.Equal(l.Lex(sym), tc.tok)
			a.Equal(tc.exp, sym.str)

			if tc.err == "" {
				a.Empty(l.errors)
			} else {
				a.Equal([]string{tc.err}, l.errors)
			}
		})
	}
}
