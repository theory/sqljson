package ast

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConstNode(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tc := range []struct {
		name     string
		node     ConstNode
		str      string
		inKeyStr string
	}{
		{"root", ConstRoot, "$", ""},
		{"current", ConstCurrent, "@", ""},
		{"last", ConstLast, "last", ""},
		{"any_array", ConstAnyArray, "[*]", ""},
		{"any_key", ConstAnyKey, "*", ".*"},
		{"true", ConstTrue, "true", ""},
		{"false", ConstFalse, "false", ""},
		{"null", ConstNull, "null", ""},
		{"unknown", -1, "ConstNode(-1)", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a.Implements((*Node)(nil), tc.node)
			a.Equal(tc.str, tc.node.String())
			a.Equal(lowestPriority, tc.node.priority())

			// Test writeTo.
			buf := new(strings.Builder)
			tc.node.writeTo(buf, false, false)
			a.Equal(tc.str, buf.String())

			// Test writeTo with knKey true.
			buf.Reset()
			tc.node.writeTo(buf, true, false)
			if tc.inKeyStr == "" {
				tc.inKeyStr = tc.str
			}
			a.Equal(tc.inKeyStr, buf.String())
		})
	}
}

func TestBinaryOperator(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tc := range []struct {
		name  string
		op    BinaryOperator
		str   string
		prior uint8
	}{
		{"and", BinaryAnd, "&&", 1},
		{"or", BinaryOr, "||", 0},
		{"equal", BinaryEqual, "==", 2},
		{"not_equal", BinaryNotEqual, "!=", 2},
		{"less", BinaryLess, "<", 2},
		{"less_equal", BinaryLessOrEqual, "<=", 2},
		{"greater", BinaryGreater, ">", 2},
		{"greater_equal", BinaryGreaterOrEqual, ">=", 2},
		{"starts_with", BinaryStartsWith, "starts with", 2},
		{"add", BinaryAdd, "+", 3},
		{"sub", BinarySub, "-", 3},
		{"mul", BinaryMul, "*", 4},
		{"div", BinaryDiv, "/", 4},
		{"mod", BinaryMod, "%", 4},
		{"subscript", BinarySubscript, "to", 6},
		{"decimal", BinaryDecimal, ".decimal()", 6},
		{"unknown", -1, "BinaryOperator(-1)", 6},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a.Equal(tc.str, tc.op.String())
			a.Equal(tc.prior, tc.op.priority())
		})
	}
}

func TestUnaryOperator(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tc := range []struct {
		name  string
		op    UnaryOperator
		str   string
		prior uint8
	}{
		{"exists", UnaryExists, "exists", 6},
		{"not", UnaryNot, "!", 6},
		{"is_unknown", UnaryIsUnknown, "is unknown", 6},
		{"plus", UnaryPlus, "+", 5},
		{"minus", UnaryMinus, "-", 5},
		{"filter", UnaryFilter, "?", 6},
		{"datetime", UnaryDateTime, ".datetime", 6},
		{"time", UnaryTime, ".time", 6},
		{"time_tz", UnaryTimeTZ, ".time_tz", 6},
		{"timestamp", UnaryTimestamp, ".timestamp", 6},
		{"timestamp_tz", UnaryTimestampTZ, ".timestamp_tz", 6},
		{"unknown", -1, "UnaryOperator(-1)", 6},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a.Equal(tc.str, tc.op.String())
			a.Equal(tc.prior, tc.op.priority())
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
			a.Equal(lowestPriority, tc.node.priority())

			// Test writeTo.
			buf := new(strings.Builder)
			tc.node.writeTo(buf, false, false)
			a.Equal(tc.str, buf.String())
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
			a.Implements((*Node)(nil), str)
			a.Equal(tc.str, str.String())
			a.Equal(lowestPriority, str.priority())
			buf := new(strings.Builder)
			str.writeTo(buf, false, false)
			a.Equal(tc.str, buf.String())

			variable := NewVariable(tc.expr)
			a.Implements((*Node)(nil), variable)
			a.Equal(tc.val, variable.Text())
			a.Equal("$"+tc.str, variable.String())
			a.Equal(lowestPriority, variable.priority())
			buf.Reset()
			variable.writeTo(buf, false, false)
			a.Equal("$"+tc.str, buf.String())

			key := NewString(tc.expr)
			a.Implements((*Node)(nil), key)
			a.Equal(tc.val, key.Text())
			a.Equal(tc.str, key.String())
			a.Equal(lowestPriority, key.priority())
			buf.Reset()
			key.writeTo(buf, false, false)
			a.Equal(tc.str, buf.String())
		})
	}
}

//nolint:dupl
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
		{"zero_dot", "0.", 0.0, "0", ""},
		{"dot_one", ".1", 0.1, "0.1", ""},
		{"zero_dot_zero", "0.0", 0.0, "0", ""},
		{"zero_dot_000", "0.000", 0.0, "0", ""},
		{"expo", "0.0010e-1", 0.0001, "0.0001", ""},
		{"pos_expo", "0.0010e+2", 0.1, "0.1", ""},
		{"dot_001", ".001", 0.001, "0.001", ""},
		{"dot_expo", "1.e1", 10, "10", ""},
		{"one_expo_3", "1e3", 1000, "1000", ""},
		{"1_dot_2e3", "1.2e3", 1200, "1200", ""},
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
			a.Implements((*Node)(nil), num)
			a.Equal(tc.num, num.Literal())
			a.Equal(tc.str, num.String())
			a.Equal(lowestPriority, num.priority())
			//nolint:testifylint
			a.Equal(tc.val, num.Float())

			// Test writeTo.
			buf := new(strings.Builder)
			num.writeTo(buf, false, false)
			a.Equal(tc.str, buf.String())

			// Test writeTo withParens true.
			buf.Reset()
			num.writeTo(buf, false, true)
			a.Equal("("+tc.str+")", buf.String())
		})
	}
}

//nolint:dupl
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
		{"underscores", "1_000_000", 1_000_000, "1000000", ""},
		{"binary", "0b100101", 37, "37", ""},
		{"octal", "0o273", 187, "187", ""},
		{"hex", "0x42F", 1071, "1071", ""},
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
			a.Implements((*Node)(nil), num)
			a.Equal(tc.num, num.Literal())
			a.Equal(tc.str, num.String())
			a.Equal(lowestPriority, num.priority())
			a.Equal(tc.val, num.Int())

			// Test writeTo.
			buf := new(strings.Builder)
			num.writeTo(buf, false, false)
			a.Equal(tc.str, buf.String())

			// Test writeTo withParens true.
			buf.Reset()
			num.writeTo(buf, false, true)
			a.Equal("("+tc.str+")", buf.String())
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
		{
			name:  "priority_parens",
			op:    BinaryAnd,
			left:  NewBinary(BinaryOr, ConstCurrent, ConstCurrent),
			right: NewBinary(BinaryOr, ConstCurrent, ConstCurrent),
			str:   "(@ || @) && (@ || @)",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			node := NewBinary(tc.op, tc.left, tc.right)
			a.Implements((*Node)(nil), node)
			a.Equal(node.op.priority(), node.priority())
			a.Equal(tc.op, node.Operator())
			a.Equal(tc.left, node.Left())
			a.Equal(tc.right, node.Right())
			if tc.err != "" {
				a.PanicsWithValue(tc.err, func() { _ = node.String() })
				return
			}
			a.Equal(tc.str, node.String())

			// Test writeTo.
			buf := new(strings.Builder)
			node.writeTo(buf, false, false)
			a.Equal(tc.str, buf.String())

			// Test writeTo withParens true
			buf.Reset()
			node.writeTo(buf, false, true)

			//nolint:exhaustive
			switch node.op {
			case BinaryAnd, BinaryOr, BinaryEqual, BinaryNotEqual, BinaryLess,
				BinaryGreater, BinaryLessOrEqual, BinaryGreaterOrEqual,
				BinaryAdd, BinarySub, BinaryMul, BinaryDiv, BinaryMod,
				BinaryStartsWith:
				a.Equal("("+tc.str+")", buf.String())
			default:
				a.Equal(tc.str, buf.String())
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
		{
			name: "priority_parens",
			op:   UnaryPlus,
			node: NewBinary(BinaryOr, ConstCurrent, ConstCurrent),
			str:  "+(@ || @)",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			node := NewUnary(tc.op, tc.node)
			a.Implements((*Node)(nil), node)
			a.Equal(node.op.priority(), node.priority())
			a.Equal(tc.op, node.Operator())
			a.Equal(tc.node, node.Operand())
			a.Equal(tc.str, node.String())

			// Test writeTo.
			buf := new(strings.Builder)
			node.writeTo(buf, false, false)
			a.Equal(tc.str, buf.String())

			// Test writeTo withParens true
			buf.Reset()
			node.writeTo(buf, false, true)

			//nolint:exhaustive
			switch node.op {
			case UnaryPlus, UnaryMinus:
				a.Equal("("+tc.str+")", buf.String())
			default:
				a.Equal(tc.str, buf.String())
			}
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
			str:   `"foo"`,
		},
		{
			name:  "two_keys",
			nodes: []Node{NewKey("foo"), NewKey("bar")},
			str:   `"foo"."bar"`,
		},
		{
			name:  "numeric",
			nodes: []Node{NewNumeric("42.2")},
			str:   `42.2`,
		},
		{
			name:  "numeric_then_key",
			nodes: []Node{NewNumeric("42.2"), NewKey("bar")},
			str:   `(42.2)."bar"`,
		},
		{
			name:  "nested_nodes",
			nodes: []Node{NewAccessorList([]Node{NewNumeric("42.2")}), NewKey("bar")},
			str:   `(42.2)."bar"`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			node := NewAccessorList(tc.nodes)
			a.Implements((*Node)(nil), node)
			a.Equal(lowestPriority, node.priority())
			a.Equal(tc.str, node.String())

			if n, ok := tc.nodes[0].(*AccessorListNode); ok {
				// Should have appended nodes.
				a.Equal(n, node)
				a.Equal(tc.nodes[1:], n.accessors[1:])
			} else {
				a.Equal(tc.nodes, node.Accessors())
			}

			// Test writeTo.
			buf := new(strings.Builder)
			node.writeTo(buf, false, false)
			a.Equal(tc.str, buf.String())
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
			a.Implements((*Node)(nil), node)
			a.Equal(tc.nodes, node.subscripts)
			a.Equal(tc.nodes, node.Subscripts())
			a.Equal(lowestPriority, node.priority())
			a.Equal(tc.str, node.String())

			// Test writeTo.
			buf := new(strings.Builder)
			node.writeTo(buf, false, false)
			a.Equal(tc.str, buf.String())
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
			a.Implements((*Node)(nil), node)
			a.Equal(lowestPriority, node.priority())
			a.Equal(tc.str, node.String())

			// Test writeTo.
			buf := new(strings.Builder)
			node.writeTo(buf, false, false)
			a.Equal(tc.str, buf.String())

			// Test writeTo with inKey true
			buf.Reset()
			node.writeTo(buf, true, false)
			a.Equal("."+tc.str, buf.String())
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
			str:     `"foo" like_regex "."`,
			match:   []string{"x", "abc", "123"},
			noMatch: []string{"", "\n"},
		},
		{
			name:    "anchor",
			node:    NewString("foo"),
			re:      `^a`,
			str:     `"foo" like_regex "^a"`,
			match:   []string{"a", "abc", "a\nb\nc"},
			noMatch: []string{"", "\na\n"},
		},
		{
			name:    "flags",
			node:    NewString("fOo"),
			re:      `^o.`,
			flag:    "ims",
			flags:   regexFlags(regexDotAll | regexICase | regexMLine),
			str:     `"fOo" like_regex "^o." flag "ism"`,
			match:   []string{"ox", "Ox", "oO", "a\no\nc"},
			noMatch: []string{"xoxo", "a\nxo"},
		},
		{
			name:    "quote",
			node:    NewString("foo"),
			re:      `xa+`,
			flag:    "iqsm",
			flags:   regexFlags(regexICase | regexQuote | regexDotAll | regexMLine),
			str:     `"foo" like_regex "xa+" flag "ismq"`,
			match:   []string{"xa+", "XA+", "\nXa+", "bmXa+"},
			noMatch: []string{`xa\+`, "x"},
		},
		{
			name: "bad_flags",
			node: NewString("foo"),
			re:   `.`,
			flag: "x",
			err:  `XQuery "x" flag (expanded regular expressions) is not implemented`,
		},
		{
			name: "bad_pattern",
			node: NewString("foo"),
			re:   `.(hi`,
			err:  "error parsing regexp: missing closing ): `.(hi`",
		},
		{
			name:    "priority_parens",
			node:    NewBinary(BinaryOr, ConstCurrent, ConstCurrent),
			re:      `xa+`,
			flag:    "iqsm",
			flags:   regexFlags(regexICase | regexQuote | regexDotAll | regexMLine),
			str:     `(@ || @) like_regex "xa+" flag "ismq"`,
			match:   []string{"xa+", "XA+", "\nXa+", "bmXa+"},
			noMatch: []string{`xa\+`, "x"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			node, err := NewRegex(tc.node, tc.re, tc.flag)
			if tc.err != "" {
				r.EqualError(err, tc.err)
				a.Nil(node)
				return
			}

			r.NoError(err)
			r.NotNil(node)
			a.Implements((*Node)(nil), node)
			a.Equal(lowestPriority, node.priority())
			a.Equal(tc.re, node.pattern)
			a.Equal(tc.flags, node.flags)
			a.Equal(tc.node, node.operand)
			a.Equal(tc.node, node.Operand())
			a.Equal(tc.str, node.String())

			// Test writeTo.
			buf := new(strings.Builder)
			node.writeTo(buf, false, false)
			a.Equal(tc.str, buf.String())

			// Test writeTo with withParens true
			buf.Reset()
			node.writeTo(buf, false, true)
			a.Equal("("+tc.str+")", buf.String())

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

func TestNewUnaryOrNumber(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tc := range []struct {
		name string
		op   UnaryOperator
		node Node
		exp  Node
		err  string
	}{
		{
			name: "plus_integer",
			op:   UnaryPlus,
			node: NewInteger("42"),
			exp:  NewInteger("42"),
		},
		{
			name: "minus_integer",
			op:   UnaryMinus,
			node: NewInteger("42"),
			exp:  NewInteger("-42"),
		},
		{
			name: "other_integer",
			op:   UnaryExists,
			node: NewInteger("42"),
			err:  "Operator must be + or - but is exists",
		},
		{
			name: "plus_accessor_integer",
			op:   UnaryPlus,
			node: NewAccessorList([]Node{NewInteger("42")}),
			exp:  NewInteger("42"),
		},
		{
			name: "minus_accessor_integer",
			op:   UnaryMinus,
			node: NewAccessorList([]Node{NewInteger("42")}),
			exp:  NewInteger("-42"),
		},
		{
			name: "minus_accessor_multi",
			op:   UnaryMinus,
			node: NewAccessorList([]Node{NewInteger("42"), NewInteger("42")}),
			exp:  NewUnary(UnaryMinus, NewAccessorList([]Node{NewInteger("42"), NewInteger("42")})),
		},
		{
			name: "plus_numeric",
			op:   UnaryPlus,
			node: NewNumeric("42.0"),
			exp:  NewNumeric("42.0"),
		},
		{
			name: "minus_numeric",
			op:   UnaryMinus,
			node: NewNumeric("42.0"),
			exp:  NewNumeric("-42.0"),
		},
		{
			name: "other_numeric",
			op:   UnaryNot,
			node: NewNumeric("42"),
			err:  "Operator must be + or - but is !",
		},
		{
			name: "plus_accessor_numeric",
			op:   UnaryPlus,
			node: NewAccessorList([]Node{NewNumeric("42.1")}),
			exp:  NewNumeric("42.1"),
		},
		{
			name: "minus_accessor_numeric",
			op:   UnaryMinus,
			node: NewAccessorList([]Node{NewNumeric("42")}),
			exp:  NewNumeric("-42"),
		},
		{
			name: "minus_accessor_multi_numeric",
			op:   UnaryMinus,
			node: NewAccessorList([]Node{NewNumeric("42"), ConstCurrent}),
			exp:  NewUnary(UnaryMinus, NewAccessorList([]Node{NewNumeric("42"), ConstCurrent})),
		},
		{
			name: "plus_other",
			op:   UnaryPlus,
			node: ConstCurrent,
			exp:  NewUnary(UnaryPlus, ConstCurrent),
		},
		{
			name: "minus_other",
			op:   UnaryMinus,
			node: ConstCurrent,
			exp:  NewUnary(UnaryMinus, ConstCurrent),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.err != "" {
				a.PanicsWithValue(tc.err, func() { NewUnaryOrNumber(tc.op, tc.node) })
				return
			}
			a.Equal(tc.exp, NewUnaryOrNumber(tc.op, tc.node))
		})
	}
}

func TestAST(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	r := require.New(t)

	for _, tc := range []struct {
		name string
		node Node
		pred bool
		err  string
	}{
		{"string", NewString("foo"), true, ""},
		{"accessor", NewAccessorList([]Node{ConstRoot}), false, ""},
		{"current", ConstCurrent, false, "@ is not allowed in root expressions"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.err != "" {
				tree, err := New(true, tc.node)
				r.EqualError(err, tc.err)
				a.Nil(tree)
				return
			}

			tree, err := New(true, tc.node)
			r.NoError(err)
			a.True(tree.lax)
			a.Equal(tc.node, tree.root)
			a.Equal(tree.root.String(), tree.String())
			a.Equal(tc.node, tree.Root())
			a.Equal(tc.pred, tree.IsPredicate())

			tree, err = New(false, tc.node)
			r.NoError(err)
			a.False(tree.lax)
			a.Equal(tc.node, tree.root)
			a.Equal("strict "+tree.root.String(), tree.String())
			a.Equal(tc.node, tree.Root())
			a.Equal(tc.pred, tree.IsPredicate())
		})
	}
}

func TestValidateNode(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	goodRegex, _ := NewRegex(ConstRoot, ".", "")
	badRegex, _ := NewRegex(ConstCurrent, ".", "")

	for _, tc := range []struct {
		name  string
		node  Node
		depth int
		inSub bool
		err   string
	}{
		{
			name: "string",
			node: NewString("foo"),
		},
		{
			name: "variable",
			node: NewVariable("foo"),
		},
		{
			name: "key",
			node: NewKey("foo"),
		},
		{
			name: "numeric",
			node: NewNumeric("42"),
		},
		{
			name: "integer",
			node: NewInteger("42"),
		},
		{
			name: "binary",
			node: NewBinary(BinaryAdd, NewInteger("42"), NewInteger("99")),
		},
		{
			name: "binary_left_fail",
			node: NewBinary(BinaryAdd, ConstCurrent, ConstRoot),
			err:  "@ is not allowed in root expressions",
		},
		{
			name: "binary_right_fail",
			node: NewBinary(BinaryAdd, ConstRoot, ConstCurrent),
			err:  "@ is not allowed in root expressions",
		},
		{
			name:  "binary_current_okay_depth",
			node:  NewBinary(BinaryAdd, ConstRoot, ConstCurrent),
			depth: 1,
		},
		{
			name: "unary",
			node: NewUnary(UnaryNot, ConstRoot),
		},
		{
			name: "unary_fail",
			node: NewUnary(UnaryNot, ConstLast),
			err:  "LAST is allowed only in array subscripts",
		},
		{
			name:  "unary_current_okay_depth",
			node:  NewUnary(UnaryNot, ConstCurrent),
			depth: 1,
		},
		{
			name: "regex",
			node: goodRegex,
		},
		{
			name: "bad_regex",
			node: badRegex,
			err:  "@ is not allowed in root expressions",
		},
		{
			name:  "regex_current_okay_depth",
			node:  badRegex,
			depth: 1,
		},
		{
			name: "current",
			node: ConstCurrent,
			err:  "@ is not allowed in root expressions",
		},
		{
			name:  "current_depth",
			node:  ConstCurrent,
			depth: 1,
		},
		{
			name: "last",
			node: ConstLast,
			err:  "LAST is allowed only in array subscripts",
		},
		{
			name:  "last_in_sub",
			node:  ConstLast,
			inSub: true,
		},
		{
			name: "array",
			node: NewArrayIndex([]Node{NewBinary(BinarySubscript, ConstRoot, ConstRoot)}),
		},
		{
			name: "array_last",
			node: NewArrayIndex([]Node{NewBinary(BinarySubscript, ConstRoot, ConstLast)}),
		},
		{
			name: "array_current",
			node: NewArrayIndex([]Node{NewBinary(BinarySubscript, ConstRoot, ConstCurrent)}),
			err:  "@ is not allowed in root expressions",
		},
		{
			name: "accessor",
			node: NewAccessorList([]Node{ConstRoot}),
		},
		{
			name: "accessor_current",
			node: NewAccessorList([]Node{ConstCurrent}),
			err:  "@ is not allowed in root expressions",
		},
		{
			name: "accessor_filter_current",
			node: NewAccessorList([]Node{ConstRoot, NewUnary(UnaryFilter, ConstCurrent)}),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateNode(tc.node, tc.depth, tc.inSub)
			if tc.err == "" {
				r.NoError(err)
			} else {
				r.EqualError(err, tc.err)
			}
		})
	}
}

func TestNodes(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tc := range []struct {
		name string
		node any
	}{
		{"ConstNode", ConstRoot},
		{"MethodNode", MethodAbs},
		{"StringNode", &StringNode{}},
		{"VariableNode", &VariableNode{}},
		{"KeyNode", &KeyNode{}},
		{"NumericNode", &NumericNode{}},
		{"IntegerNode", &IntegerNode{}},
		{"AnyNode", &AnyNode{}},
		{"BinaryNode", &BinaryNode{}},
		{"UnaryNode", &UnaryNode{}},
		{"RegexNode", &RegexNode{}},
		{"AccessorNode", &AccessorListNode{}},
		{"ArrayIndexNode", &ArrayIndexNode{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a.Implements((*Node)(nil), tc.node)
		})
	}
}
