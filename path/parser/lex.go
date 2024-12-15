package parser

// https://www.postgresql.org/docs/current/datatype-json.html#DATATYPE-JSONPATH:
// Numeric literals in SQL/JSON path expressions follow JavaScript rules, which
// are different from both SQL and JSON in some minor details. For example,
// SQL/JSON path allows .1 and 1., which are invalid in JSON. Non-decimal
// integer literals and underscore separators are supported, for example,
// 1_000_000, 0x1EEE_FFFF, 0o273, 0b100101. In SQL/JSON path (and in JavaScript,
// but not in SQL proper), there must not be an underscore separator directly
// after the radix prefix.
//
// An SQL/JSON path expression is typically written in an SQL query as an SQL
// character string literal, so it must be enclosed in single quotes, and any
// single quotes desired within the value must be doubled (see Section 4.1.2.1).
// Some forms of path expressions require string literals within them. These
// embedded string literals follow JavaScript/ECMAScript conventions: they must
// be surrounded by double quotes, and backslash escapes may be used within them
// to represent otherwise-hard-to-type characters. In particular, the way to
// write a double quote within an embedded string literal is \", and to write a
// backslash itself, you must write \\. Other special backslash sequences
// include those recognized in JavaScript strings: \b, \f, \n, \r, \t, \v for
// various ASCII control characters, and \uNNNN for a Unicode character
// identified by its 4-hex-digit code point and \u{N...} for a character code
// written with 1 to 6 hex digits.
//
// https://go.dev/ref/spec#Integer_literals
// An integer literal is a sequence of digits representing an integer constant.
// An optional prefix sets a non-decimal base: 0b or 0B for binary, 0, 0o, or 0O
// for octal, and 0x or 0X for hexadecimal [Go 1.13]. A single 0 is considered a
// decimal zero. In hexadecimal literals, letters a through f and A through F
// represent values 10 through 15.

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/smasher164/xid"
	"github.com/theory/sqljson/path/ast"
)

// position is a value that represents a source position.
type position struct {
	Offset int // byte offset, starting at 0
	Line   int // line number, starting at 1
	Column int // column number, starting at 1 (character count per line)
}

// String returns the string representation of the position.
func (pos position) String() string {
	return fmt.Sprintf("%d:%d", pos.Line, pos.Column)
}

const (
	// whitespace selects white space characters.
	whitespace = 1<<'\t' | 1<<'\n' | 1<<'\r' | 1<<' '

	// Stop lexing: EOF or error.
	stopTok = -1

	// no char read yet, not EOF.
	noChar = -1

	// Literal tokens.
	quote     = '"'
	newline   = '\n'
	backslash = '\\'
	slash     = '/'
	dollar    = '$'
	null      = rune(0)

	// Numeric bases.
	decimal = 10
	hex     = 16
	octal   = 8
	binary  = 2
)

// lexer lexes a path.
type lexer struct {
	// Collects errors while lexing.
	errors []string

	// The parser stores the parsed result here, using setResult() and
	// setPred().
	result *ast.AST
	pred   bool

	// Buffer to hold normalized string while parsing JavaScript string.
	strBuf strings.Builder

	// True if a string was parsed into gotString.
	gotString bool

	// Remaining fields borrowed from text/scanner.

	// Source buffer
	srcBuf []byte // Source buffer
	srcPos int    // reading position (srcBuf index)
	srcEnd int    // source end (srcBuf index)

	// Source position
	// srcBufOffset int // byte offset of srcBuf[0] in source
	line        int // line count
	column      int // character count
	lastLineLen int // length of last line in characters (for correct column reporting)
	lastCharLen int // length of last character in bytes

	// Token position
	tokPos int // token text tail position (srcBuf index); valid if >= 0
	tokEnd int // token text tail end (srcBuf index)

	// One character look-ahead
	ch rune // character before current srcPos

	// Start position of most recently scanned token; set by Scan.
	// Calling Init or Next invalidates the position (Line == 0).
	// The Filename field is always left untouched by the Scanner.
	// If an error is reported (via Error) and position is invalid,
	// the scanner is not inside a token. Call Pos to obtain an error
	// position in that case, or to obtain the position immediately
	// after the most recently scanned token.
	position
}

// newLexer creates a new lexer configured to lex path.
func newLexer(path string) *lexer {
	return &lexer{
		// initialize errors
		errors: []string{},

		// initialize source buffer
		srcBuf: []byte(path),
		srcEnd: len(path),

		// initialize source position
		line: 1,
		// initialize token text buffer(required for first call to next())
		tokPos: noChar,
		// initialize one character look-ahead
		ch: noChar, // no char read yet, not EOF
	}
}

func (l *lexer) resetStrBuf() {
	l.strBuf.Reset()
	l.gotString = false
}

// next reads and returns the next Unicode character. It is designed such
// that only a minimal amount of work needs to be done in the common ASCII
// case (one test to check for both ASCII and end-of-buffer, and one test
// to check for newlines).
func (l *lexer) next() rune {
	if l.srcPos == l.srcEnd {
		if l.lastCharLen > 0 {
			// previous character was not EOF
			l.column++
		}
		l.lastCharLen = 0
		return stopTok
	}

	ch, width := rune(l.srcBuf[l.srcPos]), 1
	if ch >= utf8.RuneSelf {
		// uncommon case: not ASCII
		ch, width = utf8.DecodeRune(l.srcBuf[l.srcPos:l.srcEnd])
		if ch == utf8.RuneError && width == 1 {
			// advance for correct error position
			l.srcPos += width
			l.lastCharLen = width
			l.column++
			l.Error("invalid UTF-8 encoding")
			return stopTok
		}
	}

	// advance
	l.srcPos += width
	l.lastCharLen = width
	l.column++

	// special situations
	switch ch {
	case 0:
		l.Error("invalid character NULL")
		ch = stopTok
	case '\n':
		l.line++
		l.lastLineLen = l.column
		l.column = 0
	}

	return ch
}

// peek returns the next Unicode character in the source without advancing
// the scanner. It returns EOF if the scanner's position is at the last
// character of the source.
func (l *lexer) peek() rune {
	if l.ch == noChar {
		// this code is only run for the very first character
		l.ch = l.next()
	}
	return l.ch
}

// Error implements the Error function required by the pathLexer interface
// generated by the parser grammar. It appends msg and the current position to
// l.errors.
func (l *lexer) Error(msg string) {
	l.tokEnd = l.srcPos - l.lastCharLen // make sure token text is terminated
	l.errors = append(l.errors, fmt.Sprintf("%v at %v", msg, l.pos()))
}

// errorf provides a fmt-compatible interface sending an error to [Error].
func (l *lexer) errorf(format string, args ...any) {
	l.Error(fmt.Sprintf(format, args...))
}

// pos returns the position of the character immediately after the character
// or token returned by the last call to Next or Scan. Use l.Position for the
// start position of the most recently scanned token.
//
//nolint:nonamedreturns
func (l *lexer) pos() (pos position) {
	pos.Offset = l.srcPos - l.lastCharLen
	switch {
	case l.column > 0:
		// common case: last character was not a '\n'
		pos.Line = l.line
		pos.Column = l.column
	case l.lastLineLen > 0:
		// last character was a '\n'
		pos.Line = l.line - 1
		pos.Column = l.lastLineLen
	default:
		// at the beginning of the source
		pos.Line = 1
		pos.Column = 1
	}
	return
}

// Lex implements the Lex function required by the pathLexer interface
// generated by the parser grammar. It lexes the path, returning the next
// token or Unicode character from the path. The text representation of the
// token will be stored in lval.str. It reports scanning errors (read
// and token errors) by calling l.Error.
func (l *lexer) Lex(lval *pathSymType) int {
	ch := l.peek()

	// reset token text position
	l.tokPos = -1
	l.Line = 0

redo:
	// skip white space
	for whitespace&(1<<uint(ch)) != 0 {
		ch = l.next()
	}

	// start collecting token text
	l.resetStrBuf()
	l.tokPos = l.srcPos - l.lastCharLen

	// set token position
	// (this is a slightly optimized version of the code in Pos())
	l.Offset = l.tokPos
	if l.column > 0 {
		// common case: last character was not a '\n'
		l.Line = l.line
		l.Column = l.column
	} else {
		// last character was a '\n'
		// (we cannot be at the beginning of the source
		// since we have called next() at least once)
		l.Line = l.line - 1
		l.Column = l.lastLineLen
	}

	// determine token value
	tok := ch
	switch {
	case isIdentRune(ch, 0):
		tok, ch = l.scanIdent(ch)
	case isDecimal(ch):
		tok, ch = l.scanNumber(ch, false)
	default:
		switch ch {
		case stopTok:
			break
		case '"':
			tok, ch = l.scanString(STRING_P)
		case '$':
			tok, ch = l.scanVariable()
		case '/':
			ch = l.next()
			if ch == '*' {
				l.tokPos = -1 // don't collect token text
				ch = l.scanComment(ch)
				goto redo
			}
		case '.':
			ch = l.next()
			if isDecimal(ch) {
				tok, ch = l.scanNumber(ch, true)
			}
		default:
			tok, ch = l.scanOperator(ch)
		}
	}

	l.tokEnd = l.srcPos - l.lastCharLen

	l.ch = ch
	lval.str = l.tokenText()
	return int(tok)
}

func lower(ch rune) rune     { return ('a' - 'A') | ch } // returns lower-case ch iff ch is ASCII letter
func isDecimal(ch rune) bool { return '0' <= ch && ch <= '9' }
func isHex(ch rune) bool     { return '0' <= ch && ch <= '9' || 'a' <= lower(ch) && lower(ch) <= 'f' }

// digits accepts the sequence { digit | '_' } starting with ch0.
// If base <= 10, digits accepts any decimal digit but records
// the first invalid digit >= base in *invalid if *invalid == 0.
// digits returns the first rune that is not part of the sequence
// anymore, and a bitset describing whether the sequence contained
// digits (bit 0 is set), or separators '_' (bit 1 is set).
//
//nolint:nonamedreturns
func (l *lexer) digits(ch0 rune, base int, invalid *rune) (ch rune, digSep int) {
	ch = ch0
	if base <= decimal {
		maxCh := rune('0' + base)
		for isDecimal(ch) || ch == '_' {
			ds := 1
			if ch == '_' {
				ds = 2
			} else if ch >= maxCh && *invalid == 0 {
				*invalid = ch
			}
			digSep |= ds
			ch = l.next()
		}
	} else {
		for isHex(ch) || ch == '_' {
			ds := 1
			if ch == '_' {
				ds = 2
			}
			digSep |= ds
			ch = l.next()
		}
	}
	return
}

//nolint:funlen,gocognit
func (l *lexer) scanNumber(ch rune, seenDot bool) (rune, rune) {
	base := decimal    // number base
	prefix := rune(0)  // one of 0 (decimal), '0' (0-octal), 'x', 'o', or 'b'
	digSep := 0        // bit 0: digit present, bit 1: '_' present
	invalid := rune(0) // invalid digit in literal, or 0

	// integer part
	var tok rune
	var ds int

	if !seenDot {
		tok = INT_P
		if ch == '0' {
			ch = l.next()
			switch lower(ch) {
			case 'x':
				ch = l.next()
				base, prefix = hex, 'x'
			case 'o':
				ch = l.next()
				base, prefix = octal, 'o'
			case 'b':
				ch = l.next()
				base, prefix = binary, 'b'
			case '.':
				base, prefix = octal, '0'
				digSep = 1 // leading 0
			default:
				switch {
				case ch == '_':
					l.Error("underscore disallowed at start of numeric literal")
					return stopTok, stopTok
				case isDecimal(ch):
					l.Error("trailing junk after numeric literal")
					return stopTok, stopTok
				default:
					base, prefix = octal, '0'
					digSep = 1 // leading 0
				}
			}
		}

		if ch == '_' {
			l.Error("underscore disallowed at start of numeric literal")
			return stopTok, stopTok
		}

		ch, ds = l.digits(ch, base, &invalid)
		digSep |= ds
		if digSep&1 == 0 {
			// No digits found, invalid.
			l.Error("trailing junk after numeric literal")
			return stopTok, stopTok
		}

		if ch == '.' {
			// May be numeric, though prefixes are integer-only.
			if prefix != 0 && prefix != '0' {
				// Digits found, 0x, 0o, or 0b integer looks valid, halt.
				return tok, '.'
			}

			ch = l.next()
			seenDot = true
		}
	}

	// fractional part
	if seenDot {
		tok = NUMERIC_P
		ch, ds = l.digits(ch, base, &invalid)
		digSep |= ds
	}

	// exponent
	if e := lower(ch); e == 'e' {
		if prefix != 0 && prefix != '0' {
			l.errorf("%q exponent requires decimal mantissa", ch)
			return stopTok, stopTok
		}

		ch = l.next()
		tok = NUMERIC_P
		if ch == '+' || ch == '-' {
			ch = l.next()
		}
		ch, ds = l.digits(ch, decimal, nil)
		digSep |= ds
		if ds&1 == 0 {
			l.Error("exponent has no digits")
			return stopTok, stopTok
		}
	} else if isIdentRune(e, 0) {
		l.Error("trailing junk after numeric literal")
		return stopTok, stopTok
	}

	if tok == INT_P && invalid != 0 {
		l.errorf("invalid digit %q in %s", invalid, litName(prefix))
		return stopTok, stopTok
	}

	if digSep&2 != 0 {
		l.tokEnd = l.srcPos - l.lastCharLen // make sure token text is terminated
		if i := invalidSep(l.tokenText()); i >= 0 {
			l.Error("'_' must separate successive digits")
			return stopTok, stopTok
		}
	}

	if isIdentRune(ch, 0) {
		l.Error("trailing junk after numeric literal")
		return stopTok, stopTok
	}

	return tok, ch
}

// tokenText returns the string corresponding to the most recently scanned token.
// Valid after calling Scan and in calls of Scanner.Error.
func (l *lexer) tokenText() string {
	if l.tokPos < 0 {
		// no token text
		return ""
	}

	if l.tokEnd < l.tokPos {
		// if EOF was reached, s.tokEnd is set to -1 (s.srcPos == 0)
		l.tokEnd = l.tokPos
	}

	if l.gotString {
		// A string was parsed, return it.
		return l.strBuf.String()
	}

	return string(l.srcBuf[l.tokPos:l.tokEnd])
}

// invalidSep returns the index of the first invalid separator in x, or -1.
func invalidSep(x string) int {
	x1 := ' ' // prefix char, we only care if it's 'x'
	d := '.'  // digit, one of '_', '0' (a digit), or '.' (anything else)
	i := 0

	// a prefix counts as a digit
	if len(x) >= 2 && x[0] == '0' {
		x1 = lower(rune(x[1]))
		if x1 == 'x' || x1 == 'o' || x1 == 'b' {
			d = '0'
			i = 2
		}
	}

	// mantissa and exponent
	for ; i < len(x); i++ {
		p := d // previous digit
		d = rune(x[i])
		switch {
		case d == '_':
			if p != '0' {
				return i
			}
		case isDecimal(d) || x1 == 'x' && isHex(d):
			d = '0'
		default:
			if p == '_' {
				return i - 1
			}
			d = '.'
		}
	}
	if d == '_' {
		return len(x) - 1
	}

	return -1
}

func litName(prefix rune) string {
	switch prefix {
	default:
		return "decimal literal"
	case 'x':
		return "hexadecimal literal"
	case 'o', '0':
		return "octal literal"
	case 'b':
		return "binary literal"
	}
}

// setResult creates an ast.AST and assigns it to l.result unless
// there are parser or ast.New errors.
func (l *lexer) setResult(lax bool, node ast.Node) {
	if l.hasError() {
		return
	}
	ast, err := ast.New(lax, l.pred, node)
	if err != nil {
		l.errors = append(l.errors, err.Error())
	}
	l.result = ast
}

// setPred indicates that the path being lexed is a predicate path query.
// Called by the parser grammar.
func (l *lexer) setPred() {
	l.pred = true
}

// scanVariable scans a variable name from l.scanner, assigns the resulting
// string to lval.str, and returns VARIABLE_P.
func (l *lexer) scanVariable() (rune, rune) {
	ch := l.next()
	switch {
	case ch == '"':
		// $"xyz"
		return l.scanString(VARIABLE_P)
	case isVariableRune(ch):
		// $xyz
		l.strBuf.WriteRune(ch)
		ch = l.next()
		for ; isVariableRune(ch); ch = l.next() {
			l.strBuf.WriteRune(ch)
		}

		l.gotString = true

		return VARIABLE_P, ch
	default:
		// Not a variable.
		return '$', ch
	}
}

// hasError returns true if any errors have been recorded by the lexer.
func (l *lexer) hasError() bool {
	return len(l.errors) > 0
}

// scanComment scans and discards a c-style /* */ comment. Returns Comment for
// a complete comment and 0 for an error.
func (l *lexer) scanComment(ch rune) rune {
	if ch != '*' {
		return '/'
	}

	ch = l.next() // read character after "/*"
	for {
		if ch < null {
			l.Error("unexpected end of comment")
			break
		}
		ch0 := ch
		ch = l.next()
		if ch0 == '*' && ch == '/' {
			ch = l.next()
			break
		}
	}
	return ch
}

// scanOperator scans an operator from l.scanner if there is one, or else
// returns tok. Operators scanned:
//
//   - ==
//   - >
//   - >=
//   - <
//   - <=
//   - <>, !=
//   - !
//   - &&
//   - ||
//   - **
//
// Which all mean what you'd expect mathematically and in SQL, except for
// '**', which represents the Postgres-specific '.**' any path selector.
func (l *lexer) scanOperator(ch rune) (rune, rune) {
	next := l.next() // Read the next character

	switch ch {
	case '=':
		if next == '=' {
			return EQUAL_P, l.next()
		}
	case '>':
		if next == '=' {
			return GREATEREQUAL_P, l.next()
		}
		return GREATER_P, next
	case '<':
		switch next {
		case '=':
			return LESSEQUAL_P, l.next()
		case '>':
			return NOTEQUAL_P, l.next()
		default:
			return LESS_P, next
		}
	case '!':
		if next == '=' {
			return NOTEQUAL_P, l.next()
		}
		return NOT_P, next
	case '&':
		if next == ch {
			return AND_P, l.next()
		}
	case '|':
		if next == ch {
			return OR_P, l.next()
		}
	case '*':
		if next == ch {
			return ANY_P, l.next()
		}
	default:
		return ch, next
	}

	return ch, next
}

// scanIdent scans an identifier, the first character of which is ch; remaining
// characters are scanned. Identifiers are subject to the same escapes as
// strings.
func (l *lexer) scanIdent(ch rune) (rune, rune) {
	// we know the zero'th rune is OK
	switch ch {
	case backslash:
		// An escape sequence.
		ch = l.scanEscape()
	default:
		l.strBuf.WriteRune(ch)
		ch = l.next()
	}

	// Scan the identifier as long as we have legit identifier runes.
	for isIdentRune(ch, 1) {
		switch ch {
		case backslash:
			// An escape sequence.
			ch = l.scanEscape()
		default:
			l.strBuf.WriteRune(ch)
			ch = l.next()
		}
	}

	if l.hasError() {
		return stopTok, ch
	}

	l.gotString = true
	return identToken(l.strBuf.String()), ch
}

func (l *lexer) scanString(ret rune) (rune, rune) {
	ch := l.next() // read character after quote
	for ch != quote {
		if ch == newline || ch < 0 {
			if !l.hasError() {
				l.Error("literal not terminated")
			}
			l.resetStrBuf()
			return stopTok, ch
		}
		if ch == backslash {
			ch = l.scanEscape()
		} else {
			l.strBuf.WriteRune(ch)
			ch = l.next()
		}
	}

	l.gotString = true
	return ret, l.next()
}

func (l *lexer) scanEscape() rune {
	ch := l.next() // read character after '\'
	switch ch {
	case 'b':
		l.strBuf.WriteRune('\b')
		ch = l.next()
	case 'f':
		l.strBuf.WriteRune('\f')
		ch = l.next()
	case 'n':
		l.strBuf.WriteRune('\n')
		ch = l.next()
	case 'r':
		l.strBuf.WriteRune('\r')
		ch = l.next()
	case 't':
		l.strBuf.WriteRune('\t')
		ch = l.next()
	case 'v':
		l.strBuf.WriteRune('\v')
		ch = l.next()
	case 'x':
		ch = l.scanHex()
	case 'u':
		ch = l.scanUnicode()
	case stopTok:
		l.Error("unexpected end after backslash")
		ch = stopTok
	default:
		// Everything else is literal.
		l.strBuf.WriteRune(ch)
		ch = l.next()
	}

	if ch == stopTok {
		// Reset the string.
		l.resetStrBuf()
	}

	return ch
}

// scanUnicode decodes \uNNNN and \u{NN...} UTF-16 code points into UTF-8.
func (l *lexer) scanUnicode() rune {
	// Parsing borrowed from Postgres:
	// https://github.com/postgres/postgres/blob/REL_17_2/src/backend/utils/adt/jsonpath_scan.l#L669-L718
	// and from encoding/json:
	// https://cs.opensource.google/go/go/+/refs/tags/go1.22.1:src/encoding/json/decode.go;l=1253-1272
	rr := l.decodeUnicode()
	if rr <= null {
		return rr
	}

	if utf16.IsSurrogate(rr) {
		// Should be followed by another escape.
		if l.next() != '\\' {
			l.Error("Unicode low surrogate must follow a high surrogate")
			return stopTok
		}

		if l.next() != 'u' {
			// Invalid surrogate. Backtrack to \ and return an error.
			l.srcPos -= l.lastCharLen
			l.lastCharLen = 1
			l.Error("Unicode low surrogate must follow a high surrogate")
			return stopTok
		}
		rr1 := l.decodeUnicode()
		if rr1 <= null {
			return rr1
		}

		if dec := utf16.DecodeRune(rr, rr1); dec != unicode.ReplacementChar {
			// A valid pair; encode it as UTF-8.
			l.writeUnicode(dec)
			return l.next()
		}

		// Invalid surrogate, return an error
		l.Error("Unicode low surrogate must follow a high surrogate")
		return stopTok
	}

	// \u escapes are UTF-16; convert to UTF-8
	l.writeUnicode(rr)
	return l.next()
}

// isIdentRune is a predicate controlling the characters accepted as the ith
// rune in an identifier. These follow JavaScript [identifier syntax], including
// support for \u0000 and \u{000000} unicode escapes:
//
// > In JavaScript, identifiers are commonly made of alphanumeric characters,
// > underscores (_), and dollar signs ($). Identifiers are not allowed to
// > start with numbers. However, JavaScript identifiers are not only limited
// > to ASCII — many Unicode code points are allowed as well. Namely, any
// > character in the [ID_Start] category can start an identifier, while any
// > character in the [ID_Continue] category can appear after the first
// >  character.
// >
// > In addition, JavaScript allows using Unicode escape sequences in the
// > form of \u0000 or \u{000000} in identifiers, which encode the same
// > string value as the actual Unicode characters.
//
// Variations from the spec:
//
//   - Postgres does not support literal [dollar signs], and so neither do we.
//     One can They can still be specified via '\$` or '\u0024'.
//
// Variations from Postgres:
//
//   - Postgres allows a much wider range of Unicode characters than the
//     JavaScript spec requires, including Emoji, but this function follows
//     the spec.
//
// [identifier syntax]: https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Lexical_grammar#identifiers
// [ID_Start]: https://util.unicode.org/UnicodeJsps/list-unicodeset.jsp?a=%5Cp%7BID_Start%7D
// [ID_Continue]: https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Lexical_grammar#identifiers
// [dollar signs]: https://www.postgresql.org/message-id/9F84036F-007A-432D-8DCD-1D5C3F51F76E%40justatheory.com
func isIdentRune(ch rune, i int) bool {
	return ch == '_' || ch == '\\' || (i == 0 && xid.Start(ch)) || (i > 0 && xid.Continue(ch))
}

// isVariableRune is a predicate controlling the characters accepted as a rune
// in a variable name. It follows the same conventions as isIdentRune, except
// that the first character is not treated different, because in SQL/JSON paths,
// variables always start with '$'.
func isVariableRune(ch rune) bool {
	return xid.Continue(ch)
}

// writeUnicode UTF-8 encodes r and writes it to l.strBuf. Required for UTF-16
// code points expressed with \u escapes.
func (l *lexer) writeUnicode(r rune) {
	// Should never need more than 4 max size UTF-8 characters (16 bytes) for a
	// UTF-16 code point.
	// https://github.com/postgres/postgres/blob/REL_17_2/src/src/include/mb/pg_wchar.h#L345
	const maxUnicodeEquivalentString = utf8.UTFMax * 4
	b := make([]byte, maxUnicodeEquivalentString)
	n := utf8.EncodeRune(b, r)
	l.strBuf.Write(b[:n])
}

// merge merges two runes. Seen inline in both the Postgres and encoding/json.
// Likely to be inlined by Go.
func merge(r1, r2 rune) rune {
	const four = 4
	return (r1 << four) | r2
}

func (l *lexer) scanHex() rune {
	// Parsing borrowed from the Postgres JSON Path scanner:
	// https://github.com/postgres/postgres/blob/REL_17_2/src/backend/utils/adt/jsonpath_scan.l#L720-L733
	if c1 := hexChar(l.next()); c1 >= 0 {
		if c2 := hexChar(l.next()); c2 >= 0 {
			decoded := merge(c1, c2)
			if decoded > null {
				l.strBuf.WriteRune(decoded)
				return l.next()
			}
		}
	}

	l.Error("invalid hexadecimal character sequence")
	return stopTok
}

// decodeUnicode decodes \uNNNN or \u{NN...} from s, returning the rune
// or null on error.
func (l *lexer) decodeUnicode() rune {
	var rr rune

	if ch := l.next(); ch == '{' {
		// parse '\u{NN...}'
		c := l.next()

		// Consume up to six hexadecimal characters and combine them into a
		// single rune.
		for i := 0; i < 6 && c != '}'; i, c = i+1, l.next() {
			si := hexChar(c)
			if si < null {
				l.Error("invalid Unicode escape sequence")
				return stopTok
			}

			rr = merge(rr, si)
		}

		if c != '}' {
			l.Error("invalid Unicode escape sequence")
			return stopTok
		}
	} else {
		// parse '\uNNNN'
		// Get the next four bytes.
		// l.tokPos--
		rr = hexChar(ch)
		if rr < null {
			l.Error("invalid Unicode escape sequence")
			return stopTok
		}
		for range 3 {
			c := hexChar(l.next())
			if c < null {
				l.Error("invalid Unicode escape sequence")
				return stopTok
			}

			rr = rr*hex + c
		}
	}

	if rr == null {
		// \u0000, null, not supported.
		l.Error(`\u0000 cannot be converted to text`)
		return stopTok
	}

	return rr
}

// hexVal turns a hex character into a rune. Returns -1 for an invalid hex code.
// Adapted from the Postgres hexval function encoding/json's getu4 function:
// https://github.com/postgres/postgres/blob/REL_17_2/src/backend/utils/adt/jsonpath_scan.l#L575-L596
// https://cs.opensource.google/go/go/+/refs/tags/go1.22.0:src/encoding/json/decode.go;l=1149-1170
func hexChar(c rune) rune {
	switch {
	case '0' <= c && c <= '9':
		return c - '0'
	case 'a' <= c && c <= 'f':
		return c - 'a' + decimal
	case 'A' <= c && c <= 'F':
		return c - 'A' + decimal
	default:
		return -1
	}
}

// identToken examines ident and returns the appropriate token value. If ident
// is not a jsonpath reserved word ident, it returns IDENT_P.
//
//nolint:funlen,gocyclo
func identToken(ident string) rune {
	// Start with keywords required to be lowercase.
	switch ident {
	case "null":
		return NULL_P
	case "true":
		return TRUE_P
	case "false":
		return FALSE_P
	}

	// Now try case-insensitive keywords.
	switch strings.ToLower(ident) {
	case "is":
		return IS_P
	case "to":
		return TO_P
	case "abs":
		return ABS_P
	case "lax":
		return LAX_P
	case "date":
		return DATE_P
	case "flag":
		return FLAG_P
	case "last":
		return LAST_P
	case "size":
		return SIZE_P
	case "time":
		return TIME_P
	case "type":
		return TYPE_P
	case "with":
		return WITH_P
	case "floor":
		return FLOOR_P
	case "bigint":
		return BIGINT_P
	case "double":
		return DOUBLE_P
	case "exists":
		return EXISTS_P
	case "number":
		return NUMBER_P
	case "starts":
		return STARTS_P
	case "strict":
		return STRICT_P
	case "string":
		return STRINGFUNC_P
	case "boolean":
		return BOOLEAN_P
	case "ceiling":
		return CEILING_P
	case "decimal":
		return DECIMAL_P
	case "integer":
		return INTEGER_P
	case "time_tz":
		return TIME_TZ_P
	case "unknown":
		return UNKNOWN_P
	case "datetime":
		return DATETIME_P
	case "keyvalue":
		return KEYVALUE_P
	case "timestamp":
		return TIMESTAMP_P
	case "like_regex":
		return LIKE_REGEX_P
	case "timestamp_tz":
		return TIMESTAMP_TZ_P
	default:
		return IDENT_P
	}
}
