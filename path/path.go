/*
Package path provides PostgreSQL 17-compatible SQL/JSON path parsing and
execution. It supports both SQL-standard path expressions and
PostgreSQL-specific predicate check expressions. See the README for a
description of the SQL/JSON Path language.

# Postgres Equivalents

List of the PostgreSQL jsonpath functions and their path Package equivalents:

  - @? Operator: Use [Path.Exists] with [exec.WithSilent]
  - @@ Operator: Use [Path.Match] with [exec.Silent]
  - jsonb_path_exists(): Use [Path.Exists]
  - jsonb_path_match(): Use [Path.Match]
  - jsonb_path_query() and jsonb_path_query_array(): Use [Path.Query]
  - jsonb_path_query_first(): Use [Path.First]
  - jsonb_path_exists_tz(): Use [Path.Exists] with [exec.WithTZ]
  - jsonb_path_match_tz(): Use [Path.Match] with [exec.WithTZ]
  - jsonb_path_query_tz() and jsonb_path_query_array_tz(): Use [Path.Query]
    with [exec.WithTZ]
  - jsonb_path_query_first_tz(): Use [Path.First] with [exec.WithTZ]

# Options

The path query methods take an optional list of [exec.Option] arguments.

  - [exec.WithVars] provides named values to be substituted into the
    path expression. See the WithVars example for a demonstration.

  - [exec.WithSilent] suppresses [exec.ErrVerbose] errors, including missing
    object field or array element, unexpected JSON item type, and datetime
    and numeric errors. This behavior might be helpful when searching JSON
    entities of varying structure. See the WithSilent example for a
    demonstration.

  - [exec.WithTZ] allows comparisons of date and time values that require
    timezone-aware conversions. By default such conversions are made relative
    to UTC, but can be made relative to another (user-preferred) time zone by
    using [types.ContextWithTZ] to add it to the context passed to the query
    method. See the WithTZ example for a demonstration, and [types] for more
    comprehensive examples.

# Two Types of Queries

PostgreSQL supports two flavors of path expressions, and this package follows
suit:

  - SQL-standard path expressions hew to the SQL standard, which allows
    Boolean predicates only in ?() filter expressions, and can return
    any number of results.
  - Boolean predicate check expressions are a PostgreSQL extension that allow
    path expression to be a Boolean predicate, which can return only true,
    false, and null.

This duality can sometimes cause confusion, especially when using
[Path.Exists] and the Postgres @? operator, which only work with SQL standard
expressions, and [Path.Match] and the Postgres @@ operator, which only work
with predicate check expressions.

The path package provides a couple of additional features to help navigate
this duality:

  - [Path.IsPredicate] returns true if a Path is a predicate check expression
  - [Path.PgIndexOperator] returns a string representing the appropriate
    Postgres operator to use when sending queries to the database: @? for
    SQL-standard expressions and @@ for predicate check expressions.
  - [Path.ExistsOrMatch] dispatches to the appropriate function, [Path.Exists]
    or [Path.Match], depending on whether the path is a SQL standard or
    predicate check expression.

# Errors

The path query methods return four types of errors:

  - [exec.ErrExecution]: Errors executing the query, such as array index out
    of bounds and division by zero.
  - [exec.ErrVerbose] Execution errors that can be suppressed by
    [exec.WithSilent]. Wraps [exec.ErrExecution].
  - [exec.ErrInvalid]: Usage errors due to flaws in the implementation,
    indicating a bug that needs fixing. Should be rare.
  - [exec.NULL]: Special error value returned by [Path.Exists] and [Path.Match]
    when the result is unknown.

In addition, when [context.Context.Done] is closed in the context passed to a
query function, the query will cease operation and return an
[exec.ErrExecution] that wraps the [context.Canceled] and
[context.DeadlineExceeded] error returned from [context.Context.Err].

# Examples
*/
package path

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"

	"github.com/theory/sqljson/path/ast"
	"github.com/theory/sqljson/path/exec"
	"github.com/theory/sqljson/path/parser"
	"github.com/theory/sqljson/path/types"
)

// Path provides SQL/JSON Path operations.
type Path struct {
	*ast.AST
}

// This is only here so we can import types and the documentation links work
// properly.
var _ types.DateTime = (*types.Time)(nil)

var (
	// ErrPath wraps parsing and execution errors.
	ErrPath = errors.New("path")

	// ErrScan wraps scanning errors.
	ErrScan = errors.New("scan")
)

// Parse parses path and returns the resulting Path. Returns an error on parse
// failure. Returns an [ErrPath] error on parse failure (wraps
// [parser.ErrParse]).
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

// MustQuery is syntax sugar for
// MustParse(path).MustQuery(context.Background(), json). Provided mainly for
// use in documentation examples.
func MustQuery(path string, json any, opt ...exec.Option) any {
	return MustParse(path).MustQuery(context.Background(), json, opt...)
}

// New creates and returns a new Path query defined by ast. Use [parser.Parse]
// to create ast.
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
// only with SQL-standard JSON path expressions (when [Path.IsPredicate]
// returns false), not predicate check expressions (when [Path.IsPredicate]
// returns true), which always return a value.)
//
// While the PostgreSQL jsonb_path_exists() function can return true, false,
// or null (thanks to SQL's [three-valued logic]), Exists cannot return NULL
// when the result is unknown. In such cases, Exists returns false and also
// the [exec.NULL] error value. It's a good idea to check for this error
// explicitly when the result is likely to be unknown.
//
// See the Options section for details on the optional [exec.WithVars],
// [exec.WithTZ], and [exec.WithSilent] options.
//
// [three-valued logic]: https://en.wikipedia.org/wiki/Three-valued_logic
func (path *Path) Exists(ctx context.Context, json any, opt ...exec.Option) (bool, error) {
	//nolint:wrapcheck // Okay to return unwrapped error
	return exec.Exists(ctx, path.AST, json, opt...)
}

// Match returns the result of predicate check for json. (This is useful only
// with predicate check expressions, not SQL-standard JSON path expressions
// (when [Path.IsPredicate] returns false), since it will either fail or
// return nil if the path result is not a single boolean value.)
//
// While the PostgreSQL jsonb_path_match() function can return true, false, or
// null (thanks to SQL's [three-valued logic]), Match cannot return NULL when
// the result is unknown. In such cases, Match returns false and also the
// [exec.NULL] error value. It's a good idea to check for this error
// explicitly when the result is likely to be unknown.
//
// See the Options section for details on the optional [exec.WithVars],
// [exec.WithTZ], and [exec.WithSilent] options.
func (path *Path) Match(ctx context.Context, json any, opt ...exec.Option) (bool, error) {
	//nolint:wrapcheck // Okay to return unwrapped error
	return exec.Match(ctx, path.AST, json, opt...)
}

// ExistsOrMatch dispatches SQL standard path expressions to [Exists] and
// predicate check expressions to [Match], reducing the need to know which to
// call. Results and options are the same as for those methods.
func (path *Path) ExistsOrMatch(ctx context.Context, json any, opt ...exec.Option) (bool, error) {
	//nolint:wrapcheck // Okay to return unwrapped error
	if path.IsPredicate() {
		return exec.Match(ctx, path.AST, json, opt...)
	}
	//nolint:wrapcheck // Okay to return unwrapped error
	return exec.Exists(ctx, path.AST, json, opt...)
}

// Query returns all JSON items returned by path for json. For SQL-standard
// JSON path expressions (when [Path.IsPredicate] returns false) it returns
// the values selected from json. For predicate check expressions (when
// [Path.IsPredicate] returns true) it returns the result of the predicate
// check: true, false, or nil (for an unknown result).
//
// See the Options section for details on the optional [exec.WithVars],
// [exec.WithTZ], and [exec.WithSilent] options.
func (path *Path) Query(ctx context.Context, json any, opt ...exec.Option) (any, error) {
	//nolint:wrapcheck // Okay to return unwrapped error
	return exec.Query(ctx, path.AST, json, opt...)
}

// MustQuery is like [Query], but panics on error. Mostly provided mainly for
// use in documentation examples.
func (path *Path) MustQuery(ctx context.Context, json any, opt ...exec.Option) any {
	res, err := exec.Query(ctx, path.AST, json, opt...)
	if err != nil {
		panic(err)
	}
	return res
}

// First is like [Query], but returns the first JSON item returned by path for
// json, or nil if there are no results. See the Options section for details
// on the optional [exec.WithVars], [exec.WithTZ], and [exec.WithSilent]
// options.
func (path *Path) First(ctx context.Context, json any, opt ...exec.Option) (any, error) {
	//nolint:wrapcheck // Okay to return unwrapped error
	return exec.First(ctx, path.AST, json, opt...)
}

// Scan implements sql.Scanner so Paths can be read from databases
// transparently. Currently, database types that map to string and []byte are
// supported. Please consult database-specific driver documentation for
// matching types. Returns [ErrScan] on scan failure (and may wrap
// [parser.ErrParse]).
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
func (path *Path) Value() (driver.Value, error) {
	return path.String(), nil
}

// MarshalText implements encoding.TextMarshaler.
func (path *Path) MarshalText() ([]byte, error) {
	return path.MarshalBinary()
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (path *Path) UnmarshalText(data []byte) error {
	return path.UnmarshalBinary(data)
}

// MarshalBinary implements encoding.BinaryMarshaler.
func (path *Path) MarshalBinary() ([]byte, error) {
	return []byte(path.String()), nil
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler. Returns [ErrScan] on
// scan failure (wraps [parser.ErrParse]).
func (path *Path) UnmarshalBinary(data []byte) error {
	ast, err := parser.Parse(string(data))
	if err != nil {
		return fmt.Errorf("%w: %w", ErrScan, err)
	}
	*path = Path{ast}
	return nil
}
