package exec

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/theory/sqljson/path/parser"
	"github.com/theory/sqljson/path/types"
)

func TestAddrOf(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		test  string
		value any
		noID  bool
	}{
		{
			test:  "map",
			value: map[string]any{"hi": 1},
		},
		{
			test:  "slice",
			value: []any{1, 2},
		},
		{
			test:  "vars",
			value: Vars{"x": true},
		},
		{
			test:  "int",
			value: int64(42),
			noID:  true,
		},
		{
			test:  "bool",
			value: true,
			noID:  true,
		},
		{
			test: "nil",
			noID: true,
		},
		{
			test:  "datetime",
			value: types.NewDate(time.Now()),
			noID:  true,
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)

			ptr := addrOf(tc.value)
			if tc.noID {
				a.Zero(ptr)
			} else {
				a.Equal(ptr, reflect.ValueOf(tc.value).Pointer())
			}
		})
	}
}

// deltaBetween determines the memory distance between collection and one of
// the items it contains. Used to determine keyvalue IDs at runtime because
// the memory distance can vary at runtime, but should be consistent between
// the same two literal values.
func deltaBetween(collection, item any) int64 {
	delta := int64(reflect.ValueOf(item).Pointer() - reflect.ValueOf(collection).Pointer())
	if delta < 0 {
		return -delta
	}
	return delta
}

func TestKVBaseObject(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	const tenTen = int64(10000000000) // 10^10

	// The offset of an array inside a map can very by execution, so calculate
	// it at runtime.
	mapArray := map[string]any{"x": []any{1, 4}}
	mapArrayOff := deltaBetween(mapArray, mapArray["x"])

	for _, tc := range []struct {
		test string
		base any
		path string
		exp  int64
	}{
		{
			test: "sub-map",
			base: map[string]any{"x": map[string]any{"y": 1}},
			path: "$.x",
		},
		{
			test: "sub-sub-map",
			base: map[string]any{"x": map[string]any{"y": map[string]any{"z": 1}}},
			path: "$.x.y",
		},
		{
			test: "sub-array",
			base: mapArray,
			path: "$.x",
			exp:  mapArrayOff,
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)
			r := require.New(t)

			// Use path to fetch the object from base
			path, err := parser.Parse(tc.path)
			r.NoError(err)
			obj, err := First(ctx, path, tc.base)
			r.NoError(err)

			kvBase := kvBaseObject{addr: addrOf(tc.base), id: 0}
			off := kvBase.OffsetOf(obj)

			if tc.exp > 0 {
				// We pre-calculated the id.
				a.Equal(tc.exp, off)
			} else {
				// The ID can vary at runtime (48, 96, 144, 480 are common, but so
				// are much larger numbers), so just make sure it's greater than 0
				// and less than 10000000000.
				a.Positive(off)
				a.Less(off, tenTen)
			}
		})
	}
}

func TestSetTempBaseObject(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	// Set up a base object.
	e := &Executor{baseObject: kvBaseObject{addr: uintptr(90210), id: 4}}

	// Replace it.
	obj := map[string]any{"x": 1}
	done := e.setTempBaseObject(obj, 2)
	a.Equal(reflect.ValueOf(obj).Pointer(), e.baseObject.addr)
	a.Equal(2, e.baseObject.id)

	// Restore the original.
	done()
	a.Equal(uintptr(90210), e.baseObject.addr)
	a.Equal(4, e.baseObject.id)
}

func TestExecuteKeyValueMethod(t *testing.T) {
	t.Parallel()
	// ID can vary at runtime, so figure out the value at runtime.
	vars := Vars{"foo": map[string]any{"x": true, "y": 1}}
	fooID := 10000000000 + deltaBetween(vars, vars["foo"])

	for _, tc := range []execTestCase{
		{
			test: "kv_single",
			path: "$.keyvalue()",
			json: map[string]any{"x": true},
			exp:  []any{map[string]any{"key": "x", "value": true, "id": int64(0)}},
		},
		{
			test: "kv_double",
			path: "$.keyvalue()",
			json: map[string]any{"x": true, "y": "hi"},
			exp: []any{
				map[string]any{"key": "x", "value": true, "id": int64(0)},
				map[string]any{"key": "y", "value": "hi", "id": int64(0)},
			},
			rand: true, // Results can be in any order
		},
		{
			test: "kv_sequence",
			path: "$.keyvalue().keyvalue()",
			json: map[string]any{"x": true, "y": "hi"},
			exp: []any{
				map[string]any{"id": int64(20000000000), "key": "key", "value": "x"},
				map[string]any{"id": int64(20000000000), "key": "value", "value": true},
				map[string]any{"id": int64(20000000000), "key": "id", "value": int64(0)},
				map[string]any{"id": int64(60000000000), "key": "id", "value": int64(0)},
				map[string]any{"id": int64(60000000000), "key": "key", "value": "y"},
				map[string]any{"id": int64(60000000000), "key": "value", "value": "hi"},
			},
			rand: true, // Results can be in any order
		},
		{
			test: "kv_nested",
			path: "$.keyvalue()",
			json: map[string]any{"foo": map[string]any{"x": true, "y": "hi"}},
			exp: []any{
				map[string]any{"id": int64(0), "key": "foo", "value": map[string]any{"x": true, "y": "hi"}},
			},
			rand: true, // Results can be in any order
		},
		{
			test: "kv_nested_sequence",
			path: "$.keyvalue().keyvalue()",
			json: map[string]any{"foo": map[string]any{"x": true, "y": "hi"}},
			exp: []any{
				map[string]any{"id": int64((20000000000)), "key": "id", "value": int64(0)},
				map[string]any{"id": int64(20000000000), "key": "key", "value": "foo"},
				map[string]any{"id": int64(20000000000), "key": "value", "value": map[string]any{"x": true, "y": "hi"}},
			},
			rand: true, // Results can be in any order
		},
		{
			test: "kv_multi_nested_sequence",
			path: "$.keyvalue().keyvalue()",
			json: map[string]any{"foo": map[string]any{"x": true, "y": "hi"}, "bar": 2, "baz": 1},
			exp: []any{
				map[string]any{"id": int64(20000000000), "key": "id", "value": int64(0)},
				map[string]any{"id": int64(20000000000), "key": "key", "value": "bar"},
				map[string]any{"id": int64(20000000000), "key": "value", "value": 2},
				map[string]any{"id": int64(60000000000), "key": "id", "value": int64(0)},
				map[string]any{"id": int64(60000000000), "key": "key", "value": "baz"},
				map[string]any{"id": int64(60000000000), "key": "value", "value": 1},
				map[string]any{"id": int64(100000000000), "key": "id", "value": int64(0)},
				map[string]any{"id": int64(100000000000), "key": "key", "value": "foo"},
				map[string]any{"id": int64(100000000000), "key": "value", "value": map[string]any{"x": true, "y": "hi"}},
			},
			rand: true, // Results can be in any order
		},
		{
			test: "kv_variable",
			path: "$foo.keyvalue()",
			vars: vars,
			json: `""`,
			exp: []any{
				map[string]any{"key": "x", "value": true, "id": fooID},
				map[string]any{"key": "y", "value": 1, "id": fooID},
			},
			rand: true, // Results can be in any order
		},
		{
			test: "kv_empty",
			path: "$.keyvalue()",
			json: map[string]any{},
			exp:  []any{},
		},
		{
			test: "kv_null",
			path: "$.keyvalue()",
			json: nil,
			err:  "exec: jsonpath item method .keyvalue() can only be applied to an object",
			exp:  []any{},
		},
		{
			test: "array_no_unwrap",
			path: "strict $.keyvalue()",
			json: []any{map[string]any{"x": true}},
			err:  "exec: jsonpath item method .keyvalue() can only be applied to an object",
			exp:  []any{},
		},
		{
			test: "next_error",
			path: "$.keyvalue().string()",
			json: map[string]any{"x": []any{}},
			err:  "exec: jsonpath item method .string() can only be applied to a boolean, string, numeric, or datetime value",
			exp:  []any{},
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()

			tc.run(t)
		})
	}
}

func TestExecuteKeyValueMethodUnwrap(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	r := require.New(t)
	ctx := context.Background()

	// Offset of object in a slice is non-determinate, so calculate it at runtime.
	value := []any{map[string]any{"x": true, "y": "hi"}}
	offset := deltaBetween(value, value[0])

	// Run the query; lax mode will unwrap value to execute method on its items.
	path, err := parser.Parse("$.keyvalue()")
	r.NoError(err)
	found, err := Query(ctx, path, value)
	r.NoError(err)
	a.Equal([]any{
		map[string]any{"id": offset, "key": "x", "value": true},
		map[string]any{"id": offset, "key": "y", "value": "hi"},
	}, found)
}
