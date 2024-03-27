// Package path provides a PostgreSQL-compatible implementation of SQL/JSON
// path expressions. It currently parses and normalizes paths, but execution
// has not yet been implemented.
package path

import (
	"database/sql/driver"
	"errors"
	"fmt"

	"github.com/theory/sqljson/path/ast"
	"github.com/theory/sqljson/path/parser"
)

// Path provides SQL/JSON Path operations.
type Path struct {
	*ast.AST
}

var (
	// ErrPath wraps parsing and execution errors.
	ErrPath = errors.New("path")

	// ErrScan wraps scanning errors.
	ErrScan = errors.New("scan")
)

// Parse parses path and returns the resulting Path. Returns an error on parse
// failure.
func Parse(path string) (*Path, error) {
	ast, err := parser.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrPath, err)
	}
	return &Path{ast}, nil
}

// MustParse is like Parse but panics on parse failure.
func MustParse(path string) *Path {
	ast, err := parser.Parse(path)
	if err != nil {
		panic(err)
	}
	return &Path{ast}
}

// New creates and returns a new Path compiled from ast.
func New(ast *ast.AST) *Path {
	return &Path{ast}
}

// String returns the normalized string representation of path.
func (path *Path) String() string {
	return path.AST.String()
}

// PgIndexOperator returns the indexable PostgreSQL operator used to compare a
// path to a JSON value. Returns "@?" for a SQL-standard paths and "@@" for a
// predicate check expressions.
func (path *Path) PgIndexOperator() string {
	if path.IsPredicate() {
		return "@@"
	}
	return "@?"
}

// IsPredicate returns true if path represents a PostgreSQL-style "predicate
// check" expression, and false if it's a SQL-standard path.
func (path *Path) IsPredicate() bool {
	return path.AST.IsPredicate()
}

// Matches returns the result of path evaluated against json. Returns true if
// path is a SQL-standard path that return any item or if path is a predicate
// check expression that returns true. PostgreSQL equivalents for SQL standard
// paths:
//
//	SELECT '{"a":[1,2,3,4,5]}'::jsonb @? '$.a[*] ? (@ > 2)';
//	SELECT '{"a":[1,2,3,4,5]}'::jsonb @@ 'exists($.a[*] ? (@ > 2))';
//	SELECT jsonb_path_exists('{"a":[1,2,3,4,5]}', '$.a[*] ? (@ >= $min && @ <= $max)', '{"min":2, "max":4}')
//
// PostgreSQL equivalents for predicate-check paths:
//
//	SELECT '{"a":[1,2,3,4,5]}'::jsonb @@ '$.a[*] > 2';
//	SELECT jsonb_path_match('{"a":[1,2,3,4,5]}', 'exists($.a[*] ? (@ >= $min && @ <= $max))', '{"min":2, "max":4}')
//
// If the vars is not nil, its fields provide named values to be substituted
// for variables in the jsonpath expression. If the silent is true, the
// function suppresses the following errors:
//
//   - missing object field or array element
//   - unexpected JSON item type
//   - datetime and numeric errors
//
// This behavior might be helpful when searching JSON document collections of
// varying structure.
//
// NOTE: Currently unimplemented, just returns true.
func (path *Path) Matches(json any, vars map[string]any, silent bool) bool {
	_ = json
	_ = vars
	_ = silent
	return true
}

// Select returns all JSON items returned by path evaluated against json. For
// SQL-standard path expressions it returns the JSON values selected from
// target. For predicate check expressions it returns the result of the
// predicate check: true, false, or nil. The optional vars and silent act the
// same as for [Matches].
//
// NOTE: Currently unimplemented, just returns json.
func (path *Path) Select(json any, vars map[string]any, silent bool) any {
	_ = vars
	_ = silent
	return json
}

// Scan implements sql.Scanner so Paths can be read from databases
// transparently. Currently, database types that map to string and []byte are
// supported. Please consult database-specific driver documentation for
// matching types.
func (path *Path) Scan(src any) error {
	switch src := src.(type) {
	case nil:
		return nil
	case string:
		// if an empty Path comes from a table, we return a null Path
		if src == "" {
			return nil
		}

		// see Parse for required string format
		ast, err := parser.Parse(src)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrScan, err)
		}

		*path = Path{ast}

	case []byte:
		// if an empty Path comes from a table, we return a null Path
		if len(src) == 0 {
			return nil
		}

		// Parse as a string.
		return path.Scan(string(src))

	default:
		return fmt.Errorf("%w: unable to scan type %T into Path", ErrScan, src)
	}

	return nil
}

// Value implements sql.Valuer so that Paths can be written to databases
// transparently. Currently, Paths map to strings. Please consult
// database-specific driver documentation for matching types.
func (path Path) Value() (driver.Value, error) {
	return path.String(), nil
}

// MarshalText implements encoding.TextMarshaler.
func (path Path) MarshalText() ([]byte, error) {
	return path.MarshalBinary()
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (path *Path) UnmarshalText(data []byte) error {
	return path.UnmarshalBinary(data)
}

// MarshalBinary implements encoding.BinaryMarshaler.
func (path Path) MarshalBinary() ([]byte, error) {
	return []byte(path.String()), nil
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler.
func (path *Path) UnmarshalBinary(data []byte) error {
	ast, err := parser.Parse(string(data))
	if err != nil {
		return fmt.Errorf("%w: %w", ErrScan, err)
	}
	*path = Path{ast}
	return nil
}
