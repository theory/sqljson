package ast

import (
	"errors"
	"fmt"
	"regexp/syntax"
	"strings"
)

// Use golang.org/x/tools/cmd/stringer to generate the String method for the
// regexFlag enums from their inline comments.
//go:generate stringer -linecomment -output regex_string.go -type regexFlag

// regexFlag represents a single JSON Path regex flag.
type regexFlag uint16

// https://github.com/postgres/postgres/blob/REL_18_BETA2/src/include/utils/jsonpath.h#L120-L125
//
//nolint:godot
const (
	// i flag, case insensitive
	regexICase regexFlag = 0x01 // i
	// s flag, dot matches newline
	regexDotAll regexFlag = 0x02 // s
	// m flag, ^/$ match at newlines
	regexMLine regexFlag = 0x04 // m
	// x flag, ignore whitespace in pattern
	regexWSpace regexFlag = 0x08 // x
	// q flag, no special characters
	regexQuote regexFlag = 0x10 // q
)

// regexFlags is a bit mask of regexFlag flags.
type regexFlags uint16

// newRegexFlags parses flags to create a new regexFlags.
func newRegexFlags(flags string) (regexFlags, error) {
	bitMask := regexFlag(0)

	// Parse the flags string, convert to bit mask. Duplicate flags are OK.
	for _, f := range flags {
		switch f {
		case 'i':
			bitMask |= regexICase
		case 's':
			bitMask |= regexDotAll
		case 'm':
			bitMask |= regexMLine
		case 'x':
			bitMask |= regexWSpace
		case 'q':
			bitMask |= regexQuote
		default:
			//nolint:err113,staticcheck
			return 0, fmt.Errorf(
				`Unrecognized flag character "%c" in LIKE_REGEX predicate`,
				f,
			)
		}
	}

	// Validate compatibility with Go flags.
	reFlags := regexFlags(bitMask)
	if _, err := reFlags._syntaxFlags(); err != nil {
		return 0, err
	}

	return reFlags, nil
}

// String returns the flags formatted as a SQL/JSON path 'flags ""' expression.
func (f regexFlags) String() string {
	if f == 0 {
		return ""
	}

	flags := ` flag "`
	bitMask := regexFlag(f)

	var flagsSb79 strings.Builder
	for _, flag := range []regexFlag{regexICase, regexDotAll, regexMLine, regexWSpace, regexQuote} {
		if bitMask&flag > 0 {
			flagsSb79.WriteString(flag.String())
		}
	}
	flags += flagsSb79.String()

	return flags + `"`
}

// convertRegexFlags converts from XQuery regex flags to those recognized by
// regexp/syntax.
func (f regexFlags) syntaxFlags() syntax.Flags {
	synFlags, _ := f._syntaxFlags()
	return synFlags
}

// _syntaxFlags converts from XQuery regex flags to those recognized by
// regexp/syntax. Returns an error for unsupported use of the 'x' flag.
func (f regexFlags) _syntaxFlags() (syntax.Flags, error) {
	cFlags := syntax.OneLine | syntax.ClassNL | syntax.PerlX
	bitMask := regexFlag(f)

	// Ignore case.
	if bitMask&regexICase != 0 {
		cFlags |= syntax.FoldCase
	}

	// Per XQuery spec, if 'q' is specified then 'm', 's', 'x' are ignored
	// https://www.w3.org/TR/xpath-functions-3/#flags
	if bitMask&regexQuote != 0 {
		return cFlags | syntax.Literal, nil
	}

	// From the Postgres source
	// https://github.com/postgres/postgres/blob/REL_18_BETA2/src/backend/utils/adt/jsonpath_gram.y#L669-L675
	//
	// > XQuery's 'x' mode is related to Spencer's expanded mode, but it's
	// > not really enough alike to justify treating JSP_REGEX_WSPACE as
	// > REG_EXPANDED. For now we treat 'x' as unimplemented; perhaps in
	// > future we'll modify the regex library to have an option for
	// > XQuery-style ignore-whitespace mode.
	//
	// Go regexp doesn't appear to support 'x', so we, too, treat it as
	// unimplemented.
	if bitMask&regexWSpace != 0 {
		//nolint:err113
		return 0, errors.New(
			`XQuery "x" flag (expanded regular expressions) is not implemented`,
		)
	}

	if bitMask&regexMLine != 0 {
		cFlags &= ^syntax.OneLine
	}

	if bitMask&regexDotAll != 0 {
		cFlags |= syntax.DotNL
	}

	return cFlags, nil
}

// shouldQuoteMeta returns true if the flags include the 'q' flag, in which case
// all characters in the regular expression are treated as representing
// themselves, not as metacharacters --- that is, if the pattern should be
// escaped with the use of [regexp.QuoteMeta].
func (f regexFlags) shouldQuoteMeta() bool {
	return regexFlag(f)&regexQuote != 0
}

// _syntaxFlags converts from XQuery regex flags to those recognized by
// regexp/syntax. Returns an error for unsupported use of the 'x' flag.
func (f regexFlags) goFlags() string {
	// Start flags with '(?'
	const maxFlagSize = 6
	const startSize = 2
	flags := make([]byte, startSize, maxFlagSize)
	flags[0] = '('
	flags[1] = '?'

	// need to compare same types.
	bitMask := regexFlag(f)

	// Ignore case.
	if bitMask&regexICase != 0 {
		flags = append(flags, 'i')
	}

	// Per XQuery spec, if 'q' is specified then 'm', 's' are ignored
	// https://www.w3.org/TR/xpath-functions-3/#flags
	if bitMask&regexQuote == 0 {
		if bitMask&regexDotAll != 0 {
			flags = append(flags, 's')
		}

		if bitMask&regexMLine != 0 {
			flags = append(flags, 'm')
		}
	}

	if len(flags) == startSize {
		return ""
	}

	return string(append(flags, ')'))
}

// validateRegex validates that regexp/syntax compiles pattern with flags.
func validateRegex(pattern string, flags regexFlags) error {
	// Make sure it parses.
	_, err := syntax.Parse(pattern, flags.syntaxFlags())
	if err != nil {
		//nolint:wrapcheck
		return err
	}

	// (Compile never returns an error, so skip this bit.)
	// Make sure it compiles.
	// _, err = syntax.Compile(re.Simplify())
	// if err != nil {
	// 	return err
	// }

	return nil
}
