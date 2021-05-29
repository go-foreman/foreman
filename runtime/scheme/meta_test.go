package scheme

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTypeMeta(t *testing.T) {
	meta := &TypeMeta{
		Kind:  "SomeKind",
		Group: "group",
	}
	gv := meta.GroupKind()
	//other usecases covered by FromAPIVersionAndKind and by ParseGroupVersion tests
	assert.EqualValues(t, GroupKind{
		Group: "group",
		Kind:  "SomeKind",
	}, gv)
}
