// Package ast provides an abstract syntax tree for SQL/JSON paths.
//
// Largely ported from PostgreSQL's [jsonpath.c], it provides objects for every
// node parsed from an SQL/JSON path. The [parser] constructs these nodes as it
// parses a path, and constructs an AST object from the root node.
//
// [jsonpath.c]: https://github.com/postgres/postgres/blob/adcdb2c/src/backend/utils/adt/jsonpath.c
package ast

// Use golang.org/x/tools/cmd/stringer to generate the String method for enums
// for their inline comments.

//go:generate stringer -linecomment -output ast_string.go -type ConstNode,BinaryOperator,UnaryOperator,MethodNode

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// Node represents a single node in the AST.
type Node interface {
	// String returns the properly-encoded and delimited SQL/JSON Path string
	// representation of the node.
	String() string
}

// ConstNode is a constant value parsed from the path.
type ConstNode int

//revive:disable:exported
const (
	ConstRoot     ConstNode = iota // $
	ConstCurrent                   // @
	ConstLast                      // last
	ConstAnyArray                  // [*]
	ConstAnyKey                    // *
	ConstTrue                      // true
	ConstFalse                     // false
	ConstNull                      // null
)

// BinaryOperator represents a binary operator.
type BinaryOperator int

//revive:disable:exported
const (
	BinaryAnd            BinaryOperator = iota // &&
	BinaryOr                                   // ||
	BinaryEqual                                // ==
	BinaryNotEqual                             // !=
	BinaryLess                                 // <
	BinaryGreater                              // >
	BinaryLessOrEqual                          // <=
	BinaryGreaterOrEqual                       // >=
	BinaryStartsWith                           // starts with
	BinaryAdd                                  // +
	BinarySub                                  // -
	BinaryMul                                  // *
	BinaryDiv                                  // /
	BinaryMod                                  // %
	BinarySubscript                            // to
	BinaryDecimal                              // .decimal()
)

// UnaryOperator represents a unary operator.
type UnaryOperator int

//revive:disable:exported
const (
	UnaryExists      UnaryOperator = iota // exists
	UnaryNot                              // !
	UnaryIsUnknown                        // is unknown
	UnaryPlus                             // +
	UnaryMinus                            // -
	UnaryFilter                           // ?
	UnaryDateTime                         // .datetime
	UnaryTime                             // .time
	UnaryTimeTZ                           // .time_tz
	UnaryTimestamp                        // .timestamp
	UnaryTimestampTZ                      // .timestamp_tz
)

// MethodNode represents a path method.
type MethodNode int

//revive:disable:exported
const (
	MethodAbs      MethodNode = iota // .abs()
	MethodSize                       // .size()
	MethodType                       // .type()
	MethodFloor                      // .floor()
	MethodCeiling                    // .ceiling()
	MethodDouble                     // .double()
	MethodKeyValue                   // .keyvalue()
	MethodBigint                     // .bigint()
	MethodBoolean                    // .boolean()
	MethodDate                       // .date()
	MethodInteger                    // .integer()
	MethodNumber                     // .number()
	MethodString                     // .string()
)

// quotedString represents a quoted string node, including strings, variables,
// and path keys.
type quotedString string

// Text returns the textual representation of the string.
func (n quotedString) Text() string {
	return string(n)
}

// String returns the SQL/JSON path-encoded quoted string.
func (n quotedString) String() string {
	return fmt.Sprintf("%q", string(n))
}

// StringNode represents a string parsed from the path.
type StringNode struct {
	quotedString
}

// NewString returns a new StringNode representing str.
func NewString(str string) *StringNode {
	return &StringNode{quotedString(str)}
}

// VariableNode represents a SQL/JSON path variable name.
type VariableNode struct {
	// jpiVariable
	quotedString
}

// NewVariable returns a new VariableNode named name.
func NewVariable(name string) *VariableNode {
	return &VariableNode{quotedString(name)}
}

// String returns the double-quoted representation of n, preceded by '$'.
func (n *VariableNode) String() string {
	return "$" + n.quotedString.String()
}

// KeyNode represents a SQL/JSON path key expression, e.g., '.foo'.
type KeyNode struct {
	// jpiKey
	quotedString
}

// NewKey returns a new KeyNode with key.
func NewKey(key string) *KeyNode {
	return &KeyNode{quotedString(key)}
}

// A raw string represents a raw string value.
type rawString string

// String returns the  string.
func (n rawString) String() string {
	return string(n)
}

// NumericNode represents a numeric (non-integer) value.
type NumericNode struct {
	rawString
}

// NewNumeric returns a new NumericNode representing num. Panics if num cannot
// be parsed into float64.
func NewNumeric(num string) *NumericNode {
	_, err := strconv.ParseFloat(num, 64)
	if err != nil {
		panic(err)
	}
	return &NumericNode{rawString(num)}
}

// Float returns the floating point number corresponding to n.
func (n *NumericNode) Float() float64 {
	num, _ := strconv.ParseFloat(string(n.rawString), 64)
	return num
}

// IntegerNode represents an integral value.
type IntegerNode struct {
	rawString
}

// NewInteger returns a new IntegerNode representing num. Panics if
// integer cannot be parsed into int64.
func NewInteger(integer string) *IntegerNode {
	_, err := strconv.ParseInt(integer, 10, 64)
	if err != nil {
		panic(err)
	}
	return &IntegerNode{rawString(integer)}
}

// Int returns the integer corresponding to n.
func (n *IntegerNode) Int() int64 {
	num, _ := strconv.ParseInt(string(n.rawString), 10, 64)
	return num
}

// BinaryNode represents a binary operation.
type BinaryNode struct {
	op    BinaryOperator
	left  Node
	right Node
}

// NewBinary returns a new BinaryNode where op represents the binary operator
// and left and right the operands.
func NewBinary(op BinaryOperator, left, right Node) *BinaryNode {
	return &BinaryNode{op: op, left: left, right: right}
}

// String returns the SQL/JSON path string representation of the binary
// expression.
func (n *BinaryNode) String() string {
	switch n.op {
	case BinaryDecimal:
		str := new(strings.Builder)
		str.WriteString(".decimal(")
		if n.left != nil {
			str.WriteString(n.left.String())
		}
		if n.right != nil {
			str.WriteRune(',')
			str.WriteString(n.right.String())
		}
		str.WriteRune(')')
		return str.String()
	case BinarySubscript:
		if n.right == nil {
			return n.left.String()
		}
		fallthrough
	case BinaryAnd, BinaryOr, BinaryEqual, BinaryNotEqual, BinaryLess,
		BinaryGreater, BinaryLessOrEqual, BinaryGreaterOrEqual,
		BinaryStartsWith, BinaryAdd, BinarySub, BinaryMul, BinaryDiv,
		BinaryMod:
		return n.left.String() + " " + n.op.String() + " " + n.right.String()
	default:
		panic(fmt.Sprintf("Unknown binary operator %v", n.op))
	}
}

// Operator returns the BinaryNode's BinaryOperator.
func (n *BinaryNode) Operator() BinaryOperator {
	return n.op
}

// Left returns the BinaryNode's left operand.
func (n *BinaryNode) Left() Node {
	return n.left
}

// Right returns the BinaryNode's right operand.
func (n *BinaryNode) Right() Node {
	return n.right
}

// UnaryNode represents a unary operation.
type UnaryNode struct {
	op   UnaryOperator
	node Node
}

// NewUnary returns a new UnaryNode where op represents the unary operator
// and node its operand.
func NewUnary(op UnaryOperator, node Node) *UnaryNode {
	return &UnaryNode{op: op, node: node}
}

// String returns the SQL/JSON path string representation of the unary
// expression.
func (n *UnaryNode) String() string {
	switch n.op {
	case UnaryExists:
		return "exists (" + n.node.String() + ")"
	case UnaryNot, UnaryFilter:
		return n.op.String() + "(" + n.node.String() + ")"
	case UnaryIsUnknown:
		return "(" + n.node.String() + ") is unknown"
	case UnaryPlus, UnaryMinus:
		return n.op.String() + n.node.String()
	case UnaryDateTime, UnaryTime, UnaryTimeTZ, UnaryTimestamp, UnaryTimestampTZ:
		if n.node == nil {
			return n.op.String() + "()"
		}
		return n.op.String() + "(" + n.node.String() + ")"
	default:
		return ""
	}
}

// Operator returns the UnaryNode's BinaryOperator.
func (n *UnaryNode) Operator() UnaryOperator {
	return n.op
}

// Node returns the UnaryNode's operand.
func (n *UnaryNode) Node() Node {
	return n.node
}

// listNode is the base struct for managing lists of nodes.
type listNode struct {
	nodes []Node
}

// AccessorNode represents the nodes in an accessor path expression.
type AccessorNode struct {
	*listNode
}

// NewAccessor creates a new AccessorNode consisting of nodes.
func NewAccessor(nodes []Node) *AccessorNode {
	return &AccessorNode{&listNode{nodes: nodes}}
}

// String produces JSON Path accessor path string representation of the nodes in
// n.
func (n *AccessorNode) String() string {
	val := strings.Builder{}
	maxIdx := len(n.nodes) - 1
	for i, node := range n.nodes {
		val.WriteRune('.')
		switch node.(type) {
		case *NumericNode:
			if i < maxIdx {
				val.WriteRune('(')
			}
			val.WriteString(node.String())
			if i < maxIdx {
				val.WriteRune(')')
			}
		default:
			val.WriteString(node.String())
		}
	}

	return val.String()
}

// ArrayIndexNode represents the nodes in an array index expression.
type ArrayIndexNode struct {
	*listNode
}

// NewArrayIndex creates a new ArrayIndexNode consisting of nodes.
func NewArrayIndex(nodes []Node) *ArrayIndexNode {
	return &ArrayIndexNode{&listNode{nodes: nodes}}
}

// String produces JSON Path array index string representation of the nodes in
// n.
func (n *ArrayIndexNode) String() string {
	val := strings.Builder{}

	val.WriteRune('[')
	for i, node := range n.nodes {
		if i > 0 {
			val.WriteRune(',')
		}
		val.WriteString(node.String())
	}
	val.WriteRune(']')

	return val.String()
}

// AnyNode represents any node in a path accessor with the expression
// 'first TO last'.
type AnyNode struct {
	// jpiAny
	first uint32
	last  uint32
}

// NewAny returns a new AnyNode with first as its first index and last as its
// last. If either number is negative it's considered unbounded.
func NewAny(first, last int) *AnyNode {
	n := &AnyNode{first: math.MaxUint32, last: math.MaxUint32}
	if first >= 0 {
		n.first = uint32(first)
	}
	if last >= 0 {
		n.last = uint32(last)
	}
	return n
}

// String returns the SQL/JSON path any node expression.
func (n *AnyNode) String() string {
	switch {
	case n.first == 0 && n.last == math.MaxUint32:
		return "**"
	case n.first == n.last:
		if n.first == math.MaxUint32 {
			return "**{last}"
		}
		return fmt.Sprintf("**{%v}", n.first)
	case n.first == math.MaxUint32:
		return fmt.Sprintf("**{last to %v}", n.last)
	case n.last == math.MaxUint32:
		return fmt.Sprintf("**{%v to last}", n.first)
	default:
		return fmt.Sprintf("**{%v to %v}", n.first, n.last)
	}
}

// RegexNode represents a regular expression.
type RegexNode struct {
	// jpiLikeRegex
	node    Node
	pattern string
	flags   regexFlags
}

// NewRegex returns anew RegexNode that compares node to the regular expression
// pattern configured by flags.
func NewRegex(expr Node, pattern, flags string) (*RegexNode, error) {
	f, err := newRegexFlags(flags)
	if err != nil {
		return nil, err
	}
	if err := validateRegex(pattern, f); err != nil {
		return nil, err
	}
	return &RegexNode{node: expr, pattern: pattern, flags: f}, nil
}

// String returns the RegexNode as a SQL/JSON path 'like_regex' expression.
func (n *RegexNode) String() string {
	return fmt.Sprintf("like_regex %v %q%v", n.node.String(), n.pattern, n.flags)
}

// Regexp returns a regexp.Regexp compiled from n.
func (n *RegexNode) Regexp() *regexp.Regexp {
	flags := n.flags.goFlags()
	if n.flags.shouldQuoteMeta() {
		return regexp.MustCompile(flags + regexp.QuoteMeta(n.pattern))
	}
	return regexp.MustCompile(n.flags.goFlags() + n.pattern)
}

// AST represents the complete abstract syntax tree for a parsed SQL/JSON path.
type AST struct {
	root   Node
	strict bool
}

// New creates a new AST with n as its root. If strict is true it's considered a
// strict path query.
func New(strict bool, n Node) *AST {
	return &AST{root: n, strict: strict}
}
