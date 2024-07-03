package ast

import (
	"regexp/syntax"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegexFlag(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tc := range []struct {
		flag regexFlag
		val  uint16
		str  string
	}{
		{regexICase, 0x01, "i"},
		{regexDotAll, 0x02, "s"},
		{regexMLine, 0x04, "m"},
		{regexWSpace, 0x08, "x"},
		{regexQuote, 0x10, "q"},
		{regexFlag(999), 999, "regexFlag(999)"},
	} {
		t.Run(tc.str+"_flag", func(t *testing.T) {
			t.Parallel()
			a.Equal(regexFlag(tc.val), tc.flag)
			a.Equal(tc.str, tc.flag.String())
		})
	}
}

func TestRegexFlags(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	r := require.New(t)

	for _, tc := range []struct {
		name string
		expr string
		exp  regexFlags
		str  string
		syn  syntax.Flags
		ref  string
		err  string
	}{
		{
			name: "empty",
			exp:  regexFlags(0),
			syn:  syntax.OneLine | syntax.ClassNL | syntax.PerlX,
		},
		{
			name: "i",
			expr: "i",
			exp:  regexFlags(regexICase),
			str:  ` flag "i"`,
			syn:  syntax.OneLine | syntax.ClassNL | syntax.PerlX | syntax.FoldCase,
			ref:  "(?i)",
		},
		{
			name: "s",
			expr: "s",
			exp:  regexFlags(regexDotAll),
			str:  ` flag "s"`,
			syn:  syntax.OneLine | syntax.ClassNL | syntax.PerlX | syntax.DotNL,
			ref:  "(?s)",
		},
		{
			name: "m",
			expr: "m",
			exp:  regexFlags(regexMLine),
			str:  ` flag "m"`,
			syn:  syntax.ClassNL | syntax.PerlX,
			ref:  "(?m)",
		},
		{
			name: "x",
			expr: "x",
			err:  `XQuery "x" flag (expanded regular expressions) is not implemented`,
		},
		{
			name: "q",
			expr: "q",
			exp:  regexFlags(regexQuote),
			str:  ` flag "q"`,
			syn:  syntax.OneLine | syntax.ClassNL | syntax.PerlX | syntax.Literal,
		},
		{
			name: "q",
			expr: "q",
			exp:  regexFlags(regexQuote),
			str:  ` flag "q"`,
			syn:  syntax.OneLine | syntax.ClassNL | syntax.PerlX | syntax.Literal,
		},
		{
			name: "unknown",
			expr: "y",
			err:  `Unrecognized flag character "y" in LIKE_REGEX predicate`,
		},
		{
			name: "qx",
			expr: "qx",
			exp:  regexFlags(regexQuote | regexWSpace),
			str:  ` flag "xq"`,
			syn:  syntax.OneLine | syntax.ClassNL | syntax.PerlX | syntax.Literal,
		},
		{
			name: "qi",
			expr: "qi",
			exp:  regexFlags(regexQuote | regexICase),
			str:  ` flag "iq"`,
			syn:  syntax.OneLine | syntax.ClassNL | syntax.PerlX | syntax.FoldCase | syntax.Literal,
			ref:  "(?i)",
		},
		{
			name: "qmsx",
			expr: "qmsx",
			exp:  regexFlags(regexQuote | regexMLine | regexDotAll | regexWSpace),
			str:  ` flag "smxq"`,
			syn:  syntax.OneLine | syntax.ClassNL | syntax.PerlX | syntax.Literal,
		},
		{
			name: "msi",
			expr: "msi",
			exp:  regexFlags(regexICase | regexDotAll | regexMLine),
			str:  ` flag "ism"`,
			syn:  syntax.FoldCase | syntax.ClassNL | syntax.PerlX | syntax.DotNL,
			ref:  "(?ism)",
		},
		{
			name: "dupes_okay",
			expr: "msmm",
			exp:  regexFlags(regexMLine | regexDotAll),
			str:  ` flag "sm"`,
			syn:  syntax.DotNL | syntax.ClassNL | syntax.PerlX,
			ref:  "(?sm)",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			flags, err := newRegexFlags(tc.expr)
			a.Equal(tc.exp, flags)
			if tc.err != "" {
				r.EqualError(err, tc.err)
				return
			}
			r.NoError(err)
			a.Equal(tc.str, flags.String())
			a.Equal(tc.syn, flags.syntaxFlags())
			a.Equal(tc.ref, flags.goFlags())
		})
	}
}

func TestValidateRegex(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	for _, tc := range []struct {
		name  string
		re    string
		flags regexFlags
		str   string
		err   string
	}{
		{
			name: "dot",
			re:   ".",
		},
		{
			name:  "case_insensitive",
			re:    "[abc]",
			flags: regexFlags(regexICase),
		},
		{
			name: "digits",
			re:   `\d+`,
		},
		{
			name:  "all_flags_but_x",
			re:    "[abc]",
			flags: regexFlags(regexICase | regexDotAll | regexMLine | regexQuote),
		},
		{
			name: "parse_failure",
			re:   "(oops",
			err:  "error parsing regexp: missing closing ): `(oops`",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := validateRegex(tc.re, tc.flags)
			if tc.err == "" {
				r.NoError(err)
			} else {
				r.EqualError(err, tc.err)
			}
		})
	}
}
