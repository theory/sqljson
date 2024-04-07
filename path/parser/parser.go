// Package parser parses SQL/JSON paths. It uses the same grammar as Postgres
// to support the same syntax and capabilities, with a few minor exceptions.
// The lexer use patterns borrowed PostgreSQL and from text/scanner.
package parser

import (
	"errors"
	"fmt"

	"github.com/theory/sqljson/path/ast"
)

//go:generate goyacc -v "" -o grammar.go -p path grammar.y

// ErrParse errors are returned by the parser.
var ErrParse = errors.New("parser")

// Parse parses path.
func Parse(path string) (*ast.AST, error) {
	lexer := newLexer(path)
	_ = pathParse(lexer)

	if len(lexer.errors) > 0 {
		return nil, fmt.Errorf("%w: %v", ErrParse, lexer.errors[0])
	}

	return lexer.result, nil
}
