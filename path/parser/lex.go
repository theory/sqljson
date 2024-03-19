// Package parser parses SQL/JSON paths.
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

// # Issues
//
//  - Number parsing varies from Postgres. It relies on text/scanner, which
// 	  parses Go formats, a superset of what Postgres supports. The
//    validateInt and validateNumeric functions compensate for this, but when
// 	  they find issues the errors are reported at the position after the
//    number, not at the problematic character.
//  - Some of these issues could be addressed by tweaking the position in the
//    error message, but text/scanner doesn't support backtracking.
//  - There is a circular reference between the lexer object and the
//    scanner.Scanner.
//  - The handling of UTF-8 surrogate pairs in scanUnicode consumes one too
//    many bytes if the second escape starts with a slash but is not followed
//    by a u. Ideally it would reset the scanner to before the backslash, but
//    the lack of backtracking in scanner.Scanner disallows it.
//
// These issues should be addressed by extracting the number parsing from
// text/scanner and having it do the right thing here, removing the unsupported
// syntax. This would eliminate the need for text.Scanner, and therefor the
// circular reference, but we'd need to support the position data and moving
// through the text. Could be simpler, though, iterating on the bytes in a
// string or a byte slice.

import (
	"fmt"
	"strings"
	"text/scanner"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/smasher164/xid"
)

// lexer lexes a path.
type lexer struct {
	errors  []string
	scanner *scanner.Scanner
}

// newLexer creates a new lexer configured to lex path.
func newLexer(path string) *lexer {
	l := &lexer{errors: []string{}}
	s := new(scanner.Scanner)
	s.Init(strings.NewReader(path))
	s.Filename = "path"
	s.Mode = scanner.ScanInts | scanner.ScanFloats
	s.IsIdentRune = isIdentRune

	// Yes there's a circular reference here.
	s.Error = l.scanError
	l.scanner = s

	return l
}

// scanError logs error message msg along with the position from s.
func (l *lexer) scanError(s *scanner.Scanner, msg string) {
	l.Error(fmt.Sprintf("%v at %v", msg, s.Pos()))
}

// Error logs error message msg.
func (l *lexer) Error(msg string) {
	l.errors = append(l.errors, msg)
}

// Lex lexes the path, returning the next token from the path. The text
// representation of the token will be stored in lval.str.
func (l *lexer) Lex(lval *pathSymType) int {
	tok := l.scanner.Scan()
	lval.str = l.scanner.TokenText()

	switch {
	case isIdentRune(tok, 0):
		return l.scanIdent(lval, tok)
	case tok == scanner.Int:
		return l.validateInt(lval)
	case tok == scanner.Float:
		return l.validateNumeric(lval)
	case tok == '"':
		return l.scanString(lval, STRING_P)
	case tok == '$':
		return l.scanVariable(lval)
	case tok == '/':
		t2 := l.scanComment()
		if t2 == scanner.Comment {
			return l.Lex(lval)
		}

		return t2
	default:
		return l.scanOperator(lval, tok)
	}
}

// scanVariable scans a variable name from l.scanner, assigns the resulting
// string to lval.str, and returns VARIABLE_P.
func (l *lexer) scanVariable(lval *pathSymType) int {
	tok := l.scanner.Peek()

	switch {
	case tok == '"':
		// $"xyz"
		l.scanner.Next() // Consume the '""
		return l.scanString(lval, VARIABLE_P)
	case isVariableRune(tok):
		// $xyz
		s := l.scanner

		str := new(strings.Builder)
		for isVariableRune(s.Peek()) {
			str.WriteRune(s.Next())
		}
		lval.str = str.String()

		return VARIABLE_P
	default:
		// Not a variable.
		return '$'
	}
}

// hasError returns true if any errors have been recorded by the lexer.
func (l *lexer) hasError() bool {
	return len(l.errors) > 0
}

// validateInt validates an integer. The rules for SQL jsonpath are slightly
// different than for Go, namely:
//
//   - A leading 0 is an error.
//   - Except for octal, hex, and binary literals (0o, 0b, 0x).
//   - But for those literals, underscores are allowed only after the first
//     digit, not after the letter.
//   - In other words, '0xa_b' is okay, but not '0x_ab'.
func (l *lexer) validateInt(lval *pathSymType) int {
	lval.str = l.scanner.TokenText()
	if !l.hasError() && lval.str[0] == '0' && len(lval.str) > 1 {
		// Leading 0 with subsequent characters.
		if !unicode.IsLetter(rune(lval.str[1])) {
			// Leading 0 followed by digit disallowed.
			l.scanError(l.scanner, "trailing junk after numeric literal")
		} else if len(lval.str) > 2 && lval.str[2] == '_' {
			// Underscore after letter (0o_, 0b_, 0x_) disallowed.
			l.scanError(l.scanner, "underscore disallowed at start of numeric literal")
		}
	}

	return INT_P
}

// validateNumeric validates a numeric value. The rules for SQL jsonpath are
// slightly different than for Go, namely:
//
//   - A leading 0 is an error unless it's followed by a dot (decimal).
//   - And except for octal, hex, and binary literals (0o, 0b, 0x).
//   - But for those literals, underscores are allowed only after the first
//     digit, not after the letter.
//   - In other words, '0xa_b' is okay, but not '0x_ab'.
//   - Also, Go-style p exponent in hex representations is not supported by
//     Postgres.
func (l *lexer) validateNumeric(lval *pathSymType) int {
	lval.str = l.scanner.TokenText()

	if !l.hasError() && lval.str[0] == '0' && len(lval.str) > 1 {
		// Leading 0 with subsequent characters.
		switch {
		case !unicode.IsLetter(rune(lval.str[1])):
			if lval.str[1] != '.' {
				// Leading 0 followed by digit disallowed.
				l.scanError(l.scanner, "trailing junk after numeric literal")
			}
		case len(lval.str) > 2 && lval.str[2] == '_':
			// Underscore after letter (0o_, 0b_, 0x_) disallowed.
			l.scanError(l.scanner, "underscore disallowed at start of numeric literal")
		case strings.ContainsAny(lval.str, "pP"):
			// Go-style p exponent not supported by Postgres.
			l.scanError(l.scanner, "trailing junk after numeric literal")
		}
	}

	return NUMERIC_P
}

// scanComment scans and discards a c-style /* */ comment. Returns
// scanner.Comment for a complete comment and 0 for an error.
func (l *lexer) scanComment() int {
	s := l.scanner
	if s.Peek() != '*' {
		return '/'
	}

	s.Next() // Consume '*'

	for {
		switch s.Next() {
		case '*':
			if s.Peek() == '/' {
				s.Next()
				return scanner.Comment
			}
		case scanner.EOF:
			l.scanError(s, "unexpected end of comment")
			return 0
		}
	}
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
//
//nolint:funlen,wsl
func (l *lexer) scanOperator(lval *pathSymType, tok rune) int {
	s := l.scanner
	peek := s.Peek()

	switch tok {
	case '=':
		if peek == '=' {
			s.Next()
			lval.str = "=="
			return EQUAL_P
		}
	case '>':
		if peek == '=' {
			s.Next()
			lval.str = ">="
			return GREATEREQUAL_P
		}
		lval.str = ">"
		return GREATER_P
	case '<':
		switch peek {
		case '=':
			s.Next()
			lval.str = "<="
			return LESSEQUAL_P
		case '>':
			s.Next()
			lval.str = "!="
			return NOTEQUAL_P
		default:
			lval.str = "<"
			return LESS_P
		}
	case '!':
		if peek == '=' {
			s.Next()
			lval.str = "!="
			return NOTEQUAL_P
		}
		lval.str = "!"
		return NOT_P
	case '&':
		if peek == tok {
			s.Next()
			lval.str = "&&"
			return AND_P
		}
	case '|':
		if peek == tok {
			s.Next()
			lval.str = "||"
			return OR_P
		}
	case '*':
		if peek == tok {
			s.Next()
			lval.str = "**"
			return ANY_P
		}
	default:
		return int(tok)
	}

	return int(tok)
}

const (
	quote     = '"'
	newline   = '\n'
	backslash = '\\'
	slash     = '/'
	dollar    = '$'
	null      = rune(0)
)

// scanIdent scans an identifier, the first character of which is ch; remaining
// characters are scanned. Identifiers are subject to the same escapes as
// strings.
func (l *lexer) scanIdent(lval *pathSymType, ch rune) int {
	str := new(strings.Builder)
	s := l.scanner

	// Scan the identifier as long as we have legit identifier runes.
	for i := 1; isIdentRune(ch, i); i, ch = i+1, s.Next() {
		switch ch {
		case backslash:
			// An escape sequence.
			if !l.scanEscape(str) {
				return IDENT_P
			}
		default:
			str.WriteRune(ch)
		}
	}

	// Done, grab the string and return the appropriate token.
	lval.str = str.String()

	return identToken(lval.str)
}

// scanString scans a jsonpath string. The opening double-quotation mark is
// expected ot have already been scanned, so the function scans until the
// closing quotation mark. It writes the resulting string to lval.str.
func (l *lexer) scanString(lval *pathSymType, ret int) int {
	str := new(strings.Builder)
	s := l.scanner
	ch := s.Next() // read character after quote

	// Read the string until we hit the closing quotation marks or an error.
	for ch != quote && !l.hasError() {
		if ch == newline || ch <= null {
			l.scanError(s, "literal not terminated")
			return ret
		}

		if ch == backslash {
			// An escape sequence.
			if !l.scanEscape(str) {
				return ret
			}
		} else {
			str.WriteRune(ch)
		}

		ch = s.Next()
	}

	// Done, grab the string and return.
	lval.str = str.String()

	return ret
}

// scanEscape scans an escape sequence and appends the decoded value to str.
func (l *lexer) scanEscape(str *strings.Builder) bool {
	s := l.scanner

	ch := s.Next() // read character after '\'
	switch ch {
	case 'b':
		str.WriteRune('\b')
	case 'f':
		str.WriteRune('\f')
	case 'n':
		str.WriteRune('\n')
	case 'r':
		str.WriteRune('\r')
	case 't':
		str.WriteRune('\t')
	case 'v':
		str.WriteRune('\v')
	case 'x':
		return scanHex(s, str)
	case 'u':
		return l.scanUnicode(s, str)
	case scanner.EOF:
		l.scanError(s, "unexpected end after backslash")
		return false
	default:
		// Everything else is literal.
		str.WriteRune(ch)
	}

	return true
}

// writeUnicode decodes \uNNNN and \u{NN...} UTF-16 code points into UTF-8 and
// writes it to lval.str. Returns false on error.
func (l *lexer) scanUnicode(s *scanner.Scanner, str *strings.Builder) bool {
	// Parsing borrowed from Postgres:
	// https://github.com/postgres/postgres/blob/b4a71cf/src/backend/utils/adt/jsonpath_scan.l#L669-L718
	// and from encoding/json:
	// https://cs.opensource.google/go/go/+/refs/tags/go1.22.1:src/encoding/json/decode.go;l=1253-1272
	rr := decodeUnicode(s)
	if rr <= null {
		return false
	}

	if utf16.IsSurrogate(rr) {
		// Should be followed by another escape.
		if s.Peek() != '\\' {
			s.Error(s, "Unicode low surrogate must follow a high surrogate")
			return false
		}

		// Remove backslash.
		s.Next()

		if s.Peek() != 'u' {
			// Invalid surrogate. Return an error. Ideally should backtrack to
			// \, but since there is an error it's probably no big deal.
			s.Error(s, "Unicode low surrogate must follow a high surrogate")
			return false
		}

		// Remove 'u'
		s.Next()

		rr1 := decodeUnicode(s)
		if rr1 <= null {
			return false
		}

		if dec := utf16.DecodeRune(rr, rr1); dec != unicode.ReplacementChar {
			// A valid pair; encode it as UTF-8.
			return writeUnicode(dec, str)
		}

		// Invalid surrogate, return an error
		s.Error(s, "Unicode low surrogate must follow a high surrogate")

		return false
	}

	// \u escapes are UTF-16; convert to UTF-8
	return writeUnicode(rr, str)
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

// writeUnicode UTF-8 encodes r and writes it to str. Required for UTF-16 code
// points expressed with \u escapes.
func writeUnicode(r rune, str *strings.Builder) bool {
	// Should never need more than 4 max size UTF-8 characters (16 bytes) for a
	// UTF-16 code point.
	// https://github.com/postgres/postgres/blob/c20d90a/src/include/mb/pg_wchar.h#L345
	const maxUnicodeEquivalentString = utf8.UTFMax * 4
	b := make([]byte, maxUnicodeEquivalentString)
	n := utf8.EncodeRune(b, r)
	str.Write(b[:n])

	return true
}

// merge merges two runes. Seen inline in both the Postgres and encoding/json.
// Likely to be inlined by Go.
func merge(r1, r2 rune) rune {
	const four = 4
	return (r1 << four) | r2
}

// scanHex scans a '\xNN' hex escape sequence. Returns false for invalid hex
// characters.
func scanHex(s *scanner.Scanner, str *strings.Builder) bool {
	// Parsing borrowed from the Postgres JSON Path scanner:
	// https://github.com/postgres/postgres/blob/b4a71cf/src/backend/utils/adt/jsonpath_scan.l#L720-L733
	if c1 := hexChar(s.Next()); c1 >= 0 {
		if c2 := hexChar(s.Next()); c2 >= 0 {
			decoded := merge(c1, c2)
			if decoded > null {
				str.WriteRune(decoded)
				return true
			}
		}
	}

	s.Error(s, "invalid hexadecimal character sequence")

	return false
}

// decodeUnicode decodes \uNNNN or \u{NN...} from s, returning the rune
// or null on error.
func decodeUnicode(s *scanner.Scanner) rune {
	var rr rune

	if s.Peek() == '{' {
		// parse '\u{NN...}'
		s.Next() // skip '{'
		c := s.Next()

		// Consume up to six hexadecimal characters and combine them into a
		// single rune.
		for i := 0; i < 6 && c != '}'; i, c = i+1, s.Next() {
			si := hexChar(c)
			if si < 0 {
				s.Error(s, "invalid hexadecimal character sequence")
				return null
			}

			rr = merge(rr, si)
		}

		if c != '}' {
			s.Error(s, "invalid Unicode escape sequence")
			return null
		}
	} else {
		// parse '\uNNNN'
		const sixteen = 16

		// Get the next four bytes.
		for i := 0; i < 4; i++ {
			c := hexChar(s.Next())
			if c < 0 {
				s.Error(s, "invalid hexadecimal character sequence")
				return null
			}

			rr = rr*sixteen + c
		}
	}

	if rr < 1 {
		// Invalid encoding or \u0000, null, not supported.
		s.Error(s, "invalid Unicode escape sequence")
		return null
	}

	return rr
}

// hexVal turns a hex character into a rune. Returns -1 for an invalid hex code.
// Adapted from the Postgres hexval function encoding/json's getu4 function:
// https://github.com/postgres/postgres/blob/84c18ac/src/backend/utils/adt/jsonpath_scan.l#L575-L596
// https://cs.opensource.google/go/go/+/refs/tags/go1.22.0:src/encoding/json/decode.go;l=1149-1170
func hexChar(c rune) rune {
	const decimal = 10

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
func identToken(ident string) int {
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
