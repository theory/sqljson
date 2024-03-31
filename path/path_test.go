package path

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/theory/sqljson/path/parser"
)

func TestPath(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	r := require.New(t)
	jMap := map[string]any{"foo": 1}

	checkPath := func(op string, path *Path) {
		t.Helper()
		a.NotNil(path)
		a.NotNil(path.AST)
		a.Equal(path.AST.String(), path.String())
		a.Equal(path.AST.IsPredicate(), path.IsPredicate())
		a.Equal(op, path.PgIndexOperator())

		ok, err := path.Exists(nil, nil, false)
		r.NoError(err)
		a.True(ok)

		res, err := path.Query(jMap, nil, false)
		r.NoError(err)
		a.Equal(jMap, res)
		a.NotPanics(func() { res = path.MustQuery(jMap, nil, false) })
		a.Equal(jMap, res)
	}

	for _, tc := range []struct {
		name string
		path string
		op   string
	}{
		{
			name: "root",
			path: "$",
			op:   "@?",
		},
		{
			name: "predicate",
			path: "$ == 1",
			op:   "@@",
		},
		{
			name: "filter",
			path: "$.a.b ?(@.x >= 42)",
			op:   "@?",
		},
		{
			name: "exists",
			path: "exists($.a.b ?(@.x >= 42))",
			op:   "@@",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Test Parse
			path, err := Parse(tc.path)
			r.NoError(err)
			checkPath(tc.op, path)

			// Test MustParse
			r.NotPanics(func() { path = MustParse(tc.path) })
			checkPath(tc.op, path)

			// Test New
			checkPath(tc.op, New(path.AST))

			// Test text Marshaling
			text, err := path.MarshalText()
			r.NoError(err)
			a.Equal(text, []byte(path.AST.String()))
			var txtPath Path
			r.NoError(txtPath.UnmarshalText(text))
			checkPath(tc.op, &txtPath)

			// Test binary marshaling
			bin, err := path.MarshalBinary()
			r.NoError(err)
			a.Equal(bin, []byte(path.AST.String()))
			var binPath Path
			r.NoError(binPath.UnmarshalBinary(bin))
			checkPath(tc.op, &binPath)

			// Test SQL marshaling
			val, err := path.Value()
			r.NoError(err)
			a.IsType("", val)
			a.Equal(path.String(), val)
			sqlPath := new(Path)
			r.NoError(sqlPath.Scan(val))
			checkPath(tc.op, sqlPath)

			// Test SQL binary unmarshaling
			str, ok := val.(string)
			r.True(ok)
			sqlPath = new(Path)
			r.NoError(sqlPath.Scan([]byte(str)))
			checkPath(tc.op, sqlPath)
		})
	}
}

func TestPathErrors(t *testing.T) {
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
			err:  "parser: syntax error at path:1:3",
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
