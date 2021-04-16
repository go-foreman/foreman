package scheme

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTypeMeta(t *testing.T) {
	meta := &TypeMeta{
		Kind: "SomeKind",
		Group: "group",
	}
	gv := meta.GroupKind()
	//other usecases covered by FromAPIVersionAndKind and by ParseGroupVersion tests
	assert.EqualValues(t, GroupKind{
		Group:   "group",
		Kind:    "SomeKind",
	}, gv)
}
