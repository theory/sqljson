package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInfinityModifier(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	for _, tc := range []struct {
		name string
		mod  infinityModifier
	}{
		{"infinity", Infinity},
		{"finite", Finite},
		{"-infinity", NegativeInfinity},
		{"invalid", infinityModifier(-99)},
	} {
		a.Equal(tc.name, tc.mod.String())
	}
}
