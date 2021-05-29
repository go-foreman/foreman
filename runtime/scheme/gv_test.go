package scheme

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGroup(t *testing.T) {
	t.Run("Group is ok", func(t *testing.T) {
		g := Group("test")
		assert.Equal(t, g.String(), "test")
	})
	t.Run("Group is empty", func(t *testing.T) {
		g := Group("")
		assert.Empty(t, g.String())
		assert.True(t, g.Empty())
	})
}

func TestGroupKind(t *testing.T) {
	t.Run("GK is ok", func(t *testing.T) {
		gk := GroupKind{
			Group: group,
			Kind:  "SomeTest",
		}
		assert.Equal(t, fmt.Sprintf("%s.%s", group, "SomeTest"), gk.String())
		assert.Equal(t, gk.String(), gk.Identifier())
	})

	t.Run("GK is empty", func(t *testing.T) {
		gk := GroupKind{}
		assert.True(t, gk.Empty())
	})

	t.Run("GK has empty group", func(t *testing.T) {
		gk := GroupKind{
			Kind: "SomeTest",
		}
		assert.False(t, gk.Empty())
		assert.Equal(t, "SomeTest", gk.String())
	})
}

func TestFromString(t *testing.T) {
	t.Run("str is valid", func(t *testing.T) {
		gk, err := FromString("test.SomeTest")
		assert.NoError(t, err)
		assert.Equal(t, GroupKind{
			Group: group,
			Kind:  "SomeTest",
		}, gk)
		assert.False(t, gk.Empty())
	})

	t.Run("str is invalid", func(t *testing.T) {
		gk, err := FromString("")
		assert.EqualError(t, err, "error creating GroupKind from ''")
		assert.True(t, gk.Empty())
	})
}
