// Package types provides SQL data types for SQL/JSON Path execution.
//
// It makes every effort to duplicate the behavior of PostgreSQL JSONB dates
// and times in particular in order to compatibly execute date and time
// comparisons in SQL/JSON Path expressions.
package types

import "errors"

// ErrSQLType wraps errors returned by the types package.
var ErrSQLType = errors.New("type")

// infinityModifier indicates whether a value is finite or infinite.
type infinityModifier int8

const (
	// Infinity is infinite.
	Infinity infinityModifier = 1
	// Finite is finite.
	Finite infinityModifier = 0
	// NegativeInfinity is infinitely negative.
	NegativeInfinity infinityModifier = -Infinity
)

// String returns the string representation of im.
func (im infinityModifier) String() string {
	switch im {
	case Finite:
		return "finite"
	case Infinity:
		return "infinity"
	case NegativeInfinity:
		return "-infinity"
	default:
		return "invalid"
	}
}
