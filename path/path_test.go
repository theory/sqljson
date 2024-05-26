package path

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/theory/sqljson/path/exec"
	"github.com/theory/sqljson/path/parser"
)

func TestPath(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	r := require.New(t)
	jMap := map[string]any{"foo": int64(1)}
	ctx := context.Background()

	type testCase struct {
		name string
		path string
		op   string
		json any
		exp  []any
	}

	checkPath := func(tc testCase, path *Path) {
		a.NotNil(path)
		a.NotNil(path.AST)
		a.Equal(path.AST.String(), path.String())
		a.Equal(path.AST.IsPredicate(), path.IsPredicate())
		a.Equal(tc.op, path.PgIndexOperator())

		ok, err := path.Exists(ctx, tc.json, exec.WithSilent())
		r.NoError(err)
		a.True(ok)

		res, err := path.Query(ctx, tc.json)
		r.NoError(err)
		a.Equal(tc.exp, res)
		a.NotPanics(func() { res = path.MustQuery(ctx, tc.json) })
		a.Equal(tc.exp, res)

		res, err = path.First(ctx, tc.json)
		r.NoError(err)
		a.Equal(tc.exp[0], res)

		if _, ok := tc.exp[0].(bool); ok {
			res, err = path.Match(ctx, tc.json)
			r.NoError(err)
			a.Equal(true, res)
		}
	}

	for _, tc := range []testCase{
		{
			name: "root",
			path: "$",
			op:   "@?",
			json: jMap,
			exp:  []any{jMap},
		},
		{
			name: "predicate",
			path: "$ == 1",
			op:   "@@",
			json: int64(1),
			exp:  []any{true},
		},
		{
			name: "filter",
			path: "$.a.b ?(@.x >= 42)",
			op:   "@?",
			json: map[string]any{"a": map[string]any{"b": map[string]any{"x": int64(42)}}},
			exp:  []any{map[string]any{"x": int64(42)}},
		},
		{
			name: "exists",
			path: "exists($.a.b ?(@.x >= 42))",
			op:   "@@",
			json: map[string]any{"a": map[string]any{"b": map[string]any{"x": int64(42)}}},
			exp:  []any{true},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Test Parse
			path, err := Parse(tc.path)
			r.NoError(err)
			checkPath(tc, path)

			// Test MustParse
			r.NotPanics(func() { path = MustParse(tc.path) })
			checkPath(tc, path)

			// Test New
			checkPath(tc, New(path.AST))

			// Test text Marshaling
			text, err := path.MarshalText()
			r.NoError(err)
			a.Equal(text, []byte(path.AST.String()))
			var txtPath Path
			r.NoError(txtPath.UnmarshalText(text))
			checkPath(tc, &txtPath)

			// Test binary marshaling
			bin, err := path.MarshalBinary()
			r.NoError(err)
			a.Equal(bin, []byte(path.AST.String()))
			var binPath Path
			r.NoError(binPath.UnmarshalBinary(bin))
			checkPath(tc, &binPath)

			// Test SQL marshaling
			val, err := path.Value()
			r.NoError(err)
			a.IsType("", val)
			a.Equal(path.String(), val)
			sqlPath := new(Path)
			r.NoError(sqlPath.Scan(val))
			checkPath(tc, sqlPath)

			// Test SQL binary unmarshaling
			str, ok := val.(string)
			r.True(ok)
			sqlPath = new(Path)
			r.NoError(sqlPath.Scan([]byte(str)))
			checkPath(tc, sqlPath)
		})
	}
}

func TestQueryErrors(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	r := require.New(t)
	for _, tc := range []struct {
		name string
		path string
		json any
		err  string
	}{
		{
			name: "out_of_bounds",
			path: "strict $[1]",
			json: []any{true},
			err:  "exec: jsonpath array subscript is out of bounds",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			path, err := Parse(tc.path)
			r.NoError(err)

			// Test Query
			res, err := path.Query(context.Background(), tc.json)
			r.EqualError(err, tc.err)
			r.ErrorIs(err, exec.ErrExecution)
			a.Nil(res)

			// Test First
			first, err := path.First(context.Background(), tc.json)
			r.EqualError(err, tc.err)
			r.ErrorIs(err, exec.ErrExecution)
			a.Nil(first)

			// Test MustQuery
			a.PanicsWithError(tc.err, func() {
				path.MustQuery(context.Background(), tc.json)
			})

			// Test Match
			ok, err := path.Match(context.Background(), tc.json)
			r.EqualError(err, tc.err)
			r.ErrorIs(err, exec.ErrExecution)
			a.False(ok)

			// Test Exists
			ok, err = path.Exists(context.Background(), tc.json)
			r.EqualError(err, tc.err)
			r.ErrorIs(err, exec.ErrExecution)
			a.False(ok)
		})
	}
}

func TestPathParseErrors(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	r := require.New(t)

	for _, tc := range []struct {
		name string
		path string
		err  string
	}{
		{
			name: "parse_error",
			path: "(.)",
			err:  "parser: syntax error at 1:3",
		},
		{
			name: "validation_error",
			path: "@ == 1",
			err:  "parser: @ is not allowed in root expressions",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Test Parse
			path, err := Parse(tc.path)
			r.EqualError(err, "path: "+tc.err)
			r.ErrorIs(err, ErrPath)
			a.Nil(path)

			// Test MustParse
			a.PanicsWithError(tc.err, func() { MustParse(tc.path) })

			// Test UnmarshalBinary
			scanErr := "scan: " + tc.err
			newPath := &Path{}
			err = newPath.UnmarshalBinary([]byte(tc.path))
			r.EqualError(err, scanErr)
			r.ErrorIs(err, ErrScan)
			r.ErrorIs(err, parser.ErrParse)
			a.Nil(newPath.AST)

			// Test UnmarshalText
			err = newPath.UnmarshalText([]byte(tc.path))
			r.EqualError(err, scanErr)
			r.ErrorIs(err, ErrScan)
			r.ErrorIs(err, parser.ErrParse)
			a.Nil(newPath.AST)

			// Test Scan Text
			err = newPath.Scan(tc.path)
			r.EqualError(err, scanErr)
			r.ErrorIs(err, ErrScan)
			r.ErrorIs(err, parser.ErrParse)
			a.Nil(newPath.AST)

			// Test Scan Binary
			err = newPath.Scan([]byte(tc.path))
			r.EqualError(err, scanErr)
			r.ErrorIs(err, ErrScan)
			r.ErrorIs(err, parser.ErrParse)
			a.Nil(newPath.AST)
		})
	}
}

func TestScanNilPath(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	r := require.New(t)

	for _, tc := range []struct {
		name string
		path any
	}{
		{"nil", nil},
		{"empty_string", ""},
		{"no_bytes", []byte{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			newPath := &Path{}
			r.NoError(newPath.Scan(tc.path))
			a.Nil(newPath.AST)
		})
	}

	t.Run("unknown_type", func(t *testing.T) {
		t.Parallel()
		newPath := &Path{}
		err := newPath.Scan(42)
		r.EqualError(err, "scan: unable to scan type int into Path")
		r.ErrorIs(err, ErrScan)
	})
}
