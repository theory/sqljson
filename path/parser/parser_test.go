package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/theory/sqljson/path/ast"
)

func TestParser(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	r := require.New(t)

	for _, tc := range []struct {
		name string
		path string
		ast  *ast.AST
		err  string
	}{
		{
			name: "root",
			path: "$",
			ast:  ast.New(false, ast.NewAccessor([]ast.Node{ast.ConstRoot})),
		},
		{
			name: "strict_root",
			path: "strict $",
			ast:  ast.New(true, ast.NewAccessor([]ast.Node{ast.ConstRoot})),
		},
		{
			name: "error",
			path: "$()",
			err:  "parser: syntax error at path:1:3",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			path, err := Parse(tc.path)
			if tc.err == "" {
				a.Equal(tc.ast, path)
			} else {
				r.EqualError(err, tc.err)
				r.ErrorIs(err, ErrParse)
			}
		})
	}
}
