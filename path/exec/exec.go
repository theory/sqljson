// Package exec provides the routines for SQL/JSON path execution.
package exec

import (
	"context"
	"errors"
	"fmt"

	"github.com/theory/sqljson/path/ast"
)

// Things to improve or document as different:
//   - .datetime(template)
//   - Write full docs, including examples and notes on incompatibilities
//   - Some time_tz comparisons still not quite right
//   - Allow single-digit tz offsets, e.g., `+1` instead of `+01`
//   - Allow space between seconds and offset
//   - Years > 9999?
//   - Improve .keyvalue() offsets for arrays?
//   - Less accuracy than Postgres NUMERICs: Switch to
//     github.com/shopspring/decimal?
//   - Go regexp package varies from Postgres regex
//   - Implement interfaces to be compatible with the SQL-standard
//     json_exists(), json_query(), and json_value() functions added in Postgres 17.

// Vars represents JSON path variables and their values.
type Vars map[string]any

var (
	// ErrExecution errors denote runtime execution errors.
	ErrExecution = errors.New("exec")

	// ErrVerbose errors are execution errors that can be suppressed by
	// [WithSilent].
	ErrVerbose = fmt.Errorf("%w", ErrExecution)

	// ErrInvalid errors denote invalid or unexpected execution. Generally
	// internal-only.
	ErrInvalid = errors.New("exec invalid")
)

//nolint:revive,gochecknoglobals,stylecheck
var (
	// NULL is returned when Postgres would return NULL from Match and Exists.
	NULL = errors.New("NULL")
)

// resultStatus represents the result of jsonpath expression evaluation.
type resultStatus uint8

const (
	statusOK resultStatus = iota
	statusNotFound
	statusFailed
)

// String returns a string representation of s.
func (s resultStatus) String() string {
	switch s {
	case statusOK:
		return "OK"
	case statusNotFound:
		return "NOT_FOUND"
	case statusFailed:
		return "FAILED"
	default:
		return "UNKNOWN_RESULT_STATUS"
	}
}

// failed returns true when s is statusFailed.
func (s resultStatus) failed() bool {
	return s == statusFailed
}

// valueList holds a list of jsonb values optimized for a single-value list.
type valueList struct {
	list []any
}

// newList creates a valueList with space allocated a single value.
func newList() *valueList {
	return &valueList{list: make([]any, 0, 1)}
}

// isEmpty returns true when vl is empty.
func (vl *valueList) isEmpty() bool {
	return len(vl.list) == 0
}

// append appends val to vl, allocating more space if needed.
func (vl *valueList) append(val any) {
	vl.list = append(vl.list, val)
}

// Executor represents the context for jsonpath execution.
type Executor struct {
	vars                  Vars         // variables to substitute into jsonpath
	root                  any          // for $ evaluation
	current               any          // for @ evaluation
	baseObject            kvBaseObject // "base object" for .keyvalue() evaluation
	lastGeneratedObjectID int          // "id" counter for .keyvalue() evaluation
	innermostArraySize    int          // for LAST array index evaluation
	path                  *ast.AST

	// with "true" structural errors such as absence of required json item or
	// unexpected json item type are ignored
	ignoreStructuralErrors bool

	// with "false" all suppressible errors are suppressed
	verbose bool
	// "true" enables casting between TZ and non-TZ time and timestamp types
	useTZ bool
}

// Option specifies an execution option.
type Option func(*Executor)

// WithVars specifies variables to use during execution.
func WithVars(vars Vars) Option { return func(e *Executor) { e.vars = vars } }

// WithTZ allows casting between TZ and non-TZ time and timestamp types.
func WithTZ() Option { return func(e *Executor) { e.useTZ = true } }

// WithSilent suppresses the following errors: missing object field or array
// element, unexpected JSON item type, datetime and numeric errors. This
// behavior emulates the behavior of the PostgreSQL @? and @@ operators, and
// might be helpful when searching JSON document collections of varying
// structure.
func WithSilent() Option { return func(e *Executor) { e.verbose = false } }

// newExec creates and returns a new Executor.
func newExec(path *ast.AST, opt ...Option) *Executor {
	e := &Executor{
		path:                   path,
		innermostArraySize:     -1,
		ignoreStructuralErrors: path.IsLax(),
		lastGeneratedObjectID:  1, // Reserved for IDs from vars
		verbose:                true,
	}

	for _, o := range opt {
		o(e)
	}
	return e
}

// Query returns all JSON items returned by the JSON path for the specified
// JSON value. For SQL-standard JSON path expressions it returns the JSON
// values selected from target. For predicate check expressions it returns the
// result of the predicate check: true, false, or null (false + ErrNull). The
// optional [WithVars] and [WithSilent] Options act the same as for [Exists].
func Query(ctx context.Context, path *ast.AST, value any, opt ...Option) ([]any, error) {
	exec := newExec(path, opt...)
	// if exec.verbose && exec.path.IsPredicate() {
	// 	return nil, fmt.Errorf(
	// 		"%w: Query expects a SQL standard path expression",
	// 		ErrVerbose,
	// 	)
	// }

	vals, err := exec.execute(ctx, value)
	if err != nil {
		return nil, err
	}
	return vals.list, nil
}

// First returns the first JSON item returned by the JSON path for the
// specified JSON value, or nil if there are no results. The parameters are
// the same as for [Query].
func First(ctx context.Context, path *ast.AST, value any, opt ...Option) (any, error) {
	exec := newExec(path, opt...)
	// if exec.verbose && exec.path.IsPredicate() {
	// 	return nil, fmt.Errorf(
	// 		"%w: First expects a SQL standard path expression",
	// 		ErrVerbose,
	// 	)
	// }

	vals, err := exec.execute(ctx, value)
	if err != nil {
		return nil, err
	}
	if vals.isEmpty() {
		//nolint:nilnil // nil is a valid return value, standing in for JSON null.
		return nil, nil
	}
	return vals.list[0], nil
}

// Exists checks whether the JSON path returns any item for the specified JSON
// value. (This is useful only with SQL-standard JSON path expressions, not
// predicate check expressions, since those always return a value.) If the
// [WithVars] Option is specified its fields provide named values to be
// substituted into the jsonpath expression. If the [WithSilent] Option is
// specified, the function suppresses some errors. If the [WithTZ] Option is
// specified, it allows comparisons of date/time values that require
// timezone-aware conversions. The example below requires interpretation of
// the date-only value 2015-08-02 as a timestamp with time zone, so the result
// depends on the current TimeZone setting:
//
//	Exists(
//		[]any{"2015-08-01 12:00:00-05"},
//		`$[*] ? (@.datetime() < "2015-08-02".datetime())`,
//		WithTZ(),
//	) → true
func Exists(ctx context.Context, path *ast.AST, value any, opt ...Option) (bool, error) {
	exec := newExec(path, opt...)
	// if exec.verbose && exec.path.IsPredicate() {
	// 	return false, fmt.Errorf(
	// 		"%w: Exists expects a SQL standard path expression",
	// 		ErrVerbose,
	// 	)
	// }

	res, err := exec.exists(ctx, value)
	if err != nil {
		return false, err
	}
	if res.failed() {
		return false, NULL
	}
	return res == statusOK, nil
}

// Match returns the result of a JSON path predicate check for the specified
// JSON value. (This is useful only with predicate check expressions, not
// SQL-standard JSON path expressions, since it will either fail or return
// NULL if the path result is not a single boolean value.) The optional
// [WithVars] and [WithSilent] Options act the same as for [Exists].
func Match(ctx context.Context, path *ast.AST, value any, opt ...Option) (bool, error) {
	exec := newExec(path, opt...)
	// if exec.verbose && !exec.path.IsPredicate() {
	// 	return false, fmt.Errorf(
	// 		"%w: Match expects a predicate path expression",
	// 		ErrVerbose,
	// 	)
	// }

	vals, err := exec.execute(ctx, value)
	if err != nil {
		return false, err
	}

	if len(vals.list) == 1 {
		switch val := vals.list[0].(type) {
		case nil:
			return false, NULL
		case bool:
			return val, nil
		}
	}

	if exec.verbose {
		return false, fmt.Errorf(
			"%w: single boolean result is expected",
			ErrVerbose,
		)
	}

	return false, NULL
}

func (exec *Executor) strictAbsenceOfErrors() bool { return exec.path.IsStrict() }
func (exec *Executor) autoUnwrap() bool            { return exec.path.IsLax() }
func (exec *Executor) autoWrap() bool              { return exec.path.IsLax() }

// execute executes exec.path against value, returning selected values or an error.
func (exec *Executor) execute(ctx context.Context, value any) (*valueList, error) {
	exec.root = value
	exec.current = value
	vals := newList()
	_, err := exec.query(ctx, vals, exec.path.Root(), value)
	return vals, err
}

// exists returns true if the path passed to New() returns at least one item
// for json.
func (exec *Executor) exists(ctx context.Context, json any) (resultStatus, error) {
	exec.root = json
	exec.current = json
	return exec.query(ctx, nil, exec.path.Root(), json)
}

// returnVerboseError returns statusFailed and, when exec.verbose is true, it
// also returns err. Otherwise it returns statusFailed and nil. err must be an
// ErrVerbose error.
func (exec *Executor) returnVerboseError(err error) (resultStatus, error) {
	if exec.verbose {
		return statusFailed, err
	}
	return statusFailed, nil
}

// returnError returns statusFailed and, when exec.verbose is true and err is
// an ErrVerbose error, it also returns err. Otherwise it returns statusFailed
// and nil.
func (exec *Executor) returnError(err error) (resultStatus, error) {
	if exec.verbose || !errors.Is(err, ErrVerbose) {
		return statusFailed, err
	}
	return statusFailed, nil
}
