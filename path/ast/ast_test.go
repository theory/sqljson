package ast

import (
	"fmt"
	"math"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConstNode(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tc := range []struct {
		name string
		node ConstNode
		str  string
	}{
		{"root", ConstRoot, "$"},
		{"current", ConstCurrent, "@"},
		{"last", ConstLast, "last"},
		{"any_array", ConstAnyArray, "[*]"},
		{"any_key", ConstAnyKey, "*"},
		{"true", ConstTrue, "true"},
		{"false", ConstFalse, "false"},
		{"null", ConstNull, "null"},
		{"unknown", -1, "ConstNode(-1)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a.Implements((*Node)(nil), tc.node)
			a.Equal(tc.str, tc.node.String())
		})
	}
}

func TestBinaryOperatorNode(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tc := range []struct {
		name string
		node BinaryOperator
		str  string
	}{
		{"and", BinaryAnd, "&&"},
		{"or", BinaryOr, "||"},
		{"equal", BinaryEqual, "=="},
		{"not_equal", BinaryNotEqual, "!="},
		{"less", BinaryLess, "<"},
		{"less_equal", BinaryLessOrEqual, "<="},
		{"greater", BinaryGreater, ">"},
		{"greater_equal", BinaryGreaterOrEqual, ">="},
		{"starts_with", BinaryStartsWith, "starts with"},
		{"add", BinaryAdd, "+"},
		{"sub", BinarySub, "-"},
		{"mul", BinaryMul, "*"},
		{"div", BinaryDiv, "/"},
		{"mod", BinaryMod, "%"},
		{"subscript", BinarySubscript, "to"},
		{"decimal", BinaryDecimal, ".decimal()"},
		{"unknown", -1, "BinaryOperator(-1)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a.Implements((*Node)(nil), tc.node)
			a.Equal(tc.str, tc.node.String())
		})
	}
}

func TestUnaryOperatorNode(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tc := range []struct {
		name string
		node UnaryOperator
		str  string
	}{
		{"exists", UnaryExists, "exists"},
		{"not", UnaryNot, "!"},
		{"is_unknown", UnaryIsUnknown, "is unknown"},
		{"plus", UnaryPlus, "+"},
		{"minus", UnaryMinus, "-"},
		{"filter", UnaryFilter, "?"},
		{"datetime", UnaryDateTime, ".datetime"},
		{"time", UnaryTime, ".time"},
		{"time_tz", UnaryTimeTZ, ".time_tz"},
		{"timestamp", UnaryTimestamp, ".timestamp"},
		{"timestamp_tz", UnaryTimestampTZ, ".timestamp_tz"},
		{"unknown", -1, "UnaryOperator(-1)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a.Implements((*Node)(nil), tc.node)
			a.Equal(tc.str, tc.node.String())
		})
	}
}

func TestMethodNode(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tc := range []struct {
		name string
		node MethodNode
		str  string
	}{
		{"abs", MethodAbs, ".abs()"},
		{"size", MethodSize, ".size()"},
		{"type", MethodType, ".type()"},
		{"floor", MethodFloor, ".floor()"},
		{"ceiling", MethodCeiling, ".ceiling()"},
		{"keyvalue", MethodKeyValue, ".keyvalue()"},
		{"bigint", MethodBigint, ".bigint()"},
		{"boolean", MethodBoolean, ".boolean()"},
		{"date", MethodDate, ".date()"},
		{"integer", MethodInteger, ".integer()"},
		{"number", MethodNumber, ".number()"},
		{"string", MethodString, ".string()"},
		{"unknown", -1, "MethodNode(-1)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a.Implements((*Node)(nil), tc.node)
			a.Equal(tc.str, tc.node.String())
		})
	}
}

func TestStringNodes(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tc := range []struct {
		name string
		expr string
		val  string
		str  string
	}{
		{"word", "word", "word", `"word"`},
		{"space", "hi there", "hi there", `"hi there"`},
		{"unicode", "löl", "löl", `"löl"`},
		{"backslash", `foo\nbar`, `foo\nbar`, `"foo\\nbar"`},
		{"quote", `"foo"`, `"foo"`, `"\"foo\""`},
		{"newline", "hi\nthere", "hi\nthere", `"hi\nthere"`},
		{"tab", "hi\tthere", "hi\tthere", `"hi\tthere"`},
		{"ff", "hi\fthere", "hi\fthere", `"hi\fthere"`},
		{"return", "hi\rthere", "hi\rthere", `"hi\rthere"`},
		{"vertical_tab", "hi\vthere", "hi\vthere", `"hi\vthere"`},
		{"backspace", "hi\bthere", "hi\bthere", `"hi\bthere"`},
		{"emoji", "🤘🏻🎉🐳", "🤘🏻🎉🐳", `"🤘🏻🎉🐳"`},
		{"multibyte", "\U0001D11E", "𝄞", `"𝄞"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			str := NewString(tc.expr)
			a.Equal(tc.str, str.String())

			variable := NewVariable(tc.expr)
			a.Equal(tc.val, variable.Text())
			a.Equal("$"+tc.str, variable.String())

			key := NewString(tc.expr)
			a.Equal(tc.val, key.Text())
			a.Equal(tc.str, key.String())
		})
	}
}

func TestNumericNode(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tc := range []struct {
		name string
		num  string
		val  float64
		str  string
		err  string
	}{
		{"number", "42.3", 42.3, "42.3", ""},
		{"zero_dot", "0.", 0.0, "0.", ""},
		{
			name: "max_float",
			num:  fmt.Sprintf("%v", math.MaxFloat64),
			val:  math.MaxFloat64,
			str:  fmt.Sprintf("%v", math.MaxFloat64),
		},
		{
			name: "min_float",
			num:  fmt.Sprintf("%v", math.SmallestNonzeroFloat64),
			val:  math.SmallestNonzeroFloat64,
			str:  fmt.Sprintf("%v", math.SmallestNonzeroFloat64),
		},
		{
			name: "invalid_float",
			num:  "xyz.4",
			val:  0,
			str:  "xyz.4",
			err:  `strconv.ParseFloat: parsing "xyz.4": invalid syntax`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.err != "" {
				a.PanicsWithError(tc.err, func() { NewNumeric(tc.num) })
				return
			}

			num := NewNumeric(tc.num)
			a.Equal(tc.str, num.String())
			//nolint:testifylint
			a.Equal(tc.val, num.Float())
		})
	}
}

func TestIntegerNode(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tc := range []struct {
		name string
		num  string
		val  int64
		str  string
		err  string
	}{
		{"number", "42", 42, "42", ""},
		{
			name: "max_int",
			num:  strconv.Itoa(math.MaxInt64),
			val:  math.MaxInt64,
			str:  strconv.Itoa(math.MaxInt64),
		},
		{
			name: "min_int",
			num:  strconv.Itoa(math.MinInt32),
			val:  math.MinInt32,
			str:  strconv.Itoa(math.MinInt32),
		},
		{
			name: "invalid_int",
			num:  "123x",
			val:  0,
			str:  "123x",
			err:  `strconv.ParseInt: parsing "123x": invalid syntax`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.err != "" {
				a.PanicsWithError(tc.err, func() { NewInteger(tc.num) })
				return
			}

			num := NewInteger(tc.num)
			a.Equal(tc.str, num.String())
			a.Equal(tc.val, num.Int())
		})
	}
}

func TestBinaryNode(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tc := range []struct {
		name  string
		left  Node
		op    BinaryOperator
		right Node
		str   string
		err   string
	}{
		{
			name:  "equal",
			left:  NewInteger("42"),
			op:    BinaryEqual,
			right: NewInteger("99"),
			str:   "42 == 99",
		},
		{
			name:  "equal_string",
			left:  ConstCurrent,
			op:    BinaryEqual,
			right: NewString("xyz"),
			str:   `@ == "xyz"`,
		},
		{
			name:  "not_equal",
			left:  NewInteger("42"),
			op:    BinaryNotEqual,
			right: NewInteger("99"),
			str:   "42 != 99",
		},
		{
			name:  "lt",
			left:  NewInteger("42"),
			op:    BinaryLess,
			right: NewInteger("99"),
			str:   "42 < 99",
		},
		{
			name:  "le",
			left:  NewInteger("42"),
			op:    BinaryLessOrEqual,
			right: NewInteger("99"),
			str:   "42 <= 99",
		},
		{
			name:  "gt",
			left:  NewInteger("42"),
			op:    BinaryGreater,
			right: NewInteger("99"),
			str:   "42 > 99",
		},
		{
			name:  "ge",
			left:  NewInteger("42"),
			op:    BinaryGreaterOrEqual,
			right: NewInteger("99"),
			str:   "42 >= 99",
		},
		{
			name:  "and",
			left:  NewBinary(BinaryEqual, ConstCurrent, ConstTrue),
			op:    BinaryAnd,
			right: NewBinary(BinaryEqual, NewVariable("xxx"), NewInteger("42")),
			str:   `@ == true && $"xxx" == 42`,
		},
		{
			name:  "or",
			left:  NewBinary(BinaryEqual, ConstCurrent, ConstTrue),
			op:    BinaryOr,
			right: NewBinary(BinaryEqual, NewVariable("xxx"), NewInteger("42")),
			str:   `@ == true || $"xxx" == 42`,
		},
		{
			name:  "add",
			left:  NewInteger("42"),
			op:    BinaryAdd,
			right: NewNumeric("98.6"),
			str:   `42 + 98.6`,
		},
		{
			name:  "subtract",
			left:  NewInteger("42"),
			op:    BinarySub,
			right: NewNumeric("98.6"),
			str:   `42 - 98.6`,
		},
		{
			name:  "multiply",
			left:  NewInteger("42"),
			op:    BinaryMul,
			right: NewNumeric("98.6"),
			str:   `42 * 98.6`,
		},
		{
			name:  "divide",
			left:  NewInteger("42"),
			op:    BinaryDiv,
			right: NewNumeric("98.6"),
			str:   `42 / 98.6`,
		},
		{
			name:  "modulo",
			left:  NewInteger("42"),
			op:    BinaryMod,
			right: NewInteger("12"),
			str:   `42 % 12`,
		},
		{
			name:  "starts_with",
			left:  NewString("food"),
			op:    BinaryStartsWith,
			right: NewString("foo"),
			str:   `"food" starts with "foo"`,
		},
		// case jpiStartsWith:
		{
			name:  "subscript",
			left:  NewInteger("42"),
			op:    BinarySubscript,
			right: NewInteger("99"),
			str:   "42 to 99",
		},
		{
			name:  "left_subscript",
			left:  NewInteger("42"),
			op:    BinarySubscript,
			right: nil,
			str:   "42",
		},
		{
			name:  "decimal_l_r",
			left:  NewInteger("42"),
			op:    BinaryDecimal,
			right: NewInteger("99"),
			str:   ".decimal(42,99)",
		},
		{
			name: "decimal_l",
			left: NewInteger("42"),
			op:   BinaryDecimal,
			str:  ".decimal(42)",
		},
		{
			name:  "decimal_r",
			op:    BinaryDecimal,
			right: NewInteger("99"),
			str:   ".decimal(,99)",
		},
		{
			name: "decimal",
			op:   BinaryDecimal,
			str:  ".decimal()",
		},
		{
			name: "unknown_op",
			op:   BinaryOperator(-1),
			err:  "Unknown binary operator BinaryOperator(-1)",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			node := NewBinary(tc.op, tc.left, tc.right)
			a.Equal(tc.op, node.Operator())
			a.Equal(tc.left, node.Left())
			a.Equal(tc.right, node.Right())
			if tc.err == "" {
				a.Equal(tc.str, node.String())
			} else {
				a.PanicsWithValue(tc.err, func() { _ = node.String() })
			}
		})
	}
}

func TestUnaryNode(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tc := range []struct {
		name string
		op   UnaryOperator
		node Node
		str  string
	}{
		{
			name: "exists",
			op:   UnaryExists,
			node: NewInteger("99"),
			str:  "exists (99)",
		},
		{
			name: "is_unknown",
			op:   UnaryIsUnknown,
			node: NewInteger("99"),
			str:  "(99) is unknown",
		},
		{
			name: "not",
			op:   UnaryNot,
			node: NewInteger("99"),
			str:  "!(99)",
		},
		{
			name: "plus",
			op:   UnaryPlus,
			node: NewInteger("99"),
			str:  "+99",
		},
		{
			name: "minus",
			op:   UnaryMinus,
			node: NewInteger("99"),
			str:  "-99",
		},
		{
			name: "filter",
			op:   UnaryFilter,
			node: NewInteger("99"),
			str:  "?(99)",
		},
		{
			name: "datetime",
			op:   UnaryDateTime,
			node: NewInteger("99"),
			str:  ".datetime(99)",
		},
		{
			name: "datetime_nil",
			op:   UnaryDateTime,
			str:  ".datetime()",
		},
		{
			name: "time",
			op:   UnaryTime,
			node: NewInteger("99"),
			str:  ".time(99)",
		},
		{
			name: "time_tz",
			op:   UnaryTimeTZ,
			node: NewInteger("99"),
			str:  ".time_tz(99)",
		},
		{
			name: "timestamp",
			op:   UnaryTimestamp,
			node: NewInteger("99"),
			str:  ".timestamp(99)",
		},
		{
			name: "timestamp_tz",
			op:   UnaryTimestampTZ,
			node: NewInteger("99"),
			str:  ".timestamp_tz(99)",
		},
		{
			name: "unknown_op",
			op:   UnaryOperator(-1),
			node: NewInteger("99"),
			str:  "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			node := NewUnary(tc.op, tc.node)
			a.Equal(tc.op, node.Operator())
			a.Equal(tc.node, node.Node())
			a.Equal(tc.str, node.String())
		})
	}
}

func TestAccessorNode(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tc := range []struct {
		name  string
		nodes []Node
		str   string
	}{
		{
			name:  "single_key",
			nodes: []Node{NewKey("foo")},
			str:   `."foo"`,
		},
		{
			name:  "two_keys",
			nodes: []Node{NewKey("foo"), NewKey("bar")},
			str:   `."foo"."bar"`,
		},
		{
			name:  "numeric",
			nodes: []Node{NewNumeric("42.2")},
			str:   `.42.2`,
		},
		{
			name:  "numeric_then_key",
			nodes: []Node{NewNumeric("42.2"), NewKey("bar")},
			str:   `.(42.2)."bar"`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			node := NewAccessor(tc.nodes)
			a.Equal(tc.str, node.String())
		})
	}
}

func TestArrayIndexNode(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tc := range []struct {
		name  string
		nodes []Node
		str   string
	}{
		{
			name:  "single_subscript",
			nodes: []Node{NewBinary(BinarySubscript, NewInteger("1"), NewInteger("4"))},
			str:   `[1 to 4]`,
		},
		{
			name:  "start_only",
			nodes: []Node{NewBinary(BinarySubscript, NewInteger("4"), nil)},
			str:   `[4]`,
		},
		{
			name: "two_subscripts",
			nodes: []Node{
				NewBinary(BinarySubscript, NewInteger("1"), NewInteger("4")),
				NewBinary(BinarySubscript, NewInteger("6"), NewInteger("7")),
			},
			str: `[1 to 4,6 to 7]`,
		},
		{
			name: "complex_subscripts",
			nodes: []Node{
				NewBinary(BinarySubscript, NewInteger("1"), NewInteger("2")),
				NewBinary(BinarySubscript, NewBinary(BinaryAdd, ConstCurrent, NewInteger("3")), nil),
				NewBinary(BinarySubscript, NewInteger("6"), nil),
			},
			str: `[1 to 2,@ + 3,6]`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			node := NewArrayIndex(tc.nodes)
			a.Equal(tc.str, node.String())
		})
	}
}

func TestAnyNode(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tc := range []struct {
		name  string
		first int
		last  int
		str   string
	}{
		{
			name:  "first_last",
			first: 0,
			last:  4,
			str:   `**{0 to 4}`,
		},
		{
			name:  "neg_first_last",
			first: -1,
			last:  4,
			str:   `**{last to 4}`,
		},
		{
			name:  "first_neg_last",
			first: 4,
			last:  -1,
			str:   `**{4 to last}`,
		},
		{
			name:  "zero_neg",
			first: 0,
			last:  -1,
			str:   `**`,
		},
		{
			name:  "neg_neg",
			first: -1,
			last:  -1,
			str:   `**{last}`,
		},
		{
			name:  "equal",
			first: 2,
			last:  2,
			str:   `**{2}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			node := NewAny(tc.first, tc.last)
			a.Equal(tc.str, node.String())
		})
	}
}

func TestRegexNode(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	r := require.New(t)

	for _, tc := range []struct {
		name    string
		node    Node
		re      string
		flag    string
		flags   regexFlags
		str     string
		err     string
		match   []string
		noMatch []string
	}{
		{
			name:    "dot",
			node:    NewString("foo"),
			re:      `.`,
			str:     `like_regex "foo" "."`,
			match:   []string{"x", "abc", "123"},
			noMatch: []string{"", "\n"},
		},
		{
			name:    "anchor",
			node:    NewString("foo"),
			re:      `^a`,
			str:     `like_regex "foo" "^a"`,
			match:   []string{"a", "abc", "a\nb\nc"},
			noMatch: []string{"", "\na\n"},
		},
		{
			name:    "flags",
			node:    NewString("fOo"),
			re:      `^o.`,
			flag:    "ims",
			flags:   regexFlags(regexDotAll | regexICase | regexMLine),
			str:     `like_regex "fOo" "^o." flag "ism"`,
			match:   []string{"ox", "Ox", "oO", "a\no\nc"},
			noMatch: []string{"xoxo", "a\nxo"},
		},
		{
			name:    "quote",
			node:    NewString("foo"),
			re:      `xa+`,
			flag:    "iqsm",
			flags:   regexFlags(regexICase | regexQuote | regexDotAll | regexMLine),
			str:     `like_regex "foo" "xa+" flag "ismq"`,
			match:   []string{"xa+", "XA+", "\nXa+", "bmXa+"},
			noMatch: []string{`xa\+`, "x"},
		},
		{
			name: "bad_flags",
			node: NewString("foo"),
			re:   `.`,
			flag: "x",
			err:  `parser: XQuery "x" flag (expanded regular expressions) is not implemented`,
		},
		{
			name: "bad_pattern",
			node: NewString("foo"),
			re:   `.(hi`,
			err:  "parser: error parsing regexp: missing closing ): `.(hi`",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			node, err := NewRegex(tc.node, tc.re, tc.flag)
			if tc.err != "" {
				r.EqualError(err, tc.err)
				r.ErrorIs(err, ErrAST)
				a.Nil(node)
				return
			}

			r.NoError(err)
			r.NotNil(node)
			a.Equal(tc.re, node.pattern)
			a.Equal(tc.flags, node.flags)
			a.Equal(tc.node, node.node)
			a.Equal(tc.str, node.String())

			// Make sure the regex matches what it should.
			re := node.Regexp()
			r.NotNil(re)

			for _, str := range tc.match {
				a.True(re.MatchString(str))
			}

			for _, str := range tc.noMatch {
				if !a.False(re.MatchString(str)) {
					t.Logf("Unexpectedly matched %q", str)
				}
			}
		})
	}
}

func TestAST(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tc := range []struct {
		name string
		node Node
	}{
		{"string", NewString("foo")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tree := New(true, tc.node)
			a.True(tree.strict)
			a.Equal(tc.node, tree.root)
			tree = New(false, tc.node)
			a.False(tree.strict)
			a.Equal(tc.node, tree.root)
		})
	}
}
