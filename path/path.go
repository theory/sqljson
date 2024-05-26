// Package path provides a PostgreSQL-compatible implementation of SQL/JSON
// path expressions. It currently parses and normalizes paths, but execution
// has not yet been implemented.
package path

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"

	"github.com/theory/sqljson/path/ast"
	"github.com/theory/sqljson/path/exec"
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

// Exists checks whether the path returns any item for json. (This is useful
// only with SQL-standard JSON path expressions (when [IsPredicate] returns
// false), not predicate check expressions (when [IsPredicate] returns true),
// which always return a value.)
//
// If the [exec.WithVars] Option is specified its fields provide named values
// to be substituted into the path expression. If the [exec.WithSilent] Option
// is specified, the function suppresses some errors.
//
// ∑ If the [exec.WithTZ] Option is specified, it allows comparisons of
// date/time values that require timezone-aware conversions. The example below
// requires interpretation of the date-only value 2015-08-02 as a timestamp
// with time zone, so the result depends on the current time zone
// configuration (system or TZ environment variable):
//
//	Exists(
//		[]any{"2015-08-01 12:00:00-05"},
//		`$[*] ? (@.datetime() < "2015-08-02".datetime())`,
//		WithTZ(),
//	) → true
//
// If [exec.WithSilent] is passed, the function suppresses the following
// errors:
//
//   - missing object field or array element
//   - unexpected JSON item type
//   - datetime and numeric errors
//
// This behavior might be helpful when searching JSON document collections of
// varying structure.
func (path *Path) Exists(ctx context.Context, json any, opt ...exec.Option) (bool, error) {
	//nolint:wrapcheck // Okay to return unwrapped error
	return exec.Exists(ctx, path.AST, json, opt...)
}

// Match returns the result of predicate check for json. (This is useful only
// with predicate check expressions, not SQL-standard JSON path expressions
// (when [IsPredicate] returns false), since it will either fail or return nil
// if the path result is not a single boolean value.) The optional
// [exec.WithVars] and [exec.WithSilent], and [exec.WithTZ] Options act the
// same as for [Exists].
func (path *Path) Match(ctx context.Context, json any, opt ...exec.Option) (bool, error) {
	//nolint:wrapcheck // Okay to return unwrapped error
	return exec.Match(ctx, path.AST, json, opt...)
}

// Query returns all JSON items returned by path for json. For SQL-standard
// JSON path expressions (when [IsPredicate] returns false) it returns the
// values selected from json. For predicate check expressions (when
// [IsPredicate] returns true) it returns the result of the predicate check:
// true, false, or false + ErrNull (equivalent to Postgres returning NULL).
// The optional [exec.WithVars] and [exec.WithSilent], and [exec.WithTZ]
// Options act the same as for [Exists].
func (path *Path) Query(ctx context.Context, json any, opt ...exec.Option) (any, error) {
	//nolint:wrapcheck // Okay to return unwrapped error
	return exec.Query(ctx, path.AST, json, opt...)
}

// MustQuery is like [Query], but panics on error. Mostly provided for use in
// documentation examples.
func (path *Path) MustQuery(ctx context.Context, json any, opt ...exec.Option) any {
	res, err := exec.Query(ctx, path.AST, json, opt...)
	if err != nil {
		panic(err)
	}
	return res
}

// First returns the first JSON item returned by path for json, or nil if
// there are no results. The parameters are the same as for [Query].
func (path *Path) First(ctx context.Context, json any, opt ...exec.Option) (any, error) {
	//nolint:wrapcheck // Okay to return unwrapped error
	return exec.First(ctx, path.AST, json, opt...)
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
