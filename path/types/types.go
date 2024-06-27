// Package types provides PostgresSQL-compatible data types for SQL/JSON Path
// execution.
//
// It makes every effort to duplicate the behavior of PostgreSQL JSONB dates
// and times in particular in order to compatibly execute date and time
// comparisons in SQL/JSON Path expressions.
package types

import (
	"context"
	"errors"
	"time"
)

// ErrSQLType wraps errors returned by the types package.
var ErrSQLType = errors.New("type")

// secondsPerHour contains the number of seconds in an hour (excluding leap
// seconds).
const secondsPerHour = 60 * 60

// DateTime defines the interface for all date and time data types.
type DateTime interface {
	// GoTime returns the underlying time.Time object.
	GoTime() time.Time

	// ToString returns the output appropriate for the jsonpath string()
	// method, where appropriate in the time zone in ctx.
	ToString(ctx context.Context) string
}
