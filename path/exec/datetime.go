package exec

import (
	"context"
	"errors"
	"fmt"

	"github.com/theory/sqljson/path/ast"
	"github.com/theory/sqljson/path/types"
)

// tzRequiredCast constructs an error reporting that type1 cannot be cast to
// type2 without time zone usage.
func tzRequiredCast(type1, type2 string) error {
	return fmt.Errorf(
		"%w: cannot convert value from %v to %v without time zone usage. HINT: Use WithTZ() option for time zone support",
		ErrExecution, type1, type2,
	)
}

// unknownDateTime returns 0 and an error reporting that val is not a known
// datetime type.
func unknownDateTime(val any) (int, error) {
	return 0, fmt.Errorf(
		"%w: unrecognized SQL/JSON datetime type %T",
		ErrInvalid, val,
	)
}

// compareDatetime performs a Cross-type comparison of two datetime SQL/JSON
// items. Returns <= -1 if items are incomparable. Returns an error if a cast
// requires timezone useTZ is false.
func compareDatetime(val1, val2 any, useTZ bool) (int, error) {
	switch val1 := val1.(type) {
	case *types.Date:
		return compareDate(val1, val2, useTZ)
	case *types.Time:
		return compareTime(val1, val2, useTZ)
	case *types.TimeTZ:
		return compareTimeTZ(val1, val2, useTZ)
	case *types.Timestamp:
		return compareTimestamp(val1, val2, useTZ)
	case *types.TimestampTZ:
		return compareTimestampTZ(val1, val2, useTZ)
	default:
		return unknownDateTime(val1)
	}
}

// compareDate compares val1 to val1. Returns -2 if they're incomparable and
// an error if a cast requires timezone useTZ is false.
func compareDate(val1 *types.Date, val2 any, useTZ bool) (int, error) {
	switch val2 := val2.(type) {
	case *types.Date:
		return val1.Compare(val2.Time), nil
	case *types.Timestamp:
		return val1.Compare(val2.Time), nil
	case *types.TimestampTZ:
		if !useTZ {
			return 0, tzRequiredCast("date", "timestamptz")
		}
		return val1.Compare(val2.Time.UTC()), nil
	case *types.Time, *types.TimeTZ:
		// Incomparable types
		return -2, nil
	default:
		return unknownDateTime(val2)
	}
}

// compareTime compares val1 to val1. Returns -2 if they're incomparable and
// an error if a cast requires timezone useTZ is false.
func compareTime(val1 *types.Time, val2 any, useTZ bool) (int, error) {
	switch val2 := val2.(type) {
	case *types.Time:
		return val1.Compare(val2.Time), nil
	case *types.TimeTZ:
		if !useTZ {
			return 0, tzRequiredCast("time", "timetz")
		}
		return types.NewTimeTZ(val1.Time).Compare(val2.Time), nil

	case *types.Date, *types.Timestamp, *types.TimestampTZ:
		// Incomparable types
		return -2, nil
	default:
		return unknownDateTime(val2)
	}
}

// compareTimeTZ compares val1 to val1. Returns -2 if they're incomparable and
// an error if a cast requires timezone useTZ is false.
func compareTimeTZ(val1 *types.TimeTZ, val2 any, useTZ bool) (int, error) {
	switch val2 := val2.(type) {
	case *types.Time:
		if !useTZ {
			return 0, tzRequiredCast("time", "timetz")
		}
		return val1.Compare(val2.Time), nil
	case *types.TimeTZ:
		return val1.Compare(val2.Time), nil
	case *types.Date, *types.Timestamp, *types.TimestampTZ:
		// Incomparable types
		return -2, nil
	default:
		return unknownDateTime(val2)
	}
}

// compareTimestamp compares val1 to val1. Returns -2 if they're incomparable
// and an error if a cast requires timezone useTZ is false.
func compareTimestamp(val1 *types.Timestamp, val2 any, useTZ bool) (int, error) {
	switch val2 := val2.(type) {
	case *types.Date:
		return val1.Compare(val2.Time), nil
	case *types.Timestamp:
		return val1.Compare(val2.Time), nil
	case *types.TimestampTZ:
		if !useTZ {
			return 0, tzRequiredCast("timestamp", "timestamptz")
		}
		return val1.Compare(val2.Time.UTC()), nil
	case *types.Time, *types.TimeTZ:
		// Incomparable types
		return -2, nil
	default:
		return unknownDateTime(val2)
	}
}

// compareTimestampTZ compares val1 to val1. Returns -2 if they're
// incomparable and an error if a cast requires timezone useTZ is false.
func compareTimestampTZ(val1 *types.TimestampTZ, val2 any, useTZ bool) (int, error) {
	switch val2 := val2.(type) {
	case *types.Date:
		if !useTZ {
			return 0, tzRequiredCast("date", "timestamptz")
		}
		return val1.Compare(val2.Time.UTC()), nil
	case *types.Timestamp:
		if !useTZ {
			return 0, tzRequiredCast("timestamp", "timestamptz")
		}
		return val1.Compare(val2.Time.UTC()), nil
	case *types.TimestampTZ:
		return val1.Compare(val2.Time), nil
	case *types.Time, *types.TimeTZ:
		// Incomparable types
		return -2, nil
	default:
		return unknownDateTime(val2)
	}
}

// executeDateTimeMethod implements .datetime() and related methods.
//
// Converts a string into a date/time value. The actual type is determined at
// run time. If an argument is provided to .datetime(), it  should be used as
// the template to parse the string, but that feature is currently
// unimplemented, so it instead returns an error.
//
// In all other cases, it calls [types.ParseTime], which attempts a number of
// formats fitting ISO, and the first to succeed determines the type.
//
// .time(), .time_tz(), .timestamp(), .timestamp_tz() take an optional time
// precision.
func (exec *Executor) executeDateTimeMethod(
	ctx context.Context,
	node *ast.UnaryNode,
	value any,
	found *valueList,
) (resultStatus, error) {
	op := node.Operator()

	datetime, ok := value.(string)
	if !ok {
		return exec.returnVerboseError(fmt.Errorf(
			"%w: jsonpath item method %v() can only be applied to a string",
			ErrVerbose, op,
		))
	}

	arg := node.Operand()
	var timeVal types.DateTime
	var err error

	// .datetime(template) has an argument, the rest of the methods don't have
	// an argument.  So we handle that separately.
	if op == ast.UnaryDateTime && arg != nil {
		err = exec.parseDateTimeFormat(datetime, arg)
	} else {
		timeVal, err = exec.parseDateTime(op, datetime, arg)
	}
	if err != nil {
		return exec.returnError(err)
	}

	// The parsing above processes the entire input string and returns the
	// best fitted datetime type. So, if this call is for a specific datatype,
	// then we do the conversion here. Return an error for incompatible types.
	switch op {
	case ast.UnaryDateTime:
		// Nothing to do for DATETIME
	case ast.UnaryDate:
		timeVal, err = exec.castDate(datetime, timeVal)
	case ast.UnaryTime:
		timeVal, err = exec.castTime(datetime, timeVal)
	case ast.UnaryTimeTZ:
		timeVal, err = exec.castTimeTZ(datetime, timeVal)
	case ast.UnaryTimestamp:
		timeVal, err = exec.castTimestamp(datetime, timeVal)
	case ast.UnaryTimestampTZ:
		timeVal, err = exec.castTimestampTZ(datetime, timeVal)
	case ast.UnaryExists, ast.UnaryNot, ast.UnaryIsUnknown, ast.UnaryPlus, ast.UnaryMinus, ast.UnaryFilter:
		return statusFailed, fmt.Errorf("%w: unrecognized jsonpath item type: %T", ErrInvalid, op)
	}

	if err != nil {
		return exec.returnError(err)
	}

	next := node.Next()
	if next == nil && found == nil {
		return statusOK, nil
	}

	return exec.executeNextItem(ctx, node, next, timeVal, found)
}

// parseDateTimeFormat parses datetime with arg format and returns the
// resulting [types.DateTime] or an error.
//
// Or it will eventually. Currently it is unimplemented and returns an error.
func (exec *Executor) parseDateTimeFormat(_ string, _ ast.Node) error {
	// func (exec *Executor) parseDateTimeFormat(datetime string, arg ast.Node) (types.DateTime, error) {
	// XXX: Requires a format parser, so defer for now.
	return fmt.Errorf(
		"%w: .datetime(template) is not yet supported",
		ErrExecution,
	)

	// var str *ast.StringNode
	// str, ok := arg.(*ast.StringNode)
	// if !ok {
	// 	return nil, fmt.Errorf(
	// 		"%w: invalid jsonpath item type for .datetime() argument",
	// 		ErrExecution,
	// 	)
	// }
	// timeVal, ok := types.ParseDateTime(str.Text(), datetime)
}

// parseDateTime extracts an optional precision from arg, if it's not nil, the
// passes it along with datetime to [types.ParseTime] to parse datetime and
// apply precision to the resulting [types.DateTime] value.
func (exec *Executor) parseDateTime(op ast.UnaryOperator, datetime string, arg ast.Node) (types.DateTime, error) {
	// Check for optional precision for methods other than .datetime() and
	// .date()
	precision := -1
	if op != ast.UnaryDateTime && op != ast.UnaryDate && arg != nil {
		var err error
		precision, err = getNodeInt32(op.String()+"()", arg, "time precision")
		if err != nil {
			if !errors.Is(err, ErrVerbose) {
				return nil, err
			}
			return nil, err
		}
		const maxTimestampPrecision = 6
		if precision < 0 {
			return nil, fmt.Errorf(
				"%w: time precision of jsonpath item method %v() is invalid",
				ErrVerbose, op,
			)
		}
		if precision > maxTimestampPrecision {
			// pg: issues a warning
			precision = maxTimestampPrecision
		}
	}

	// Parse the value.
	timeVal, ok := types.ParseTime(datetime, precision)
	if !ok {
		return nil, fmt.Errorf(
			`%w: %v format is not recognized: "%v"`,
			ErrVerbose, op.String()[1:], datetime,
		)
	}

	return timeVal, nil
}

// notRecognized creates an error when the format of datetime is not able to
// be parsed into a [types.DateTime].
func notRecognized(op ast.UnaryOperator, datetime string) error {
	return fmt.Errorf(
		`%w: %v format is not recognized: "%v"`,
		ErrVerbose, op.String()[1:], datetime,
	)
}

// castDate casts timeVal to [types.Date]. The datetime param is used in error
// messages.
func (exec *Executor) castDate(datetime string, timeVal types.DateTime) (*types.Date, error) {
	// Convert result type to date
	switch tv := timeVal.(type) {
	case *types.Date:
		// Nothing to do for DATE
		return tv, nil
	case *types.Time, *types.TimeTZ:
		// Incompatible.
		return nil, notRecognized(ast.UnaryDate, datetime)
	case *types.Timestamp:
		return types.NewDate(tv.Time), nil
	case *types.TimestampTZ:
		if !exec.useTZ {
			return nil, tzRequiredCast("timestamptz", "date")
		}
		return types.NewDate(tv.Time.UTC()), nil
	default:
		return nil, fmt.Errorf("%w: type %T not supported", ErrInvalid, tv)
	}
}

// castTime casts timeVal to [types.Time]. The datetime param is used in error
// messages.
func (exec *Executor) castTime(datetime string, timeVal types.DateTime) (*types.Time, error) {
	switch tv := timeVal.(type) {
	case *types.Date:
		return nil, notRecognized(ast.UnaryTime, datetime)
	case *types.Time:
		// Nothing to do for time
		return tv, nil
	case *types.TimeTZ:
		if !exec.useTZ {
			return nil, tzRequiredCast("timetz", "time")
		}
		return types.NewTime(tv.Time), nil
	case *types.Timestamp:
		return types.NewTime(tv.Time), nil
	case *types.TimestampTZ:
		if !exec.useTZ {
			return nil, tzRequiredCast("timestamptz", "time")
		}
		return types.NewTime(tv.Time.UTC()), nil
	default:
		return nil, fmt.Errorf("%w: type %T not supported", ErrInvalid, tv)
	}
}

// castTimeTZ casts timeVal to [types.TimeTZ]. The datetime param is used in
// error messages.
func (exec *Executor) castTimeTZ(datetime string, timeVal types.DateTime) (*types.TimeTZ, error) {
	switch tv := timeVal.(type) {
	case *types.Date, *types.Timestamp:
		return nil, notRecognized(ast.UnaryTimeTZ, datetime)
	case *types.Time:
		if !exec.useTZ {
			return nil, tzRequiredCast("time", "timetz")
		}
		return types.NewTimeTZ(tv.Time.UTC()), nil
	case *types.TimeTZ:
		// Nothing to do for TIMETZ
		return tv, nil
	case *types.TimestampTZ:
		// Retain the offset.
		return types.NewTimeTZ(tv.Time), nil
	default:
		return nil, fmt.Errorf("%w: type %T not supported", ErrInvalid, tv)
	}
}

// castTimestamp casts timeVal to [types.Timestamp]. The datetime param is
// used in error messages.
func (exec *Executor) castTimestamp(datetime string, timeVal types.DateTime) (*types.Timestamp, error) {
	switch tv := timeVal.(type) {
	case *types.Date:
		return types.NewTimestamp(tv.Time), nil
	case *types.Time, *types.TimeTZ:
		return nil, notRecognized(ast.UnaryTimestamp, datetime)
	case *types.Timestamp:
		// Nothing to do for TIMESTAMP
		return tv, nil
	case *types.TimestampTZ:
		if !exec.useTZ {
			return nil, tzRequiredCast("timestamptz", "timestamp")
		}
		return types.NewTimestamp(tv.Time.UTC()), nil
	default:
		return nil, fmt.Errorf("%w: type %T not supported", ErrInvalid, tv)
	}
}

// castTimestampTZ casts timeVal to [types.TimestampTZ]. The datetime param is
// used in error messages.
func (exec *Executor) castTimestampTZ(datetime string, timeVal types.DateTime) (*types.TimestampTZ, error) {
	switch tv := timeVal.(type) {
	case *types.Date:
		if !exec.useTZ {
			return nil, tzRequiredCast("date", "timestamptz")
		}
		return types.NewTimestampTZ(tv.Time), nil
	case *types.Time, *types.TimeTZ:
		return nil, notRecognized(ast.UnaryTimestampTZ, datetime)
	case *types.Timestamp:
		if !exec.useTZ {
			return nil, tzRequiredCast("timestamp", "timestamptz")
		}
		return types.NewTimestampTZ(tv.Time.UTC()), nil
	case *types.TimestampTZ:
		// Nothing to do for TIMESTAMPTZ
		return tv, nil
	default:
		return nil, fmt.Errorf("%w: type %T not supported", ErrInvalid, tv)
	}
}
