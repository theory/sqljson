package parser

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

//nolint:paralleltest // Setting a global so cannot run in parallel.
func TestGrammarStuff(t *testing.T) {
	a := assert.New(t)

	pathErrorVerbose = true
	t.Cleanup(func() { pathErrorVerbose = false })

	p := &pathParserImpl{char: 42}
	a.Equal(42, p.Lookahead())
	a.Equal("tok-57386", pathTokname(DECIMAL_P))
	a.Equal("TO_P", pathTokname(4))
	a.Equal("state-42", pathStatname(42))

	a.Equal("syntax error: unexpected TO_P", pathErrorMessage(4, 4))
	a.Equal("syntax error: unexpected TO_P", pathErrorMessage(1, 4))
	a.Equal(
		"syntax error: unexpected TO_P, expecting OR_P or AND_P or ')'",
		pathErrorMessage(int(pathPact[0]), 4),
	)

	rx := regexp.MustCompile(`^syntax error: unexpected (?:\w+|'.'|\$[a-z]+|tok-\d+)(?:, expecting .+)?$`)
	for tok := range pathToknames[3:] {
		for state := range pathPact {
			a.Regexp(rx, pathErrorMessage(state, tok))
		}
	}
}
