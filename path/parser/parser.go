// Package parser parses SQL/JSON paths.
package parser

import (
	"errors"
	"fmt"
	"strings"

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
		return nil, fmt.Errorf(
			"%w: %v", ErrParse, strings.Join(lexer.errors, "\n"),
		)
	}

	return lexer.result, nil
}
