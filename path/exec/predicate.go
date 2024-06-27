package exec

import (
	"context"

	"github.com/theory/sqljson/path/ast"
)

// predOutcome represents the result of jsonpath predicate evaluation.
type predOutcome uint8

const (
	predFalse predOutcome = iota
	predTrue
	predUnknown
)

// String prints a string representation of p. Used for debugging.
func (p predOutcome) String() string {
	switch p {
	case predFalse:
		return "FALSE"
	case predTrue:
		return "TRUE"
	case predUnknown:
		return "UNKNOWN"
	default:
		return "UNKNOWN_PREDICATE_OUTCOME"
	}
}

// predFrom converts book to a predOutcome, returning predTrue if ok is true
// and predFalse if ok is false.
func predFrom(ok bool) predOutcome {
	if ok {
		return predTrue
	}
	return predFalse
}

// predicateCallback defines the interface to carry out a specific type of
// predicate comparison.
type predicateCallback func(ctx context.Context, node ast.Node, left, right any) (predOutcome, error)

// executePredicate executes a unary or binary predicate.
//
// Predicates have existence semantics, because their operands are item
// sequences. Pairs of items from the left and right operand's sequences are
// checked. Returns true only if any pair satisfying the condition is found.
// In strict mode, even if the desired pair has already been found, all pairs
// still need to be examined to check the absence of errors. Returns
// executePredicate (analogous to SQL NULL) if any error occurs.
func (exec *Executor) executePredicate(
	ctx context.Context,
	pred, left, right ast.Node,
	value any,
	unwrapRightArg bool,
	callback predicateCallback,
) (predOutcome, error) {
	hasErr := false
	found := false

	// Left argument is always auto-unwrapped.
	lSeq := newList()
	res, err := exec.executeItemOptUnwrapResultSilent(ctx, left, value, true, lSeq)
	if res == statusFailed {
		return predUnknown, err
	}

	rSeq := newList()
	if right != nil {
		// Right argument is conditionally auto-unwrapped.
		res, err := exec.executeItemOptUnwrapResultSilent(ctx, right, value, unwrapRightArg, rSeq)
		if res == statusFailed {
			return predUnknown, err
		}
	} else {
		// Right arg is nil.
		rSeq.append(nil)
	}

	for _, lVal := range lSeq.list {
		// Loop over right arg sequence.
		for _, rVal := range rSeq.list {
			res, err := callback(ctx, pred, lVal, rVal)
			if err != nil {
				return predUnknown, err
			}
			switch res {
			case predUnknown:
				if exec.strictAbsenceOfErrors() {
					return predUnknown, nil
				}
				hasErr = true
			case predTrue:
				if !exec.strictAbsenceOfErrors() {
					return predTrue, nil
				}
				found = true
			case predFalse:
				// Do nothing
			}
		}
	}

	if found { // possible only in strict mode
		return predTrue, nil
	}

	if hasErr { //  possible only in lax mode
		return predUnknown, nil
	}

	return predFalse, nil
}
