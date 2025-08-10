package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDateTime(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		test string
		obj  any
	}{
		{"date", &Date{}},
		{"time", &Time{}},
		{"timetz", &TimeTZ{}},
		{"timestamp", &Timestamp{}},
		{"timestamptz", &TimestampTZ{}},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)

			a.Implements((*DateTime)(nil), tc.obj)
		})
	}
}
