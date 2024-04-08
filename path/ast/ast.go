// Package ast provides an abstract syntax tree for SQL/JSON paths.
//
// Largely ported from PostgreSQL's [jsonpath.c], it provides objects for every
// node parsed from an SQL/JSON path. The [parser] constructs these nodes as it
// parses a path, and constructs an AST object from the root node.
//
// Note that errors returned by AST are not wrapped, as they're expected to be
// wrapped by parser.
//
// The complete list of types that implement Node:
//
//   - [ConstNode]
//   - [MethodNode]
//   - [StringNode]
//   - [VariableNode]
//   - [KeyNode]
//   - [NumericNode]
//   - [AnyNode]
//   - [BinaryNode]
//   - [UnaryNode]
//   - [RegexNode]
//   - [AccessorListNode]
//   - [ArrayIndexNode]
//
// Here's a starter recursive function for processing nodes.
//
//	func processNode(node ast.Node) {
//		switch node := node.(type) {
//		case ast.ConstNode:
//		case ast.MethodNode:
//		case *ast.StringNode:
//		case *ast.VariableNode:
//		case *ast.KeyNode:
//		case *ast.NumericNode:
//		case *ast.AnyNode:
//		case *ast.BinaryNode:
//			processNode(node.Left())
//			processNode(node.Right())
//		case *ast.UnaryNode:
//			processNode(node.Operand())
//		case *ast.RegexNode:
//			processNode(node.Operand())
//		case *ast.AccessorListNode:
//			for _, n := range node.Accessors() {
//				processNode(n)
//			}
//		case *ast.ArrayIndexNode:
//			for _, n := range node.Subscripts() {
//				processNode(n)
//			}
//		}
//	}
//
// [jsonpath.c]: https://github.com/postgres/postgres/blob/adcdb2c/src/backend/utils/adt/jsonpath.c
package ast

// Use golang.org/x/tools/cmd/stringer to generate the String method for enums
// for their inline comments.

//go:generate stringer -linecomment -output ast_string.go -type ConstNode,BinaryOperator,UnaryOperator,MethodNode

import (
	"encoding/json"
	"errors"
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

	// writeTo writes the string representation of a node to buf. inKey is true
	// when the node is a key in an accessor list and withParens requires
	// parentheses to be printed around the node.
	writeTo(buf *strings.Builder, inKey, withParens bool)

	// priority returns the operational priority of the node relative to other
	// nodes. Priority ranges from 0 for highest to 6 for lowest.
	priority() uint8
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

// writeTo writes the string representation of n to buf. If n is ConstAnyKey and
// inKey is true, it will be preceded by '.'.
func (n ConstNode) writeTo(buf *strings.Builder, inKey, _ bool) {
	if n == ConstAnyKey && inKey {
		buf.WriteRune('.')
	}
	buf.WriteString(n.String())
}

// lowestPriority is the lowest priority returned by priority, and the default
// for most nodes.
const lowestPriority = uint8(6)

// priority returns the priority of the ConstNode, which is always 6.
func (ConstNode) priority() uint8 { return lowestPriority }

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

// Priority returns the priority of the operator.
//
//nolint:gomnd,exhaustive
func (op BinaryOperator) priority() uint8 {
	switch op {
	case BinaryOr:
		return 0
	case BinaryAnd:
		return 1
	case BinaryEqual, BinaryNotEqual, BinaryLess, BinaryGreater,
		BinaryLessOrEqual, BinaryGreaterOrEqual, BinaryStartsWith:
		return 2
	case BinaryAdd, BinarySub:
		return 3
	case BinaryMul, BinaryDiv, BinaryMod:
		return 4
	default:
		return lowestPriority
	}
}

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

// Priority returns the priority of the operator.
//
//nolint:gomnd,exhaustive
func (op UnaryOperator) priority() uint8 {
	switch op {
	case UnaryPlus, UnaryMinus:
		return 5
	default:
		return lowestPriority
	}
}

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

// writeTo writes the string representation of n to buf.
func (n MethodNode) writeTo(buf *strings.Builder, _, _ bool) {
	buf.WriteString(n.String())
}

// priority returns the priority of the MethodNode, which is always 6.
func (MethodNode) priority() uint8 { return lowestPriority }

// quotedString represents a quoted string node, including strings, variables,
// and path keys.
type quotedString string

// Text returns the textual representation of the string.
func (n quotedString) Text() string {
	return string(n)
}

// String returns the SQL/JSON path-encoded quoted string.
func (n quotedString) String() string {
	return strconv.Quote(string(n))
}

// writeTo writes String to buf.
func (n quotedString) writeTo(buf *strings.Builder, _, _ bool) {
	buf.WriteString(n.String())
}

// priority returns the priority of the quotedString, which is always 6.
func (quotedString) priority() uint8 { return lowestPriority }

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

// writeTo writes String to buf.
func (n VariableNode) writeTo(buf *strings.Builder, _, _ bool) {
	buf.WriteString(n.String())
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

// writeTo writes the key to buf, prepended with '.' if inKey is true.
func (n *KeyNode) writeTo(buf *strings.Builder, inKey, _ bool) {
	if inKey {
		buf.WriteRune('.')
	}
	buf.WriteString(n.String())
}

// NumericNode represents a numeric value, and distinguishes between an
// integer and a float.
type NumericNode struct {
	literal string
	parsed  string
	isInt   bool
}

// NewNumeric returns a new NumberNode representing num. Panics if num cannot
// be parsed into a 64-bit integer when isInt is true or a JSON-compatible
// float64 when isInt is false.
func NewNumeric(num string, isInt bool) *NumericNode {
	self := &NumericNode{literal: num, isInt: isInt}
	if isInt {
		val, err := strconv.ParseInt(num, 0, 64)
		if err != nil {
			panic(err)
		}
		self.parsed = strconv.FormatInt(val, 10)
		return self
	}

	f, err := strconv.ParseFloat(num, 64)
	if err != nil {
		panic(err)
	}

	// https://www.postgresql.org/docs/current/datatype-json.html#DATATYPE-JSONPATH:
	//
	// > Numeric literals in SQL/JSON path expressions follow JavaScript rules,
	// > which are different from both SQL and JSON in some minor details. For
	// > example, SQL/JSON path allows .1 and 1., which are invalid in JSON.
	// > Non-decimal integer literals and underscore separators are supported,
	// > for example, 1_000_000, 0x1EEE_FFFF, 0o273, 0b100101. In SQL/JSON path
	// > (and in JavaScript, but not in SQL proper), there must not be an
	// > underscore separator directly after the radix prefix.
	//
	// Rely on JSON semantics, a subset of the JavaScript.
	str, err := json.Marshal(f)
	if err != nil {
		panic(err)
	}

	self.parsed = string(str)
	return self
}

// Literal returns the literal text of the number as passed to the
// constructor.
func (n *NumericNode) Literal() string {
	return n.literal
}

// String returns the normalized string representation of the number.
func (n *NumericNode) String() string {
	return n.parsed
}

// IsInt returns the normalized string representation of the number.
func (n *NumericNode) IsInt() bool {
	return n.isInt
}

// Float64 returns the floating point number corresponding to n. For maximum
// accuracy, should only be called when n.IsInt() is false.
func (n *NumericNode) Float64() float64 {
	if n.isInt {
		val, _ := strconv.ParseInt(n.parsed, 0, 64)
		return float64(val)
	}
	num, _ := strconv.ParseFloat(n.parsed, 64)
	return num
}

// Int64 returns the integer corresponding to n. For maximum accuracy, should
// only be called when n.IsInt() is true.
func (n *NumericNode) Int64() int64 {
	if !n.isInt {
		num, _ := strconv.ParseFloat(n.parsed, 64)
		return int64(num)
	}
	val, _ := strconv.ParseInt(n.parsed, 0, 64)
	return val
}

// JSON returns a json.Number representation of n.
func (n *NumericNode) JSON() json.Number {
	return json.Number(n.parsed)
}

// writeTo writes String to buf, surrounded by parentheses if withParens is
// true.
func (n *NumericNode) writeTo(buf *strings.Builder, _, withParens bool) {
	if withParens {
		buf.WriteRune('(')
	}
	buf.WriteString(n.String())
	if withParens {
		buf.WriteRune(')')
	}
}

// priority returns the priority of the NumberNode, which is always 6.
func (*NumericNode) priority() uint8 { return lowestPriority }

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
	buf := new(strings.Builder)
	n.writeTo(buf, false, false)
	return buf.String()
}

// writeTo writes the SQL/JSON path string representation of the binary
// expression to buf. If withParens is true and the binary operation is neither
// BinaryDecimal nor BinarySubscript, parentheses will be written around the
// expression.
func (n *BinaryNode) writeTo(buf *strings.Builder, _, withParens bool) {
	switch n.op {
	case BinaryDecimal:
		buf.WriteString(".decimal(")
		if n.left != nil {
			buf.WriteString(n.left.String())
		}
		if n.right != nil {
			buf.WriteRune(',')
			buf.WriteString(n.right.String())
		}
		buf.WriteRune(')')
	case BinarySubscript:
		buf.WriteString(n.left.String())
		if n.right != nil {
			buf.WriteString(" " + n.op.String() + " " + n.right.String())
		}
	case BinaryAnd, BinaryOr, BinaryEqual, BinaryNotEqual, BinaryLess,
		BinaryGreater, BinaryLessOrEqual, BinaryGreaterOrEqual,
		BinaryAdd, BinarySub, BinaryMul, BinaryDiv, BinaryMod,
		BinaryStartsWith:
		if withParens {
			buf.WriteRune('(')
		}

		n.left.writeTo(buf, false, n.left.priority() <= n.priority())
		buf.WriteString(" " + n.op.String() + " ")
		n.right.writeTo(buf, false, n.right.priority() <= n.priority())

		if withParens {
			buf.WriteRune(')')
		}
	default:
		panic(fmt.Sprintf("Unknown binary operator %v", n.op))
	}
}

// priority returns the priority of n.op.
func (n BinaryNode) priority() uint8 { return n.op.priority() }

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
	op      UnaryOperator
	operand Node
}

// NewUnary returns a new UnaryNode where op represents the unary operator
// and node its operand.
func NewUnary(op UnaryOperator, node Node) *UnaryNode {
	return &UnaryNode{op: op, operand: node}
}

// String returns the SQL/JSON path string representation of the unary
// expression.
func (n *UnaryNode) String() string {
	buf := new(strings.Builder)
	n.writeTo(buf, false, false)
	return buf.String()
}

// priority returns the priority of n.op.
func (n UnaryNode) priority() uint8 { return n.op.priority() }

// writeTo writes the SQL/JSON path string representation of the unary
// expression to buf. If withParens is true and the binary operation is
// UnaryPlus or UnaryMinus, parentheses will be written around the expression.
func (n *UnaryNode) writeTo(buf *strings.Builder, _, withParens bool) {
	switch n.op {
	case UnaryExists:
		buf.WriteString("exists (" + n.operand.String() + ")")
	case UnaryNot, UnaryFilter:
		buf.WriteString(n.op.String() + "(" + n.operand.String() + ")")
	case UnaryIsUnknown:
		buf.WriteString("(" + n.operand.String() + ") is unknown")
	case UnaryPlus, UnaryMinus:
		if withParens {
			buf.WriteRune('(')
		}

		buf.WriteString(n.op.String())
		n.operand.writeTo(buf, false, n.operand.priority() <= n.priority())

		if withParens {
			buf.WriteRune(')')
		}
	case UnaryDateTime, UnaryTime, UnaryTimeTZ, UnaryTimestamp, UnaryTimestampTZ:
		if n.operand == nil {
			buf.WriteString(n.op.String() + "()")
		} else {
			buf.WriteString(n.op.String() + "(" + n.operand.String() + ")")
		}
	default:
		// Write nothing.
	}
}

// Operator returns the UnaryNode's BinaryOperator.
func (n *UnaryNode) Operator() UnaryOperator {
	return n.op
}

// Operand returns the UnaryNode's operand.
func (n *UnaryNode) Operand() Node {
	return n.operand
}

// AccessorListNode represents the nodes in an accessor path expression.
type AccessorListNode struct {
	accessors []Node
}

// NewAccessorList creates a new AccessorNode consisting of nodes. If the first node
// in nodes is an AccessorNode, it will be returned with the remaining nodes
// appended to it.
func NewAccessorList(nodes []Node) *AccessorListNode {
	if acc, ok := nodes[0].(*AccessorListNode); ok {
		// Append items to existing list.
		acc.accessors = append(acc.accessors, nodes[1:]...)
		return acc
	}

	return &AccessorListNode{accessors: nodes}
}

// Accessors returns all of the accessor nodes in n.
func (n *AccessorListNode) Accessors() []Node { return n.accessors }

// String produces JSON Path accessor path string representation of the nodes in
// n.
func (n *AccessorListNode) String() string {
	buf := new(strings.Builder)
	n.writeTo(buf, false, false)
	return buf.String()
}

// writeTo writes the SQL/JSON path string representation of n to buf.
func (n *AccessorListNode) writeTo(buf *strings.Builder, _, _ bool) {
	maxIdx := len(n.accessors) - 1
	for i, node := range n.accessors {
		node.writeTo(buf, i > 0, i < maxIdx)
	}
}

// priority returns the priority of the AccessorNode, which is always 6.
func (*AccessorListNode) priority() uint8 { return lowestPriority }

// ArrayIndexNode represents the nodes in an array index expression.
type ArrayIndexNode struct {
	subscripts []Node
}

// NewArrayIndex creates a new ArrayIndexNode consisting of subscripts.
// which must be BinaryNodes using the BinarySubscript operator.
func NewArrayIndex(subscripts []Node) *ArrayIndexNode {
	return &ArrayIndexNode{subscripts: subscripts}
}

// Subscripts returns all of the subscript nodes in n.
func (n *ArrayIndexNode) Subscripts() []Node { return n.subscripts }

// String produces JSON Path array index string representation of the nodes in
// n.
func (n *ArrayIndexNode) String() string {
	buf := new(strings.Builder)
	n.writeTo(buf, false, false)
	return buf.String()
}

// writeTo writes the SQL/JSON path representation of n to buf.
func (n *ArrayIndexNode) writeTo(buf *strings.Builder, _, _ bool) {
	buf.WriteRune('[')
	for i, node := range n.subscripts {
		if i > 0 {
			buf.WriteRune(',')
		}
		buf.WriteString(node.String())
	}
	buf.WriteRune(']')
}

// priority returns the priority of the ArrayIndexNode, which is always 6.
func (*ArrayIndexNode) priority() uint8 { return lowestPriority }

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
	buf := new(strings.Builder)
	n.writeTo(buf, false, false)
	return buf.String()
}

// writeTo writes the SQL/JSON path representation of n to buf.
// If inKey is true it will be preceded by a '.'.
func (n *AnyNode) writeTo(buf *strings.Builder, inKey, _ bool) {
	if inKey {
		buf.WriteRune('.')
	}
	switch {
	case n.first == 0 && n.last == math.MaxUint32:
		buf.WriteString("**")
	case n.first == n.last:
		if n.first == math.MaxUint32 {
			buf.WriteString("**{last}")
		} else {
			buf.WriteString(fmt.Sprintf("**{%v}", n.first))
		}
	case n.first == math.MaxUint32:
		buf.WriteString(fmt.Sprintf("**{last to %v}", n.last))
	case n.last == math.MaxUint32:
		buf.WriteString(fmt.Sprintf("**{%v to last}", n.first))
	default:
		buf.WriteString(fmt.Sprintf("**{%v to %v}", n.first, n.last))
	}
}

// priority returns the priority of the AnyNode, which is always 6.
func (*AnyNode) priority() uint8 { return lowestPriority }

// RegexNode represents a regular expression.
type RegexNode struct {
	// jpiLikeRegex
	operand Node
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
	return &RegexNode{operand: expr, pattern: pattern, flags: f}, nil
}

// String returns the RegexNode as a SQL/JSON path 'like_regex' expression.
func (n *RegexNode) String() string {
	buf := new(strings.Builder)
	n.writeTo(buf, false, false)
	return buf.String()
}

// writeTo writes the SQL/JSON path representation of n to buf. If withParens it
// will be wrapped in parentheses.
func (n *RegexNode) writeTo(buf *strings.Builder, _, withParens bool) {
	if withParens {
		buf.WriteRune('(')
	}

	n.operand.writeTo(buf, false, n.operand.priority() <= n.priority())
	buf.WriteString(fmt.Sprintf(" like_regex %q%v", n.pattern, n.flags))

	if withParens {
		buf.WriteRune(')')
	}
}

// priority returns the priority of the RegexNode, which is always 6.
func (*RegexNode) priority() uint8 { return lowestPriority }

// Regexp returns a regexp.Regexp compiled from n.
func (n *RegexNode) Regexp() *regexp.Regexp {
	flags := n.flags.goFlags()
	if n.flags.shouldQuoteMeta() {
		return regexp.MustCompile(flags + regexp.QuoteMeta(n.pattern))
	}
	return regexp.MustCompile(n.flags.goFlags() + n.pattern)
}

// Operand returns the RegexNode's operand.
func (n *RegexNode) Operand() Node {
	return n.operand
}

// AST represents the complete abstract syntax tree for a parsed SQL/JSON path.
type AST struct {
	root Node
	lax  bool
}

// New creates a new AST with n as its root. If lax is true it's considered a
// lax path query.
func New(lax bool, n Node) (*AST, error) {
	if err := validateNode(n, 0, false); err != nil {
		return nil, err
	}
	return &AST{root: n, lax: lax}, nil
}

// String returns the SQL/JSON Path-encoded string representation of the path.
func (a *AST) String() string {
	buf := new(strings.Builder)
	if !a.lax {
		buf.WriteString("strict ")
	}
	a.root.writeTo(buf, false, true)
	return buf.String()
}

// Root returns the root node of the AST.
func (a *AST) Root() Node {
	return a.root
}

// IsPredicate returns true if the AST represents a PostgreSQL-style "predicate
// check" path.
func (a *AST) IsPredicate() bool {
	_, ok := a.root.(*AccessorListNode)
	return !ok
}

// validateNode recursively validates nodes. It's based on the Postgres
// flattenJsonPathParseItem function, but does not turn the AST into a binary
// representation, just does a second pass to detect any further issues.
//
//nolint:gocognit
func validateNode(node Node, depth int, inSubscript bool) error {
	argDepth := 0
	switch node := node.(type) {
	case StringNode, *VariableNode, *KeyNode, *NumericNode:
		// Nothing to do.
	case *BinaryNode:
		if err := validateNode(node.left, depth+argDepth, inSubscript); err != nil {
			return err
		}
		if err := validateNode(node.right, depth+argDepth, inSubscript); err != nil {
			return err
		}
	case *UnaryNode:
		if node.op == UnaryFilter {
			argDepth++
		}
		if err := validateNode(node.operand, depth+argDepth, inSubscript); err != nil {
			return err
		}
	case *RegexNode:
		if err := validateNode(node.operand, depth, inSubscript); err != nil {
			return err
		}
	case ConstNode:
		//nolint:exhaustive
		switch node {
		case ConstCurrent:
			if depth <= 0 {
				//nolint:goerr113
				return errors.New("@ is not allowed in root expressions")
			}
		case ConstLast:
			if !inSubscript {
				//nolint:goerr113
				return errors.New("LAST is allowed only in array subscripts")
			}
		}
	case *ArrayIndexNode:
		for _, n := range node.subscripts {
			if err := validateNode(n, depth+argDepth, true); err != nil {
				return err
			}
		}
	case *AccessorListNode:
		for _, n := range node.accessors {
			if err := validateNode(n, depth, inSubscript); err != nil {
				return err
			}
		}
	}

	return nil
}

// NewUnaryOrNumber returns a new node for op ast.UnaryPlus or ast.UnaryMinus.
// If node is numeric and not the first item in an AccessorArray, it returns a
// ast.NumericNode. Otherwise it returns a UnaryNode.
func NewUnaryOrNumber(op UnaryOperator, node Node) Node {
	switch node := node.(type) {
	case *NumericNode:
		//nolint:exhaustive
		switch op {
		case UnaryPlus:
			// Just a positive number, return it.
			return node
		case UnaryMinus:
			// Just a negative number, return it with the minus sign.
			return NewNumeric("-"+node.literal, node.isInt)
		default:
			panic(fmt.Sprintf("Operator must be + or - but is %v", op))
		}
	case *AccessorListNode:
		// If node is an accessor with a single node, just use that node.
		if len(node.accessors) == 1 {
			return NewUnaryOrNumber(op, node.accessors[0])
		}
	}

	return NewUnary(op, node)
}
